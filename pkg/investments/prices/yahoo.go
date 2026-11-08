package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultYahooBaseURL is the public chart endpoint
const DefaultYahooBaseURL = "https://query1.finance.yahoo.com"

// Yahoo fetches latest prices for stocks, ETFs and crypto from the public chart endpoint (no API key).
// The endpoint is unofficial and may change; failures are reported per symbol.
type Yahoo struct {
	BaseURL string
	Client  *http.Client
}

// Name returns the price source name
func (y *Yahoo) Name() string {
	return "yahoo"
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// Fetch requests each symbol sequentially (the endpoint is one symbol per request)
func (y *Yahoo) Fetch(ctx context.Context, identifiers []string) map[string]Result {
	results := make(map[string]Result, len(identifiers))

	for _, identifier := range identifiers {
		quote, err := y.fetchOne(ctx, identifier)
		results[identifier] = Result{Quote: quote, Err: err}
	}

	return results
}

func (y *Yahoo) fetchOne(ctx context.Context, symbol string) (Quote, error) {
	base := y.BaseURL

	if base == "" {
		base = DefaultYahooBaseURL
	}

	client := y.Client

	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	requestUrl := fmt.Sprintf("%s/v8/finance/chart/%s?range=1d&interval=1d", base, url.PathEscape(symbol))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)

	if err != nil {
		return Quote{}, err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Expbook/1.0)")
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)

	if err != nil {
		return Quote{}, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))

	if err != nil {
		return Quote{}, err
	}

	if response.StatusCode == http.StatusNotFound {
		return Quote{}, ErrNoQuote
	}

	if response.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("yahoo returned http status %d", response.StatusCode)
	}

	var parsed yahooChartResponse

	if err := json.Unmarshal(body, &parsed); err != nil {
		return Quote{}, fmt.Errorf("yahoo response could not be read: %w", err)
	}

	if parsed.Chart.Error != nil || len(parsed.Chart.Result) == 0 {
		return Quote{}, ErrNoQuote
	}

	meta := parsed.Chart.Result[0].Meta
	price, err := floatToScaled(meta.RegularMarketPrice)

	if err != nil {
		return Quote{}, err
	}

	quoteTime := meta.RegularMarketTime

	if quoteTime <= 0 {
		quoteTime = time.Now().Unix()
	}

	return Quote{Price: price, Time: quoteTime, Currency: meta.Currency}, nil
}
