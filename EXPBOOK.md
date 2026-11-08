# Expbook

Personal expense and investment tracker, built on [ezBookkeeping](https://github.com/mayswind/ezbookkeeping) (see `UPSTREAM.md`).

## Run locally

    ./run-local.sh        # creates .env with a random secret, builds if needed, serves on :8080

Requires Go and Node (see `go.mod`, `package.json`).

## How work is driven

- **Spec Kit** (`.specify/`, `.cursor/skills/speckit-*`): `/speckit-specify` -> `/speckit-plan` -> `/speckit-tasks` -> `/speckit-implement`. Rules: `.specify/memory/constitution.md`.
- **Impeccable** (`.cursor/skills/impeccable`): design review and polish (`/impeccable critique`, `audit`, `polish`). Context: `PRODUCT.md`, `DESIGN.md`. Check UI code with `npx impeccable detect src/`.
- **Taste Skill** (`.agents/skills/design-taste-frontend`, git-ignored because it is ~43 MB): restore with `npx skills experimental_install` (uses `skills-lock.json`). It is aimed at marketing pages, so use it for browser checks and art direction, not to restyle app screens.

Spec Kit and Impeccable skills are committed except the downloadable engine binary, which Impeccable re-downloads on first run.
