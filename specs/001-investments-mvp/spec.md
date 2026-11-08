# Feature Specification: Investments MVP

**Feature Branch**: `001-investments-mvp` (work happens on `main`; the directory name is the identifier)
**Created**: 2026-11-08
**Status**: Draft
**Input**: User description: "Add investment tracking to my expense tracker: holdings, buys and sells, SIPs, dividends, cost basis, profit and loss, XIRR, allocation, and a net worth that combines cash and investments. Manual entry and CSV import first, broker sync later."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Record trades and see holdings (Priority: P1)

The owner records that they bought or sold units of a stock, ETF, mutual fund or crypto asset, on a date, at a price, with optional fees, and then sees a holdings table showing for each security: quantity held, average cost, total cost, latest price, current value, and unrealised gain (amount and percent).

**Why this priority**: Without trades and holdings there is no investment tracking at all. Every other story builds on it.

**Independent Test**: Add one security, record two buys and one partial sell, set a price, and verify the holdings row shows the expected quantity, average cost, value and gain. Delivers a usable portfolio view on its own.

**Acceptance Scenarios**:

1. **Given** no securities, **When** the owner adds a security (name, symbol, type, currency) and a buy of 10 units at 100 with fee 5, **Then** the holdings table shows 10 units, total cost 1005, average cost 100.50.
2. **Given** that holding, **When** the owner records a sell of 4 units at 130, **Then** quantity is 6, average cost is unchanged at 100.50, and realised profit of 118.00 (4 x 130 - 4 x 100.50) is shown.
3. **Given** a holding, **When** the latest price is 120, **Then** current value is 720 and unrealised gain is 117.00 (+19.40%).
4. **Given** a sell quantity larger than the quantity held on that date, **When** the owner saves it, **Then** the app refuses with a clear message and nothing is saved.
5. **Given** a security with no price yet, **When** the owner views holdings, **Then** value and gain show as unavailable (not zero) and the row explains why.

---

### User Story 2 - Import trades from CSV safely (Priority: P1)

The owner uploads a CSV of past trades (for example exported from a broker or spreadsheet), reviews a preview with any problems flagged, and imports. Importing the same file again does not create duplicates.

**Why this priority**: Years of history cannot realistically be typed by hand, and the constitution requires idempotent imports.

**Independent Test**: Import a 5-row CSV, then import it again, and verify the holdings and the trade count are identical after the second import.

**Acceptance Scenarios**:

1. **Given** a valid CSV, **When** the owner previews it, **Then** each row is shown as valid or with a specific error (row number and reason) before anything is saved.
2. **Given** a previewed CSV with valid rows, **When** the owner confirms, **Then** all valid rows are saved and securities that do not exist yet are created.
3. **Given** a file already imported, **When** the same file is imported again, **Then** zero new trades are created and the owner is told how many rows were skipped as duplicates.
4. **Given** a CSV with an invalid row, **When** the owner imports, **Then** invalid rows are skipped and reported; valid rows are not blocked.

---

### User Story 3 - Prices, performance and allocation (Priority: P2)

The owner refreshes prices on demand (and they also refresh on a schedule) so values stay current. They see portfolio-level totals (invested, current value, unrealised and realised gain, XIRR) and an allocation breakdown by security type and by currency.

**Why this priority**: Makes the holdings meaningful and answers "how are my investments doing", but the app is already useful with manually entered prices.

**Independent Test**: With a configured price source (or a fake one), refresh prices and verify the holdings values and portfolio XIRR update; verify allocation percentages sum to 100%.

**Acceptance Scenarios**:

1. **Given** securities with a price source, **When** the owner taps Refresh prices, **Then** latest prices and their timestamps update and a per-security success or failure message is shown.
2. **Given** one security fails to refresh, **When** the refresh finishes, **Then** other securities are still updated and the failure is reported, never silently ignored.
3. **Given** a portfolio with dated buys, sells and dividends, **When** the owner views performance, **Then** the annualised return (XIRR) is shown, or marked unavailable when it cannot be computed (for example a single day of history).
4. **Given** holdings in several currencies, **When** the owner views totals, **Then** totals are shown in the owner's base currency using the app's existing exchange rates, and the allocation views add up to 100%.
5. **Given** no network access, **When** the owner uses the app, **Then** recording, importing and viewing all still work; only price refresh reports a network failure.

---

### User Story 4 - Net worth including investments (Priority: P2)

The owner sees one net worth figure that combines their existing cash and account balances with the current value of investments, and can see the split between the two.

**Why this priority**: This is the headline promise of one ledger, but it depends on holdings and prices.

**Independent Test**: With known account balances and known holdings value, verify net worth equals their sum in the base currency.

**Acceptance Scenarios**:

1. **Given** account balances and holdings, **When** the owner opens the investments overview, **Then** net worth = account assets minus liabilities plus current investment value, with the split visible.
2. **Given** an investment trade linked to an existing account, **When** it is viewed, **Then** the link is shown but the account balance is not changed by the trade (see Assumptions).

---

### User Story 5 - Dividends and SIPs (Priority: P3)

The owner records dividends or interest received and recurring SIP purchases. Dividends count toward return (XIRR and realised income) but not toward quantity held. A SIP is a regular recurring buy entered as ordinary buy trades.

**Why this priority**: Improves accuracy of returns but is not needed for a first useful release.

**Independent Test**: Add a dividend and verify it appears in income totals and raises XIRR but does not change quantity.

**Acceptance Scenarios**:

