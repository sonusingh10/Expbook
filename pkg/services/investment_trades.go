package services

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// InvestmentTradeService represents investment trade service (Expbook)
type InvestmentTradeService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize an investment trade service singleton instance
var (
	InvestmentTrades = &InvestmentTradeService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

func mapLedgerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, investments.ErrSellExceedsHoldings):
		return errs.ErrInvestmentSellExceedsHoldings
	case errors.Is(err, investments.ErrInvalidTrade):
		return errs.ErrInvestmentTradeValueInvalid
	}

	return err
}

// GetAllTradesByUid returns all non-deleted trades of the user, oldest first
func (s *InvestmentTradeService) GetAllTradesByUid(c core.Context, uid int64) ([]*models.InvestmentTrade, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var trades []*models.InvestmentTrade
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).OrderBy("trade_time asc, trade_id asc").Find(&trades)

	return trades, err
}

// GetTradesBySecurityId returns all non-deleted trades of one security, oldest first
func (s *InvestmentTradeService) GetTradesBySecurityId(c core.Context, uid int64, securityId int64) ([]*models.InvestmentTrade, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if securityId <= 0 {
		return nil, errs.ErrSecurityIdInvalid
	}

	var trades []*models.InvestmentTrade
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=? AND security_id=?", uid, false, securityId).OrderBy("trade_time asc, trade_id asc").Find(&trades)

	return trades, err
}

// GetTradeByTradeId returns one trade of the user
func (s *InvestmentTradeService) GetTradeByTradeId(c core.Context, uid int64, tradeId int64) (*models.InvestmentTrade, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if tradeId <= 0 {
		return nil, errs.ErrInvestmentTradeIdInvalid
	}

	trade := &models.InvestmentTrade{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(tradeId).Where("uid=? AND deleted=?", uid, false).Get(trade)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrInvestmentTradeNotFound
	}

	return trade, nil
}

func loadTradesInSession(sess datastore.DatabaseSession, uid int64, securityId int64) ([]*models.InvestmentTrade, error) {
	var trades []*models.InvestmentTrade
	err := sess.Where("uid=? AND deleted=? AND security_id=?", uid, false, securityId).Find(&trades)

	return trades, err
}

func toLedger(trades []*models.InvestmentTrade, excludeTradeId int64) []investments.Trade {
	ledger := make([]investments.Trade, 0, len(trades))

	for _, t := range trades {
		if t.TradeId == excludeTradeId {
			continue
		}

		ledger = append(ledger, t.ToLedgerTrade())
	}

	return ledger
}

func securityExistsInSession(sess datastore.DatabaseSession, uid int64, securityId int64) error {
	exists, err := sess.Cols("uid", "deleted", "security_id").Where("uid=? AND deleted=? AND security_id=?", uid, false, securityId).Limit(1).Exist(&models.Security{})

	if err != nil {
		return err
	} else if !exists {
		return errs.ErrSecurityNotFound
	}

	return nil
}

// normaliseTrade zeroes the fields that do not apply to the trade type
func normaliseTrade(trade *models.InvestmentTrade) {
	if investments.TradeType(trade.TradeType) == investments.TradeTypeDividend {
		trade.Quantity = 0
		trade.Price = 0
	} else {
		trade.Amount = 0
	}

	if trade.Source == "" {
		trade.Source = models.InvestmentTradeSourceManual
	}
}

// CreateTrade validates the whole history of the security (a sell may never exceed holdings) and saves a new trade
func (s *InvestmentTradeService) CreateTrade(c core.Context, trade *models.InvestmentTrade) error {
	if trade.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	normaliseTrade(trade)

	ledgerTrade := trade.ToLedgerTrade()

	if ledgerTrade.Type < investments.TradeTypeBuy || ledgerTrade.Type > investments.TradeTypeDividend {
		return errs.ErrInvestmentTradeTypeInvalid
	}

	if err := ledgerTrade.Validate(); err != nil {
		return mapLedgerError(err)
	}

	return s.UserDataDB(trade.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		if err := securityExistsInSession(sess, trade.Uid, trade.SecurityId); err != nil {
			return err
		}

		existing, err := loadTradesInSession(sess, trade.Uid, trade.SecurityId)

		if err != nil {
			return err
		}

		trade.TradeId = s.GenerateUuid(uuid.UUID_TYPE_INVESTMENT_TRADE)

		if trade.TradeId < 1 {
			return errs.ErrSystemIsBusy
		}

		candidate := append(toLedger(existing, 0), trade.ToLedgerTrade())

		if _, err := investments.Replay(candidate); err != nil {
			return mapLedgerError(err)
		}

		now := time.Now().Unix()
		trade.Deleted = false
		trade.CreatedUnixTime = now
		trade.UpdatedUnixTime = now

		if trade.ExternalId == "" {
			trade.ExternalId = "m:" + uuidString(trade.TradeId)
		}

		_, err = sess.Insert(trade)

		return err
	})
}

func uuidString(id int64) string {
	return strconv.FormatInt(id, 10)
}

