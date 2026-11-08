package investments

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func scaled(t *testing.T, s string) int64 {
	t.Helper()
	v, err := ParseScaled(s)
	assert.NoError(t, err)
	return v
}

func day(offset int) int64 {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 0, offset).Unix()
}

func buy(t *testing.T, id int64, d int, qty, price string, fees int64) Trade {
	return Trade{ID: id, Time: day(d), Type: TradeTypeBuy, Quantity: scaled(t, qty), Price: scaled(t, price), Fees: fees}
}

func sell(t *testing.T, id int64, d int, qty, price string, fees int64) Trade {
	return Trade{ID: id, Time: day(d), Type: TradeTypeSell, Quantity: scaled(t, qty), Price: scaled(t, price), Fees: fees}
}

func dividend(id int64, d int, amountMinor int64) Trade {
	return Trade{ID: id, Time: day(d), Type: TradeTypeDividend, Amount: amountMinor}
}

// Spec scenario US1: buy 10 @ 100 fee 5, sell 4 @ 130 -> realised 118.00, avg cost 100.50, price 120 -> gain 117.00
func TestReplaySpecScenario(t *testing.T) {
	trades := []Trade{
		buy(t, 1, 0, "10", "100", 500),
	}

	h, err := Replay(trades)
	assert.NoError(t, err)
	assert.Equal(t, scaled(t, "10"), h.Quantity)
	assert.Equal(t, int64(100500), h.TotalCost)
	assert.Equal(t, scaled(t, "100.5"), h.AverageCost)
	assert.Equal(t, int64(0), h.Realised)

	trades = append(trades, sell(t, 2, 10, "4", "130", 0))
	h, err = Replay(trades)
	assert.NoError(t, err)
	assert.Equal(t, scaled(t, "6"), h.Quantity)
	assert.Equal(t, int64(60300), h.TotalCost)
	assert.Equal(t, scaled(t, "100.5"), h.AverageCost)
	assert.Equal(t, int64(11800), h.Realised)

	v := h.Value(scaled(t, "120"))
	assert.Equal(t, int64(72000), v.Value)
	assert.Equal(t, int64(11700), v.Unrealised)
	assert.NotNil(t, v.UnrealisedPercent)
	assert.InDelta(t, 19.4029850, *v.UnrealisedPercent, 0.0001)
}

func TestReplayMultipleBuysAverageCost(t *testing.T) {
	// SIP style: three monthly buys
	h, err := Replay([]Trade{
		buy(t, 1, 0, "10", "100", 0),
		buy(t, 2, 30, "10", "110", 0),
		buy(t, 3, 60, "10", "90", 0),
	})

	assert.NoError(t, err)
	assert.Equal(t, scaled(t, "30"), h.Quantity)
	assert.Equal(t, int64(300000), h.TotalCost)
	assert.Equal(t, scaled(t, "100"), h.AverageCost)
}

func TestReplaySellEverythingResetsCost(t *testing.T) {
	h, err := Replay([]Trade{
		buy(t, 1, 0, "3", "33.33", 0),
		sell(t, 2, 5, "3", "40", 100),
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), h.Quantity)
	assert.Equal(t, int64(0), h.TotalCost)
	assert.Equal(t, int64(0), h.AverageCost)
	// proceeds 12000 - 100 = 11900; cost 9999 -> 1901
	assert.Equal(t, int64(1901), h.Realised)
}

func TestReplayPartialSellsDoNotDrift(t *testing.T) {
	// three equal partial sells from a position whose cost does not divide evenly must leave exactly zero cost
	h, err := Replay([]Trade{
		buy(t, 1, 0, "3", "10.01", 0),
		sell(t, 2, 1, "1", "11", 0),
		sell(t, 3, 2, "1", "11", 0),
		sell(t, 4, 3, "1", "11", 0),
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), h.Quantity)
	assert.Equal(t, int64(0), h.TotalCost)
	// proceeds 3300 - cost 3003 = 297 in total (each leg rounded; allow +-1 minor unit total)
	assert.InDelta(t, 297, h.Realised, 1)
}