1. **Given** a holding, **When** the owner records a dividend of 50, **Then** quantity and average cost do not change and total dividends increase by 50.
2. **Given** a monthly SIP across three months entered or imported as buys, **When** the owner views the holding, **Then** average cost reflects all three purchases.

---

### Edge Cases

- Selling more than held (including selling before any buy by date) is rejected.
- Zero or negative quantity or price on a buy or sell is rejected; a dividend needs a positive amount.
- Fractional quantities (mutual fund units, crypto) up to 8 decimal places are supported without drift.
- Trades entered out of date order produce the same holdings as if entered in order.
- Deleting a trade recomputes holdings; deleting a security that still has trades is refused.
- A security priced in a currency that has no exchange rate shows its values in its own currency and is excluded from the base-currency totals with a visible warning.
- Duplicate rows inside the same CSV file count as duplicates.
- A price refresh returning an implausible value (zero, negative, missing) is ignored and reported.
- Very long histories (thousands of trades) still load the holdings view quickly.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The owner MUST be able to create, edit and delete securities with name, symbol, type (stock, ETF, mutual fund, crypto, other), currency and an optional price source identifier.
- **FR-002**: The owner MUST be able to record buy, sell and dividend trades with date, quantity (not for dividends), price per unit or amount, fees and an optional note and linked account, and edit or delete them.
- **FR-003**: The system MUST reject a sell that exceeds the quantity held at that date, and any non-positive quantity or price on buys and sells.
- **FR-004**: The system MUST compute, per security, quantity held, average cost, total cost, realised profit and loss, dividends received, latest price, current value and unrealised gain (amount and percent), using the average-cost method.
- **FR-005**: The system MUST produce the same results regardless of the order in which trades were entered.
- **FR-006**: The system MUST show clearly when a value cannot be computed (no price, no exchange rate) instead of showing zero.
- **FR-007**: The system MUST compute portfolio totals in the owner's base currency using the app's existing exchange rates, and show unconverted holdings separately.
- **FR-008**: The system MUST compute the annualised return (XIRR) for each security and for the whole portfolio from dated cash flows plus current value, and report unavailable when not computable.
- **FR-009**: The system MUST show allocation by security type and by currency, summing to 100%.
- **FR-010**: The system MUST let the owner set a price manually for a security and refresh prices from a configured source on demand and on a daily schedule.
- **FR-011**: A failed price refresh for one security MUST NOT block others and MUST be reported with the reason.
- **FR-012**: The system MUST import trades from CSV with a preview step, row-level validation messages, creation of missing securities, and idempotent behaviour so re-importing never duplicates trades.
- **FR-013**: Every imported or synced trade MUST record its source and an external identifier so it can be traced and removed.
- **FR-014**: The system MUST show net worth as account balances plus investment value, with the split visible.
- **FR-015**: All investment screens MUST work on desktop and mobile layouts, in light and dark mode, with designed empty, loading and error states.
- **FR-016**: The feature MUST work without network access except for price refresh, and MUST NOT send transaction or holding data to third parties.
- **FR-017**: Credentials for any price provider MUST be read from configuration or environment, never stored in source or in the database in plain text.
- **FR-018**: The data model MUST work on SQLite and remain compatible with the MySQL and PostgreSQL support already in the app.

### Key Entities

- **Security**: something that can be held. Name, symbol, type, currency, optional price source identifier, latest price and when it was fetched.
- **Trade**: one dated event on a security. Kind (buy, sell, dividend), quantity, price per unit or amount, fees, optional linked account, note, source (manual, csv, broker) and external identifier.
- **Price**: a dated price for a security (latest and history), with where it came from.
- **Holding** (derived, not stored): the computed position and performance of a security from its trades and latest price.
- **Portfolio summary** (derived): totals, XIRR, allocation, and net worth split.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The owner can record a trade in under 30 seconds on a phone and see the holdings update immediately.
- **SC-002**: Importing a 500-row CSV completes with a clear result summary in under 10 seconds.
- **SC-003**: Re-importing the same file creates exactly 0 new trades.
- **SC-004**: For a reference set of test portfolios, quantity, average cost, realised and unrealised profit match hand-calculated values exactly to the minor currency unit, and XIRR matches a reference to within 0.01 percentage points.
- **SC-005**: Holdings and the portfolio summary load in under 2 seconds with 5,000 trades.
- **SC-006**: Every investment screen passes the design checks defined in `DESIGN.md` (contrast, sign-before-color, empty states) with no P0 or P1 findings.
- **SC-007**: With the network disabled, all recording and viewing flows still work.

## Assumptions

- Single user, self-hosted; there is no sharing between users.
- Cost basis uses the **average cost** method. FIFO and tax-lot reporting are out of scope.
- Trades are **memo records** and do not create or change ordinary account transactions or balances. Linking a trade to an account is informational. Reconciling cash movement automatically is a later phase.
- The owner's base currency is the one already set in the app; the owner mixes INR with foreign currencies, so multi-currency totals use the existing exchange rates.
- Price sources (decided in the plan): Yahoo-style chart data for stocks and ETFs (including NSE/BSE tickers), AMFI NAV file for Indian mutual funds, and manual pricing for everything else. All are optional and configured, and none need an API key.
- SIPs are represented as repeated buy trades; automatic generation of future SIP trades is out of scope.
- Corporate actions (splits, bonuses, mergers) are handled by manually adjusting quantity and price through trades; automatic handling is out of scope.
- Broker sync is a separate later feature (see `specs/002-broker-sync`) and reuses the same trade records.
- Tax reports and capital-gains statements are out of scope.
