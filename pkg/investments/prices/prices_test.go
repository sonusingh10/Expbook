package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestYahooFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("User-Agent"))

		switch {
		case strings.Contains(r.URL.Path, "RELIANCE.NS"):
			_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"INR","regularMarketPrice":2891.45,"regularMarketTime":1760000000}}],"error":null}}`))
		case strings.Contains(r.URL.Path, "BAD"):
			_, _ = w.Write([]byte(`{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found"}}}`))
		case strings.Contains(r.URL.Path, "ZERO"):
			_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"USD","regularMarketPrice":0}}],"error":null}}`))
		case strings.Contains(r.URL.Path, "GONE"):
			w.WriteHeader(http.StatusNotFound)
		case strings.Contains(r.URL.Path, "BROKEN"):
			_, _ = w.Write([]byte(`<html>not json</html>`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	provider := &Yahoo{BaseURL: server.URL}
	assert.Equal(t, "yahoo", provider.Name())

	results := provider.Fetch(context.Background(), []string{"RELIANCE.NS", "BAD", "ZERO", "GONE", "BROKEN", "ERR"})

	assert.Len(t, results, 6)

	ok := results["RELIANCE.NS"]
	assert.NoError(t, ok.Err)
	assert.Equal(t, int64(289145000000), ok.Quote.Price)
	assert.Equal(t, "INR", ok.Quote.Currency)
	assert.Equal(t, int64(1760000000), ok.Quote.Time)

	assert.ErrorIs(t, results["BAD"].Err, ErrNoQuote)
	assert.ErrorIs(t, results["ZERO"].Err, ErrImplausiblePrice)
	assert.ErrorIs(t, results["GONE"].Err, ErrNoQuote)
	assert.Error(t, results["BROKEN"].Err)
	assert.Error(t, results["ERR"].Err)
}

func TestYahooSymbolIsPathEscaped(t *testing.T) {
	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	provider := &Yahoo{BaseURL: server.URL}
	provider.Fetch(context.Background(), []string{"A/B?x=1"})

	assert.Equal(t, "/v8/finance/chart/A%2FB%3Fx=1", requestedPath)
}

func TestYahooNetworkFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // nothing listens any more

	provider := &Yahoo{BaseURL: url, Client: &http.Client{Timeout: time.Second}}
	results := provider.Fetch(context.Background(), []string{"AAPL"})

	assert.Error(t, results["AAPL"].Err)
}

const amfiSample = `Scheme Code;ISIN Div Payout/ ISIN Growth;ISIN Div Reinvestment;Scheme Name;Net Asset Value;Date

Open Ended Schemes(Debt Scheme - Banking and PSU Fund)

Aditya Birla Sun Life Mutual Fund

119551;INF209KA12Z1;INF209KA13Z9;Aditya Birla Sun Life Banking & PSU Debt Fund  - DIRECT - IDCW;113.5812;07-Nov-2025
120503;INF846K01EW2;-;Axis ELSS Tax Saver Fund - Direct Plan - Growth;95.4421;07-Nov-2025
130000;-;-;Broken Fund;N.A.;07-Nov-2025
130001;-;-;Zero Fund;0.0000;07-Nov-2025
`

func TestParseAmfiNav(t *testing.T) {
	navs, err := ParseAmfiNav(strings.NewReader(amfiSample))
	assert.NoError(t, err)
	assert.Len(t, navs, 2)

	axis := navs["120503"]
	assert.Equal(t, int64(9544210000), axis.Price)
	assert.Equal(t, "INR", axis.Currency)
	assert.Equal(t, time.Date(2025, 11, 7, 12, 0, 0, 0, time.UTC).Unix(), axis.Time)

	_, hasBroken := navs["130000"]
	assert.False(t, hasBroken)

	_, err = ParseAmfiNav(strings.NewReader("garbage\nno rows here"))
	assert.Error(t, err)
}

func TestAmfiFetch(t *testing.T) {
	downloads := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads++
		_, _ = w.Write([]byte(amfiSample))
	}))
	defer server.Close()

	provider := &Amfi{URL: server.URL}
	assert.Equal(t, "amfi", provider.Name())

	results := provider.Fetch(context.Background(), []string{"120503", " 119551 ", "999999"})

	assert.Equal(t, 1, downloads, "the file is downloaded once for all schemes")
	assert.NoError(t, results["120503"].Err)
	assert.NoError(t, results[" 119551 "].Err)
	assert.ErrorIs(t, results["999999"].Err, ErrNoQuote)
}

func TestAmfiDownloadFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	results := (&Amfi{URL: server.URL}).Fetch(context.Background(), []string{"120503"})
	assert.Error(t, results["120503"].Err)
}

func TestFloatToScaled(t *testing.T) {
	v, err := floatToScaled(0.00001234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1234), v)

	for _, bad := range []float64{0, -1} {
		_, err = floatToScaled(bad)
		assert.ErrorIs(t, err, ErrImplausiblePrice)
	}
}
