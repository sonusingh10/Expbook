// Package prices fetches latest security prices from free, keyless public sources.
// Only symbols are ever sent to a provider, never quantities, amounts or account data.
package prices

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/mayswind/ezbookkeeping/pkg/investments"
)

// Quote is one latest price
type Quote struct {
	Price    int64  // scaled 1e8
	Time     int64  // unix seconds of the quote
	Currency string // currency the provider quoted in (may be empty if unknown)
}

// Provider fetches latest quotes for provider-specific identifiers
type Provider interface {
	// Name is the price_source value this provider handles
	Name() string

	// Fetch returns a quote or an error for every requested identifier. A failure for one
	// identifier must not prevent the others from being returned.
	Fetch(ctx context.Context, identifiers []string) map[string]Result
}

// Result is a quote or the reason it could not be fetched
type Result struct {
	Quote Quote
	Err   error
}

// ErrNoQuote is returned when the provider has no price for an identifier
var ErrNoQuote = errors.New("provider has no price for this symbol")

// ErrImplausiblePrice is returned when a provider returns zero, negative or non-finite prices
var ErrImplausiblePrice = errors.New("provider returned an implausible price")

func floatToScaled(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, ErrImplausiblePrice
	}

	scaled, err := investments.ParseScaled(strconv.FormatFloat(value, 'f', investments.ScaleDigits, 64))

	if err != nil || scaled <= 0 {
		return 0, ErrImplausiblePrice
	}

	return scaled, nil
}
