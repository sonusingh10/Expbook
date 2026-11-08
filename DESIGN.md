---
name: Expbook
description: A calm, precise ledger for spending and investing, extending the inherited ezBookkeeping look.
colors:
  copper-primary: "#c67e48"
  copper-deep: "#b67443"
  paper-background: "#faf8f4"
  surface-white: "#ffffff"
  ink: "#413935"
  stone-secondary: "#8c8c8c"
  stone-deep: "#595754"
  ledger-teal: "#009688"
  alert-red: "#ff3b30"
  warning-amber: "#ff9500"
  info-blue: "#2196f3"
  success-green: "#4cd964"
typography:
  display:
    fontFamily: "Inter, sans-serif"
    fontSize: "1.75rem"
    fontWeight: 600
    lineHeight: 1.2
  title:
    fontFamily: "Inter, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 600
    lineHeight: 1.3
  body:
    fontFamily: "Inter, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Inter, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 500
    lineHeight: 1.3
rounded:
  sm: "0.375rem"
  md: "0.625rem"
  lg: "0.75rem"
  xl: "1rem"
spacing:
  "1": "0.25rem"
  "2": "0.5rem"
  "3": "0.75rem"
  "4": "1rem"
  "5": "1.25rem"
  "6": "1.5rem"
components:
  button-primary:
    backgroundColor: "{colors.copper-primary}"
    textColor: "{colors.surface-white}"
    rounded: "{rounded.md}"
  button-primary-hover:
    backgroundColor: "{colors.copper-deep}"
    textColor: "{colors.surface-white}"
  card:
    backgroundColor: "{colors.surface-white}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    padding: "{spacing.5}"
---

# Design System: Expbook

<!-- Scan mode: extracted from the inherited ezBookkeeping implementation (src/desktop-main.ts theme, src/styles/desktop/_variable.scss, src/styles/desktop/amount-color.scss). Investment screens extend these tokens; they do not introduce a second palette. -->

## Overview

**Creative North Star: "The Quiet Ledger"**

A well-kept paper ledger on a warm desk: calm surfaces, one copper accent, and numbers that are easy to trust. The interface stays out of the way during fast capture and becomes a dense, readable table during review. Personality comes from precision (aligned numerals, consistent units, clear signs), not decoration.

The inherited system is already warm and restrained: a paper-toned page, white raised cards, soft warm-tinted shadows and a copper accent. Investment features inherit all of it. Density rises on investment tables, but the visual language does not change.

**Key Characteristics:**
- Warm paper background with white cards; no heavy borders, soft warm shadows.
- One accent (copper) used for primary actions and selected states only.
- Tabular, right-aligned numerals for every amount.
- Gain and loss are conveyed by sign and icon first, color second.
- Same patterns on desktop (Vuetify) and mobile (Framework7), adapted per platform.

## Colors

A warm neutral base with a single copper accent and a restrained set of status colors.

