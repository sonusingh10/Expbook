# 004 Portfolio trend

## Problem
The app showed today's totals only. There was no picture of how total assets and investments change over time.

## What the user gets
- A line chart "Portfolio trend" at the top of the Home page (desktop and mobile) and of the Investments page (desktop and mobile).
- Three lines in the display currency: **Total assets** (all accounts plus investments), **Investments** (market value) and **Cost of investments**.
- Range buttons 6M, 1Y, 3Y, All. The headline shows today's total and the change over the range.
- Hover, touch or the arrow keys read the values at a date. Amounts follow the "hide amounts" setting.

## Rules
- Accounts use the same daily balances and the same "hide account from overview" setting as the Home net-assets widget.
- Investments are valued at each date from the trades known on that date and the latest recorded price at or before it.
- Where a security has no recorded price yet at a date, its cost stands in for value. Those parts are drawn dashed and a note says so. Nothing is invented.
- The server computes in the profile currency; the browser converts to the display currency with the latest rates, so the chart follows the display currency without another request.
- With no rate for the display currency, investments are left out and the card says why.
- With no trades, only the accounts are charted.

## Out of scope
- Historical market prices. Only prices already recorded are used (every price refresh and manual price adds one). Backfilling past NAVs from AMFI or Yahoo is a follow-up.
- Historical exchange rates: every conversion uses today's rate.
- Per-security lines.

## Acceptance
1. The last point of Total assets equals Net worth on the Investments page.
2. Switching the display currency re-draws the chart in that currency.
3. A security bought months ago with one recent price shows cost (dashed) before the price date and value after it.
4. Hiding an account on the Home overview removes it from the chart.
