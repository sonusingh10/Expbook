package services

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/httpclient"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/investments/prices"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// InvestmentPriceService refreshes security prices from the configured providers (Expbook)
type InvestmentPriceService struct {
	ServiceUsingDB
	ServiceUsingUuid
	providers map[string]prices.Provider
}

// Initialize an investment price service singleton instance
var (
	InvestmentPrices = &InvestmentPriceService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

const priceRefreshTimeout = 90 * time.Second

func (s *InvestmentPriceService) getProviders() map[string]prices.Provider {
	if s.providers != nil {
		return s.providers
	}

	client := httpclient.NewHttpClient(15000, "", false, "Mozilla/5.0 (compatible; Expbook/1.0)", false, nil)

	return map[string]prices.Provider{
		models.SecurityPriceSourceYahoo: &prices.Yahoo{Client: client},
		models.SecurityPriceSourceAmfi:  &prices.Amfi{Client: client},
	}
}

// reconcileQuote converts a provider quote into the security's currency, or explains why it cannot be used
func reconcileQuote(security *models.Security, quote prices.Quote) (int64, error) {
	price := quote.Price
	currency := quote.Currency

	// London quotes are in pence (GBp / GBX): convert to pounds
	if currency == "GBp" || strings.EqualFold(currency, "GBX") {
		price = investments.RoundRat(new(big.Rat).SetFrac(big.NewInt(price), big.NewInt(100)))
		currency = "GBP"
	}

	if price <= 0 {
		return 0, prices.ErrImplausiblePrice
	}

	if currency != "" && !strings.EqualFold(currency, security.Currency) {
		return 0, errors.New("provider quoted in " + strings.ToUpper(currency) + " but the security currency is " + security.Currency)
	}

	return price, nil
}

// RefreshPrices refreshes all non-manual securities of one user. One failing security never blocks the others.
func (s *InvestmentPriceService) RefreshPrices(c core.Context, uid int64) ([]*models.SecurityPriceRefreshResult, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	securities, err := Securities.GetAllSecuritiesByUid(c, uid)

	if err != nil {
		return nil, err
	}

	results := []*models.SecurityPriceRefreshResult{}
	bySource := map[string][]*models.Security{}

	for _, security := range securities {
		if security.PriceSource == models.SecurityPriceSourceManual || security.PriceSource == "" {
			continue
		}

		bySource[security.PriceSource] = append(bySource[security.PriceSource], security)
	}

	providers := s.getProviders()
	ctx, cancel := context.WithTimeout(context.Background(), priceRefreshTimeout)
	defer cancel()

	for source, group := range bySource {
		provider, ok := providers[source]

		if !ok {
			for _, security := range group {
				results = append(results, &models.SecurityPriceRefreshResult{SecurityId: security.SecurityId, Symbol: security.Symbol, Message: "unknown price source " + source})
			}

			continue
		}

		seen := map[string]bool{}
		var identifiers []string

		for _, security := range group {
			if !seen[security.PriceSymbol] {
				seen[security.PriceSymbol] = true
				identifiers = append(identifiers, security.PriceSymbol)
			}
		}

		fetched := provider.Fetch(ctx, identifiers)

		for _, security := range group {
			result := &models.SecurityPriceRefreshResult{SecurityId: security.SecurityId, Symbol: security.Symbol}
			results = append(results, result)

			item, found := fetched[security.PriceSymbol]

			if !found {
				result.Message = "provider returned no result"
				continue
			}

			if item.Err != nil {
				result.Message = item.Err.Error()
				continue
			}

			price, err := reconcileQuote(security, item.Quote)

			if err != nil {
				result.Message = err.Error()
				continue
			}

			if err := Securities.SetLatestPrice(c, uid, security.SecurityId, price, item.Quote.Time, source); err != nil {
				result.Message = "could not save price: " + err.Error()
				continue
			}

			result.Ok = true
			result.Price = investments.FormatScaled(price)
			result.PriceTime = item.Quote.Time
		}
	}

	return results, nil
}

// RefreshAllUsers refreshes prices for every user that owns a non-manual security (used by the cron job)
func (s *InvestmentPriceService) RefreshAllUsers(c core.Context) error {
	var firstErr error

	for index := 0; index < s.UserDataDBCount(); index++ {
		var rows []*models.Security
		err := s.UserDataDBByIndex(index).NewSession(c).Cols("uid").Where("deleted=? AND price_source<>?", false, models.SecurityPriceSourceManual).GroupBy("uid").Find(&rows)

		if err != nil {
			log.Errorf(c, "[investment_prices.RefreshAllUsers] failed to list users with securities, because %s", err.Error())

			if firstErr == nil {
				firstErr = err
			}

			continue
		}

		for _, row := range rows {
			results, err := s.RefreshPrices(c, row.Uid)

			if err != nil {
				log.Errorf(c, "[investment_prices.RefreshAllUsers] failed to refresh prices for user \"uid:%d\", because %s", row.Uid, err.Error())

				if firstErr == nil {
					firstErr = err
				}

				continue
			}

			failed := 0

			for _, r := range results {
				if !r.Ok {
					failed++
					log.Warnf(c, "[investment_prices.RefreshAllUsers] price of security \"id:%d\" was not refreshed, because %s", r.SecurityId, r.Message)
				}
			}

			log.Infof(c, "[investment_prices.RefreshAllUsers] refreshed prices for user \"uid:%d\": %d updated, %d failed", row.Uid, len(results)-failed, failed)
		}
	}

	return firstErr
}
