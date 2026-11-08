# Implementation Plan: Broker Sync (Fyers)

**Feature**: [spec.md](spec.md) | **Date**: 2026-11-08

## Summary

Phase 1 reuses the server's existing idempotent import core (`InvestmentTradeService.ImportTrades` with `source=broker`, exposed as `POST /api/v1/investments/trades/sync.json`) and adds a dependency-free Python converter in `scripts/broker/`. No server change was needed. Phase 2 would add a server-side Fyers connector behind the same core.

## Constitution Check

| Principle | How the plan complies |
|---|---|
| I. Self-hosted, privacy first | The converter runs on the owner's machine and talks only to their Expbook server. It never contacts the broker. |
| II. SQLite default | No schema change. |
| III. No hardcoded secrets | Token from `EXPBOOK_TOKEN`, URL from `EXPBOOK_URL`. Fixtures are synthetic. Phase 2 broker tokens: encrypted at rest, key from env. |
| IV. Tests for money maths | Quantities and prices stay decimal strings end to end (no float arithmetic in the converter); 11 unit tests plus a live idempotency run. |
| V. Minimal upstream edits | Zero upstream files touched. |
| VI. Design system | No UI in phase 1. Phase 2 UI follows `DESIGN.md`. |
| VII. Idempotent imports | `externalId = fyers:<trade number>`; unique `(uid, source, external_id)` on the server. |

## Design Decisions

### Why a script first
Broker sessions need an interactive login (Fyers opens a browser). Doing that inside the server means OAuth handling, token storage and refresh, which is real security surface. A file-based phase delivers the value (trades in Expbook, no re-typing) with none of that surface, and it validates the mapping on real data before any server code depends on it.

### Mapping
| Fyers | Expbook sync item |
|---|---|
| `symbol` `NSE:SBIN-EQ` | `symbol` `SBIN` (exchange prefix and series suffix dropped) |
| `side` 1 / -1 | `tradeType` 1 buy / 2 sell |
| `tradedQty`, `tradePrice` | `quantity`, `price` as plain decimal strings |
| `orderDateTime` | `time` = Indian calendar day at 12:00 UTC |
| `tradeNumber` | `externalId` `fyers:<tradeNumber>` |
| segment other than 10, FUT/CE/PE symbols | skipped and reported |
| holdings `quantity`, `costPrice` | buy on `--opening-date`, id `fyers-opening:<symbol>:<date>` |

### Known limits (carried into the spec)
Fees are 0, corporate actions are not in the trade book, mutual funds are skipped, symbols are not unified across exchanges (NSE and BSE codes for one company stay separate).

### Phase 2 outline (not built)
1. `pkg/investments/brokers` with a `Broker` interface (`FetchTrades(ctx, from, to)`, `FetchHoldings(ctx)`), a Fyers implementation using the official OAuth app flow.
2. Settings screen: connect, disconnect, last sync status. Tokens encrypted with AES-256-GCM using a key from `EBK_INVESTMENTS_BROKER_KEY`.
3. A daily cron job after market close that calls the same `ImportTrades` core and records a per-run summary.
4. Behind a feature flag in `ezbookkeeping.ini`; the app works unchanged when it is off.

## Project Structure

```text
scripts/broker/
├── fyers_to_expbook.py          # converter and optional poster
├── test_fyers_to_expbook.py     # unit tests (python3 -m unittest)
└── fixtures/
    ├── fyers_tradebook_sample.json   # synthetic DEMO trades
    └── fyers_holdings_sample.json    # synthetic DEMO holdings
specs/002-broker-sync/{spec,plan,tasks,quickstart}.md
```
