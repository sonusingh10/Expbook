package models

import "github.com/mayswind/ezbookkeeping/pkg/investments"

// InvestmentHistoryRequest asks for the portfolio value over time (Expbook)
type InvestmentHistoryRequest struct {
	// StartTime is the first date to chart (unix seconds). 0 means "from the first trade".
	StartTime int64 `form:"startTime" binding:"min=0"`
	// StepDays is the spacing between points, 7 when omitted
	StepDays int `form:"stepDays" binding:"min=0,max=31"`
}

// InvestmentHistoryPoint is the portfolio at one date, in the base currency (minor units)
type InvestmentHistoryPoint struct {
	Time      int64 `json:"time"`
	Invested  int64 `json:"invested"`
	Value     int64 `json:"value"`
	Estimated bool  `json:"estimated"`
}

// InvestmentHistoryResponse is the portfolio history in the user's default currency
type InvestmentHistoryResponse struct {
	BaseCurrency          string                    `json:"baseCurrency"`
	RatesAvailable        bool                      `json:"ratesAvailable"`
	UnconvertedCurrencies []string                  `json:"unconvertedCurrencies"`
	FirstTradeTime        int64                     `json:"firstTradeTime"`
	Points                []*InvestmentHistoryPoint `json:"points"`
}

// ToInvestmentHistoryPoints converts computed points to their response form
func ToInvestmentHistoryPoints(points []investments.HistoryPoint) []*InvestmentHistoryPoint {
	resp := make([]*InvestmentHistoryPoint, len(points))

	for i, p := range points {
		resp[i] = &InvestmentHistoryPoint{Time: p.Time, Invested: p.Invested, Value: p.Value, Estimated: p.Estimated}
	}

	return resp
}
