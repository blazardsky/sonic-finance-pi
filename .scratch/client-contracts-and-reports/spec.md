Status: ready-for-agent

# Gifts, Client Contracts, Reminders, and the Full Yearly Report

## Problem Statement

Buying a gift for someone shows its price to anyone glancing at a shared screen. The record-management screens (Expenses, Incomes, Categories, Clients, Recurring Expenses) are plain lists that don't scale well visually. A Client is just a name — there's no way to see what they've paid in total, default their usual Income Category, or track an ongoing agreement (a Contract) against what's actually arrived. A manual, non-automated payment the household makes by hand every month (like a bank transfer on the 5th) has no way to be checked off and is easy to forget or double-do. And there's no single place to see a whole year, month by month, broken down by Category, with the medians that say what a "normal" month looks like. Finally, an Expense or Income can currently be saved with no Payer at all, which undermines the "whose money was it" question the app otherwise always answers.

## Solution

1. A protected Gift Category (reusing the existing, unused `Regali` category) whose Expenses are blurred by default in lists and grids.
2. The record-management list screens swapped to shadcn/ui's Table component.
3. Clients gain a default Income Category, an all-time "total earned" figure, and one or more Contracts — each a total the household expects from that Client over a date range — with running "should have," "got," and "invoice this month" figures.
4. A household-managed list of Dashboard Reminders — simple on/off toggles for manual actions, each resetting to off at the start of the next month.
5. A new, dedicated full yearly report: expenses by Category for every month, a running total, a total excluding tax, Savings at the start of the year, and medians scoped to that year.
6. Payer becomes a required field on Expense and Income going forward.

## User Stories

**Gift spoiler**

1. As a household member, I want an Expense that's a gift for someone else to be categorized as Gift, so its nature is recorded the same way any other Expense's Category is.
2. As a household member, I want a Gift Expense's amount blurred by default wherever it appears in a list or grid, so someone glancing at my screen can't see what I spent.
3. As a household member, I want to reveal a blurred amount by hovering over it (desktop) or tapping it (mobile), so I can still check it myself.
4. As a household member, I want a revealed amount to blur again once I move on, so it doesn't stay exposed after I've looked away.
5. As a household member, I want Gift to behave like any other protected Base category — I can hide it once I stop using it, but never rename or delete it — so the spoiler logic never silently breaks.

**Table swap**

6. As a household member, I want the Expenses, Incomes, Categories, Clients, and Recurring Expenses screens to use a real table, so dense record lists are easier to scan.
7. As a household member, I want the table swap to be visual only — same rows, same columns, same actions — so nothing I rely on today changes behavior.

**Client defaults and totals**

8. As a household member, I want to set a default Income Category on a Client, so I don't have to repick "Freelance" or "Stipendio" every time I log an Income from them.
9. As a household member, I want the default Category to only prefill the picker, not lock it, so an unusual Income from that Client can still be categorized differently.
10. As a household member, I want to see how much I've earned in total from a given Client, computed from their received Incomes, so I know the relationship's real value without adding it up by hand.

**Client contracts**

11. As a household member, I want to record a Contract with a Client — a total amount and a start/end month — so I can track an ongoing agreement against what's actually arrived.
12. As a household member, I want a Client to be able to have more than one Contract over time (e.g. renewed each year), so a past agreement doesn't block recording a new one.
13. As a household member, I want two Contracts for the same Client to never overlap in time, so "how much should I have" is never ambiguous about which Contract a given month belongs to.
14. As a household member, I want to see how much I should have received from a Contract by now, based on a straight-line share of the time elapsed, so I can tell at a glance if I'm behind.
15. As a household member, I want to link an Income to the Contract it counts toward, so the app can tell contract income apart from anything else from the same Client.
16. As a household member, I want an Income from a Client with no Contract link to still count as Extra — real income from them, just outside the agreed total — so the app doesn't lose track of one-off work.
17. As a household member, I want to see how much I should invoice this month, recomputed from whatever's left of the total and however many months remain, so the figure adapts if I invoiced more or less than expected last month.
18. As a household member, I want a Contract that's already past its end month to clearly show what's still owed as overdue rather than dividing a shortfall by zero remaining months.

**Dashboard reminders**

19. As a household member, I want to create a short, labeled Reminder (like "Paid the rent transfer"), so I can track a manual action the app doesn't automate.
20. As a household member, I want more than one Reminder at a time, so I can track several manual actions independently.
21. As a household member, I want to toggle a Reminder on once I've done the thing it names, so I have a quick visual check on the Dashboard.
22. As a household member, I want every Reminder to read as off again starting on the first of the next month, without me having to remember to reset it, so it's ready to track the same action next month.
23. As a household member, I want to delete a Reminder I no longer need, so the Dashboard doesn't accumulate stale toggles.

