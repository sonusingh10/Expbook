# Tasks: Investments MVP

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [contracts/api.md](contracts/api.md)
Format: `[ID] [P?] [Story] description`. `[P]` means it can run in parallel.

## Phase 1: Core maths (blocks everything; test-first)

- [x] T001 [US1] `pkg/investments/decimal.go`: scale constants, parse and format decimal strings, gross amount with `math/big` (+ tests)
- [x] T002 [US1] `pkg/investments/ledger.go`: trade ordering, average-cost replay, sell validation, holding result (+ tests with reference portfolios)
- [x] T003 [P] [US3] `pkg/investments/xirr.go`: Newton plus bisection, unavailable cases (+ tests)
- [x] T004 [P] [US3] `pkg/investments/portfolio.go`: currency conversion, totals, allocation summing to 100% (+ tests)
- [x] T005 [P] [US2] `pkg/investments/csv.go`: header mapping, row validation, idempotency key (+ tests)

## Phase 2: Persistence and API (US1, US2)

- [x] T006 Add uuid types, error subcategory and `pkg/errs/investment.go`
- [x] T007 [P] Models: `security.go`, `investment_trade.go`, `security_price.go` with request and response types
- [x] T008 Services: securities, trades (replay validation on create, modify and delete), import core (dry run, idempotent)
- [x] T009 API handlers in `pkg/api/investments.go`; summary builds on `pkg/investments`
- [x] T010 Wire `SyncStructs` in `cmd/database.go` and routes in `cmd/webserver.go`
- [x] T011 Service tests on in-memory SQLite: add, sell guard, delete guard, import twice equals once

## Phase 3: Prices (US3)

- [x] T012 [P] `pkg/investments/prices`: provider interface, `yahoo`, `amfi`, tests with `httptest`
- [x] T013 Refresh service (per-security results, plausibility check, price history rows)
- [x] T014 Cron job `RefreshSecurityPrices` (daily) and settings flag; manual `prices/refresh.json`

## Phase 4: Frontend (US1 to US5)

- [x] T015 [P] `src/models/investment.ts`, API client methods in `src/lib/services.ts`
- [x] T016 [P] `src/lib/investments.ts` formatting helpers (sign, percent, tabular classes) + vitest
- [x] T017 `src/stores/investment.ts` (securities, trades, summary, refresh state)
- [x] T018 Desktop `ListPage.vue`: summary cards, holdings table, empty, loading and error states
- [x] T019 Desktop dialogs: security edit, trade edit, CSV import with preview
- [x] T020 Mobile `ListPage.vue`, `TradeEditPage.vue` and `SecurityEditPage.vue`. Deviation: CSV import is desktop-only for now (picking a file on a phone and reading a preview table is a poor fit; the API is shared)
- [x] T021 Router and menu entries (desktop and mobile), `en.json` strings
- [x] T022 Net worth card (accounts plus investments, split shown) on the desktop investments page. Deviation: not added to the configurable overview dashboard or to mobile

## Phase 5: Verification

- [x] T023 `go test ./pkg/...`, `npm run lint`, `npm test`, `./build.sh` succeed
- [x] T024 Live smoke test against the running server (create, trade, import twice, refresh). Deviation: refresh was exercised live only for the failure path (no outbound network in the test environment); the success path and partial failure are covered by `investment_prices_test.go` and `prices_test.go` with fake providers
- [x] T025 Walk the spec's acceptance scenarios and mark them in `quickstart.md` (browser walk-through items are completed in the ux-pass step)

## Dependencies

T001 -> T002 -> T008 -> T009. T003, T004, T005 depend on T001. T012 is independent of Phase 2.
Frontend T015 can start once `contracts/api.md` is stable. T023 to T025 run last.
