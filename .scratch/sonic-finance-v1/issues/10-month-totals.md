# 10: Month totals

**What to build:** Opening the app answers the first question it exists to answer: where does this month stand? Money in, money out, the difference — and the ability to walk back through previous months.

**Blocked by:** 05, 09

**Status:** ready-for-human

- [x] A month view shows total in, total out, and the difference
- [x] Only received Incomes contribute to money in
- [x] Navigation between months, any month, past or current
- [x] The calendar month is the reporting unit; arbitrary date ranges are out of scope
- [x] All month reads go through one path, so later tickets have a single place to hook into
- [x] Tests use a fixed clock and never read the wall clock

## Comments

Handoff from ticket 09 (Income), which is done. One thing this ticket inherits, and it is the assertion 09 could not make:

- **Ticket 09's last box is open and this ticket closes it.** It reads "tests cover an unpaid Income being excluded from totals and then included once paid" — and there was no total to be excluded from, because this is the ticket that builds it. Ticket 09 asserts the *state* those sums filter on (an unpaid Income round-trips with an empty `payment_date`, a payment date can be added later and taken back again); what is still owed is the *arithmetic*: **an unpaid Income absent from a month's money-in, and the same Income present once it has a payment date.** That test belongs here, and it is what the second box above amounts to in practice.
- **Unpaid has exactly one spelling: `payment_date IS NULL`.** Both of an Income's dates cross the API as `""` and are stored NULL, and the columns `CHECK (payment_date <> '')` so an empty string cannot get in by another route. So the filter every sum needs is `payment_date IS NOT NULL` and nothing more elaborate — no `!= ''`, no `COALESCE`. ADR-0003 calls omitting it the single most likely bug in this codebase; the column shape is what makes forgetting it fail loudly rather than quietly overcount.
- **`income` has no date of its own for a month to filter on except the payment date.** A received Income belongs to the month its money arrived — `payment_date` — not the month its invoice went out. `invoice_sent_date` is for "how long has this been sitting", which is ticket 14's question, not this one's.


---

Implemented. `reports.go` holds the endpoint and the one read path; `web/src/Month.tsx` is the screen, and it is what the app now opens on.