**Full yearly report**

24. As a household member, I want a dedicated report page showing every month of a year broken down by Category, so I can see spending patterns across the whole year at once.
25. As a household member, I want a running total alongside the month-by-month breakdown, so I can see the year accumulate as it goes.
26. As a household member, I want a total for the year excluding the Taxes category, so I can see what I actually spent living, separate from tax payments.
27. As a household member, I want to see my Savings figure as it stood at the start of the reported year, so I have a baseline for how the year changed my position.
28. As a household member, I want to see cumulative income, expense, and what's left at each point through the year, so I can see the year's trajectory, not just its endpoint.
29. As a household member, I want median Expense, Income, and Net figures scoped to the reported year's own completed months, so "what was typical in 2026" doesn't shift depending on what year I'm viewing it from.

**Payer required**

30. As a household member, I want Payer required on every new Expense and Income, so "whose money was it" is never left unanswered going forward.
31. As a household member, I want existing Expenses/Incomes with no Payer left untouched, so this doesn't retroactively break records I already saved.

## Implementation Decisions

### Schema (schemaVersion 10 → 11, one migration step)

- The existing `Regali` category is promoted in place, exactly like `Investimenti` was: `applies_to = 'both'`, new protected `code = 'gift'` (constant `codeGift`). No new category row.
- `client.default_category_id INTEGER REFERENCES category(id)`, nullable.
- New `contract` table: `id`, `client_id` (FK, required), `start_month`, `end_month` (both `YYYY-MM` text, same layout as everywhere else in the app), `total_cents`. Overlap between two Contracts for the same Client is refused at write time (application-level check, not a DB constraint — matching how Recurring's own date ranges aren't DB-constrained either).
- `income.contract_id INTEGER REFERENCES contract(id)`, nullable — same shape as `expense.recurring_id` and the new `holding_id` columns: a money row pointing back at what it belongs to.
- New `reminder` table: `id`, `label`, `enabled` (boolean), `set_for_month` (nullable `YYYY-MM` text).

### Gift

- Spoiler blur applies to any Expense resolving to the Gift category, wherever it's shown in a list or grid (Recent entries, Month's list, the new report, etc.) — the amount only, not date/store/category/note.
- Reveal is hover (desktop) or tap (mobile), scoped to that row for the current viewing session — not persisted, not remembered across navigation.

### Table swap

- shadcn's Table component is added (`npx shadcn add table`, not yet present in `web/src/components/ui/`).
- Applies to Expenses, Incomes, Categories, Clients, and Recurring Expenses. Month's category breakdown and Dashboard's Recent-entries lists are unaffected — they're summary widgets, not record-management screens.
- Structural/visual swap only: same rows, same columns, same row actions. No sorting or pagination added in this pass.

### Client defaults and totals

- `default_category_id` only prefills the Income form's category picker when that Client is selected; it remains freely editable per Income.
- Total earned is computed at read time — `SUM` of that Client's received Incomes (`payment_date IS NOT NULL`, per ADR-0003) — not stored.

### Contracts

- `GET /api/clients/{id}/contracts` lists every Contract for that Client (past and current), each with:
  - `expected_so_far_cents`: `total_cents × (months elapsed ÷ total months in the contract)`, capped at `total_cents` once `end_month` has passed.
  - `received_cents`: sum of that Contract's linked Incomes with `payment_date IS NOT NULL` — actual cash in hand.
  - `accounted_cents`: sum of that Contract's linked Incomes regardless of payment status — includes an invoice already sent but not yet paid, so it isn't suggested for invoicing twice.
  - `invoice_target_this_month_cents`: `(total_cents − accounted_cents) ÷ months remaining` (ADR-0012); once `end_month` has passed, this is shown as the full remaining shortfall, labeled overdue, rather than divided by zero or a negative month count.
- An Income's `contract_id` is optional; when a Client has an active Contract, the Income form offers linking to it, defaulting to unlinked (Extra) rather than auto-guessing.
- "Extra" Incomes (no `contract_id`) still count toward the Client's all-time "total earned," just not toward any Contract's figures.

### Reminders

