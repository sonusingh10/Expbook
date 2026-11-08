package investments

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testRates() Rates {
	// reference = INR; 1 INR = 0.012 USD, 1 INR = 0.011 EUR  (so 1 USD = 83.33 INR)
	return Rates{
		"INR": big.NewRat(1, 1),
		"USD": big.NewRat(12, 1000),
		"EUR": big.NewRat(11, 1000),
	}
}

func input(t *testing.T, id int64, symbol string, typ SecurityType, currency string, price string, trades ...Trade) HoldingInput {
	h, err := Replay(trades)
	assert.NoError(t, err)

	in := HoldingInput{SecurityID: id, Name: symbol, Symbol: symbol, Type: typ, Currency: currency, Holding: h, Trades: trades}

	if price != "" {
		in.Price = scaled(t, price)
		in.PriceTime = day(400)
	}

	return in
}

func TestRatesConvert(t *testing.T) {
	rates := testRates()

	v, ok := rates.Convert(12000, "USD", "INR")
	assert.True(t, ok)
	assert.Equal(t, int64(1000000), v) // 120.00 USD = 10,000.00 INR

	v, ok = rates.Convert(5000, "INR", "INR")
	assert.True(t, ok)
	assert.Equal(t, int64(5000), v)

	_, ok = rates.Convert(5000, "JPY", "INR")
	assert.False(t, ok)
}

func TestBuildSummaryTotalsInBaseCurrency(t *testing.T) {
	inr := input(t, 1, "RELIANCE", SecurityTypeStock, "INR", "120",
		buy(t, 1, 0, "10", "100", 0),
	)
	usd := input(t, 2, "VOO", SecurityTypeETF, "USD", "10",
		buy(t, 2, 0, "10", "8", 0),
	)

	s := BuildSummary([]HoldingInput{inr, usd}, "INR", testRates(), day(365))

	assert.Equal(t, "INR", s.BaseCurrency)
	assert.Len(t, s.Holdings, 2)

	// INR holding: cost 1000.00, value 1200.00
	// USD holding: cost 80.00 USD = 6666.67 INR, value 100.00 USD = 8333.33 INR
	assert.Equal(t, int64(100000+666667), s.Totals.Invested)
	assert.Equal(t, int64(120000+833333), s.Totals.Value)
	assert.Equal(t, s.Totals.Value-s.Totals.Invested, s.Totals.Unrealised)
	assert.Empty(t, s.UnconvertedCurrencies)
	assert.NotNil(t, s.Totals.XIRR)
	assert.Greater(t, *s.Totals.XIRR, 0.0)
}

func TestBuildSummaryAllocationSumsTo100(t *testing.T) {
	a := input(t, 1, "A", SecurityTypeStock, "INR", "1", buy(t, 1, 0, "1", "1", 0))
	b := input(t, 2, "B", SecurityTypeETF, "INR", "1", buy(t, 2, 0, "1", "1", 0))
	c := input(t, 3, "C", SecurityTypeMutualFund, "INR", "1", buy(t, 3, 0, "1", "1", 0))

	s := BuildSummary([]HoldingInput{a, b, c}, "INR", testRates(), day(30))

	total := 0.0

	for _, slice := range s.AllocationByType {
		total += slice.Percent
	}

	assert.InDelta(t, 100.0, total, 1e-9)
	assert.Len(t, s.AllocationByType, 3)

	currencyTotal := 0.0

	for _, slice := range s.AllocationByCurrency {
		currencyTotal += slice.Percent
	}

	assert.InDelta(t, 100.0, currencyTotal, 1e-9)
}

func TestAllocateLargestRemainder(t *testing.T) {
	result := allocate(map[string]int64{"a": 1, "b": 1, "c": 1})

	total := 0.0

	for _, r := range result {
		total += r.Percent
	}

	assert.InDelta(t, 100.0, total, 1e-9)
	assert.Equal(t, []Allocation{{"a", 1, 33.34}, {"b", 1, 33.33}, {"c", 1, 33.33}}, result)

	assert.Empty(t, allocate(map[string]int64{}))
	assert.Empty(t, allocate(map[string]int64{"a": 0}))
}

func TestBuildSummaryMissingPriceIsNotZero(t *testing.T) {
	noPrice := input(t, 1, "NOPRICE", SecurityTypeStock, "INR", "", buy(t, 1, 0, "10", "100", 0))

	s := BuildSummary([]HoldingInput{noPrice}, "INR", testRates(), day(30))

	assert.Len(t, s.Holdings, 1)
	assert.False(t, s.Holdings[0].PriceAvailable)
	assert.Equal(t, WarningNoPrice, s.Holdings[0].Warning)
	assert.Nil(t, s.Holdings[0].UnrealisedPercent)
	assert.Nil(t, s.Holdings[0].XIRR)
	assert.Equal(t, int64(0), s.Totals.Value)
	assert.Equal(t, int64(100000), s.Totals.Invested)
	assert.Nil(t, s.Totals.XIRR, "portfolio XIRR must be unavailable when a held security has no price")
}

func TestBuildSummaryMissingRateIsReported(t *testing.T) {
	jpy := input(t, 1, "TOYOTA", SecurityTypeStock, "JPY", "10", buy(t, 1, 0, "10", "5", 0))
	inr := input(t, 2, "INFY", SecurityTypeStock, "INR", "20", buy(t, 2, 0, "10", "10", 0))

	s := BuildSummary([]HoldingInput{jpy, inr}, "INR", testRates(), day(30))

	assert.Equal(t, []string{"JPY"}, s.UnconvertedCurrencies)
	assert.Equal(t, WarningNoRate, s.Holdings[0].Warning)
	assert.Equal(t, int64(20000), s.Totals.Value, "only the convertible holding counts toward totals")
	assert.Equal(t, int64(10000), s.Totals.Invested)
}

func TestBuildSummaryClosedPositionKeepsRealised(t *testing.T) {
	closed := input(t, 1, "OLD", SecurityTypeStock, "INR", "50",
		buy(t, 1, 0, "10", "10", 0),
		sell(t, 2, 100, "10", "15", 0),
		dividend(3, 50, 1000),
	)

	s := BuildSummary([]HoldingInput{closed}, "INR", testRates(), day(200))

	assert.Equal(t, int64(5000), s.Totals.Realised)
	assert.Equal(t, int64(1000), s.Totals.Dividends)
	assert.Equal(t, int64(0), s.Totals.Value)
	assert.NotNil(t, s.Holdings[0].XIRR, "closed positions have an XIRR from their flows")
	assert.NotNil(t, s.Totals.XIRR)
}

func TestBuildSummaryEmpty(t *testing.T) {
	s := BuildSummary(nil, "INR", testRates(), day(0))

	assert.Empty(t, s.Holdings)
	assert.Nil(t, s.Totals.XIRR)
	assert.Empty(t, s.AllocationByType)
}
