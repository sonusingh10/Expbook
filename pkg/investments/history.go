package investments

import "sort"

// PricePoint is one recorded unit price (scaled 1e8) of a security
type PricePoint struct {
	Time  int64
	Price int64
}

// HistoryInput is everything needed to value one security at past dates
type HistoryInput struct {
	Currency string
	Trades   []Trade
	Prices   []PricePoint // any order
}

// HistoryPoint is the portfolio at one date, in the base currency (minor units)
type HistoryPoint struct {
	Time     int64
	Invested int64 // cost basis of the units held at that date
	Value    int64 // market value at that date
	// Estimated is true when at least one holding had no recorded price yet, so its cost basis stands in for its value.
	// The chart draws such points differently so a guess is never mistaken for a market value.
	Estimated bool
}

// HistoryTimes returns the dates to chart: start, start+step, ... and always end as the last one.
// It returns at most maxPoints dates (the step is widened when needed) and nothing when end is before start.
func HistoryTimes(start int64, end int64, step int64, maxPoints int) []int64 {
	if step <= 0 || end < start || maxPoints < 2 {
		return nil
	}

	span := end - start

	if span/step+2 > int64(maxPoints) {
		step = span/int64(maxPoints-2) + 1
	}

	times := make([]int64, 0, span/step+2)

	for t := start; t < end; t += step {
		times = append(times, t)
	}

	return append(times, end)
}

// priceAt returns the last price recorded at or before t (prices must be sorted by time)
func priceAt(prices []PricePoint, t int64) (int64, bool) {
	i := sort.Search(len(prices), func(i int) bool { return prices[i].Time > t })

	if i == 0 {
		return 0, false
	}

	return prices[i-1].Price, true
}

// BuildHistory values the portfolio at each time. Trades after a date never affect that date.
// Securities without an exchange rate to base are left out of every point, never counted as zero cost or value
// in a way that hides them: their currencies are returned so the caller can tell the user.
func BuildHistory(inputs []HistoryInput, base string, rates Rates, times []int64) ([]HistoryPoint, []string) {
	sorted := make([][]PricePoint, len(inputs))
	ledgers := make([][]Trade, len(inputs))

	for i, in := range inputs {
		prices := make([]PricePoint, len(in.Prices))
		copy(prices, in.Prices)
		sort.SliceStable(prices, func(a, b int) bool { return prices[a].Time < prices[b].Time })
		sorted[i] = prices
		ledgers[i] = SortTrades(in.Trades)
	}

	unconverted := map[string]bool{}
	points := make([]HistoryPoint, 0, len(times))

	for _, t := range times {
		point := HistoryPoint{Time: t}

		for i, in := range inputs {
			upTo := ledgers[i][:sort.Search(len(ledgers[i]), func(k int) bool { return ledgers[i][k].Time > t })]

			if len(upTo) == 0 {
				continue
			}

			holding, err := Replay(upTo)

			if err != nil || holding.Quantity == 0 {
				continue
			}

			cost, ok := rates.Convert(holding.TotalCost, in.Currency, base)

			if !ok {
				unconverted[in.Currency] = true
				continue
			}

			value := cost
			price, hasPrice := priceAt(sorted[i], t)

			if hasPrice {
				converted, ok := rates.Convert(GrossMinor(holding.Quantity, price), in.Currency, base)

				if ok {
					value = converted
				} else {
					point.Estimated = true
				}
			} else {
				point.Estimated = true
			}

			point.Invested += cost
			point.Value += value
		}

		points = append(points, point)
	}

	missing := make([]string, 0, len(unconverted))

	for currency := range unconverted {
		missing = append(missing, currency)
	}

	sort.Strings(missing)

	return points, missing
}