- `GET /api/reminders` returns each Reminder with `enabled` collapsed to `false` whenever `set_for_month` isn't the real current month — read off the browser's clock, no scheduler, same pattern `materialise` already uses (ADR-0005).
- `POST /api/reminders` creates one with a label (starts disabled). `PATCH /api/reminders/{id}` toggles it: enabling sets `enabled = true, set_for_month = <current month>`; disabling sets `enabled = false`. `DELETE /api/reminders/{id}` removes one outright — nothing else references a Reminder, so unlike Category/Client this doesn't need a hide/archive path.
- Shown on the Dashboard as a small list of labeled toggles.

### Full yearly report

- New dedicated page (not an extension of `Year.tsx`), reachable per year, using shadcn's Chart component (added by the daily/weekly-charts ticket in the other spec if that lands first; added here otherwise).
- One new endpoint returning, for a given year: per-month-per-category Expense totals from a single `(month, category)`-grouped query (ADR-0011, not twelve separate breakdown calls), a running cumulative total, the year's total Expense excluding the Taxes category, Savings as of December 31st of the prior year (the existing Savings formula, cut off at that date instead of today), cumulative Income/Expense/Net through each month of the year, and median Expense/Income/Net computed over that year's own completed months (distinct from Budget's rolling 12-month window — see `CONTEXT.md`).

### Payer required

- `POST /api/expenses` and `POST /api/incomes`, and any update that sets `payer`, reject an empty or whitespace-only value. No schema change — the column is already `NOT NULL DEFAULT ''`.
- Existing rows with an empty Payer are left exactly as they are; only new writes are validated. Frontend forms mark Payer as a required field.

## Testing Decisions

- Every new/changed endpoint (Gift category resolution, Client totals, Contract math, Reminder toggling, the yearly report) is tested at the same single seam the rest of the backend uses: black-box HTTP tests via the shared `testApp` harness, no mocking.
- Prior art to follow directly:
  - `categories_test.go`'s migration/promotion tests, for verifying `Regali` is promoted in place with no duplicate row (same shape as the existing `Investimenti` promotion test).
  - `clients_test.go`, extended for `default_category_id` and the computed total-earned figure.
  - A new `contract_test.go`, following `recurring_test.go`'s clock-controlled style for `expected_so_far_cents` and `invoice_target_this_month_cents` at various points in a contract's span, including the past-end-month overdue case.
  - A new `reminder_test.go`, using the same `a.moveClock`/`a.setNow` pattern as `recurring_test.go` to verify a Reminder collapses to disabled once the month rolls over.
  - `reports_test.go`'s helper pattern (`a.month`, `a.year`, …) for the new yearly-report endpoint.
  - `expense_test.go`/`income_test.go`, extended to assert an empty Payer is now rejected on create/update.
- No automated tests for the Gift spoiler blur or the Table swap — both are frontend-only visual behavior, and this codebase has no frontend test runner today (unchanged from the other spec's testing decision).

## Out of Scope

- A second, differently-named Gift category — `Regali` is reused for both directions.
- Sorting or pagination on the newly-tabled list screens.
- Auto-guessing which Contract an Income belongs to — linking is always an explicit choice, defaulting to unlinked.
- A payment schedule on a Contract (e.g. "€500 due on the 1st of each month") — only a total and a date range.
- Reminders tied to a specific Recurring expense record — they're freestanding labels, unrelated to Recurring generation.
- A history of past Reminder toggles (e.g. "I paid rent on time 11 of the last 12 months") — only the current month's state is tracked.
- Backfilling Payer on existing Expense/Income rows that have none.
- Protecting `Stipendio` or `Altro` — settled in this conversation as unnecessary, since no code resolves either by identity (ADR-0008).

## Further Notes

- Builds on ADR-0003 (received Income only, for total-earned and Contract "received"), ADR-0005 (no-scheduler, clock-read pattern, reused for Reminders), ADR-0008 (Base category protection, reused for Gift), and the two new ADRs recorded this session: ADR-0011 (the yearly report is its own page with one grouped query) and ADR-0012 (a Contract's invoice target recomputes from the shortfall rather than a fixed schedule).
- New/updated `CONTEXT.md` terms: Gift, Contract, Reminder, and a clarifying note on Budget distinguishing it from the yearly report's own medians.
- This spec assumes the other in-flight spec's schema migration (schemaVersion 9→10, Investments) lands first, since this one bumps schemaVersion again (10→11) on top of it — sequence accordingly when ticketing.
- All new UI strings go through the existing `web/src/lib/strings.ts` (`t.`) pattern, Italian-first, matching the household's existing seeded vocabulary.
