package services

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

const testUid = int64(1)

// setupInvestmentTestDb creates a real SQLite database in a temp dir and syncs the investment tables
func setupInvestmentTestDb(t *testing.T) core.Context {
	t.Helper()

	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      filepath.Join(t.TempDir(), "test.db"),
			MaxIdleConnection: 2,
			MaxOpenConnection: 4,
		},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
		UuidServerId:      1,
	}

	assert.NoError(t, datastore.InitializeDataStore(config))
	assert.NoError(t, uuid.InitializeUuidGenerator(config))
	assert.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.Security), new(models.SecurityPrice), new(models.InvestmentTrade)))

	return core.NewNullContext()
}

func newSecurity(t *testing.T, c core.Context, symbol string, currency string) *models.Security {
	t.Helper()

	security := &models.Security{
		Uid:          testUid,
		Name:         symbol + " name",
		Symbol:       symbol,
		SecurityType: int32(investments.SecurityTypeStock),
		Currency:     currency,
	}

	assert.NoError(t, Securities.CreateSecurity(c, security))
	assert.NotZero(t, security.SecurityId)

	return security
}

func newTrade(t *testing.T, security *models.Security, tradeType investments.TradeType, daysAgo int, qty string, price string, fees int64) *models.InvestmentTrade {
	t.Helper()

	quantity, err := investments.ParseScaled(qty)
	assert.NoError(t, err)
	unitPrice, err := investments.ParseScaled(price)
	assert.NoError(t, err)

	return &models.InvestmentTrade{
		Uid:        testUid,
		SecurityId: security.SecurityId,
		TradeType:  int32(tradeType),
		TradeTime:  time.Now().AddDate(0, 0, -daysAgo).Unix(),
		Quantity:   quantity,
		Price:      unitPrice,
		Fees:       fees,
	}
}

func TestSecurityCreateAndDuplicateSymbol(t *testing.T) {
	c := setupInvestmentTestDb(t)

	security := newSecurity(t, c, "abc", "inr")

	stored, err := Securities.GetSecurityBySecurityId(c, testUid, security.SecurityId)
	assert.NoError(t, err)
	assert.Equal(t, "ABC", stored.Symbol, "symbols are stored upper-case")
	assert.Equal(t, "INR", stored.Currency)
	assert.Equal(t, models.SecurityPriceSourceManual, stored.PriceSource)

	duplicate := &models.Security{Uid: testUid, Name: "x", Symbol: " Abc ", SecurityType: 1, Currency: "INR"}
	assert.Equal(t, errs.ErrSecuritySymbolAlreadyExists, Securities.CreateSecurity(c, duplicate))

	invalidType := &models.Security{Uid: testUid, Name: "x", Symbol: "ZZZ", SecurityType: 9, Currency: "INR"}
	assert.Equal(t, errs.ErrSecurityTypeInvalid, Securities.CreateSecurity(c, invalidType))

	invalidSource := &models.Security{Uid: testUid, Name: "x", Symbol: "ZZZ", SecurityType: 1, Currency: "INR", PriceSource: "bogus"}
	assert.Equal(t, errs.ErrSecurityPriceSourceInvalid, Securities.CreateSecurity(c, invalidSource))

	// a provider source defaults its price symbol to the symbol
	provider := &models.Security{Uid: testUid, Name: "p", Symbol: "tcs", SecurityType: 1, Currency: "INR", PriceSource: models.SecurityPriceSourceYahoo}
	assert.NoError(t, Securities.CreateSecurity(c, provider))
	assert.Equal(t, "TCS", provider.PriceSymbol)
}

func TestSecurityModifyAndSymbolConflict(t *testing.T) {
	c := setupInvestmentTestDb(t)

	a := newSecurity(t, c, "AAA", "INR")
	b := newSecurity(t, c, "BBB", "INR")

	b.Symbol = "AAA"
	assert.Equal(t, errs.ErrSecuritySymbolAlreadyExists, Securities.ModifySecurity(c, b))

	b.Symbol = "CCC"
	b.Name = "Renamed"
	assert.NoError(t, Securities.ModifySecurity(c, b))

	stored, err := Securities.GetSecurityBySecurityId(c, testUid, b.SecurityId)
	assert.NoError(t, err)
	assert.Equal(t, "CCC", stored.Symbol)
	assert.Equal(t, "Renamed", stored.Name)

	// modifying A with its own symbol is not a conflict
	assert.NoError(t, Securities.ModifySecurity(c, a))
}

