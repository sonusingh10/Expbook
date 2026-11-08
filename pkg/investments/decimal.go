// Package investments contains the pure (database free) money maths used by the
// Expbook investment features: fixed-point decimals, the average-cost ledger,
// XIRR, portfolio summaries and CSV parsing.
package investments

import (
	"errors"
	"math/big"
	"strings"
)

// Scale is the fixed-point factor used for quantities and unit prices (8 decimal places)
const Scale = int64(100000000)

// ScaleDigits is the number of decimal places represented by Scale
const ScaleDigits = 8

// MinorUnitFactor converts major currency units to minor units (same convention as ezBookkeeping amounts)
const MinorUnitFactor = int64(100)

// ErrInvalidDecimal is returned when a decimal string cannot be parsed
var ErrInvalidDecimal = errors.New("invalid decimal number")

// ParseScaled parses a decimal string like "10.5" or "-0.00000001" into an integer scaled by 1e8.
// It rejects thousands separators, exponents, and more than 8 fractional digits so nothing is silently rounded.
func ParseScaled(s string) (int64, error) {
	return parseWithDigits(s, ScaleDigits)
}

// ParseMinor parses a decimal string in major units (for example "123.45") into minor units (x100).
// More than 2 fractional digits are rejected.
func ParseMinor(s string) (int64, error) {
	return parseWithDigits(s, 2)
}

func parseWithDigits(s string, digits int) (int64, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, ErrInvalidDecimal
	}

	negative := false

	if s[0] == '-' || s[0] == '+' {
		negative = s[0] == '-'
		s = s[1:]
	}

	if s == "" {
		return 0, ErrInvalidDecimal
	}

	intPart := s
	fracPart := ""

	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart = s[:dot]
		fracPart = s[dot+1:]
	}

	if intPart == "" && fracPart == "" {
		return 0, ErrInvalidDecimal
	}

	if len(fracPart) > digits {
		return 0, ErrInvalidDecimal
	}

	for _, ch := range intPart + fracPart {
		if ch < '0' || ch > '9' {
			return 0, ErrInvalidDecimal
		}
	}

	fracPart += strings.Repeat("0", digits-len(fracPart))

	value, ok := new(big.Int).SetString(intPart+fracPart, 10)

	if !ok || !value.IsInt64() {
		return 0, ErrInvalidDecimal
	}

	result := value.Int64()

	if negative {
		result = -result
	}

	return result, nil
}

// FormatScaled formats an integer scaled by 1e8 as a decimal string without trailing zeros
func FormatScaled(v int64) string {
	return formatWithDigits(v, ScaleDigits)
}

// FormatMinor formats minor units as a decimal string with exactly 2 fractional digits
func FormatMinor(v int64) string {
	s := formatWithDigits(v, 2)

	if !strings.Contains(s, ".") {
		return s + ".00"
	}

	if len(s)-strings.IndexByte(s, '.')-1 == 1 {
		return s + "0"
	}

	return s
}

func formatWithDigits(v int64, digits int) string {
	negative := v < 0
	abs := new(big.Int).Abs(big.NewInt(v))
	str := abs.String()

	if len(str) <= digits {
		str = strings.Repeat("0", digits-len(str)+1) + str
	}

	intPart := str[:len(str)-digits]
	fracPart := strings.TrimRight(str[len(str)-digits:], "0")

	result := intPart

	if fracPart != "" {
		result += "." + fracPart
	}

	if negative && result != "0" {
		result = "-" + result
	}

	return result
}

// GrossAmount returns quantity x price as an exact rational number of minor units.
// quantity and price are both scaled by 1e8 and amounts by 100, so the result is q*p*100/1e16 = q*p/1e14.
func GrossAmount(quantity int64, price int64) *big.Rat {
	numerator := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(price))
	denominator := new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil)

	return new(big.Rat).SetFrac(numerator, denominator)
}

// RoundRat rounds a rational number to the nearest integer, halves away from zero
func RoundRat(r *big.Rat) int64 {
	if r.Sign() == 0 {
		return 0
	}

	num := new(big.Int).Set(r.Num())
	den := r.Denom()
	negative := num.Sign() < 0
	num.Abs(num)

	twice := new(big.Int).Mul(num, big.NewInt(2))
	twice.Add(twice, den)
	den2 := new(big.Int).Mul(den, big.NewInt(2))
	quotient := new(big.Int).Div(twice, den2)

	result := quotient.Int64()

	if negative {
		result = -result
	}

	return result
}

// GrossMinor returns quantity x price rounded to whole minor units
func GrossMinor(quantity int64, price int64) int64 {
	return RoundRat(GrossAmount(quantity, price))
}

// PriceFromAmount derives a unit price (scaled 1e8) from a total amount in minor units and a quantity (scaled 1e8).
// Returns 0 when quantity is not positive.
func PriceFromAmount(amountMinor int64, quantity int64) int64 {
	if quantity <= 0 {
		return 0
	}

	// price8 = (amountMinor / 100) / (quantity / 1e8) * 1e8 = amountMinor * 1e16 / (100 * quantity)
	numerator := new(big.Int).Mul(big.NewInt(amountMinor), new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil))
	denominator := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(MinorUnitFactor))

	return RoundRat(new(big.Rat).SetFrac(numerator, denominator))
}
