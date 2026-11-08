package investments

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
)

// TradeType is the kind of an investment trade
type TradeType int

const (
	// TradeTypeBuy adds units and cost
	TradeTypeBuy TradeType = 1
	// TradeTypeSell removes units at average cost and realises profit
	TradeTypeSell TradeType = 2
	// TradeTypeDividend records income without changing units
	TradeTypeDividend TradeType = 3
)

// ErrSellExceedsHoldings is returned when a sell would take the quantity below zero
var ErrSellExceedsHoldings = errors.New("sell quantity exceeds quantity held")

// ErrInvalidTrade is returned when a trade has a non-positive quantity or price (or amount for dividends)
var ErrInvalidTrade = errors.New("trade has invalid quantity, price or amount")

// SellExceedsError carries the offending trade id
type SellExceedsError struct {
	TradeID int64
}

// Error returns the message
func (e *SellExceedsError) Error() string {
	return fmt.Sprintf("%s (trade id:%d)", ErrSellExceedsHoldings.Error(), e.TradeID)
}

// Unwrap lets errors.Is match ErrSellExceedsHoldings
func (e *SellExceedsError) Unwrap() error {
	return ErrSellExceedsHoldings
}

// Trade is one dated event on a security. Quantity and Price are scaled by 1e8, Amount and Fees are minor units.
type Trade struct {
	ID       int64
	Time     int64
	Type     TradeType
	Quantity int64
	Price    int64
	Amount   int64
	Fees     int64
}

// Holding is the replayed position of one security (average-cost method)
type Holding struct {
	Quantity     int64 // units held, scaled 1e8
	TotalCost    int64 // cost basis of the units held, minor units
	AverageCost  int64 // cost per unit including fees, scaled 1e8 (0 when nothing is held)
	Realised     int64 // realised profit and loss, minor units
	Dividends    int64 // dividends received, minor units
	BoughtCost   int64 // sum of gross buys plus fees, minor units
	SoldProceeds int64 // sum of gross sells minus fees, minor units
	TradeCount   int
}

// Validate checks the numeric fields of one trade
func (t Trade) Validate() error {
	switch t.Type {
	case TradeTypeBuy, TradeTypeSell:
		if t.Quantity <= 0 || t.Price <= 0 || t.Fees < 0 {
			return ErrInvalidTrade
		}
	case TradeTypeDividend:
		if t.Amount <= 0 {
			return ErrInvalidTrade
		}
	default:
		return ErrInvalidTrade
	}

	return nil
}

func (t Trade) typeOrder() int {
	switch t.Type {
	case TradeTypeBuy:
		return 0
	case TradeTypeDividend:
		return 1
	default:
		return 2
	}
}

// SortTrades returns a copy ordered by time, then buys before dividends before sells, then id.
// This ordering is what makes results independent of the order trades were entered.
func SortTrades(trades []Trade) []Trade {
	sorted := make([]Trade, len(trades))
	copy(sorted, trades)

	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]

		if a.Time != b.Time {
			return a.Time < b.Time
		}

		if a.typeOrder() != b.typeOrder() {
			return a.typeOrder() < b.typeOrder()
		}

		return a.ID < b.ID
	})

	return sorted
}

// Replay computes the holding from all trades of one security using the average-cost method.
// It returns a *SellExceedsError when the history ever sells more than is held.
func Replay(trades []Trade) (Holding, error) {
	var h Holding

	cost := new(big.Rat)
	quantity := int64(0)

	for _, t := range SortTrades(trades) {
		if err := t.Validate(); err != nil {
			return Holding{}, err
		}

		h.TradeCount++

		switch t.Type {
		case TradeTypeBuy:
			buyCost := GrossMinor(t.Quantity, t.Price) + t.Fees
			cost.Add(cost, new(big.Rat).SetInt64(buyCost))
			quantity += t.Quantity
			h.BoughtCost += buyCost

		case TradeTypeSell:
			if t.Quantity > quantity {
				return Holding{}, &SellExceedsError{TradeID: t.ID}
			}

			proceeds := GrossMinor(t.Quantity, t.Price) - t.Fees

			removed := new(big.Rat).Mul(cost, new(big.Rat).SetFrac(big.NewInt(t.Quantity), big.NewInt(quantity)))
			realised := new(big.Rat).Sub(new(big.Rat).SetInt64(proceeds), removed)
			h.Realised += RoundRat(realised)
			h.SoldProceeds += proceeds

			cost.Sub(cost, removed)
			quantity -= t.Quantity

		case TradeTypeDividend:
			h.Dividends += t.Amount
		}
	}

	h.Quantity = quantity

	if quantity == 0 {
		h.TotalCost = 0
		h.AverageCost = 0
	} else {
		h.TotalCost = RoundRat(cost)
		perUnit := new(big.Rat).Mul(cost, new(big.Rat).SetFrac(new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil), big.NewInt(quantity)))
		h.AverageCost = RoundRat(perUnit)
	}

	return h, nil
}

// Valuation is the market value and unrealised result of a holding at a price
type Valuation struct {
	Value             int64    // minor units
	Unrealised        int64    // minor units
	UnrealisedPercent *float64 // nil when cost is zero
}

// Value values a holding at a unit price (scaled 1e8)
func (h Holding) Value(price int64) Valuation {
	value := GrossMinor(h.Quantity, price)
	v := Valuation{Value: value, Unrealised: value - h.TotalCost}

	if h.TotalCost > 0 {
		pct := float64(v.Unrealised) / float64(h.TotalCost) * 100
		v.UnrealisedPercent = &pct
	}

	return v
}

// CashFlows converts trades into dated cash flows (minor units; negative = money out) for XIRR
func CashFlows(trades []Trade) []CashFlow {
	flows := make([]CashFlow, 0, len(trades))

	for _, t := range SortTrades(trades) {
		switch t.Type {
		case TradeTypeBuy:
			flows = append(flows, CashFlow{Time: t.Time, Amount: -float64(GrossMinor(t.Quantity, t.Price) + t.Fees)})
		case TradeTypeSell:
			flows = append(flows, CashFlow{Time: t.Time, Amount: float64(GrossMinor(t.Quantity, t.Price) - t.Fees)})
		case TradeTypeDividend:
			flows = append(flows, CashFlow{Time: t.Time, Amount: float64(t.Amount)})
		}
	}

	return flows
}
