package services

import (
	"testing"

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
