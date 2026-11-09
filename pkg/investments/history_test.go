package investments

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHistoryTimes(t *testing.T) {
	assert.Equal(t, []int64{0, 10, 20, 25}, HistoryTimes(0, 25, 10, 100))
	assert.Equal(t, []int64{5}, HistoryTimes(5, 5, 10, 100), "a zero-length range is just its end point")
	assert.Nil(t, HistoryTimes(10, 5, 10, 100), "end before start")
	assert.Nil(t, HistoryTimes(0, 10, 0, 100), "no step")

	wide := HistoryTimes(0, 1_000_000, 1, 50)
	assert.LessOrEqual(t, len(wide), 50, "the step is widened so the point count is capped")
	assert.Equal(t, int64(1_000_000), wide[len(wide)-1])
}

func TestBuildHistoryUsesOnlyTradesAndPricesKnownAtEachDate(t *testing.T) {
	rates := Rates{"INR": big.NewRat(1, 1)}
	in := HistoryInput{
		Currency: "INR",
		Trades: []Trade{
			{ID: 1, Time: 100, Type: TradeTypeBuy, Quantity: scaled(t, "10"), Price: scaled(t, "100")},
			{ID: 2, Time: 300, Type: TradeTypeBuy, Quantity: scaled(t, "10"), Price: scaled(t, "200")},
			{ID: 3, Time: 500, Type: TradeTypeSell, Quantity: scaled(t, "20"), Price: scaled(t, "300")},
		},
		Prices: []PricePoint{{Time: 400, Price: scaled(t, "250")}, {Time: 200, Price: scaled(t, "150")}}, // unsorted on purpose
	}

	points, missing := BuildHistory([]HistoryInput{in}, "INR", rates, []int64{50, 100, 250, 350, 450, 600})
	assert.Empty(t, missing)
	assert.Len(t, points, 6)

	// before the first trade: nothing held
	assert.Equal(t, HistoryPoint{Time: 50}, points[0])

	// at t=100 the buy exists but no price is recorded yet: cost stands in for value, flagged as an estimate
	assert.Equal(t, HistoryPoint{Time: 100, Invested: 1000_00, Value: 1000_00, Estimated: true}, points[1])

	// t=250: price 150 recorded at t=200; the second buy (t=300) must not count yet
	assert.Equal(t, HistoryPoint{Time: 250, Invested: 1000_00, Value: 1500_00}, points[2])

	// t=350: 20 units, cost 1000+2000, still priced at 150
	assert.Equal(t, HistoryPoint{Time: 350, Invested: 3000_00, Value: 3000_00}, points[3])

	// t=450: the price of 250 recorded at t=400 applies
	assert.Equal(t, HistoryPoint{Time: 450, Invested: 3000_00, Value: 5000_00}, points[4])

	// t=600: everything sold, nothing held
	assert.Equal(t, HistoryPoint{Time: 600}, points[5])
}

func TestBuildHistoryConvertsToBaseAndReportsMissingRates(t *testing.T) {
	// 1 EUR = 100 INR = 2 PLN
	rates := Rates{"EUR": big.NewRat(1, 1), "INR": big.NewRat(100, 1), "PLN": big.NewRat(2, 1)}

	inINR := HistoryInput{Currency: "INR", Trades: []Trade{{ID: 1, Time: 1, Type: TradeTypeBuy, Quantity: scaled(t, "1"), Price: scaled(t, "1000")}}, Prices: []PricePoint{{Time: 1, Price: scaled(t, "2000")}}}
	inJPY := HistoryInput{Currency: "JPY", Trades: []Trade{{ID: 2, Time: 1, Type: TradeTypeBuy, Quantity: scaled(t, "1"), Price: scaled(t, "50")}}}

	points, missing := BuildHistory([]HistoryInput{inINR, inJPY}, "PLN", rates, []int64{10})

	// 1000 INR cost = 20 PLN; 2000 INR value = 40 PLN; JPY has no rate and is left out, not counted as zero silently
	assert.Equal(t, int64(20_00), points[0].Invested)
	assert.Equal(t, int64(40_00), points[0].Value)
	assert.False(t, points[0].Estimated)
	assert.Equal(t, []string{"JPY"}, missing)
}
