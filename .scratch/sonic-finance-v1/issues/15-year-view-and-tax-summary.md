# 15: Year view and tax summary

**What to build:** The actual figures, as against the forecasts the invoicing software already gives: for a given year, what was really received in freelance income and what was really paid in tax.

**Blocked by:** 10

**Status:** ready-for-human

- [x] A year view with year navigation, showing the shape of the whole year
- [x] A Tax year can be recorded on tax Expenses, defaulting to the year of the expense date and editable — because tax on 2026's income is paid during 2027
- [x] The Tax year field is surfaced only for Expenses in a tax Category
- [x] The summary reads "received X, paid Y in tax, net Z%" — see ADR-0008
- [x] X is received Income in the `freelance` Base category only: employment income arrives already taxed and gifts are not income, and including either makes the percentage meaningless
- [x] Y is Expenses in the `taxes` Base category, attributed by **Tax year, not payment date**
- [x] One total; no per-tax-type breakdown
- [x] Tests cover tax paid in one year attributing to the previous year's summary

## Notes

- **No schema change.** `tax_year INTEGER` has been on `expense` since schema step 2 — ticket 05's migration created more columns than it wrote, deliberately, because a migration case can never be edited afterwards. This ticket is the one that fills it. `schemaVersion` stays at 9.

- **Two endpoints, as the spec's API list has them.** `GET /api/reports/year/{yyyy}` answers the shape of the year; `GET /api/reports/tax/{yyyy}` answers the year in tax terms. They are not one endpoint because the second is not a sum of the first: received is cash in the year, tax paid is attributed by Tax year, and the two halves are counted differently on purpose. The year screen fetches both in one `Promise.all`.

- **A year is twelve months read through the one month path.** `readMonthRow` is what `readMonth` was, minus the Category breakdown; `monthTotals` now embeds `monthRow` and adds `by_category`, so a month report's JSON is byte-identical to what it was. The year loops `readMonthRow`, which is what makes a year view generate the Recurring expenses each of its twelve months owes — ADR-0005's "a month nobody opened at the time is not silently empty forever" is most of what a year view is for, and `TestAYearGeneratesTheRecurringExpensesItsMonthsOwe` pins it, second read included.

- **The year carries no Category breakdown, and the split is why.** An earlier draft had `months` as twelve full `monthTotals`, on the reasoning that `readMonth` had already computed the breakdowns. Both reviews called it: twelve runs of the heaviest query in the app, on ARMv6, for a screen that reads none of them and would have to sum them back together to say anything. The month screen is one tap away and answers that question for the month actually being asked about.

- **A Tax year is an override of a default, not the only place the default lives.** `checkExpense` derives it on the way in — defaulted to the year of `occurred_on`, cleared outside a tax Category — so a typed tax Expense always stores one. `materialise` copies template fields and knows nothing about tax, so a *generated* one stores NULL, and `handleTaxSummary` therefore reads `COALESCE(e.tax_year, CAST(substr(e.occurred_on, 1, 4) AS INTEGER))`. That is one rule stated in two places rather than two rules: a tax Expense that says nothing belongs to the year it was paid in.

- **The bug that hole was.** Before the COALESCE, a recurring tax payment — a monthly INPS instalment is the obvious one — generated Expenses with `tax_year` NULL, which `WHERE tax_year = ?` matched for no year at all: the money landed in the year view's `expense_cents` and vanished from the tax figure. Both review axes found it independently. Fixing it in the query rather than at generation also covers the rows already generated on the Pi. `TestARecurringTaxPaymentIsCountedByTheSummary` pins both halves — the instalments counted, and a typed Tax year still overriding one of them.

- **`isTaxCategory` resolves by `code`,** like every other read of a Base category (ADR-0008): the household cannot rename Tasse, but no report should depend on that being true. Hidden is not part of the question — a tax payment recorded under a Category since hidden is still a tax payment, and `TestOnlyTaxCategoryExpensesCountAsTaxPaid` hides it and checks the summary still counts it.

- **`net_percent` is null, not zero, when nothing was received.** A percentage of nothing is not 0%, and a screen reading "net 0%" over an empty year would be a wrong answer where there is no answer. It is the one float in the codebase, and it is not money — the note says so where it is declared.

- **The Tax year field is surfaced by `base` on the expense side.** The API deliberately does not publish a Category's `code`, so `Expenses.tsx` asks whether the chosen Category is a Base one: there are two, and the other is income-only, so it can never be what the Expense form has chosen. The bounds are the browser's — `type=number` with `min`/`max` refuses a half-typed `202` in the phone's own language, and clearing the field snaps back to the payment's year rather than showing empty.

- **Tests** (`reports_test.go`, `expense_test.go`, through the one HTTP seam): the twelve months and their totals, and only the year's own months; a year refused for `26`, `2026-03`, `twenty`, `20264` and empty; twelve months' rent generated by looking, idempotent on a second read; received/paid/net and the percentage; **tax paid in 2027 attributing to 2026's summary**, with a defaulted 2027 payment staying in 2027; freelance income only, against all four other seeded income Categories; unpaid freelance income excluded and then landing in the year it was paid; a hidden tax Category still counted; a year with nothing received having no percentage, including tax paid against no income; the Tax year defaulting, editable, surviving an unrelated PATCH, cleared outside a tax Category and restored on the way back, and refused as `26`; and the recurring instalment case above. Clean under `-race`.

- **Verified against the real shipped binary** (clock 2026-09-03): a €45.000 freelance income and a €300 monthly tax instalment recurring from January reading `received 4500000, tax_paid 270000, net_percent 94` — nine instalments generated by the look itself; the year totalling `276200` out with the itemised shop in it; `months[2]` carrying four keys and no breakdown; the month report's JSON unchanged, breakdown and all; and `26`, `2026-03`, `twenty` refused with 400 on both endpoints.

## Beyond the letter of the ticket

- **The Tax year is cleared when an Expense leaves a tax Category.** Neither the ticket nor the spec says what happens then. The alternative is a stale year sitting on a row nothing reads, waiting to be wrong on the day it is moved back — so the field follows the Category, and moving one back gives it the default again. `TestATaxYearFollowsTheCategoryTheExpenseIsMovedTo` fixes the choice; one branch to drop if the household would rather it were sticky.

- **A pair of bars per month on the year screen.** The ticket asks for "the shape of the whole year", and twelve rows of two numbers is a table rather than a shape. Each month gets an income bar and an expense bar scaled to the biggest month of the year, so the shape is visible without reading twenty-four numbers. Eight lines, and one component to delete if it reads as decoration.

## Not built, and not asked for by this ticket

- **A per-tax-type breakdown** (IVA / INPS / IRPEF). Explicitly out of scope in the ticket, the spec and ADR-0008: with a flat Category list it needs either protected categories per Italian tax name or the two-level categories the design deferred. One total was what was asked for.
- **A year-level Category breakdown.** See the note above on why the year carries none. It is `readBreakdown` over a year rather than a month whenever someone asks for it — one query, not a redesign.
- **Tapping a month in the year view to open that month.** It wants the current month lifted out of `Month.tsx` into `App.tsx`, and the nav is already two taps. Worth doing when the nav is rebuilt.
- **Anything past the current year.** The forward arrow stops there, as the month screen's does: a year that has not happened has no totals, and generating for it is refused anyway.