// ModifyTrade validates the affected securities' histories and updates the trade
func (s *InvestmentTradeService) ModifyTrade(c core.Context, trade *models.InvestmentTrade) error {
	if trade.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	normaliseTrade(trade)

	ledgerTrade := trade.ToLedgerTrade()

	if ledgerTrade.Type < investments.TradeTypeBuy || ledgerTrade.Type > investments.TradeTypeDividend {
		return errs.ErrInvestmentTradeTypeInvalid
	}

	if err := ledgerTrade.Validate(); err != nil {
		return mapLedgerError(err)
	}

	trade.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(trade.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		old := &models.InvestmentTrade{}
		has, err := sess.ID(trade.TradeId).Where("uid=? AND deleted=?", trade.Uid, false).Get(old)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrInvestmentTradeNotFound
		}

		if err := securityExistsInSession(sess, trade.Uid, trade.SecurityId); err != nil {
			return err
		}

		if old.SecurityId != trade.SecurityId {
			// the old security must stay valid without this trade
			oldHistory, err := loadTradesInSession(sess, trade.Uid, old.SecurityId)

			if err != nil {
				return err
			}

			if _, err := investments.Replay(toLedger(oldHistory, trade.TradeId)); err != nil {
				return mapLedgerError(err)
			}
		}

		history, err := loadTradesInSession(sess, trade.Uid, trade.SecurityId)

		if err != nil {
			return err
		}

		candidate := append(toLedger(history, trade.TradeId), trade.ToLedgerTrade())

		if _, err := investments.Replay(candidate); err != nil {
			return mapLedgerError(err)
		}

		updatedRows, err := sess.ID(trade.TradeId).Cols("security_id", "trade_type", "trade_time", "quantity", "price", "amount", "fees", "account_id", "comment", "updated_unix_time").Where("uid=? AND deleted=?", trade.Uid, false).Update(trade)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrInvestmentTradeNotFound
		}

		return nil
	})
}

// DeleteTrade soft-deletes a trade if the remaining history is still valid
func (s *InvestmentTradeService) DeleteTrade(c core.Context, uid int64, tradeId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	updateModel := &models.InvestmentTrade{
		Deleted:         true,
		DeletedUnixTime: time.Now().Unix(),
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		old := &models.InvestmentTrade{}
		has, err := sess.ID(tradeId).Where("uid=? AND deleted=?", uid, false).Get(old)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrInvestmentTradeNotFound
		}

		history, err := loadTradesInSession(sess, uid, old.SecurityId)

		if err != nil {
			return err
		}

		if _, err := investments.Replay(toLedger(history, tradeId)); err != nil {
			return mapLedgerError(err)
		}

		deletedRows, err := sess.ID(tradeId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrInvestmentTradeNotFound
		}

		return nil
	})
}

