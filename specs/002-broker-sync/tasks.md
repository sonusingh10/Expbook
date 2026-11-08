# Tasks: Broker Sync (Fyers)

## Phase 1: file-based sync (built)

- [x] T001 Confirm the server sync endpoint (`trades/sync.json`, `source=broker`, idempotent, dry run) exists and is covered by tests from the investments MVP
- [x] T002 Inspect the Fyers MCP tool schemas read-only (`get_trade_history`, `get_holdings`); no broker call was made
- [x] T003 Write the converter `scripts/broker/fyers_to_expbook.py` (trades, holdings as opening balances, `--out`, `--post`, `--dry-run`)
- [x] T004 Synthetic fixtures labelled DEMO, including a derivative, a duplicate and a bad row
- [x] T005 Unit tests (11): mapping, noon-UTC dates, ETF detection, skips with reasons, stable ids, fallback id, opening balances, unknown file shape, decimals
- [x] T006 Live check against the local server: dry run, real run (5 trades, 3 new securities), second run creates 0 and skips 5; holdings 12 / 100 / 8 units
- [x] T007 `quickstart.md` with the exact commands and the Fyers MCP prompts to fetch the files

## Phase 2: in-app pull (specified, not built)

- [ ] T101 `Broker` interface and Fyers connector (OAuth app flow), with fakes for tests
- [ ] T102 Encrypted token storage (AES-256-GCM, key from `EBK_INVESTMENTS_BROKER_KEY`), never logged
- [ ] T103 Settings screen (desktop and mobile): connect, disconnect, status, last sync result
- [ ] T104 Daily cron job after market close using the same `ImportTrades` core
- [ ] T105 Optional: set Yahoo price source (`SYMBOL.NS`) automatically for synced NSE equities
- [ ] T106 Optional: import charges (brokerage, STT, GST) and attach them as trade fees
- [ ] T107 Corporate actions: manual adjustment entry for splits and bonuses

## Checkpoints

- Phase 1 is complete when the converter tests pass and the live idempotency run shows 0 created on the second run (done).
- Phase 2 must not start until the converter has been run once on the owner's real Fyers export and the mapping is confirmed.
