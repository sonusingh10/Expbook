package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// buildCsv makes n rows over `symbols` securities: every security trades once per day, buying 10 units
// four days out of five and selling 3 on the fifth, so every sell is covered by earlier buys
func buildCsv(n int, symbols int) string {
	var b strings.Builder
	b.WriteString("date,type,symbol,name,security_type,currency,quantity,price,amount,fees,external_id,comment\n")
	start := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < n; i++ {
		day := start.AddDate(0, 0, i/symbols).Format("2006-01-02")
		symbol := fmt.Sprintf("SYM%02d", i%symbols)
		tradeType := "buy"
		qty := "10"

		if (i/symbols)%5 == 4 {
			tradeType = "sell"
			qty = "3"
		}

		fmt.Fprintf(&b, "%s,%s,%s,%s,stock,INR,%s,%d.25,,1,,\n", day, tradeType, symbol, symbol, qty, 100+i%50)
	}

	return b.String()
}

// SC-002: a 500-row CSV imports in well under 10 seconds; SC-003: importing it again creates nothing.
func TestImport500RowsIsFastAndIdempotent(t *testing.T) {
	c := setupInvestmentTestDb(t)
	rows, err := investments.ParseCSV(buildCsv(500, 10), "INR")
	assert.NoError(t, err)

	started := time.Now()
	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, rows, false)
	elapsed := time.Since(started)

	assert.NoError(t, err)
	assert.Equal(t, 500, result.Created, "%+v", result.Failed)
	assert.Less(t, elapsed, 10*time.Second)
	t.Logf("500-row import took %s", elapsed)

	again, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, rows, false)
	assert.NoError(t, err)
	assert.Equal(t, 0, again.Created)
	assert.Equal(t, 500, again.Skipped)
}

// SC-005: the portfolio summary stays under 2 seconds with 5,000 trades.
func TestSummaryWith5000TradesIsFast(t *testing.T) {
	c := setupInvestmentTestDb(t)
	rows, err := investments.ParseCSV(buildCsv(5000, 25), "INR")
	assert.NoError(t, err)

	started := time.Now()
	result, err := InvestmentTrades.ImportTrades(c, testUid, models.InvestmentTradeSourceCsv, rows, false)
	assert.NoError(t, err)
	assert.Equal(t, 5000, result.Created)
	t.Logf("5000-row import took %s", time.Since(started))

	started = time.Now()
	summary, err := InvestmentTrades.BuildSummary(c, testUid, "INR", RatesFromLatest(nil, "INR"))
	elapsed := time.Since(started)

	assert.NoError(t, err)
	assert.Len(t, summary.Holdings, 25)
	assert.Less(t, elapsed, 2*time.Second)
	t.Logf("summary over 5000 trades took %s", elapsed)
}
