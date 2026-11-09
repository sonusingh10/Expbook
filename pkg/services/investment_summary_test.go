package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func TestRatesFromLatest(t *testing.T) {
	// reference USD: 1 USD = 83 INR = 0.8 EUR
	resp := &models.LatestExchangeRateResponse{
		BaseCurrency: "USD",
		ExchangeRates: models.LatestExchangeRateSlice{
			{Currency: "INR", Rate: "83"},
			{Currency: "EUR", Rate: "0.8"},
			{Currency: "BAD", Rate: "not a number"},
			{Currency: "ZERO", Rate: "0"},
		},
	}

	rates := RatesFromLatest(resp, "INR")
	converted, ok := rates.Convert(100, "USD", "INR")
	assert.True(t, ok)
	assert.Equal(t, int64(8300), converted)

	_, ok = rates.Convert(100, "BAD", "INR")
	assert.False(t, ok, "unparseable rates are dropped, never treated as zero")
	_, ok = rates.Convert(100, "ZERO", "INR")
	assert.False(t, ok)

	// no response: only the fallback currency converts (to itself)
	fallback := RatesFromLatest(nil, "INR")
	_, ok = fallback.Convert(100, "USD", "INR")
	assert.False(t, ok)
	same, ok := fallback.Convert(100, "INR", "INR")
	assert.True(t, ok)
	assert.Equal(t, int64(100), same)
}

func TestBuildSummaryFromDatabase(t *testing.T) {
	c := setupInvestmentTestDb(t)

	inr := newSecurity(t, c, "INFY", "INR")
	usd := newSecurity(t, c, "AAPL", "USD")
	noPrice := newSecurity(t, c, "NOPX", "INR")

	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, inr, investments.TradeTypeBuy, 100, "10", "100", 0)))
	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, usd, investments.TradeTypeBuy, 100, "2", "50", 0)))
	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, noPrice, investments.TradeTypeBuy, 100, "1", "10", 0)))

	price := func(s string) int64 { v, _ := investments.ParseScaled(s); return v }
	assert.NoError(t, Securities.SetLatestPrice(c, testUid, inr.SecurityId, price("120"), 1, "manual"))
	assert.NoError(t, Securities.SetLatestPrice(c, testUid, usd.SecurityId, price("60"), 1, "manual"))

	rates := RatesFromLatest(&models.LatestExchangeRateResponse{
		BaseCurrency:  "INR",
		ExchangeRates: models.LatestExchangeRateSlice{{Currency: "USD", Rate: "0.01"}},
	}, "INR")

	summary, err := InvestmentTrades.BuildSummary(c, testUid, "INR", rates)
	assert.NoError(t, err)
	assert.Len(t, summary.Holdings, 3)

	// INFY: 10 x 120 = 1200.00 INR; AAPL: 2 x 60 = 120.00 USD = 12000.00 INR at 0.01 USD per INR
	assert.Equal(t, int64(1200_00+12000_00), summary.Totals.Value)
	assert.Equal(t, int64(1000_00+10000_00+10_00), summary.Totals.Invested, "cost of unpriced holdings is still invested capital")
	assert.Equal(t, int64(1000_00+10000_00), summary.Totals.PricedCost, "the unrealised percentage must only use the cost of priced holdings")

	resp := models.ToInvestmentSummaryResponse(summary, true)
	warnings := map[string]string{}

	for _, h := range resp.Holdings {
		warnings[h.Symbol] = h.Warning
	}

	assert.Equal(t, investments.WarningNoPrice, warnings["NOPX"])
	assert.Equal(t, "", warnings["INFY"])
}

func TestBuildHistoryFromDatabase(t *testing.T) {
	c := setupInvestmentTestDb(t)

	sec := newSecurity(t, c, "INFY", "INR")
	assert.NoError(t, InvestmentTrades.CreateTrade(c, newTrade(t, sec, investments.TradeTypeBuy, 60, "10", "100", 0)))

	price := func(s string) int64 { v, _ := investments.ParseScaled(s); return v }
	// one price recorded 10 days ago, none before it
	assert.NoError(t, Securities.SetLatestPrice(c, testUid, sec.SecurityId, price("150"), time.Now().AddDate(0, 0, -10).Unix(), "manual"))

	rates := RatesFromLatest(nil, "INR")
	now := time.Now().Unix()
	week := int64(7 * 86400)

	points, missing, firstTrade, err := InvestmentTrades.BuildHistory(c, testUid, "INR", rates, 0, now, week)
	assert.NoError(t, err)
	assert.Empty(t, missing)
	assert.NotZero(t, firstTrade)
	assert.GreaterOrEqual(t, len(points), 8)
	assert.Equal(t, now, points[len(points)-1].Time, "the last point is now")

	first, last := points[0], points[len(points)-1]
	assert.Equal(t, int64(1000_00), first.Invested)
	assert.True(t, first.Estimated, "no price was recorded yet, so cost stands in for value")
	assert.Equal(t, int64(1000_00), first.Value)

	assert.False(t, last.Estimated)
	assert.Equal(t, int64(1500_00), last.Value, "10 units at the recorded price of 150")
	assert.Equal(t, int64(1000_00), last.Invested)

	// a start after the first trade is honoured, and one before it shows zeros
	recent, _, _, err := InvestmentTrades.BuildHistory(c, testUid, "INR", rates, now-14*86400, now, week)
	assert.NoError(t, err)
	assert.Equal(t, now-14*86400, recent[0].Time)
	assert.Len(t, recent, 3)

	early, _, _, err := InvestmentTrades.BuildHistory(c, testUid, "INR", rates, now-100*86400, now, week)
	assert.NoError(t, err)
	assert.Equal(t, now-100*86400, early[0].Time)
	assert.Zero(t, early[0].Invested, "nothing was bought yet")

	prices, err := Securities.GetAllPricesByUid(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, prices, 1)

	_, err = Securities.GetAllPricesByUid(c, 0)
	assert.Error(t, err)

	// no trades: an empty history, not an error
	empty, _, none, err := InvestmentTrades.BuildHistory(c, testUid+1, "INR", rates, 0, now, week)
	assert.NoError(t, err)
	assert.Empty(t, empty)
	assert.Zero(t, none)
}
