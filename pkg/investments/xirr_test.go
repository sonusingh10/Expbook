package investments

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func date(year int, month time.Month, d int) int64 {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC).Unix()
}

func TestXIRRSimpleTenPercent(t *testing.T) {
	rate, ok := XIRR([]CashFlow{
		{Time: date(2024, 1, 1), Amount: -1000},
		{Time: date(2024, 1, 1) + 365*24*3600, Amount: 1100},
	})

	assert.True(t, ok)
	assert.InDelta(t, 0.10, rate, 1e-6)
}

// Reference example from the Microsoft Excel XIRR documentation
func TestXIRRExcelReference(t *testing.T) {
	rate, ok := XIRR([]CashFlow{
		{Time: date(2008, 1, 1), Amount: -10000},
		{Time: date(2008, 3, 1), Amount: 2750},
		{Time: date(2008, 10, 30), Amount: 4250},
		{Time: date(2009, 2, 15), Amount: 3250},
		{Time: date(2009, 4, 1), Amount: 2750},
	})

	assert.True(t, ok)
	assert.InDelta(t, 0.373362535, rate, 1e-6)
}

func TestXIRRNegativeReturn(t *testing.T) {
	rate, ok := XIRR([]CashFlow{
		{Time: date(2024, 1, 1), Amount: -1000},
		{Time: date(2025, 1, 1), Amount: 800},
	})

	assert.True(t, ok)
	assert.Less(t, rate, 0.0)
	assert.InDelta(t, -0.2, rate, 0.01)
}

func TestXIRRUnordered(t *testing.T) {
	rate, ok := XIRR([]CashFlow{
		{Time: date(2024, 1, 1) + 365*24*3600, Amount: 1100},
		{Time: date(2024, 1, 1), Amount: -1000},
	})

	assert.True(t, ok)
	assert.InDelta(t, 0.10, rate, 1e-6)
}

func TestXIRRUnavailableCases(t *testing.T) {
	// no flows or single flow
	_, ok := XIRR(nil)
	assert.False(t, ok)
	_, ok = XIRR([]CashFlow{{Time: date(2024, 1, 1), Amount: -100}})
	assert.False(t, ok)

	// no sign change
	_, ok = XIRR([]CashFlow{{Time: date(2024, 1, 1), Amount: -100}, {Time: date(2025, 1, 1), Amount: -50}})
	assert.False(t, ok)

	// span shorter than a day
	_, ok = XIRR([]CashFlow{{Time: date(2024, 1, 1), Amount: -100}, {Time: date(2024, 1, 1) + 3600, Amount: 110}})
	assert.False(t, ok)
}

func TestXIRRFromTrades(t *testing.T) {
	trades := []Trade{
		buy(t, 1, 0, "10", "100", 0),
	}

	flows := CashFlows(trades)
	// holding worth 1100 one year later
	flows = append(flows, CashFlow{Time: day(365), Amount: 110000})

	rate, ok := XIRR(flows)
	assert.True(t, ok)
	assert.InDelta(t, 0.10, rate, 1e-6)
}
