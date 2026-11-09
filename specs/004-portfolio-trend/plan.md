# Plan: portfolio trend

## Backend
- `GET /api/v1/investments/history.json?startTime=&stepDays=` returns `{baseCurrency, ratesAvailable, unconvertedCurrencies, firstTradeTime, points[{time, invested, value, estimated}]}` in the profile currency (minor units).
- `pkg/investments/history.go`: `HistoryTimes` (even grid, capped at 600 points, always ends now) and `BuildHistory` (replays the trades known at each date with the average-cost method, values at the latest earlier price, flags estimates).
- `pkg/services/investment_history.go`: loads securities, trades and the full price history, builds the inputs.
- A `startTime` of 0 means the first trade; a start earlier than the first trade is allowed (zeros) so the chart lines up with account history.

## Frontend (`src/expbook/portfolio-trend/`)
- `lib.ts`: pure helpers (merge, carry-forward, scale, paths, nearest point) with Vitest tests.
- `store.ts`: loads the existing `asset_trends.json` (all history) and the new history; the account totals and rows are computed values so they follow the display currency, rates and the hidden-account setting.
- `PortfolioTrendCard.vue`: dependency-free SVG chart (the mobile app has no chart library); one component for desktop and mobile.

## Touched upstream lines (marked `// Expbook`)
- `cmd/webserver.go`: one route.
- `src/views/desktop/HomePage.vue`, `src/views/mobile/HomePage.vue`: the card above the dashboard.
- `src/views/desktop/investments/ListPage.vue`, `src/views/mobile/investments/ListPage.vue`: the card above the totals.
- `vite.config.ts`: the card goes in the `common` chunk, because both apps use it (a chunk of its own creates a cycle with the vendor chunk).
- `src/locales/en.json`: new strings.