func TestSecurityDeleteBlockedByTrades(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	trade := newTrade(t, security, investments.TradeTypeBuy, 10, "5", "100", 0)
	assert.NoError(t, InvestmentTrades.CreateTrade(c, trade))

	assert.Equal(t, errs.ErrSecurityInUseCannotBeDeleted, Securities.DeleteSecurity(c, testUid, security.SecurityId))

	assert.NoError(t, InvestmentTrades.DeleteTrade(c, testUid, trade.TradeId))
	assert.NoError(t, Securities.DeleteSecurity(c, testUid, security.SecurityId))

	_, err := Securities.GetSecurityBySecurityId(c, testUid, security.SecurityId)
	assert.Equal(t, errs.ErrSecurityNotFound, err)
}

func TestSetLatestPrice(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	price, _ := investments.ParseScaled("123.45")
	assert.NoError(t, Securities.SetLatestPrice(c, testUid, security.SecurityId, price, 0, models.SecurityPriceSourceManual))

	stored, err := Securities.GetSecurityBySecurityId(c, testUid, security.SecurityId)
	assert.NoError(t, err)
	assert.Equal(t, price, stored.LatestPrice)
	assert.NotZero(t, stored.LatestPriceTime)

	assert.Equal(t, errs.ErrSecurityPriceInvalid, Securities.SetLatestPrice(c, testUid, security.SecurityId, 0, 0, "manual"))
	assert.Equal(t, errs.ErrSecurityNotFound, Securities.SetLatestPrice(c, testUid, 999, price, 0, "manual"))

	var history []*models.SecurityPrice
	assert.NoError(t, Securities.UserDataDB(testUid).NewSession(c).Where("uid=?", testUid).Find(&history))
	assert.Len(t, history, 1)
}

func TestTradeCreateSellGuard(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, security, investments.TradeTypeBuy, 20, "10", "100", 500)))

	tooMuch := newTrade(t, security, investments.TradeTypeSell, 10, "11", "130", 0)
	assert.Equal(t, errs.ErrInvestmentSellExceedsHoldings, InvestmentTrades.CreateTrade(c, tooMuch))

	// nothing was stored
	all, err := InvestmentTrades.GetAllTradesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, all, 1)

	// selling before the buy by date is also refused
	early := newTrade(t, security, investments.TradeTypeSell, 30, "1", "130", 0)
	assert.Equal(t, errs.ErrInvestmentSellExceedsHoldings, InvestmentTrades.CreateTrade(c, early))

	ok := newTrade(t, security, investments.TradeTypeSell, 10, "4", "130", 0)
	assert.NoError(t, InvestmentTrades.CreateTrade(c, ok))

	all, err = InvestmentTrades.GetAllTradesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	ledger := toLedger(all, 0)
	holding, err := investments.Replay(ledger)
	assert.NoError(t, err)
	assert.Equal(t, int64(11800), holding.Realised)
}

func TestTradeValidation(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	zeroQty := newTrade(t, security, investments.TradeTypeBuy, 1, "0", "100", 0)
	assert.Equal(t, errs.ErrInvestmentTradeValueInvalid, InvestmentTrades.CreateTrade(c, zeroQty))

	zeroPrice := newTrade(t, security, investments.TradeTypeBuy, 1, "1", "0", 0)
	assert.Equal(t, errs.ErrInvestmentTradeValueInvalid, InvestmentTrades.CreateTrade(c, zeroPrice))

	badDividend := &models.InvestmentTrade{Uid: testUid, SecurityId: security.SecurityId, TradeType: int32(investments.TradeTypeDividend), TradeTime: time.Now().Unix()}
	assert.Equal(t, errs.ErrInvestmentTradeValueInvalid, InvestmentTrades.CreateTrade(c, badDividend))

	badType := &models.InvestmentTrade{Uid: testUid, SecurityId: security.SecurityId, TradeType: 9, TradeTime: time.Now().Unix()}
	assert.Equal(t, errs.ErrInvestmentTradeTypeInvalid, InvestmentTrades.CreateTrade(c, badType))

	unknownSecurity := newTrade(t, security, investments.TradeTypeBuy, 1, "1", "1", 0)
	unknownSecurity.SecurityId = 12345
	assert.Equal(t, errs.ErrSecurityNotFound, InvestmentTrades.CreateTrade(c, unknownSecurity))
}

