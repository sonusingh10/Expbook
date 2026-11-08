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

## Where Expbook differs from upstream (look and defaults)

- Money in is green and money out is red by default (upstream defaults to the opposite pairing). Both are still selectable in Settings.
- Green, red, warning amber and copper were tuned so text reaches WCAG AA (4.5:1) on desktop in light and dark. Light-mode copper is `#a65f2d` (was `#c67e48`); dark mode keeps `#c67e48` with dark text on it. Mobile still uses `#c67e48` for the Framework7 theme colour.
- Keyboard focus shows a 2px ring on desktop (upstream removed outlines).
- CI runs the Impeccable detector on the investment screens (`.github/workflows/design-detect.yml`).
- Broker import: `scripts/broker/` and `specs/002-broker-sync/`.
