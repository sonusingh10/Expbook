# Upstream credit

Expbook is built on [ezBookkeeping](https://github.com/mayswind/ezbookkeeping)
by MayswindLab, licensed under the MIT License (see `LICENSE`).

- Upstream commit imported: `7b3eaf73e8bfb6773db4f71212aefd5441500e2f` (branch `main`)
- Imported on: 2026-11-08

The first commit in this repository is the untouched upstream snapshot, so
`git diff <first-commit>..HEAD` shows every Expbook change. To pull upstream
updates, diff upstream against that SHA and merge selectively.

Expbook-only modules are kept in separate files so upstream merges stay small: `pkg/investments`, the investment services/API/models, `src/expbook/`, `scripts/broker/`. The few upstream lines they touch are marked `// Expbook`; `specs/003-display-currency/plan.md` and `specs/004-portfolio-trend/plan.md` list the display-currency and portfolio-trend ones.
