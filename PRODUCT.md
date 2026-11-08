# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users
A single person (the owner, Sonu) tracking their own money. Daily situation: logging an expense in a few seconds on a phone, and sitting down at a laptop weekly or monthly to review spending, net worth and how investments are doing. Uses both mobile and desktop about equally. Not a team or family product; there is no multi-user collaboration need.

## Product Purpose
Expbook is a private, self-hosted tracker that puts expenses and investments in one place. It is built on ezBookkeeping (accounts, categories, transactions, imports, charts) and adds an investment layer: holdings, buys and sells, SIPs, dividends, cost basis, unrealised and realised profit and loss, XIRR, allocation, and a combined net worth. Success means the owner can answer "where did my money go this month" and "what am I worth and how are my investments doing" from one trustworthy screen, without handing data to a third party.

## Positioning
Spending and investing in one self-hosted ledger that the owner fully controls, instead of a bookkeeping app with no portfolio view or a portfolio app with no spending. Data never leaves the owner's server except optional price lookups.

## Operating Context
- Runs on the owner's own machine or server; SQLite by default.
- Currency is mixed: INR plus one or more foreign currencies, so multi-currency and exchange rates matter everywhere amounts appear.
- Investments are entered manually or imported from CSV first; broker sync is a later phase and reuses the same records.
- Prices come from an optional, user-configured free market-data source, refreshed on a schedule.

## Capabilities and Constraints
- Existing (from ezBookkeeping): two-level accounts and categories, transactions with attachments, scheduled transactions, filters and search, charts, import from many formats, multi-currency, dark mode, PWA, 2FA.
- To add: investment accounts, securities (stocks, ETFs, mutual funds, crypto), trades and SIPs, price snapshots, holdings table, P&L, XIRR, allocation chart, net worth that includes investments.
- Constraints: no hardcoded secrets, tests for all money maths, English strings first, minimal edits to inherited upstream files (see `.specify/memory/constitution.md`).
- Undecided: the exact free price source per market (decided in the investments spec), final broker integration.

## Brand Commitments
Name is "Expbook". No logo or brand assets exist yet; none should be invented without the owner. Upstream ezBookkeeping credit stays in `UPSTREAM.md` and `LICENSE`.

## Evidence on Hand
No real user data, screenshots or testimonials exist in the repo. The only visual evidence is the inherited ezBookkeeping interface (Vuetify on desktop, Framework7 on mobile). Do not fabricate sample portfolios as if they were real; use clearly labelled demo data.

## Product Principles
1. Trust over flash: every number must be traceable to the transaction or price that produced it.
2. Fast capture, calm review: logging takes seconds; reviewing is quiet, dense and readable.
3. One ledger: spending and investing share accounts, currencies and net worth rather than living in separate silos.
4. Private by default: works offline for recording, no telemetry, optional network only for prices.
5. Respect the incumbent: extend the existing app's patterns before inventing new ones.

## Accessibility & Inclusion
Targets WCAG AA contrast, keyboard operation on desktop, and legible numerals. Never rely on red or green alone to show gain versus loss; pair color with a sign or icon. Money and percentages must be readable at small mobile sizes.