func TestReplayRejectsSellMoreThanHeld(t *testing.T) {
	_, err := Replay([]Trade{
		buy(t, 1, 0, "5", "100", 0),
		sell(t, 2, 1, "6", "100", 0),
	})

	assert.ErrorIs(t, err, ErrSellExceedsHoldings)

	var sellErr *SellExceedsError
	assert.ErrorAs(t, err, &sellErr)
	assert.Equal(t, int64(2), sellErr.TradeID)
}

func TestReplayRejectsSellBeforeAnyBuyByDate(t *testing.T) {
	_, err := Replay([]Trade{
		sell(t, 1, 0, "1", "100", 0),
		buy(t, 2, 5, "5", "100", 0),
	})

	assert.ErrorIs(t, err, ErrSellExceedsHoldings)
}

func TestReplaySameTimestampBuyBeforeSell(t *testing.T) {
	h, err := Replay([]Trade{
		sell(t, 1, 0, "5", "100", 0),
		buy(t, 2, 0, "5", "100", 0),
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), h.Quantity)
}

func TestReplayRejectsInvalidTrades(t *testing.T) {
	invalid := []Trade{
		{ID: 1, Time: day(0), Type: TradeTypeBuy, Quantity: 0, Price: scaled(t, "1")},
		{ID: 2, Time: day(0), Type: TradeTypeBuy, Quantity: scaled(t, "1"), Price: 0},
		{ID: 3, Time: day(0), Type: TradeTypeBuy, Quantity: scaled(t, "-1"), Price: scaled(t, "1")},
		{ID: 4, Time: day(0), Type: TradeTypeSell, Quantity: scaled(t, "1"), Price: scaled(t, "1"), Fees: -1},
		{ID: 5, Time: day(0), Type: TradeTypeDividend, Amount: 0},
		{ID: 6, Time: day(0), Type: TradeType(9)},
	}

	for _, trade := range invalid {
		_, err := Replay([]Trade{trade})
		assert.ErrorIs(t, err, ErrInvalidTrade, "trade %d", trade.ID)
	}
}

func TestReplayDividendDoesNotChangeQuantity(t *testing.T) {
	h, err := Replay([]Trade{
		buy(t, 1, 0, "10", "100", 0),
		dividend(2, 30, 5000),
	})

	assert.NoError(t, err)
	assert.Equal(t, scaled(t, "10"), h.Quantity)
	assert.Equal(t, int64(100000), h.TotalCost)
	assert.Equal(t, int64(5000), h.Dividends)
}

func TestReplayOrderIndependent(t *testing.T) {
	base := []Trade{
		buy(t, 1, 0, "10", "100", 100),
		buy(t, 2, 10, "5.5", "104.25", 50),
		sell(t, 3, 20, "3", "120", 70),
		dividend(4, 25, 1234),
		buy(t, 5, 30, "0.12345678", "99.99", 0),
		sell(t, 6, 40, "7", "125.5", 30),
	}

	expected, err := Replay(base)
	assert.NoError(t, err)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 50; i++ {
		shuffled := make([]Trade, len(base))
		copy(shuffled, base)
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })

		actual, err := Replay(shuffled)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	}
}

func TestReplayFractionalUnits(t *testing.T) {
	h, err := Replay([]Trade{
		buy(t, 1, 0, "123.4567", "45.6789", 0),
		sell(t, 2, 10, "23.4567", "50", 0),
	})

	assert.NoError(t, err)
	assert.Equal(t, scaled(t, "100"), h.Quantity)
}

func TestCashFlows(t *testing.T) {
	flows := CashFlows([]Trade{
		buy(t, 1, 0, "10", "100", 500),
		dividend(2, 5, 2500),
		sell(t, 3, 10, "4", "130", 100),
	})

	assert.Len(t, flows, 3)
	assert.Equal(t, -100500.0, flows[0].Amount)
	assert.Equal(t, 2500.0, flows[1].Amount)
	assert.Equal(t, 51900.0, flows[2].Amount)
}

func TestValueWithZeroCost(t *testing.T) {
	v := Holding{Quantity: scaled(t, "1")}.Value(scaled(t, "10"))
	assert.Equal(t, int64(1000), v.Value)
	assert.Nil(t, v.UnrealisedPercent)
}
