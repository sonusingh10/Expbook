package investments

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseCSVValidRows(t *testing.T) {
	content := "date,type,symbol,quantity,price,fees,currency,name,security_type,external_id,note\n" +
		"2024-01-15,Buy,reliance,10,2500.5,20.5,INR,Reliance Industries,stock,,first lot\n" +
		"2024-02-01,SELL,RELIANCE,4,2600,10,,,,TRD-77,\n" +
		"2024-03-01,dividend,RELIANCE,,,,INR,,,,\n"

	// dividend row above lacks an amount -> error; add a real one
	content += "2024-03-02,dividend,RELIANCE,,,,INR,,,,\n"

	results, err := ParseCSV(content, "INR")
	assert.NoError(t, err)
	assert.Len(t, results, 4)

	first := results[0].Trade
	assert.NotNil(t, first)
	assert.Equal(t, "RELIANCE", first.Symbol)
	assert.Equal(t, TradeTypeBuy, first.Type)
	assert.Equal(t, int64(1000000000), first.Quantity)
	assert.Equal(t, int64(250050000000), first.Price)
	assert.Equal(t, int64(2050), first.Fees)
	assert.Equal(t, "Reliance Industries", first.Name)
	assert.Equal(t, SecurityTypeStock, first.SecurityType)
	assert.Equal(t, time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix(), first.Time)
	assert.Equal(t, "first lot", first.Comment)
	assert.Contains(t, first.ExternalID, "h:")

	second := results[1].Trade
	assert.NotNil(t, second)
	assert.Equal(t, TradeTypeSell, second.Type)
	assert.Equal(t, "TRD-77", second.ExternalID)
	assert.Equal(t, "INR", second.Currency, "default currency applies")
	assert.Equal(t, "RELIANCE", second.Name, "name falls back to symbol")

	assert.Nil(t, results[2].Trade)
	assert.Contains(t, results[2].Error, "amount")
	assert.Equal(t, 4, results[2].Line)
}

func TestParseCSVDividendAndDerivedPrice(t *testing.T) {
	content := "Date,Action,Ticker,Units,Amount\n" +
		"2024-01-15,Purchase,VOO,4,520.00\n" +
		"2024-02-15,Dividend,VOO,,12.34\n"

	results, err := ParseCSV(content, "USD")
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	assert.NotNil(t, results[0].Trade)
	assert.Equal(t, int64(13000000000), results[0].Trade.Price, "price derived from amount / quantity")

	assert.NotNil(t, results[1].Trade)
	assert.Equal(t, TradeTypeDividend, results[1].Trade.Type)
	assert.Equal(t, int64(1234), results[1].Trade.Amount)
}

func TestParseCSVRowErrors(t *testing.T) {
	content := "date,type,symbol,quantity,price\n" +
		"15/01/2024,buy,A,1,1\n" + // bad date
		"2024-01-15,hold,A,1,1\n" + // bad type
		"2024-01-15,buy,,1,1\n" + // no symbol
		"2024-01-15,buy,A,0,1\n" + // zero quantity
		"2024-01-15,buy,A,1,-5\n" + // negative price
		"2024-01-15,buy,A,1.123456789,1\n" + // too many decimals
		"2024-01-15,buy,A,1,1,extra\n" + // extra field is tolerated
		"\n" + // blank line is skipped
		"2024-01-15,buy,A,1,\n" // no price
	results, err := ParseCSV(content, "INR")
	assert.NoError(t, err)

	errorsFound := 0
	valid := 0

	for _, r := range results {
		if r.Error != "" {
			errorsFound++
			assert.NotZero(t, r.Line)
		} else {
			valid++
		}
	}

	assert.Equal(t, 7, errorsFound)
	assert.Equal(t, 1, valid)
}

func TestParseCSVThousandsSeparatorRejected(t *testing.T) {
	content := "date,type,symbol,quantity,price\n2024-01-15,buy,A,1,\"1,234.50\"\n"
	results, err := ParseCSV(content, "INR")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotEmpty(t, results[0].Error, "ambiguous number formats must be rejected, not guessed")
}

func TestParseCSVHeaderProblems(t *testing.T) {
	_, err := ParseCSV("", "INR")
	assert.ErrorIs(t, err, ErrCsvEmpty)

	_, err = ParseCSV("   \n  ", "INR")
	assert.ErrorIs(t, err, ErrCsvEmpty)

	_, err = ParseCSV("date,symbol,quantity\n2024-01-01,A,1\n", "INR")
	assert.ErrorIs(t, err, ErrCsvMissingColumns)
	assert.Contains(t, err.Error(), "type")
}

func TestParseCSVBomAndTimestamps(t *testing.T) {
	content := "\ufeffdate,type,symbol,quantity,price\n" +
		"2024-01-15T10:30:00Z,buy,A,1,1\n" +
		"2024-01-16 09:15,buy,B,1,1\n"

	results, err := ParseCSV(content, "INR")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix(), results[0].Trade.Time)
	assert.Equal(t, time.Date(2024, 1, 16, 9, 15, 0, 0, time.UTC).Unix(), results[1].Trade.Time)
}

func TestTradeFingerprintStableAndDistinct(t *testing.T) {
	a := &ParsedTrade{Time: 1, Type: TradeTypeBuy, Symbol: "A", Currency: "INR", Quantity: 1, Price: 2}
	b := &ParsedTrade{Time: 1, Type: TradeTypeBuy, Symbol: "A", Currency: "INR", Quantity: 1, Price: 2}
	c := &ParsedTrade{Time: 1, Type: TradeTypeBuy, Symbol: "A", Currency: "INR", Quantity: 1, Price: 3}

	assert.Equal(t, TradeFingerprint(a), TradeFingerprint(b))
	assert.NotEqual(t, TradeFingerprint(a), TradeFingerprint(c))
	assert.LessOrEqual(t, len(TradeFingerprint(a)), MaxExternalIDLength)
}

func TestParseCSVIdenticalRowsShareFingerprint(t *testing.T) {
	content := "date,type,symbol,quantity,price\n2024-01-15,buy,A,1,1\n2024-01-15,buy,A,1,1\n"
	results, err := ParseCSV(content, "INR")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, results[0].Trade.ExternalID, results[1].Trade.ExternalID)
}