func TestTradeDividendIgnoresQuantity(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	dividend := &models.InvestmentTrade{
		Uid: testUid, SecurityId: security.SecurityId, TradeType: int32(investments.TradeTypeDividend),
		TradeTime: time.Now().Unix(), Amount: 5000, Quantity: 999, Price: 999,
	}

	assert.NoError(t, InvestmentTrades.CreateTrade(c, dividend))

	stored, err := InvestmentTrades.GetTradeByTradeId(c, testUid, dividend.TradeId)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stored.Quantity)
	assert.Equal(t, int64(0), stored.Price)
	assert.Equal(t, int64(5000), stored.Amount)
}

func TestTradeDeleteBuyThatSupportsSellIsRefused(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	buy := newTrade(t, security, investments.TradeTypeBuy, 20, "10", "100", 0)
	assert.NoError(t, InvestmentTrades.CreateTrade(c, buy))
	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, security, investments.TradeTypeSell, 10, "5", "110", 0)))

	assert.Equal(t, errs.ErrInvestmentSellExceedsHoldings, InvestmentTrades.DeleteTrade(c, testUid, buy.TradeId))

	_, err := InvestmentTrades.GetTradeByTradeId(c, testUid, buy.TradeId)
	assert.NoError(t, err, "refused delete must leave the trade in place")

	assert.Equal(t, errs.ErrInvestmentTradeNotFound, InvestmentTrades.DeleteTrade(c, testUid, 4242))
}

func TestTradeModifyValidatesHistory(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	buy := newTrade(t, security, investments.TradeTypeBuy, 20, "10", "100", 0)
	assert.NoError(t, InvestmentTrades.CreateTrade(c, buy))
	sell := newTrade(t, security, investments.TradeTypeSell, 10, "5", "110", 0)
	assert.NoError(t, InvestmentTrades.CreateTrade(c, sell))

	// shrinking the buy below the sell is refused
	shrunk := *buy
	shrunk.Quantity, _ = investments.ParseScaled("4")
	assert.Equal(t, errs.ErrInvestmentSellExceedsHoldings, InvestmentTrades.ModifyTrade(c, &shrunk))

	// growing the sell beyond holdings is refused
	grown := *sell
	grown.Quantity, _ = investments.ParseScaled("11")
	assert.Equal(t, errs.ErrInvestmentSellExceedsHoldings, InvestmentTrades.ModifyTrade(c, &grown))

	// a valid edit works
	edited := *sell
	edited.Quantity, _ = investments.ParseScaled("6")
	edited.Comment = "adjusted"
	assert.NoError(t, InvestmentTrades.ModifyTrade(c, &edited))

	stored, err := InvestmentTrades.GetTradeByTradeId(c, testUid, sell.TradeId)
	assert.NoError(t, err)
	assert.Equal(t, "adjusted", stored.Comment)
	assert.Equal(t, edited.Quantity, stored.Quantity)

	missing := *sell
	missing.TradeId = 777
	assert.Equal(t, errs.ErrInvestmentTradeNotFound, InvestmentTrades.ModifyTrade(c, &missing))
}

