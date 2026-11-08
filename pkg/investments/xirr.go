package investments

import (
	"math"
	"sort"
)

// CashFlow is a dated amount. Negative means money paid out, positive means money received.
type CashFlow struct {
	Time   int64 // unix seconds
	Amount float64
}

const (
	secondsPerYear = 365.0 * 24 * 3600
	minXirrSpan    = int64(24 * 3600)
	xirrLowerBound = -0.999999
	xirrUpperBound = 1000.0
)

// XIRR returns the annualised internal rate of return of the cash flows (365-day year, like spreadsheets).
// ok is false when the rate cannot be determined: no sign change, a span shorter than a day, or no convergence.
func XIRR(flows []CashFlow) (rate float64, ok bool) {
	if len(flows) < 2 {
		return 0, false
	}

	sorted := make([]CashFlow, len(flows))
	copy(sorted, flows)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Time < sorted[j].Time })

	hasNegative, hasPositive := false, false

	for _, f := range sorted {
		if f.Amount < 0 {
			hasNegative = true
		} else if f.Amount > 0 {
			hasPositive = true
		}
	}

	if !hasNegative || !hasPositive {
		return 0, false
	}

	start := sorted[0].Time

	if sorted[len(sorted)-1].Time-start < minXirrSpan {
		return 0, false
	}

	years := make([]float64, len(sorted))

	for i, f := range sorted {
		years[i] = float64(f.Time-start) / secondsPerYear
	}

	npv := func(r float64) float64 {
		total := 0.0

		for i, f := range sorted {
			total += f.Amount / math.Pow(1+r, years[i])
		}

		return total
	}

	derivative := func(r float64) float64 {
		total := 0.0

		for i, f := range sorted {
			total -= years[i] * f.Amount / math.Pow(1+r, years[i]+1)
		}

		return total
	}

	// Newton iteration first
	r := 0.1

	for iteration := 0; iteration < 100; iteration++ {
		value := npv(r)
		slope := derivative(r)

		if slope == 0 || math.IsNaN(slope) || math.IsInf(slope, 0) {
			break
		}

		next := r - value/slope

		if next <= xirrLowerBound || next >= xirrUpperBound || math.IsNaN(next) {
			break
		}

		if math.Abs(next-r) < 1e-10 {
			if math.Abs(npv(next)) < 1e-6*totalAbs(sorted) {
				return next, true
			}

			break
		}

		r = next
	}

	// Bisection fallback
	lo, hi := xirrLowerBound, xirrUpperBound
	fLo, fHi := npv(lo), npv(hi)

	if math.IsNaN(fLo) || math.IsNaN(fHi) || fLo*fHi > 0 {
		return 0, false
	}

	for iteration := 0; iteration < 300; iteration++ {
		mid := (lo + hi) / 2
		fMid := npv(mid)

		if math.Abs(fMid) < 1e-9*totalAbs(sorted) || (hi-lo) < 1e-12 {
			return mid, true
		}

		if fLo*fMid <= 0 {
			hi = mid
		} else {
			lo, fLo = mid, fMid
		}
	}

	return 0, false
}

func totalAbs(flows []CashFlow) float64 {
	total := 0.0

	for _, f := range flows {
		total += math.Abs(f.Amount)
	}

	if total == 0 {
		return 1
	}

	return total
}
