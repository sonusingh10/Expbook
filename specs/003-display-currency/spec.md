# 003 Display currency

## Problem
Every total in the app is shown in the profile currency. A person with income in PLN and spending in INR cannot quickly see the same totals in another currency without changing the profile.

## What the user gets
- A currency button in the top bar on desktop, and in the Home navbar on mobile.
- Choosing a currency re-shows totals everywhere in that currency: dashboard widgets, account totals, statistics, transaction list totals, investment totals and net worth.
- The profile currency is not changed. Choosing the profile currency again clears the choice.
- The choice is saved on the server per account and follows the user to other devices.
- The menu shows "Rates as of <date>".

## Rules
- Individual amounts stay in their own currency (a transaction in INR is still shown in INR, a holding in USD in USD). Only totals are converted.
- The server never converts. The browser converts with the latest exchange rates it already loads.
- No rate means no conversion: the amount stays in its original currency with its original code, and the net worth total shows "-" rather than a wrong number.
- Forms that create things (new account, new security, import) keep using the profile currency.

## Out of scope
- Insights explorer and custom charts (they keep the profile currency).
- Historical rates: every conversion uses today's rate.
- Mobile picker outside the Home page (conversion applies on all mobile pages; the button is only on Home).

## Acceptance
1. With profile INR and display PLN, dashboard net assets equals the INR figure times INR-to-PLN.
2. A second browser signing in to the same account shows PLN without any action.
3. Picking INR removes the override; the chip shows INR and totals match the profile.
4. Another user is not affected.
