# Quickstart: display currency

1. Run the app (`./run-local.sh`) and sign in.
2. Click the currency button at the top right (desktop) or in the Home navbar (mobile).
3. Pick PLN. The page reloads and totals show in PLN. The button is highlighted while the display currency differs from the profile currency.
4. Open the same account in another browser: the button already shows PLN.
5. Pick the profile currency to go back.

Checks: `go test ./pkg/services/ -run "DisplayCurrency|Preference"`, `npx vitest run src/expbook`.
