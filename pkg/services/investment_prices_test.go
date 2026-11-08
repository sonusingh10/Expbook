package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/investments/prices"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

type fakeProvider struct {
	name    string
	results map[string]prices.Result
	calls   int
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Fetch(ctx context.Context, identifiers []string) map[string]prices.Result {
	f.calls++
	out := map[string]prices.Result{}

	for _, id := range identifiers {
		if r, ok := f.results[id]; ok {
			out[id] = r
		}
	}

	return out
}

func priceOf(t *testing.T, s string) int64 {
	v, err := investments.ParseScaled(s)
	assert.NoError(t, err)

	return v
}

func TestReconcileQuote(t *testing.T) {
	inr := &models.Security{Currency: "INR"}
	gbp := &models.Security{Currency: "GBP"}

	price, err := reconcileQuote(inr, prices.Quote{Price: priceOf(t, "100"), Currency: "INR"})
	assert.NoError(t, err)
	assert.Equal(t, priceOf(t, "100"), price)

	// unknown provider currency is accepted
	_, err = reconcileQuote(inr, prices.Quote{Price: priceOf(t, "100")})
	assert.NoError(t, err)

	// pence to pounds
	price, err = reconcileQuote(gbp, prices.Quote{Price: priceOf(t, "1234.5"), Currency: "GBp"})
	assert.NoError(t, err)
	assert.Equal(t, priceOf(t, "12.345"), price)

	// a quote in another currency must not be stored silently
	_, err = reconcileQuote(inr, prices.Quote{Price: priceOf(t, "100"), Currency: "USD"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "USD")
}

func TestRefreshPricesPartialFailure(t *testing.T) {
	c := setupInvestmentTestDb(t)

	good := &models.Security{Uid: testUid, Name: "Good", Symbol: "GOOD", SecurityType: 1, Currency: "INR", PriceSource: models.SecurityPriceSourceYahoo, PriceSymbol: "GOOD.NS"}
	bad := &models.Security{Uid: testUid, Name: "Bad", Symbol: "BAD", SecurityType: 1, Currency: "INR", PriceSource: models.SecurityPriceSourceYahoo, PriceSymbol: "BAD.NS"}
	wrongCcy := &models.Security{Uid: testUid, Name: "Foreign", Symbol: "FOR", SecurityType: 1, Currency: "INR", PriceSource: models.SecurityPriceSourceYahoo, PriceSymbol: "FOR"}
	manual := &models.Security{Uid: testUid, Name: "Manual", Symbol: "MAN", SecurityType: 1, Currency: "INR"}

	for _, s := range []*models.Security{good, bad, wrongCcy, manual} {
		assert.NoError(t, Securities.CreateSecurity(c, s))
	}

	provider := &fakeProvider{name: "yahoo", results: map[string]prices.Result{
		"GOOD.NS": {Quote: prices.Quote{Price: priceOf(t, "250.5"), Time: 1760000000, Currency: "INR"}},
		"BAD.NS":  {Err: errors.New("boom")},
		"FOR":     {Quote: prices.Quote{Price: priceOf(t, "10"), Time: 1760000000, Currency: "USD"}},
	}}

	service := &InvestmentPriceService{
		ServiceUsingDB:   Securities.ServiceUsingDB,
		ServiceUsingUuid: Securities.ServiceUsingUuid,
		providers:        map[string]prices.Provider{"yahoo": provider},
	}

	results, err := service.RefreshPrices(c, testUid)
	assert.NoError(t, err)
	assert.Equal(t, 1, provider.calls)
	assert.Len(t, results, 3, "manual securities are not refreshed")

	byId := map[int64]*models.SecurityPriceRefreshResult{}

	for _, r := range results {
		byId[r.SecurityId] = r
	}

	assert.True(t, byId[good.SecurityId].Ok)
	assert.Equal(t, "250.5", byId[good.SecurityId].Price)
	assert.False(t, byId[bad.SecurityId].Ok)
	assert.Equal(t, "boom", byId[bad.SecurityId].Message)
	assert.False(t, byId[wrongCcy.SecurityId].Ok)
	assert.Contains(t, byId[wrongCcy.SecurityId].Message, "USD")

	stored, err := Securities.GetSecurityBySecurityId(c, testUid, good.SecurityId)
	assert.NoError(t, err)
	assert.Equal(t, priceOf(t, "250.5"), stored.LatestPrice)
	assert.Equal(t, int64(1760000000), stored.LatestPriceTime)

	untouched, err := Securities.GetSecurityBySecurityId(c, testUid, bad.SecurityId)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), untouched.LatestPrice, "a failed refresh must not overwrite or invent a price")

	// the cron entry point finds the user and refreshes too
	assert.NoError(t, service.RefreshAllUsers(c))
}

func TestRefreshPricesUnknownSource(t *testing.T) {
	c := setupInvestmentTestDb(t)

	security := &models.Security{Uid: testUid, Name: "X", Symbol: "X", SecurityType: 1, Currency: "INR", PriceSource: models.SecurityPriceSourceAmfi, PriceSymbol: "1"}
	assert.NoError(t, Securities.CreateSecurity(c, security))

	service := &InvestmentPriceService{
		ServiceUsingDB:   Securities.ServiceUsingDB,
		ServiceUsingUuid: Securities.ServiceUsingUuid,
		providers:        map[string]prices.Provider{},
	}

	results, err := service.RefreshPrices(c, testUid)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.False(t, results[0].Ok)
	assert.Contains(t, results[0].Message, "unknown price source")
}
