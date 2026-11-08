package investments

import (
	"math/big"
	"sort"
)

// SecurityType classifies a security
type SecurityType int

const (
	// SecurityTypeStock is a listed share
	SecurityTypeStock SecurityType = 1
	// SecurityTypeETF is an exchange traded fund
	SecurityTypeETF SecurityType = 2
	// SecurityTypeMutualFund is a mutual fund
	SecurityTypeMutualFund SecurityType = 3
	// SecurityTypeCrypto is a crypto asset
	SecurityTypeCrypto SecurityType = 4
	// SecurityTypeOther is anything else
	SecurityTypeOther SecurityType = 5
)

// Rates holds how many units of each currency equal one unit of a common reference currency.
// The reference currency itself must be present with rate 1.
type Rates map[string]*big.Rat

// Convert converts minor units between currencies. ok is false when either currency has no rate.
func (r Rates) Convert(amount int64, from string, to string) (int64, bool) {
	if from == to {
		return amount, true
	}

	fromRate, hasFrom := r[from]
	toRate, hasTo := r[to]

	if !hasFrom || !hasTo || fromRate.Sign() <= 0 || toRate.Sign() <= 0 {
		return 0, false
	}

	converted := new(big.Rat).SetInt64(amount)
	converted.Mul(converted, toRate)
	converted.Quo(converted, fromRate)

	return RoundRat(converted), true
}

// HoldingInput is everything needed to summarise one security
type HoldingInput struct {
	SecurityID int64
	Name       string
	Symbol     string
	Type       SecurityType
	Currency   string
	Holding    Holding
	Price      int64 // latest unit price scaled 1e8, 0 when unknown
	PriceTime  int64
	Trades     []Trade
}

// HoldingView is the computed view of one security in its own currency
type HoldingView struct {
	SecurityID        int64
	Name              string
	Symbol            string
	Type              SecurityType
	Currency          string
	Holding           Holding
	Price             int64
	PriceTime         int64
	PriceAvailable    bool
	Value             int64
	Unrealised        int64
	UnrealisedPercent *float64
	XIRR              *float64
	Warning           string
}

// Allocation is one slice of an allocation breakdown
type Allocation struct {
	Key     string
	Value   int64   // base currency minor units
	Percent float64 // percentages of all slices add up to exactly 100
}

// Totals are portfolio totals in the base currency
type Totals struct {
	Invested   int64 // cost basis of units currently held
	PricedCost int64 // the part of Invested that belongs to holdings with a price; the denominator of the unrealised percentage
	Value      int64
	Unrealised int64
	Realised   int64
	Dividends  int64
	XIRR       *float64
}

// Summary is the portfolio summary
type Summary struct {
	BaseCurrency          string
	Holdings              []HoldingView
	Totals                Totals
	AllocationByType      []Allocation
	AllocationByCurrency  []Allocation
	UnconvertedCurrencies []string
}

// Warning codes carried in HoldingView.Warning
const (
	WarningNoPrice = "no_price"
	WarningNoRate  = "no_exchange_rate"
)