// ImportTrades runs the idempotent import core shared by CSV import and broker sync.
// Rows that fail parsing are passed in with their Error set. With dryRun nothing is written.
func (s *InvestmentTradeService) ImportTrades(c core.Context, uid int64, source string, rows []investments.RowResult, dryRun bool) (*models.InvestmentImportResponse, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var result *models.InvestmentImportResponse

	err := s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		var err error
		result, err = s.importInSession(sess, uid, source, rows, dryRun)

		return err
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

type importCandidate struct {
	trade *investments.ParsedTrade
}

func (s *InvestmentTradeService) importInSession(sess datastore.DatabaseSession, uid int64, source string, rows []investments.RowResult, dryRun bool) (*models.InvestmentImportResponse, error) {
	response := &models.InvestmentImportResponse{DryRun: dryRun, Rows: []*models.InvestmentImportRowResponse{}}
	rowByLine := map[int]*models.InvestmentImportRowResponse{}

	setRow := func(line int, symbol string, status string, message string) {
		row := &models.InvestmentImportRowResponse{Line: line, Symbol: symbol, Status: status, Message: message}
		rowByLine[line] = row
	}

	// existing securities by symbol
	var securities []*models.Security

	if err := sess.Where("uid=? AND deleted=?", uid, false).Find(&securities); err != nil {
		return nil, err
	}

	securityBySymbol := map[string]*models.Security{}

	for _, sec := range securities {
		securityBySymbol[sec.Symbol] = sec
	}

	// existing external ids for this source
	var existingTrades []*models.InvestmentTrade

	if err := sess.Cols("uid", "deleted", "source", "external_id").Where("uid=? AND deleted=? AND source=?", uid, false, source).Find(&existingTrades); err != nil {
		return nil, err
	}

	seen := map[string]bool{}

	for _, t := range existingTrades {
		seen[t.ExternalId] = true
	}

	// pass 1: classify in file order
	candidatesBySymbol := map[string][]*importCandidate{}
	pendingCurrency := map[string]string{} // currency of the first row of a symbol that does not exist yet
	var symbolOrder []string

	for _, row := range rows {
		if row.Trade == nil {
			setRow(row.Line, "", models.InvestmentImportRowError, row.Error)
			continue
		}

		t := row.Trade

		if seen[t.ExternalID] {
			setRow(t.Line, t.Symbol, models.InvestmentImportRowDuplicate, "already imported")
			continue
		}

		expectedCurrency := pendingCurrency[t.Symbol]

		if sec, ok := securityBySymbol[t.Symbol]; ok {
			expectedCurrency = sec.Currency
		}

		if expectedCurrency != "" && expectedCurrency != t.Currency {
			setRow(t.Line, t.Symbol, models.InvestmentImportRowError, "currency "+t.Currency+" differs from the security currency "+expectedCurrency)
			continue
		}

		pendingCurrency[t.Symbol] = t.Currency

		seen[t.ExternalID] = true

		if _, ok := candidatesBySymbol[t.Symbol]; !ok {
			symbolOrder = append(symbolOrder, t.Symbol)
		}

		candidatesBySymbol[t.Symbol] = append(candidatesBySymbol[t.Symbol], &importCandidate{trade: t})
	}

	// pass 2: per security, accept rows chronologically while the history stays valid
	type acceptedRow struct {
		trade    *investments.ParsedTrade
		security *models.Security
	}

	var accepted []*acceptedRow
	newSecurityCount := int64(0)
	placeholderId := int64(0)

	for _, symbol := range symbolOrder {
		candidates := candidatesBySymbol[symbol]

		sort.SliceStable(candidates, func(i, j int) bool {
			a, b := candidates[i].trade, candidates[j].trade

			if a.Time != b.Time {
				return a.Time < b.Time
			}

			return a.Line < b.Line
		})

		security := securityBySymbol[symbol]
		var history []investments.Trade
		isNew := security == nil

		if !isNew {
			stored, err := loadTradesInSession(sess, uid, security.SecurityId)

			if err != nil {
				return nil, err
			}

			history = toLedger(stored, 0)
		}

		acceptedForSymbol := 0

		for _, cand := range candidates {
			t := cand.trade
			ledgerTrade := investments.Trade{
				ID:       -int64(t.Line), // negative ids never clash with stored ids and keep the order deterministic
				Time:     t.Time,
				Type:     t.Type,
				Quantity: t.Quantity,
				Price:    t.Price,
				Amount:   t.Amount,
				Fees:     t.Fees,
			}

			if err := ledgerTrade.Validate(); err != nil {
				setRow(t.Line, symbol, models.InvestmentImportRowError, "trade values are invalid")
				continue
			}

			next := append(append([]investments.Trade{}, history...), ledgerTrade)

			// buys and dividends can never make a history invalid; only sells need the full replay
			if t.Type == investments.TradeTypeSell {
				if _, err := investments.Replay(next); err != nil {
					setRow(t.Line, symbol, models.InvestmentImportRowError, "sell quantity exceeds the quantity held at that date")
					continue
				}
			}

			history = next
			acceptedForSymbol++

			if security == nil && acceptedForSymbol == 1 {
				placeholderId--

				security = &models.Security{
					Uid:          uid,
					Name:         t.Name,
					Symbol:       symbol,
					SecurityType: int32(t.SecurityType),
					Currency:     t.Currency,
					PriceSource:  models.SecurityPriceSourceManual,
				}

				if err := validateSecurityFields(security); err != nil {
					return nil, err
				}

				if !dryRun {
					if err := Securities.createSecurityInSession(sess, security); err != nil {
						return nil, err
					}
				} else {
					security.SecurityId = placeholderId
				}

				newSecurityCount++
				securityBySymbol[symbol] = security
			}

			accepted = append(accepted, &acceptedRow{trade: t, security: security})
			setRow(t.Line, symbol, models.InvestmentImportRowValid, "")
		}
	}

	// pass 3: write
	for _, a := range accepted {
		t := a.trade

		if dryRun {
			continue
		}

		model := &models.InvestmentTrade{
			Uid:        uid,
			SecurityId: a.security.SecurityId,
			TradeType:  int32(t.Type),
			TradeTime:  t.Time,
			Quantity:   t.Quantity,
			Price:      t.Price,
			Amount:     t.Amount,
			Fees:       t.Fees,
			Source:     source,
			ExternalId: t.ExternalID,
			Comment:    t.Comment,
		}

		normaliseTrade(model)
		model.TradeId = s.GenerateUuid(uuid.UUID_TYPE_INVESTMENT_TRADE)

		if model.TradeId < 1 {
			return nil, errs.ErrSystemIsBusy
		}

		now := time.Now().Unix()
		model.CreatedUnixTime = now
		model.UpdatedUnixTime = now

		if _, err := sess.Insert(model); err != nil {
			return nil, err
		}
	}

	// summarise
	lines := make([]int, 0, len(rowByLine))

	for line := range rowByLine {
		lines = append(lines, line)
	}

	sort.Ints(lines)

	for _, line := range lines {
		row := rowByLine[line]
		response.Rows = append(response.Rows, row)

		switch row.Status {
		case models.InvestmentImportRowValid:
			response.Created++
		case models.InvestmentImportRowDuplicate:
			response.Skipped++
		default:
			response.Failed++
		}
	}

	response.CreatedSecurities = int(newSecurityCount)

	return response, nil
}
