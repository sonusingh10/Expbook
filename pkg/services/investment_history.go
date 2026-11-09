package services

import (
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// maxHistoryPoints bounds the size of a history response
const maxHistoryPoints = 600

// GetAllPricesByUid returns every recorded price of the user (the price history), oldest first
func (s *SecurityService) GetAllPricesByUid(c core.Context, uid int64) ([]*models.SecurityPrice, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var prices []*models.SecurityPrice
	err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).OrderBy("price_time asc").Find(&prices)

	return prices, err
}

// BuildHistory values the portfolio at weekly (or stepSeconds) dates between start and end in the base currency.
// It returns the points, the currencies that could not be converted, and the time of the first trade (0 when none).
func (s *InvestmentTradeService) BuildHistory(c core.Context, uid int64, base string, rates investments.Rates, start int64, end int64, stepSeconds int64) ([]investments.HistoryPoint, []string, int64, error) {
	securities, err := Securities.GetAllSecuritiesByUid(c, uid)

	if err != nil {
		return nil, nil, 0, err
	}

	trades, err := s.GetAllTradesByUid(c, uid)

	if err != nil {
		return nil, nil, 0, err
	}

	prices, err := Securities.GetAllPricesByUid(c, uid)

	if err != nil {
		return nil, nil, 0, err
	}

	tradesBySecurity := map[int64][]investments.Trade{}
	firstTrade := int64(0)

	for _, trade := range trades {
		tradesBySecurity[trade.SecurityId] = append(tradesBySecurity[trade.SecurityId], trade.ToLedgerTrade())

		if firstTrade == 0 || trade.TradeTime < firstTrade {
			firstTrade = trade.TradeTime
		}
	}

	pricesBySecurity := map[int64][]investments.PricePoint{}

	for _, price := range prices {
		pricesBySecurity[price.SecurityId] = append(pricesBySecurity[price.SecurityId], investments.PricePoint{Time: price.PriceTime, Price: price.Price})
	}

	inputs := make([]investments.HistoryInput, 0, len(securities))

	for _, security := range securities {
		inputs = append(inputs, investments.HistoryInput{
			Currency: security.Currency,
			Trades:   tradesBySecurity[security.SecurityId],
			Prices:   pricesBySecurity[security.SecurityId],
		})
	}

	if firstTrade == 0 {
		return []investments.HistoryPoint{}, []string{}, 0, nil
	}

	// a start before the first trade is allowed (zeros) so the caller can line up with other series
	if start <= 0 {
		start = firstTrade
	}

	times := investments.HistoryTimes(start, end, stepSeconds, maxHistoryPoints)
	points, missing := investments.BuildHistory(inputs, base, rates, times)

	sort.Slice(points, func(i, j int) bool { return points[i].Time < points[j].Time })

	return points, missing, firstTrade, nil
}