// BuildSummary computes holding views and base-currency totals.
// Holdings without a price or without an exchange rate are never counted as zero: they are flagged and left out of totals.
func BuildSummary(inputs []HoldingInput, base string, rates Rates, now int64) Summary {
	summary := Summary{BaseCurrency: base, Holdings: make([]HoldingView, 0, len(inputs))}

	unconverted := map[string]bool{}
	typeValues := map[string]int64{}
	currencyValues := map[string]int64{}

	var portfolioFlows []CashFlow
	portfolioXirrValid := true
	hasPortfolioFlows := false

	for _, in := range inputs {
		view := HoldingView{
			SecurityID: in.SecurityID,
			Name:       in.Name,
			Symbol:     in.Symbol,
			Type:       in.Type,
			Currency:   in.Currency,
			Holding:    in.Holding,
			Price:      in.Price,
			PriceTime:  in.PriceTime,
		}

		held := in.Holding.Quantity > 0
		view.PriceAvailable = in.Price > 0

		if held && !view.PriceAvailable {
			view.Warning = WarningNoPrice
		}

		var valuation Valuation

		if view.PriceAvailable {
			valuation = in.Holding.Value(in.Price)
			view.Value = valuation.Value
			view.Unrealised = valuation.Unrealised
			view.UnrealisedPercent = valuation.UnrealisedPercent
		}

		// per-security XIRR in the security currency
		flows := CashFlows(in.Trades)

		if held && view.PriceAvailable {
			flows = append(flows, CashFlow{Time: now, Amount: float64(valuation.Value)})
		}

		if !held || view.PriceAvailable {
			if rate, ok := XIRR(flows); ok {
				r := rate
				view.XIRR = &r
			}
		}

		// base currency totals
		_, hasRate := rates.Convert(1, in.Currency, base)

		if !hasRate {
			if held || in.Holding.TradeCount > 0 {
				unconverted[in.Currency] = true

				if view.Warning == "" {
					view.Warning = WarningNoRate
				}
			}

			summary.Holdings = append(summary.Holdings, view)
			continue
		}

		if in.Holding.TradeCount > 0 {
			hasPortfolioFlows = true
		}

		realised, _ := rates.Convert(in.Holding.Realised, in.Currency, base)
		dividends, _ := rates.Convert(in.Holding.Dividends, in.Currency, base)
		summary.Totals.Realised += realised
		summary.Totals.Dividends += dividends

		if held {
			cost, _ := rates.Convert(in.Holding.TotalCost, in.Currency, base)
			summary.Totals.Invested += cost

			if view.PriceAvailable {
				value, _ := rates.Convert(valuation.Value, in.Currency, base)
				summary.Totals.PricedCost += cost
				summary.Totals.Value += value
				summary.Totals.Unrealised += value - cost
				typeValues[typeKey(in.Type)] += value
				currencyValues[in.Currency] += value
			} else {
				portfolioXirrValid = false
			}
		}

		for _, f := range CashFlows(in.Trades) {
			converted, _ := rates.Convert(int64(f.Amount), in.Currency, base)
			portfolioFlows = append(portfolioFlows, CashFlow{Time: f.Time, Amount: float64(converted)})
		}

		summary.Holdings = append(summary.Holdings, view)
	}

	if hasPortfolioFlows && portfolioXirrValid {
		portfolioFlows = append(portfolioFlows, CashFlow{Time: now, Amount: float64(summary.Totals.Value)})

		if rate, ok := XIRR(portfolioFlows); ok {
			r := rate
			summary.Totals.XIRR = &r
		}
	}

	summary.AllocationByType = allocate(typeValues)
	summary.AllocationByCurrency = allocate(currencyValues)

	for currency := range unconverted {
		summary.UnconvertedCurrencies = append(summary.UnconvertedCurrencies, currency)
	}

	sort.Strings(summary.UnconvertedCurrencies)

	return summary
}

func typeKey(t SecurityType) string {
	switch t {
	case SecurityTypeStock:
		return "stock"
	case SecurityTypeETF:
		return "etf"
	case SecurityTypeMutualFund:
		return "mutual_fund"
	case SecurityTypeCrypto:
		return "crypto"
	default:
		return "other"
	}
}

// allocate converts values into slices whose percentages (two decimals) sum to exactly 100
// using the largest remainder method. Slices are sorted by value descending, then key.
func allocate(values map[string]int64) []Allocation {
	total := int64(0)

	for _, v := range values {
		if v > 0 {
			total += v
		}
	}

	if total <= 0 {
		return []Allocation{}
	}

	type slice struct {
		key       string
		value     int64
		basis     int64 // hundredths of a percent, floor
		remainder *big.Rat
	}

	slices := make([]slice, 0, len(values))
	assigned := int64(0)

	for key, value := range values {
		if value <= 0 {
			continue
		}

		exact := new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(value), big.NewInt(10000)), big.NewInt(total))
		floor := new(big.Int).Div(exact.Num(), exact.Denom())
		remainder := new(big.Rat).Sub(exact, new(big.Rat).SetInt(floor))

		slices = append(slices, slice{key: key, value: value, basis: floor.Int64(), remainder: remainder})
		assigned += floor.Int64()
	}

	sort.Slice(slices, func(i, j int) bool {
		if c := slices[i].remainder.Cmp(slices[j].remainder); c != 0 {
			return c > 0
		}

		return slices[i].key < slices[j].key
	})

	for i := int64(0); i < 10000-assigned && int(i) < len(slices); i++ {
		slices[i].basis++
	}

	result := make([]Allocation, len(slices))

	for i, s := range slices {
		result[i] = Allocation{Key: s.key, Value: s.value, Percent: float64(s.basis) / 100}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Value != result[j].Value {
			return result[i].Value > result[j].Value
		}

		return result[i].Key < result[j].Key
	})

	return result
}
