# Plan: display currency

## Isolation (so upstream updates stay easy)
Everything new lives in files upstream does not have:

| Layer | New files |
|---|---|
| Backend | `pkg/models/expbook_preference.go`, `pkg/services/expbook_preferences.go`, `pkg/api/expbook_preferences.go`, test `pkg/services/expbook_preferences_test.go` |
| Frontend | `src/expbook/display-currency/` (`lib.ts`, `store.ts`, `picker.ts`, `investments.ts`, `DisplayCurrencyChip.vue`, `DisplayCurrencyLink.vue`, `DisplayCurrencyPopover.vue`, tests) |

Table `expbook_preference` (uid, pref_key, pref_value, updated_unix_time) is generic, so a later preference needs no migration.

## Why not the upstream cloud settings
They are opt-in ("Settings Sync") and use a fixed key list on both server and client, so using them would mean editing several upstream files and would not work for users who keep sync off.

## API
- `GET  /api/v1/expbook/preferences/display_currency.json` returns `{currency}` ("" when none).
- `POST /api/v1/expbook/preferences/display_currency/update.json` with `{currency}` ("" clears).

## Frontend flow
1. The store reads a browser cache (keyed by user name) so the first paint is already right.
2. On mount the button loads the exchange rates if missing and asks the server; if the saved value differs from the cache, the page reloads once.
3. Picking a currency saves it, then reloads the page. Many screens cache totals when data loads, so one reload keeps them all consistent.
4. `useDisplayCurrencyStore().currency` is the single value the display sites read.

## Upstream lines touched (all marked `// Expbook` or listed here)
- `cmd/database.go` (table sync), `cmd/webserver.go` (two routes).
- `vite.config.ts`: `src/expbook/**/*.ts` joins the `common` chunk. Without it the stores and the module end up in chunks that import each other and the app fails to start.
- `src/locales/en.json`: 5 keys.
- `MainPageLayout.vue` and mobile `HomePage.vue`: mount the button.
- Display sites switched from `userStore.currentUserDefaultCurrency` to the display store: `stores/{account,statistics,overview,transaction}.ts`, `views/base/{accounts/AccountListPageBase,overview/*WidgetBase (4),statistics/StatisticsTransactionPageBase,transactions/TransactionListPageBase}.ts`, `MonthlyIncomeAndExpenseChart.vue`, `NetAssetsTrendWidget.vue`, `TransactionCalendarHeatmapWidget.vue`.
- Investments pages (ours).

When merging upstream, re-run `rg "displayCurrencyStore" src` and check each hit; any new upstream site that shows a total should read the display store too.