func TestTradeEntryOrderDoesNotMatter(t *testing.T) {
	c := setupInvestmentTestDb(t)
	security := newSecurity(t, c, "ABC", "INR")

	// enter the later sell... no: it must be refused first, then accepted once the earlier buy exists
	buy := newTrade(t, security, investments.TradeTypeBuy, 20, "10", "100", 0)
	sell := newTrade(t, security, investments.TradeTypeSell, 10, "5", "110", 0)

	assert.Equal(t, errs.ErrInvestmentSellExceedsHoldings, InvestmentTrades.CreateTrade(c, sell))
	assert.NoError(t, InvestmentTrades.CreateTrade(c, buy))

	sell2 := newTrade(t, security, investments.TradeTypeSell, 10, "5", "110", 0)
	assert.NoError(t, InvestmentTrades.CreateTrade(c, sell2))
}

const importCsv = "date,type,symbol,quantity,price,fees,currency,name,security_type\n" +
	"2024-01-10,buy,INFY,10,1500,20,INR,Infosys,stock\n" +
	"2024-02-10,buy,INFY,5,1600,20,INR,Infosys,stock\n" +
	"2024-03-10,sell,INFY,3,1700,10,INR,Infosys,stock\n" +
	"2024-01-15,buy,VOO,2,400,0,USD,Vanguard S&P 500,etf\n"

func parseImport(t *testing.T, content string) []investments.RowResult {
	t.Helper()

	rows, err := investments.ParseCSV(content, "INR")
	assert.NoError(t, err)

	return rows
}

func TestImportDryRunWritesNothing(t *testing.T) {
	c := setupInvestmentTestDb(t)

	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, importCsv), true)
	assert.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Equal(t, 4, result.Created)
	assert.Equal(t, 2, result.CreatedSecurities)
	assert.Equal(t, 0, result.Failed)

	securities, err := Securities.GetAllSecuritiesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Empty(t, securities)

	trades, err := InvestmentTrades.GetAllTradesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Empty(t, trades)
}

func TestImportTwiceEqualsOnce(t *testing.T) {
	c := setupInvestmentTestDb(t)
	rows := parseImport(t, importCsv)

	first, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, rows, false)
	assert.NoError(t, err)
	assert.Equal(t, 4, first.Created)
	assert.Equal(t, 0, first.Skipped)
	assert.Equal(t, 2, first.CreatedSecurities)

	securities, err := Securities.GetAllSecuritiesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, securities, 2)

	tradesAfterFirst, err := InvestmentTrades.GetAllTradesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, tradesAfterFirst, 4)

	second, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, importCsv), false)
	assert.NoError(t, err)
	assert.Equal(t, 0, second.Created)
	assert.Equal(t, 4, second.Skipped)
	assert.Equal(t, 0, second.CreatedSecurities)

	tradesAfterSecond, err := InvestmentTrades.GetAllTradesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, tradesAfterSecond, 4, "re-importing the same file must not create trades")

	securities, err = Securities.GetAllSecuritiesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, securities, 2)

	// holdings from the import: 10+5-3 = 12 units. Cost of 15 units = 15000.00+20.00+8000.00+20.00 = 23040.00,
	// 12/15 of it remains = 18432.00 = 1843200 minor units
	var infy *models.Security

	for _, sec := range securities {
		if sec.Symbol == "INFY" {
			infy = sec
		}
	}

	assert.NotNil(t, infy)
	stored, err := InvestmentTrades.GetTradesBySecurityId(c, testUid, infy.SecurityId)
	assert.NoError(t, err)

	holding, err := investments.Replay(toLedger(stored, 0))
	assert.NoError(t, err)
	assert.Equal(t, int64(12)*investments.Scale, holding.Quantity)
	assert.Equal(t, int64(1843200), holding.TotalCost)
}

func TestImportDuplicateRowsInsideFile(t *testing.T) {
	c := setupInvestmentTestDb(t)

	content := "date,type,symbol,quantity,price\n" +
		"2024-01-10,buy,AAA,1,10\n" +
		"2024-01-10,buy,AAA,1,10\n"

	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Skipped)
}

