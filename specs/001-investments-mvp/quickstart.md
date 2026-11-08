# Quickstart: Investments MVP

## Run it

```sh
./run-local.sh            # builds if needed, creates .env with a random secret, serves on http://localhost:8080
```

Sign up, then open **Investments** (chart icon in the desktop top bar; chart icon in the mobile home navbar).

## Try it in two minutes

1. **Add Security**: name `Infosys`, symbol `INFY`, type Stock, currency INR, price source Manual.
2. **Add Trade**: buy 10 at 100 with fees 5. Holdings shows quantity 10, average cost 100.5.
3. **Set price** 120. Value 1,200, gain +195 (+19.40%).
4. **Import CSV**: *Download template*, edit it, choose the file, press **Preview**, then **Import**. Import the same file again: nothing is created.
5. Add a USD security. Totals convert to your default currency with the exchange rates the app already has; if a rate is missing the holding is flagged and left out of totals.

CSV columns: `date,type,symbol,name,security_type,currency,quantity,price,amount,fees,external_id,comment`
(`date`, `type` = buy/sell/dividend and `symbol` are required; buys and sells need `quantity` plus `price` or `amount`; dividends need `amount`).

## Price sources

| Source | Needs | Notes |
|---|---|---|
| Manual | nothing | You type the price. |
| Yahoo Finance | price symbol such as `INFY.NS`, `AAPL` | Unofficial keyless endpoint; may be rate limited or change. Only the symbol is sent. |
| AMFI | AMFI scheme code | Indian mutual fund NAV from `amfi.com`. |

A daily job (`enable_refresh_security_prices` in `conf/ezbookkeeping.ini`, on by default) refreshes every non-manual security; **Refresh prices** does it on demand. One failing security never blocks the others, and a failed refresh never overwrites a price.

## Verify

```sh
go test ./pkg/investments/... ./pkg/services/...   # maths, import, prices, summary, performance budgets
npm test && npm run lint
```

Known unrelated failures: two upstream timezone tests in `pkg/utils` (`*_AmbiguousMidnight`) depend on the host timezone database and fail on this machine without involving Expbook code.

## Acceptance walk-through

| Scenario (spec) | How it is checked | Status |
|---|---|---|
| US1.1 buy 10 @ 100, fee 5: cost 1005, average 100.50 | `ledger_test.go`, smoke test (INFY cost 60300 for 6 units at 100.5) | verified |
| US1 sell more than held is rejected, also when editing or deleting an earlier trade | `investments_test.go`, smoke test | verified |
| US2 preview shows row-level errors, nothing saved; re-import creates 0 | `investments_test.go` (`TestImportDryRunWritesNothing`, `TestImportTwiceEqualsOnce`), smoke test | verified |
| US3 refresh reports per-security success or failure | `investment_prices_test.go`, smoke test (failure path live) | verified |
| US3 XIRR matches reference | `xirr_test.go` (Excel reference 0.373362535) | verified |
| US4 net worth = accounts + investments with split visible | desktop investments page, first card | built; browser check in ux-pass |
| US5 dividend leaves quantity and average cost unchanged | `ledger_test.go` | verified |
| SC-002 500-row import under 10 s | `TestImport500RowsIsFastAndIdempotent` (about 8 ms) | verified |
| SC-005 summary with 5,000 trades under 2 s | `TestSummaryWith5000TradesIsFast` (about 17 ms) | verified |
| SC-006 design checks | Impeccable critique and audit | ux-pass |