- **`GET /api/reports/month/{yyyy-mm}` answers four fields**: the month, `income_cents`, `expense_cents`, `net_cents`. The difference is computed server-side rather than left to the client — it is the number the household actually reads, and there is no second opinion to be had about it.
- **The assertion ticket 09 owed is paid.** `TestAnUnpaidIncomeIsAbsentFromTheMonthUntilItIsPaid` records an invoice with no payment date, reads the month and finds money in at zero, PATCHes a payment date on, reads it again and finds the amount — then takes the payment back and watches it leave again. Ticket 09's last box is ticked and points here.
- **Unpaid is spelled once, and said out loud.** The Income sum filters `payment_date IS NOT NULL AND substr(payment_date, 1, 7) = ?`. The first half is redundant against the second — a NULL date matches no month — and it stays: ADR-0003 calls the missing filter the single most likely bug here, and every query that sums Income is meant to name the rule rather than rely on a `substr` happening to imply it.
- **An Income belongs to the month its money arrived.** `payment_date`, never `invoice_sent_date`, which answers ticket 14's question instead. `TestAnIncomeBelongsToTheMonthItWasPaidNotTheMonthItWasInvoiced` holds it in both directions across a month boundary.
- **`readMonth` is the single path**, and the comment on it says why: ticket 11's Category breakdown reads the same month, and ticket 13 materialises that month's Recurring expenses before anything is summed (ADR-0005). One function to hook, not one per report. No empty hook was added for it — there is nothing to generate yet, and a stub would be scaffolding bought on credit.
- **A month that is not a month is a 400, not zeros.** `substr` against `"2026-3"` matches nothing, so summing it would answer an empty month — indistinguishable from a real one in which nothing happened. `time.Parse` with the zero-padded `2006-01` refuses `2026-13`, `2026-00`, `2026-3`, `2026`, `marzo` and `2026-09-02` alike.
- **No handler here reads the clock.** Which month to show is entirely the browser's question — the phone has a correct clock and the Pi has no RTC (the spec's own note) — so the server has no sense of "this month" to be wrong about, and the tests need no clock at all beyond the fixed one every test already runs at.
- **No schema change.** Both sums are over columns that already exist, so `schemaVersion` stays at 7.
- **Tests** (`reports_test.go`, through the one HTTP seam): the three numbers together; the unpaid/paid assertion above; payment month vs invoice month; the four-date boundary test where neither 28 February nor 1 April leaks into March; an empty month as zeros rather than an error; a negative difference when more went out than came in; and six refusals. Clean under `-race`.
- **Frontend**: `Month.tsx` — three rows between `‹` and `›`, and a native `<input type="month">` as the heading. The arrows are for the month either side; the picker is for *any* month, because comparing against a month three years back should not be thirty-six taps. Both are capped at the current month — `max` on the input, `disabled` on the arrow — since there is no such thing as next month's total; `month >= thisMonth()` is a string comparison, which is a date comparison because `YYYY-MM` sorts. `<input type="month">` is the same native-picker bargain the two entry screens already make with `type="date"`, and it is why nothing here formats a month name. A slow response for a month already navigated away from is discarded rather than allowed to overwrite the one on screen, and a failed one clears the numbers instead of leaving the previous month's three totals sitting under this month's heading.
- **The two totals borrow the nav's own words** — *Entrate* and *Spese*, the screens they are the sum of — rather than introducing *Uscite* as a second word for an Expense. One concept, one word, which is what CONTEXT.md's Avoid lists are for. Only *Differenza* is new.
- **`formatCents` was broken for negative cents and is now fixed**, which is what the difference row turned up: the euros and the cents are two integers and the sign was being formatted into both, so −65,43 printed as `-65,-43`. Fixed in `money.ts` rather than worked around in the screen, because ticket 15's year view will want the same number.
- **`today()` moved from `Expenses.tsx` to `money.ts`** and gained `thisMonth` and `shiftMonth` for company, rather than local date helpers in a second file. `shiftMonth` uses `Date` for the carrying, so December is what January minus one month is without this code having to know it.
- **The month view is what the app opens on** (`App.tsx`), replacing Expenses as the default screen — the ticket's own first sentence. Expenses is still first in the nav after it.
- **Verified against the real shipped binary**: unauthenticated 401, an empty month as zeros, an Expense and a received Income summing and differencing correctly, an unpaid invoice absent and then present the moment it was paid, a neighbouring month still zero, a month that spent more than it received answering `net_cents: -65000`, a month in 2019 answering zeros, five malformed months each answering 400 with the rule, and the Italian strings, the month input and the cleared-on-failure path all in the built bundle.

## Handoff to ticket 13

**`readMonth(db, month)` will have to widen, and that is expected.** It takes no clock, because nothing in this ticket has a use for one — the browser decides which month to ask for, and the server only sums what the path names. Ticket 13 needs one: "nothing is ever generated beyond the current month" and the clock guard are both questions about `now`, and `app.go` already threads `now` into every handler that needs it, so `handleMonthReport(db, now)` → `readMonth(db, now, month)` is the shape. Adding the parameter here, unused, would have been scaffolding bought on credit.

It also takes a `*sql.DB` rather than a transaction, so ADR-0005's generate-then-read will not be atomic without one. Idempotent generation is what makes that safe, and idempotence is a box ticket 13 carries anyway — but if it prefers a transaction, this is the function to change and nothing outside it needs to know.

## Not built, and not asked for by this ticket

- **The Category breakdown and the recent-entries list** are ticket 11's, which extends this same month read.
- **Recurring materialisation.** `readMonth` is where it goes and says so; there are no Recurring expenses to generate until ticket 12 creates the table.
- **Arbitrary date ranges.** The spec rules them out: calendar month and calendar year only. The month picker is not one — it chooses which calendar month to report, which is story 55's "any past month".
