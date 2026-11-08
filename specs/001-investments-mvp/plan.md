# Implementation Plan: Investments MVP

**Feature**: [spec.md](spec.md) | **Date**: 2026-11-08

## Summary

Add an investment layer to the vendored ezBookkeeping. A pure Go package computes
holdings, cost basis, P&L and XIRR from stored trades. Two new tables hold
securities and trades (plus a small price history). New REST endpoints sit beside
the existing ones. A new Investments section is added to the Vue desktop and
mobile apps. Price refresh uses free, keyless providers behind one interface.

## Technical Context

- **Language/Version**: Go (version in `go.mod`), TypeScript with Vue 3.
- **Storage**: existing xorm datastore; new tables created through the same
  `SyncStructs` mechanism as upstream tables. SQLite is the default, with no
  vendor-specific SQL.
- **Testing**: `go test ./pkg/...` (testify, already a dependency) and `vitest`.
- **Target Platform**: self-hosted server plus PWA (desktop and mobile layouts).
- **Constraints**: offline-capable except price refresh; no secrets in source;
  money maths exact to the minor unit.

## Constitution Check

| Principle | How the plan complies |
|---|---|
| I. Self-hosted, privacy first | Only price lookups leave the server and they send only symbols, never amounts or quantities. |
| II. SQLite default | No new SQL dialect features; simple indexed tables; computations in Go. |
| III. No hardcoded secrets | Providers used need no key. Any future key comes from `EBK_INVESTMENTS_*` env vars. |
| IV. Tests for money maths | `pkg/investments` is written test-first with reference portfolios (spec SC-004). |
| V. Minimal upstream edits | New code lives in new files. Upstream files touched: `cmd/database.go`, `cmd/webserver.go`, `pkg/errs/error.go`, `pkg/uuid/uuid_type.go`, `pkg/settings/setting.go`, `pkg/cron/*`, router and menu files, `en.json`. Each edit is a few lines and marked `// Expbook`. |
| VI. Design system | UI follows `DESIGN.md` (tabular numerals, sign before color, no second palette). |
| VII. Idempotent imports | Unique key on `(uid, source, external_id)`; CSV rows without an id use a SHA-256 of the normalised row. |

## Design Decisions

### Number representation
- Quantity: integer scaled by 1e8 (up to 8 decimal places, as crypto needs).
- Price: integer scaled by 1e8 in the security's currency.
- Fees and dividend amounts: integer minor units (x100), the same convention
  ezBookkeeping uses for transaction amounts.
- Gross amount = round-half-up(quantity x price / 1e14) using `math/big`, so no
  float error. Cost basis is carried as an exact rational during replay and
  rounded only when reported.

### Average-cost replay
Trades are sorted by (time, kind order buy < dividend < sell, id) and replayed:
- buy: `cost += gross + fees`, `qty += q`
- sell: `removed = cost x q / qty`, `realised += (gross - fees) - removed`,
  `cost -= removed`, `qty -= q`; error if `q > qty`
- dividend: `dividends += amount`
This makes results independent of entry order and lets every create, edit or
delete be validated by replaying the whole history of that security.

### XIRR
Cash flows per security: buys negative (gross + fees), sells positive
(gross - fees), dividends positive, plus current value as a final positive flow
dated "now". Solved with Newton iteration and a bisection fallback. Returns
"unavailable" when there is no sign change, history shorter than one day, or no
convergence. Portfolio XIRR converts every flow with today's exchange rate
(documented simplification).

### Currency conversion
Backend reads the app's existing latest exchange rates and converts holding
values to the user's default currency. A currency without a rate stays in its own
currency, is excluded from totals, and is listed in `unconvertedCurrencies`.

### Price providers
One interface `PriceProvider { Fetch(ctx, symbols) map[string]Quote }` with:
- `yahoo`: chart endpoint, keyless, covers NSE/BSE (`RELIANCE.NS`), US and EU tickers, crypto (`BTC-USD`).
- `amfi`: the public AMFI `NAVAll.txt` file for Indian mutual funds, keyed by scheme code.
- `manual`: owner sets the price.
Providers are selected per security (`price_source`, `price_symbol`). Base URLs are
injectable so tests use `httptest`. A daily cron job and the Refresh button call the
same service function.

### Import
CSV columns (header required): `date, type, symbol, quantity, price, fees, currency, name, security_type, external_id, account, note`.
Only `date, type, symbol` and the quantity/price (or amount for dividends) are mandatory.
`import.json` supports `dryRun` for the preview. The broker phase calls the same
core with `source = broker` and real external ids.

### Not done on purpose
- Existing transaction import framework is not reused: it targets account
  transactions and its column mapping UI does not fit trades. A small dedicated
  importer is simpler and fully testable.
- No separate `investment_account` table. A trade may reference an existing
  account id for information only, which avoids duplicating account management.

## Project Structure

```text
pkg/investments/            pure logic: decimal, ledger, xirr, portfolio, csv   (+ _test.go)
pkg/investments/prices/     provider interface, yahoo, amfi                       (+ _test.go)
pkg/models/security.go, investment_trade.go, security_price.go
pkg/services/securities.go, investment_trades.go, investment_prices.go
pkg/api/investments.go
pkg/errs/investment.go
pkg/cron/cron_jobs.go       + RefreshSecurityPricesJob (edit)
cmd/database.go, cmd/webserver.go   (table sync and routes, edits)
src/models/investment.ts
src/lib/investments.ts      formatting helpers (sign, tabular) and client-side types
src/stores/investment.ts
src/views/desktop/investments/ListPage.vue (+ dialogs)
src/views/mobile/investments/ListPage.vue, TradeEditPage.vue
src/router/desktop.ts, mobile.ts, menu entries (edits)
src/locales/en.json         (edit)
```

## Phases

1. **Core maths** (test-first): `pkg/investments` decimal, ledger, xirr, portfolio, csv.
2. **Persistence and API**: models, services, handlers, routes, errors, table sync.
3. **Prices**: providers, refresh service, cron job.
4. **Frontend**: models, API client, store, desktop page and dialogs, mobile pages.
5. **Verification**: Go and vitest suites, a live smoke test with real HTTP calls
   against a running server, and Impeccable/Taste checks (spec SC-006).

## Risks

- Yahoo's endpoint is unofficial and may change. Mitigation: provider interface,
  manual pricing always available, failures reported per security.
- Upstream drift: kept small by the minimal-edit rule and the untouched first commit.
