package models

import (
	"github.com/mayswind/ezbookkeeping/pkg/investments"
)

// InvestmentHoldingResponse is one security row of the portfolio summary (Expbook).
// Quantities and prices are decimal strings; money is integer minor units in the security currency.
type InvestmentHoldingResponse struct {
	SecurityId        int64    `json:"securityId,string"`
	Name              string   `json:"name"`
	Symbol            string   `json:"symbol"`
	SecurityType      int32    `json:"securityType"`
	Currency          string   `json:"currency"`
	Quantity          string   `json:"quantity"`
	AverageCost       string   `json:"averageCost"`
	TotalCost         int64    `json:"totalCost"`
	Price             string   `json:"price"`
	PriceTime         int64    `json:"priceTime"`
	PriceAvailable    bool     `json:"priceAvailable"`
	Value             int64    `json:"value"`
	Unrealised        int64    `json:"unrealised"`
	UnrealisedPercent *float64 `json:"unrealisedPercent"`
	Realised          int64    `json:"realised"`
	Dividends         int64    `json:"dividends"`
	Xirr              *float64 `json:"xirr"`
	TradeCount        int      `json:"tradeCount"`
	Warning           string   `json:"warning"`
}

// InvestmentTotalsResponse is the portfolio total in the base currency, in minor units
type InvestmentTotalsResponse struct {
	Invested   int64    `json:"invested"`
	PricedCost int64    `json:"pricedCost"`
	Value      int64    `json:"value"`
	Unrealised int64    `json:"unrealised"`
	Realised   int64    `json:"realised"`
	Dividends  int64    `json:"dividends"`
	Xirr       *float64 `json:"xirr"`
}

// InvestmentAllocationResponse is one slice of an allocation breakdown
type InvestmentAllocationResponse struct {
	Key     string  `json:"key"`
	Value   int64   `json:"value"`
	Percent float64 `json:"percent"`
}

// InvestmentSummaryResponse is the response of the portfolio summary
type InvestmentSummaryResponse struct {
	BaseCurrency          string                          `json:"baseCurrency"`
	RatesAvailable        bool                            `json:"ratesAvailable"`
	Holdings              []*InvestmentHoldingResponse    `json:"holdings"`
	Totals                *InvestmentTotalsResponse       `json:"totals"`
	AllocationByType      []*InvestmentAllocationResponse `json:"allocationByType"`
	AllocationByCurrency  []*InvestmentAllocationResponse `json:"allocationByCurrency"`
	UnconvertedCurrencies []string                        `json:"unconvertedCurrencies"`
}

// ToInvestmentSummaryResponse converts a computed summary to its view-object
func ToInvestmentSummaryResponse(summary investments.Summary, ratesAvailable bool) *InvestmentSummaryResponse {
	resp := &InvestmentSummaryResponse{
		BaseCurrency:          summary.BaseCurrency,
		RatesAvailable:        ratesAvailable,
		Holdings:              make([]*InvestmentHoldingResponse, 0, len(summary.Holdings)),
		AllocationByType:      toAllocationResponses(summary.AllocationByType),
		AllocationByCurrency:  toAllocationResponses(summary.AllocationByCurrency),
		UnconvertedCurrencies: summary.UnconvertedCurrencies,
		Totals: &InvestmentTotalsResponse{
			Invested:   summary.Totals.Invested,
			PricedCost: summary.Totals.PricedCost,
			Value:      summary.Totals.Value,
			Unrealised: summary.Totals.Unrealised,
			Realised:   summary.Totals.Realised,
			Dividends:  summary.Totals.Dividends,
			Xirr:       summary.Totals.XIRR,
		},
	}

	if resp.UnconvertedCurrencies == nil {
		resp.UnconvertedCurrencies = []string{}
	}

	for _, h := range summary.Holdings {
		price := ""

		if h.PriceAvailable {
			price = investments.FormatScaled(h.Price)
		}

		resp.Holdings = append(resp.Holdings, &InvestmentHoldingResponse{
			SecurityId:        h.SecurityID,
			Name:              h.Name,
			Symbol:            h.Symbol,
			SecurityType:      int32(h.Type),
			Currency:          h.Currency,
			Quantity:          investments.FormatScaled(h.Holding.Quantity),
			AverageCost:       investments.FormatScaled(h.Holding.AverageCost),
			TotalCost:         h.Holding.TotalCost,
			Price:             price,
			PriceTime:         h.PriceTime,
			PriceAvailable:    h.PriceAvailable,
			Value:             h.Value,
			Unrealised:        h.Unrealised,
			UnrealisedPercent: h.UnrealisedPercent,
			Realised:          h.Holding.Realised,
			Dividends:         h.Holding.Dividends,
			Xirr:              h.XIRR,
			TradeCount:        h.Holding.TradeCount,
			Warning:           h.Warning,
		})
	}

	return resp
}

func toAllocationResponses(items []investments.Allocation) []*InvestmentAllocationResponse {
	out := make([]*InvestmentAllocationResponse, 0, len(items))

	for _, item := range items {
		out = append(out, &InvestmentAllocationResponse{Key: item.Key, Value: item.Value, Percent: item.Percent})
	}

	return out
}