func TestImportRejectsBadRowsButKeepsGoodOnes(t *testing.T) {
	c := setupInvestmentTestDb(t)

	content := "date,type,symbol,quantity,price,currency\n" +
		"2024-01-10,buy,AAA,10,10,INR\n" +
		"not-a-date,buy,AAA,1,10,INR\n" + // parse error
		"2024-02-10,sell,AAA,50,12,INR\n" + // sells more than held
		"2024-02-11,sell,AAA,4,12,INR\n" + // fine
		"2024-02-12,buy,AAA,1,10,USD\n" // currency differs from the first row of the same new security

	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(result.Rows))
	assert.Equal(t, 2, result.Created, "the first buy and the valid sell")
	assert.GreaterOrEqual(t, result.Failed, 2)

	statuses := map[int]string{}

	for _, row := range result.Rows {
		statuses[row.Line] = row.Status
	}

	assert.Equal(t, models.InvestmentImportRowValid, statuses[2])
	assert.Equal(t, models.InvestmentImportRowError, statuses[3])
	assert.Equal(t, models.InvestmentImportRowError, statuses[4])
	assert.Equal(t, models.InvestmentImportRowValid, statuses[5])
}

func TestImportSecondFileRespectsExistingHoldings(t *testing.T) {
	c := setupInvestmentTestDb(t)

	first := "date,type,symbol,quantity,price\n2024-01-10,buy,AAA,10,10\n"
	_, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, first), false)
	assert.NoError(t, err)

	second := "date,type,symbol,quantity,price\n" +
		"2024-03-10,sell,AAA,11,10\n" + // exceeds stored holdings
		"2023-12-01,sell,AAA,1,10\n" // before the stored buy

	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, second), false)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 2, result.Failed)
}

func TestImportSourcesAreIndependent(t *testing.T) {
	c := setupInvestmentTestDb(t)

	content := "date,type,symbol,quantity,price,external_id\n2024-01-10,buy,AAA,10,10,ORDER-1\n"

	broker, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceBroker, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, broker.Created)

	again, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceBroker, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, again.Skipped)

	// the same external id coming from csv is a different source
	csvResult, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, csvResult.Created)
}

func TestImportDeletedTradeCanBeReimported(t *testing.T) {
	c := setupInvestmentTestDb(t)

	content := "date,type,symbol,quantity,price\n2024-01-10,buy,AAA,10,10\n"
	_, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, content), false)
	assert.NoError(t, err)

	trades, err := InvestmentTrades.GetAllTradesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, trades, 1)
	assert.NoError(t, InvestmentTrades.DeleteTrade(c, testUid, trades[0].TradeId))

	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Created, "the user removed it, so importing again brings it back")
}

func TestImportCurrencyMismatchWithExistingSecurity(t *testing.T) {
	c := setupInvestmentTestDb(t)
	newSecurity(t, c, "AAA", "INR")

	content := "date,type,symbol,quantity,price,currency\n2024-01-10,buy,AAA,10,10,USD\n"
	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, parseImport(t, content), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Failed)
	assert.Contains(t, result.Rows[0].Message, "currency")
}

func TestSecurityCurrencyLockedOnceTradesExist(t *testing.T) {
	c := setupInvestmentTestDb(t)

	security := newSecurity(t, c, "LOCK", "INR")

	// no trades yet: changing currency is fine
	edit := &models.Security{SecurityId: security.SecurityId, Uid: testUid, Name: "Lock", Symbol: "LOCK", SecurityType: 1, Currency: "USD"}
	assert.NoError(t, Securities.ModifySecurity(c, edit))

	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, security, investments.TradeTypeBuy, 5, "1", "10", 0)))

	edit = &models.Security{SecurityId: security.SecurityId, Uid: testUid, Name: "Lock", Symbol: "LOCK", SecurityType: 1, Currency: "EUR"}
	assert.Equal(t, errs.ErrSecurityCurrencyCannotBeChanged, Securities.ModifySecurity(c, edit))

	// other fields may still change when the currency stays the same
	edit = &models.Security{SecurityId: security.SecurityId, Uid: testUid, Name: "Renamed", Symbol: "LOCK", SecurityType: 1, Currency: "USD"}
	assert.NoError(t, Securities.ModifySecurity(c, edit))
}
