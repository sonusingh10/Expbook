# API Contract: Investments

Base path `/api/v1`, same envelope and auth as other endpoints
(`{"success": true, "result": ...}`). Ids are strings in JSON like upstream.
Quantities and prices are exchanged as decimal strings (for example `"10.5"`,
`"123.4567"`); fees and amounts as integers in minor units.

| Method | Path | Purpose |
|---|---|---|
| GET | `/investments/securities/list.json` | All securities |
| POST | `/investments/securities/add.json` | Create a security |
| POST | `/investments/securities/modify.json` | Edit a security |
| POST | `/investments/securities/delete.json` | Delete (refused if it has trades) |
| POST | `/investments/securities/set_price.json` | Set a manual price |
| GET | `/investments/trades/list.json?securityId=` | List trades, newest first |
| POST | `/investments/trades/add.json` | Create a trade (validated by replay) |
| POST | `/investments/trades/modify.json` | Edit a trade (validated by replay) |
| POST | `/investments/trades/delete.json` | Delete a trade (validated by replay) |
| POST | `/investments/trades/import.json` | CSV preview or import (`dryRun`) |
| POST | `/investments/trades/sync.json` | Idempotent batch of external trades (broker phase) |
| GET | `/investments/summary.json` | Holdings, totals, allocation |
| POST | `/investments/prices/refresh.json` | Refresh prices from providers |

## Key shapes

`trades/add.json` request: `securityId, tradeType (buy|sell|dividend), time, quantity, price, amount, fees, accountId?, comment?`

`trades/import.json` request: `{ "csv": "...", "dryRun": true }`
response: `{ "rows": [{ "line": 2, "status": "valid|duplicate|error", "message": "" }], "created": 0, "skipped": 0, "failed": 0, "createdSecurities": 0 }`

`summary.json` response:
`{ "baseCurrency": "INR", "holdings": [...], "totals": { "invested": 0, "value": 0, "unrealised": 0, "realised": 0, "dividends": 0, "xirr": null }, "allocationByType": [...], "allocationByCurrency": [...], "unconvertedCurrencies": [] }`

`prices/refresh.json` response: `{ "results": [{ "securityId": "1", "ok": true, "price": "123.4", "message": "" }] }`

## Errors

New normal-error subcategory `Investment` (22): security not found, trade not found,
sell exceeds holdings, invalid quantity/price, security in use, duplicate symbol,
import content empty or malformed.
