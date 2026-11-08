# Feature Specification: Broker Sync (Fyers)

**Feature Branch**: `002-broker-sync` (work happens on `main`; the directory name is the identifier)
**Created**: 2026-11-08
**Status**: Phase 1 built (file-based sync); phase 2 (in-app, scheduled) specified, not built
**Input**: User description: "Investment entry is manual plus CSV first, broker sync later." The owner trades through Fyers (NSE/BSE) and already has a Fyers MCP server in Cursor.

## Scope and phases

| Phase | What | Status |
|---|---|---|
| 1 | A converter script turns saved Fyers trade-book and holdings JSON into the existing `trades/sync.json` payload and can post it. The server endpoint (`source=broker`, idempotent on `external_id`) already exists from the investments MVP. | built and tested |
| 2 | Pull from the broker inside Expbook (button and schedule), with OAuth handled by the server. | specified below, not built |

Phase 1 deliberately keeps the broker out of the server: no broker credentials are stored by Expbook, and the owner stays in control of when data moves.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bring Fyers trades into Expbook (Priority: P1)

The owner exports or fetches their Fyers trade history, runs one command, and sees the trades in Investments. Running it again changes nothing.

**Independent Test**: Run the converter on the sample trade book with `--post`, check the holdings, run it again, and check that zero trades were created the second time.

**Acceptance Scenarios**:

1. **Given** a Fyers trade book with buys and sells of cash-market equities, **When** the owner runs the converter with `--post`, **Then** each trade appears once with source "broker", the securities that did not exist are created, and holdings match the broker.
2. **Given** the same file, **When** it is run again, **Then** the server reports every trade as a duplicate and creates none.
3. **Given** `--dry-run`, **When** the owner runs it, **Then** the server validates and reports counts and nothing is saved.
4. **Given** rows the converter cannot map (derivatives, currency, commodity, unknown side, missing quantity), **When** it runs, **Then** each is listed with its position and reason on stderr, the rest are still converted, and nothing is guessed.
5. **Given** a sell whose buys are older than the history supplied, **When** the sync runs, **Then** the server refuses that row (sell exceeds holdings) and says which line, rather than creating a negative holding.

### User Story 2 - Opening balances for older history (Priority: P2)

The owner has holdings bought years ago but only this financial year's trade book. They supply the broker holdings and an opening date, and Expbook records each holding as a buy at the broker's average cost on that date.

**Acceptance Scenarios**:

1. **Given** holdings of 8 units at average cost 3200.25 and `--opening-date 2025-04-01`, **When** synced, **Then** a buy of 8 at 3200.25 dated that day exists with the id `fyers-opening:<symbol>:<date>`.
2. **Given** zero-quantity holdings, **When** converted, **Then** they are skipped.
3. **Given** opening balances plus the later trade book, **When** both are synced in one run, **Then** later sells validate against the opening quantity.

### User Story 3 - Prices for synced securities (Priority: P2)

Synced securities are created with a manual price. The owner sets a price source once per security (Yahoo `SYMBOL.NS`) or sets a price by hand.

**Acceptance Scenarios**:

1. **Given** a synced security with no price, **When** the owner views holdings, **Then** value and gain show as unavailable with the existing warning (never zero).

### User Story 4 - Scheduled in-app pull (Priority: P3, phase 2, not built)

The owner connects Fyers once in Settings and Expbook pulls new trades daily after market close.

**Acceptance Scenarios**:

1. **Given** a connected broker, **When** the scheduled job runs, **Then** new trades are synced through the same idempotent path and a summary is logged.
2. **Given** an expired broker session, **When** the job runs, **Then** the failure is reported in the UI and nothing is partially saved.

## Requirements

- **FR-001**: The converter MUST produce one sync item per cash-market equity trade, with `externalId = fyers:<trade number>` (at most 64 characters), symbol without exchange prefix or series suffix, currency INR.
- **FR-002**: The converter MUST NOT map derivatives, currency or commodity trades. They are skipped and listed.
- **FR-003**: Trade dates MUST be the Indian calendar day of the trade, stored at 12:00 UTC like every other trade.
- **FR-004**: When a trade has no trade number, the id MUST be built from the order number, quantity and price so the same row always gets the same id.
- **FR-005**: The Expbook API token MUST come from `EXPBOOK_TOKEN` and the server URL from `EXPBOOK_URL`. Tokens and broker data MUST NOT be written to the repository. Sample fixtures are synthetic and labelled `DEMO`.
- **FR-006**: The converter MUST NOT contact the broker. Getting the JSON is a separate, owner-driven step.
- **FR-007**: Opening balances MUST use the broker's average cost and carry a comment saying so.
- **FR-008** (phase 2): Broker OAuth tokens MUST be stored encrypted with a key from the environment, and the app MUST work without the broker configured.
- **FR-009**: Charges (brokerage, STT, GST) are not in the Fyers trade book, so synced fees are 0. The owner can add fees later by editing the trade or by a charges import (out of scope here).

## Edge Cases

- Same trade appears in two exports: skipped by id, in the file and on the server.
- A symbol changes (corporate action, rename): the old and new symbols stay separate securities. Merge is out of scope.
- Splits and bonuses: the broker trade book does not contain them; holdings can drift until an adjustment trade is entered by hand. Documented, not solved.
- Mutual funds bought through Fyers have a different identity (scheme code) and are skipped; use CSV import.
- A file with an unknown shape fails with the top-level keys it found, not with a stack trace.

## Success Criteria

- **SC-001**: Syncing the sample file twice leaves the trade count unchanged the second time (verified live against the local server).
- **SC-002**: Every skipped or failed row is reported with its position and a reason.
- **SC-003**: No credential appears in the repository, the payload file, or the command line.

## Assumptions

- The Fyers MCP tools (`get_trade_history`, `get_holdings`) return the same fields as the Fyers API v3 trade book and holdings (`symbol`, `side`, `tradedQty`, `tradePrice`, `orderDateTime`, `tradeNumber`, `segment`; `quantity`, `costPrice`). The converter accepts common aliases and reports unknown shapes clearly. It was tested on synthetic fixtures in that shape; it has not yet been run on the owner's live data.
- The Fyers MCP server exposes `get_trade_history(from_date, to_date, exchange_type, segment_type, symbol)` and `get_holdings()` (inspected read-only; no broker call was made while building this phase).