### Primary
- **Copper** (#c67e48): primary buttons, active navigation, selected chips, focus ring (at 16% alpha). `--ebk-primary-color`.
- **Deep Copper** (#b67443): hover and pressed state of Copper.

### Secondary
- **Stone** (#8c8c8c) and **Deep Stone** (#595754): secondary text, inactive icons, secondary buttons.

### Tertiary
- **Ledger Teal** (#009688): positive money flows by default (income, gains) via `--ebk-amount-color-green`.
- **Alert Red** (#ff3b30; amount red `rgb(212, 63, 63)`): errors and negative flows (expenses, losses).
- **Warning Amber** (#ff9500), **Info Blue** (#2196f3), **Success Green** (#4cd964): status messages only, never used for portfolio data.

### Neutral
- **Paper** (#faf8f4): page background.
- **Surface White** (#ffffff): cards, dialogs, tables. In dark mode `rgb(28, 26, 24)` with raised `rgb(39, 36, 33)`.
- **Ink** (#413935): primary text on light surfaces.

### Named Rules
**The One Voice Rule.** Copper appears on at most one primary action per view plus selected state. Its rarity is how users find the next step.

**The Sign Before Color Rule.** Every gain or loss shows `+` or `-` (and an arrow icon where space allows). Color reinforces the sign and is never the only signal, because red and teal must not be the sole carrier of meaning.

**The No Second Palette Rule.** Charts and allocation views use the existing theme colors and tints of Copper and Stone. Do not add rainbow chart palettes for investments.

## Typography

**Display Font:** Inter (with system sans-serif fallback)
**Body Font:** Inter (with system sans-serif fallback)
**Label/Mono Font:** Inter with tabular numerals for amounts

**Character:** Neutral, legible and compact. The same family at different weights does all the work; hierarchy comes from size and weight, not from a second typeface.

### Hierarchy
- **Display** (600, 1.75rem, 1.2): net worth and the single headline figure on a dashboard.
- **Title** (600, 1.125rem, 1.3): card and section titles.
- **Body** (400, 0.9375rem, 1.5): default text and table cells. Prose lines stay under 70ch.
- **Label** (500, 0.8125rem, 1.3): column headers, chips, helper text.

### Named Rules
**The Tabular Numerals Rule.** All amounts, quantities, prices and percentages use tabular numerals (`font-variant-numeric: tabular-nums`) and are right-aligned in tables so columns scan vertically.

## Layout

Desktop uses a fixed sidebar (16rem) and a 4rem top bar with a fluid content area; mobile uses Framework7 pages with a bottom tab bar and a floating add action. Spacing follows a 4px-based scale (0.25rem to 1.5rem steps). Content density is moderate on dashboards and higher on holdings and transaction tables. Tables on mobile collapse to stacked rows with the amount and gain as the right-hand pair.

## Elevation & Depth

Hybrid: surfaces are mostly flat and tonal (Paper under Surface White), with soft warm-tinted shadows to lift cards and dialogs.

### Shadow Vocabulary
- **Small** (`0 1px 2px rgba(68,48,34,0.05), 0 3px 10px rgba(68,48,34,0.035)`): resting cards.
- **Medium** (`0 7px 24px rgba(68,48,34,0.1)`): menus and popovers.
- **Large** (`0 16px 44px rgba(68,48,34,0.16)`): dialogs.
- **Focus ring** (`0 0 0 3px rgba(198,126,72,0.16)`): keyboard focus.

### Named Rules
**The Flat-By-Default Rule.** Elements are flat at rest. Larger shadows appear only on overlays (menus, dialogs), never to decorate dashboard cards.

## Shapes

Gently rounded: 6px for small controls, 10px for buttons and inputs, 12px for cards, 16px for large containers and dialogs. Borders are hairlines using the ink color at low alpha. No pill shapes except chips and avatars.

## Components

### Buttons
- **Shape:** 10px radius (`--ebk-radius-md`).
- **Primary:** Copper background, white text.
- **Hover / Focus:** Deep Copper on hover; focus ring on keyboard focus; transition 150ms ease.
- **Secondary / Ghost:** Stone text or outline; no copper.

### Chips
- **Style:** tinted background at low alpha with matching text.
- **State:** selected chips use Copper tint; used for security type and filters.

### Cards / Containers
- **Corner Style:** 12px.
- **Background:** Surface White (dark: raised surface).
- **Shadow Strategy:** Small shadow at rest.
- **Border:** none by default; hairline in dark mode.
- **Internal Padding:** 1.25rem (`--ebk-space-5`).

### Inputs / Fields
- **Style:** Vuetify outlined fields on desktop; Framework7 list inputs on mobile; 10px radius.
- **Focus:** Copper focus ring.
- **Error / Disabled:** Alert Red message under the field; disabled at reduced opacity.

### Navigation
- Desktop: left sidebar, active item tinted with Copper. Mobile: bottom tab bar. The Investments section appears as one top-level entry on both.

### Holdings Table (planned signature component)
Right-aligned tabular columns for quantity, average cost, price, value, gain (amount and percent). The gain column shows sign, arrow and color together. Totals sit in a sticky footer row.

## Do's and Don'ts

### Do:
- **Do** reuse Vuetify and Framework7 components and the `--ebk-*` tokens before writing custom CSS.
- **Do** show amounts in the account's currency with tabular numerals, right-aligned.
- **Do** design empty, loading and error states for every new screen.
- **Do** keep light and dark mode both working; check contrast at WCAG AA.

### Don't:
- **Don't** use color alone for gain or loss.
- **Don't** add a second accent color or a rainbow chart palette.
- **Don't** nest cards inside cards or add gradient text and decorative gradients.
- **Don't** hardcode hex values in components; use theme colors and tokens.
