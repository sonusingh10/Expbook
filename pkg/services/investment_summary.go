package services

import (
	"math/big"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// RatesFromLatest converts the exchange rate response (units of currency per 1 unit of the response's
// reference currency) into investment rates. Without a response only fallbackBase (rate 1) is known.
func RatesFromLatest(resp *models.LatestExchangeRateResponse, fallbackBase string) investments.Rates {
	if resp == nil || resp.BaseCurrency == "" {
		return investments.Rates{fallbackBase: big.NewRat(1, 1)}
	}

	reference := resp.BaseCurrency
	rates := investments.Rates{reference: big.NewRat(1, 1)}

	for _, item := range resp.ExchangeRates {
		if item == nil || item.Currency == reference {
			continue
		}

		rate, ok := new(big.Rat).SetString(item.Rate)

		if !ok || rate.Sign() <= 0 {
			continue
		}

		rates[item.Currency] = rate
	}

	return rates
}

// BuildSummary replays all trades of the user and builds the portfolio summary in the base currency
func (s *InvestmentTradeService) BuildSummary(c core.Context, uid int64, base string, rates investments.Rates) (investments.Summary, error) {
	securities, err := Securities.GetAllSecuritiesByUid(c, uid)

	if err != nil {
		return investments.Summary{}, err
	}

	trades, err := s.GetAllTradesByUid(c, uid)

	if err != nil {
		return investments.Summary{}, err
	}

	bySecurity := map[int64][]investments.Trade{}

	for _, trade := range trades {
		bySecurity[trade.SecurityId] = append(bySecurity[trade.SecurityId], trade.ToLedgerTrade())
	}

	inputs := make([]investments.HoldingInput, 0, len(securities))

	for _, security := range securities {
		ledger := investments.SortTrades(bySecurity[security.SecurityId])
		holding, err := investments.Replay(ledger)

		if err != nil {
			// stored data was validated on write, so this only happens if the database was edited by hand
			return investments.Summary{}, mapLedgerError(err)
		}

		inputs = append(inputs, investments.HoldingInput{
			SecurityID: security.SecurityId,
			Name:       security.Name,
			Symbol:     security.Symbol,
			Type:       investments.SecurityType(security.SecurityType),
			Currency:   security.Currency,
			Holding:    holding,
			Price:      security.LatestPrice,
			PriceTime:  security.LatestPriceTime,
			Trades:     ledger,
		})
	}

	return investments.BuildSummary(inputs, base, rates, time.Now().Unix()), nil
}
