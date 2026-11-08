package investments

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScaled(t *testing.T) {
	valid := map[string]int64{
		"10":          1000000000,
		"10.5":        1050000000,
		"0.00000001":  1,
		"-0.5":        -50000000,
		"+2":          200000000,
		".5":          50000000,
		"5.":          500000000,
		" 123.4567 ":  12345670000,
		"0":           0,
		"92233720368": 9223372036800000000,
	}

	for input, expected := range valid {
		actual, err := ParseScaled(input)
		assert.NoError(t, err, input)
		assert.Equal(t, expected, actual, input)
	}

	invalid := []string{"", " ", "-", ".", "1,000", "1e3", "abc", "1.123456789", "1..2", "--1", "9223372036854775808", "92233720369"}

	for _, input := range invalid {
		_, err := ParseScaled(input)
		assert.ErrorIs(t, err, ErrInvalidDecimal, input)
	}
}

func TestParseMinor(t *testing.T) {
	actual, err := ParseMinor("123.45")
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), actual)

	actual, err = ParseMinor("5")
	assert.NoError(t, err)
	assert.Equal(t, int64(500), actual)

	_, err = ParseMinor("1.234")
	assert.ErrorIs(t, err, ErrInvalidDecimal)
}

func TestFormatScaled(t *testing.T) {
	assert.Equal(t, "10", FormatScaled(1000000000))
	assert.Equal(t, "10.5", FormatScaled(1050000000))
	assert.Equal(t, "0.00000001", FormatScaled(1))
	assert.Equal(t, "-0.5", FormatScaled(-50000000))
	assert.Equal(t, "0", FormatScaled(0))
}

func TestFormatMinor(t *testing.T) {
	assert.Equal(t, "118.00", FormatMinor(11800))
	assert.Equal(t, "100.50", FormatMinor(10050))
	assert.Equal(t, "0.05", FormatMinor(5))
	assert.Equal(t, "-1.20", FormatMinor(-120))
	assert.Equal(t, "0.00", FormatMinor(0))
}

func TestParseFormatRoundTrip(t *testing.T) {
	for _, v := range []int64{0, 1, 99999999, 100000000, 123456789012, -7} {
		parsed, err := ParseScaled(FormatScaled(v))
		assert.NoError(t, err)
		assert.Equal(t, v, parsed)
	}
}

func TestGrossMinor(t *testing.T) {
	quantity, _ := ParseScaled("10")
	price, _ := ParseScaled("100")
	assert.Equal(t, int64(100000), GrossMinor(quantity, price))

	// 3 units at 33.335 = 100.005 -> 100.01 (half rounds up)
	quantity, _ = ParseScaled("3")
	price, _ = ParseScaled("33.335")
	assert.Equal(t, int64(10001), GrossMinor(quantity, price))

	// fractional mutual fund units: 123.4567 x 45.6789 = 5639.36625363 -> 5639.37
	quantity, _ = ParseScaled("123.4567")
	price, _ = ParseScaled("45.6789")
	assert.Equal(t, int64(563937), GrossMinor(quantity, price))

	// very small price (crypto) does not lose precision: 1,000,000 x 0.00001234 = 12.34
	quantity, _ = ParseScaled("1000000")
	price, _ = ParseScaled("0.00001234")
	assert.Equal(t, int64(1234), GrossMinor(quantity, price))
}

func TestRoundRat(t *testing.T) {
	assert.Equal(t, int64(3), RoundRat(big.NewRat(5, 2)))
	assert.Equal(t, int64(-3), RoundRat(big.NewRat(-5, 2)))
	assert.Equal(t, int64(2), RoundRat(big.NewRat(7, 4)))
	assert.Equal(t, int64(-2), RoundRat(big.NewRat(-7, 4)))
	assert.Equal(t, int64(0), RoundRat(big.NewRat(1, 3)))
	assert.Equal(t, int64(0), RoundRat(new(big.Rat)))
}

func TestPriceFromAmount(t *testing.T) {
	quantity, _ := ParseScaled("4")
	// 520.00 for 4 units = 130 per unit
	assert.Equal(t, int64(13000000000), PriceFromAmount(52000, quantity))
	assert.Equal(t, int64(0), PriceFromAmount(52000, 0))
}
