# Expbook Constitution

Expbook is a personal, self-hosted expense and investment tracker built on a
vendored copy of [ezBookkeeping](https://github.com/mayswind/ezbookkeeping)
(MIT). See `UPSTREAM.md` for the imported upstream commit.

## Core Principles

### I. Self-Hosted and Privacy First
All financial data stays on the user's own machine or server. The app MUST work
fully offline for recording and reviewing data. Network calls (price feeds,
exchange rates, broker sync) are optional, user-configured, and MUST send no
personal transaction data to third parties. No analytics or telemetry.

### II. SQLite by Default
SQLite is the default and always-supported database. Every schema change MUST
ship as a migration that works on SQLite and MUST NOT break MySQL or PostgreSQL
support that upstream already provides.

### III. No Hardcoded Secrets (NON-NEGOTIABLE)
API keys, tokens, passwords, signing keys and connection strings MUST NOT
appear in source code, tests, fixtures or committed config. They come from
environment variables (`EBK_<SECTION>_<KEY>`) or untracked files such as `.env`.
`.env.example` documents the variable names only. Broker and price-provider
credentials follow the same rule.

### IV. Tests for Money Maths (NON-NEGOTIABLE)
Any code computing amounts, cost basis, realised or unrealised P&L, XIRR,
currency conversion or rounding MUST have unit tests written with it, including
edge cases (partial sells, zero quantity, multiple currencies, leap years,
negative returns). Money is stored as integers in minor units or fixed-point
decimals, never binary floats.

### V. Minimal, Marked Changes to Upstream Code
New functionality goes in new files and packages (for example
`pkg/**/investment*.go` and `src/**/investments/`). Edits to files inherited
from upstream MUST be small and wired in through existing extension points
(routes, menus, migrations, locale files). The first commit of the repository
is the untouched upstream snapshot, so `git diff` against it always shows
exactly what Expbook changed.

### VI. Design System Is the Source of Truth for UI
All user-facing UI work follows `PRODUCT.md` and `DESIGN.md`. New screens MUST
work on mobile and desktop, in light and dark mode, and have designed empty,
loading and error states. UI changes are reviewed with Impeccable
(`critique`, `audit`, `polish`) and the Taste Skill browser check before they
count as done.

### VII. Idempotent, Auditable Imports
CSV import and broker sync MUST be idempotent: re-running the same import never
duplicates trades. Every imported record keeps its source and an external id so
the user can trace and undo it.

## Technology and Quality Constraints

- Backend: Go (version in `go.mod`), following the existing `pkg/` layering
  (models, services, API handlers, cron jobs).
- Frontend: Vue 3 and TypeScript with Vite, linted by the existing ESLint config.
- Required checks before merging: `go test ./...`, `npm run lint`,
  `npm test` where tests exist, and `npx impeccable detect src/` for UI changes.
- UI strings go through the existing i18n locale files; English is required
  for every new string, other languages may follow later.

## Development Workflow

- Work follows Spec Kit: `/speckit-specify`, `/speckit-clarify`, `/speckit-plan`,
  `/speckit-tasks`, `/speckit-implement`. Non-trivial features need a spec
  before code.
- Commits are small and focused. The `main` branch stays buildable.
- Investment features start with manual entry and CSV import. Broker sync is a
  separate spec that reuses the same tables.

## Governance

This constitution supersedes other practices in this repository. Amendments are
made by editing this file with a version bump and a short rationale in the
commit message. Every spec, plan and review checks compliance with the
principles above; deviations MUST be justified in the plan's complexity notes.

**Version**: 1.0.0 | **Ratified**: 2026-11-08 | **Last Amended**: 2026-11-08
