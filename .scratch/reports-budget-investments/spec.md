Status: ready-for-agent

# Trends, Budget, and Investments/Savings

## Problem Statement

The household can see what happened in a given month or year, but has no sense of *trend* within a month, no forward-looking read on where the year is heading (the Dashboard's income/expense targets are a fixed, made-up €50k with no connection to real history), and no sense of whether a given month's spending is normal for them. There is also no way to record or track money that is saved or invested — every euro that leaves the household today is implicitly assumed to be spent, with no way to see how much has been put aside or invested, or how an investment portfolio is allocated.

## Solution

1. A daily expense trend, by Category, on the Dashboard (rolling window), and a weekly one, by Category, on the Month page (scoped to the viewed month).
2. A yearly Estimate on the Dashboard — projected income, expense, and tax for the rest of the year, replacing the fixed hardcoded targets — computed from this year's pace so far against last year's, clearly labeled as a projection.
3. A computed Budget (typical monthly spend) and a household-settable Target and savings Goal, shown on the Month page for the current month.
4. A new protected `Investments` base category and a `Holding` concept, so buying/selling a stock, ETF, crypto asset, or bond is recorded exactly like any other Expense/Income, with a dedicated Buy/Sell form and a portfolio percentage breakdown.
5. A computed, ledger-free `Savings` figure — the household's uninvested surplus — with a one-time starting balance for pre-app savings.
6. Support for a recurring investment (an accumulation plan / PAC), reusing the existing Recurring expense machinery.

All of the above builds on ADR-0009 (Investments count in ordinary totals; only Budget/Target/Estimate exclude them) and ADR-0010 (the Estimate is the app's first and only projected figure, always labeled as such).

## User Stories

**Daily/weekly trend charts**

1. As a household member, I want to see a daily expense trend on the Dashboard, broken down by Category, so I can spot which days and categories are driving recent spending.
2. As a household member, I want to see a weekly expense trend for the month I'm viewing on the Month page, broken down by Category, so I can see whether spending is front- or back-loaded within the month.
3. As a household member, I want a week that only partially falls inside the viewed month shown as its real date range rather than hidden or merged into a neighboring week, so the chart never quietly drops days.
4. As a household member, I want the trend charts to use the same category names and colors as the rest of the app, so they're easy to cross-reference with the Month page's existing category breakdown.
5. As a household member, I want the daily chart's category breakdown to account for Items the same way the rest of the app does, so a book bought during a grocery run still counts under Books, not Groceries.

**Yearly Estimate**

6. As a household member, I want the Dashboard to show a projected year-end income, expense, and tax figure instead of a fixed made-up target, so the progress bars mean something.
7. As a household member, I want the Estimate clearly labeled as a projection, so I never mistake it for money that has actually moved.
8. As a household member with less than a full year of history, I want the Estimate to simply not show, rather than show a number built on no real prior-year data.
9. As a household member, I want the Estimate to ignore Investments activity, so a one-off stock purchase or sale doesn't distort my projected income or expenses.

**Budget / Target / Goal**

10. As a household member, I want to see my typical (median) monthly spending on the Month page, computed automatically from history, so I know what "normal" looks like for my household.
11. As a household member, I want to set my own spending ceiling (Target) for the current month, starting equal to the computed Budget, so I can hold myself to a tighter number when I choose to.
12. As a household member, I want to set a monthly savings Goal by hand, so I have something concrete to aim for.
13. As a household member with fewer than 3 completed months of history, I want the Budget to simply not show, so a "median" computed from one or two data points doesn't mislead me.
14. As a household member, I want Investments activity excluded from the Budget/Target calculation, so buying stock doesn't inflate what looks like my normal living expense.
15. As a household member, I want Budget/Target/Goal to only appear when I'm viewing the current real month, so there's no ambiguity about what "this month's target" means for a past or future month.

**Investments & Holdings**

16. As a household member, I want to record buying a stock, ETF, crypto asset, or bond as a purchase of a named Holding, so I can track where my invested money has gone.
17. As a household member, I want to record selling a Holding, so my portfolio reflects reality when I cash out.
18. As a household member, I want each Holding to have a name and a type (ETF, crypto, stock, bond, or other), so I can tell my holdings apart and see them grouped by kind.
19. As a household member, I want a percentage breakdown of my portfolio by Holding, computed from money actually put in net of what's been sold, so I can see my real allocation without tracking live prices.
20. As a household member, I want buying/selling a Holding to still show up in my ordinary Month, Year, and Recent totals, so I never lose sight of money that left my pocket and might not come back.
21. As a household member, I want a dedicated Buy/Sell form on the Savings page, so I don't have to remember to pick "Investments" and a Holding on the generic Expense/Income screens.
22. As a household member, I want the list of Holdings to be a fixed, household-maintained list, like Category, so a typo doesn't silently split one holding's percentage across two rows.
23. As a developer, I want the existing unused "Investimenti" category promoted in place to the new protected Investments base category (not a new, separately-named category created), so the category list doesn't end up with two confusingly similar entries.

**Savings**

24. As a household member, I want to see a single Savings figure — money that's neither been spent nor invested — computed automatically, so I don't have to log savings by hand.
25. As a household member, I want to enter a one-time starting balance for money I saved before I started using this app, so my Savings figure reflects my real position from day one.

**Recurring investments**

26. As a household member, I want to set up a recurring investment (an accumulation plan) that buys into a Holding automatically each month, so I don't have to remember to log it by hand.
27. As a household member, I want the Holding field to only appear on the Recurring Expense form when I've picked the Investments category, so the form doesn't clutter every other recurring expense with an irrelevant field.
28. As a household member, I want a generated recurring investment purchase to be indistinguishable from one I typed by hand, consistent with how every other recurring expense already works.

## Implementation Decisions

### Schema (schemaVersion 9 → 10, one migration step)

- New `holding` table: `id INTEGER PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL CHECK (type IN ('etf','crypto','stock','bond','other'))`. No `hidden`/archive column for this version (see Out of Scope).
- `expense.holding_id INTEGER REFERENCES holding(id)`, nullable. Meaningful only when the row's category is the Investments base category — same relationship `tax_year` already has to the Taxes base category. Not copied by anything; set directly on insert like `category_id`.
- `income.holding_id INTEGER REFERENCES holding(id)`, nullable. Same meaning, for a sell.
- `recurring_expense.holding_id INTEGER REFERENCES holding(id)`, nullable. Copied by `materialise` into each generated Expense's `holding_id`, in the same statement that already copies `category_id`, `store`, `payer`, `payment_method`, `note` — unlike `tax_year`, which `materialise` deliberately leaves NULL, `holding_id` is a genuine template property.
- The existing `category` row named `Investimenti` (id 14 in the current seed, `applies_to = 'income'`, no code, zero real-world usage) is updated in place: `applies_to = 'both'`, `code = 'investments'` (new constant `codeInvestments`, alongside `codeFreelance`/`codeTaxes`). No new category row is inserted, and no other existing category row is touched.
- No new tables for Savings, Budget, Target, or Goal — see below.

### Investments & Holdings

- Buying a Holding = an ordinary `POST /api/expenses` with `category_id` resolving to the Investments base category and `holding_id` set.
- Selling a Holding = an ordinary `POST /api/incomes`, same shape.
- No new create/update endpoints for buy/sell — the dedicated Buy/Sell form on the Savings page is a frontend affordance that POSTs to the two existing endpoints with the category pre-selected and a Holding picker in place of a free Category picker.
- `holding_id` is not cross-validated against `category_id` server-side (no new validation beyond what `tax_year` already lacks) — a `holding_id` set on a non-Investments row is simply never read by any report. This mirrors existing precedent rather than adding new enforcement nobody asked for.
- New Holding CRUD, mirroring Category's existing shape: `GET /api/holdings` (list), `POST /api/holdings` (create: name + type), `PATCH /api/holdings/{id}` (rename / change type). No delete/hide endpoint in this version.
- The Investments base category is protected exactly like Freelance and Taxes (ADR-0008's pattern): rename and delete refused by its `code`, hiding allowed.
- Every report that sums Expense/Income today (Month, Year, Dashboard actuals, Recent entries) is **unchanged** — Investments rows flow through them exactly like any other category (ADR-0009). Only the two new reports below (Budget/Target, Estimate) add an explicit `code != 'investments'` exclusion.
- Portfolio breakdown: `SUM(expense.amount_cents) − SUM(income.amount_cents)`, grouped by `holding_id`, for rows under the Investments category — the same shape as `readBreakdown`'s category grouping, applied to Holdings instead of Categories. A holding fully sold off nets to zero and is simply not listed (same "a zero remainder is dropped" rule `readBreakdown` already applies to categories).

### Savings

- No `savings` table and no ledger. Computed as: `SUM(all Income, all time, excluding Investments) − SUM(all Expense, all time, excluding Investments) + starting_balance_cents`.
- `starting_balance_cents` is a single value in the existing `setting` table (new key, e.g. `savings_starting_balance_cents`), read/written via the existing `getSetting`/`setSetting` helpers — no schema change.
- Exposed as part of a single `GET /api/reports/savings` response alongside the Holdings breakdown, so the Savings page loads in one request: `{ savings_cents, starting_balance_cents, holdings: [{ holding_id, name, type, net_cents, percent }] }`.
- Setting the starting balance: extend the existing `PUT /api/settings` payload/response (currently `{ payers, payment_methods }`) with `savings_starting_balance_cents`, rather than adding a new endpoint — same screen, same request, one more field.

### Budget / Target / Goal

- **Budget**: computed, not stored. Median of the trailing 12 *completed* calendar months' `expense_cents` (current in-progress month excluded), itself computed excluding the Investments category. Requires at least 3 completed months of history; below that, the API returns "unavailable" rather than a number from too little data.
- **Target**: a single stored value (new `setting` key, e.g. `target_cents`), read/written through the same extended `/api/settings` payload as the Savings starting balance. Defaults to the current Budget's value the first time it's read if no Target has ever been set; once set, it's the household's own number and is never silently recomputed.
- **Goal**: a single stored value (new `setting` key, e.g. `savings_goal_cents`), same mechanism as Target.
- All three exposed via one new `GET /api/reports/budget` response: `{ available: bool, budget_cents, target_cents, goal_cents }`. `available` is `false` when fewer than 3 completed months exist; `target_cents`/`goal_cents` are independent of `available` since they're just settings.
- Shown on the Month page only when the viewed month equals the real current month (`thisMonth()`) — Budget/Target/Goal describe "now," not an arbitrary historical or future month.

### Yearly Estimate

- `estimate = lastYearFullYearTotal × (thisYearYTD / lastYearYTDAtSamePoint)`, computed independently for income, expense, and tax-paid.
- YTD cutoff is by whole completed months (e.g. if today is in September, YTD means January–August for both years) — this reuses the existing `GET /api/reports/year/{year}` response's per-month breakdown for both this year and last year, so no new lower-level query is needed for the cutoff itself.
- Income/expense figures exclude the Investments category (same exclusion as Budget); tax uses the existing `taxSummary` shape (`tax_paid_cents`, attributed by Tax year, per ADR-0008).
- New `GET /api/reports/estimate/{year}` response: `{ year, available: bool, income_estimate_cents, expense_estimate_cents, tax_estimate_cents }`. `available` is `false` when there's no prior year at all, or when last year's YTD-at-the-same-point for a given metric is zero (undefined ratio) — in that case that metric (not necessarily the whole response) is omitted/null.
- Replaces `Dashboard.tsx`'s hardcoded `ESTIMATE_INCOME_CENTS`/`ESTIMATE_EXPENSE_CENTS`: the progress bars now target the computed `income_estimate_cents`/`expense_estimate_cents` instead. When `available` is `false`, the progress bar for that figure is simply not shown (per ADR-0010 — no Estimate is better than a fabricated one).
- The Estimate is visually and textually distinguished from actual totals wherever shown (e.g. a "stima" label/footnote) — it must never be presented as an unqualified number the way `year.income_cents` is.

### Daily/weekly trend charts

- New `GET /api/reports/month/{month}/daily` — per-day, per-category Expense totals for one month, same Item-inclusive attribution as `readBreakdown` (one query, `GROUP BY` on day and category instead of category alone — no additional query cost over the existing per-month breakdown). Powers the Month page's weekly chart: the frontend buckets the returned days into ISO weeks (Monday-start) client-side; a week is not re-fetched from the server as a separate concept.
- New `GET /api/reports/daily` — a fixed trailing window (constant, e.g. 30 days ending today) of per-day, per-category Expense totals, independent of month navigation. Powers the Dashboard's daily chart.
- Both use shadcn's Chart component (Recharts) — not yet installed in `web/src/components/ui/`; add via `npx shadcn add chart` per `web/components.json`'s `radix-nova` style.
- Investments rows are **not** excluded from either chart (ADR-0009) — a stock purchase appears as a normal bar/segment like any other category.

### Recurring investments

- `RecurringExpenses.tsx`'s form gets a Holding picker rendered conditionally (`isInvestmentRecurring && <HoldingPicker />`), exactly mirroring `Expenses.tsx`'s existing `isTaxExpense && <tax_year field>` block.
- No other change to the Recurring domain: skip, edit, and delete all already operate on whichever fields a template has, `holding_id` included, with no special-casing needed beyond the copy in `materialise`.

## Testing Decisions

- Every new/changed endpoint is tested at the same single seam the rest of the backend already uses exclusively: black-box HTTP tests against a real handler and a real (temporary) SQLite database, through the shared `testApp` harness (`cmd/app_test.go`'s `a.get`/`a.post`/`a.patch`/etc.). No mocking, no handler-internals testing — this is the only seam that exists in this codebase and every existing report/CRUD test already follows it.
- Prior art to follow directly:
  - `reports_test.go` (`a.month`, `a.year`, `a.breakdown`, `a.tax` helper pattern) for the new `daily`, `estimate`, `budget`, and `savings` report endpoints — add matching `a.daily(...)`, `a.estimate(...)`, `a.budget(...)`, `a.savings(...)` helpers.
  - `categories_test.go` for Holdings CRUD (list/create/rename), adapted for `type`'s enum constraint.
  - `recurring_test.go`'s clock-controlled `materialise` tests for verifying a recurring investment's `holding_id` is copied into each generated Expense, using the same `a.moveClock`/`a.setNow` pattern already used to test rent generation.
  - `expense_test.go`/`income_test.go` for `holding_id` on ordinary buy/sell rows, following the same shape as their existing `tax_year` tests.
- Migration correctness (the `Investimenti` category promoted in place, not duplicated) is tested the same way `migrate_test.go`-style tests already verify seed data — a fresh database at the new `schemaVersion` should have exactly one category with `code = 'investments'`, `applies_to = 'both'`, and no second `Investimenti`-named row.
- No frontend automated tests are added — the repository has no frontend test runner today (`web/package.json` has no `test` script), and this spec doesn't introduce one. Frontend changes are verified manually against a running instance, per `CLAUDE.md`.

## Out of Scope

- Real-time or live pricing/valuation of Holdings — only money contributed/withdrawn is tracked, never market value.
- Multiple portfolios or any grouping above a single flat Holding list.
- A starting balance for a pre-existing Holding position — represented instead as a backdated buy Expense (harmless, since Investments is already excluded from Budget/Target math).
- Target-allocation tracking (a percentage you're aiming for) — only actual contributed percentage is shown.
- A `hidden`/archive flag on `holding` — a fully-sold Holding simply nets to zero and drops out of the breakdown; deletion/archival is deferred.
- Any change to Payer, settling-up, or multi-user support (ADR-0001 stands untouched).
- Alerts or notifications when a Target is exceeded — display-only in this version.
- Any currency other than EUR.
- A second, independent "weekly" backend endpoint — weekly is a client-side bucketing of daily data.
- Per-category Budget or Target — both are whole-household totals only.

## Further Notes

- Builds on ADR-0001 (no accounts/balances between people — unaffected), ADR-0002 (Item-inclusive category attribution, reused as-is for the daily breakdown), ADR-0003/ADR-0004 (cash-basis totals — why the Estimate is the deliberate, labeled exception), ADR-0005 (Recurring generation/`materialise` — reused unchanged for PAC support), ADR-0008 (Base category protection pattern — reused for the new Investments category).
- New/updated glossary terms already recorded in `CONTEXT.md`: Holding, Savings, Budget, Target, Goal, Estimate; the Base category entry now also names Investments.
- New ADRs already recorded: `docs/adr/0009-investments-count-as-ordinary-expense-income.md`, `docs/adr/0010-yearly-estimate-is-a-labeled-projection.md`.
- This spec covers six related but separable pieces of work (daily/weekly charts, the Estimate, Budget/Target/Goal, Investments/Holdings, Savings, recurring investments). When splitting into tickets, the schema migration and Holding CRUD are the natural first ticket everything else in the Investments/Savings area depends on; the daily/weekly charts and the Estimate have no dependency on the Investments work and can proceed independently.
- All new UI strings go through the existing `web/src/lib/strings.ts` (`t.`) pattern, Italian-first, matching the household-facing language already seeded (`Tasse`, `Alimentari`, etc.).
