# 10: Month totals

**What to build:** Opening the app answers the first question it exists to answer: where does this month stand? Money in, money out, the difference — and the ability to walk back through previous months.

**Blocked by:** 05, 09

**Status:** ready-for-agent

- [ ] A month view shows total in, total out, and the difference
- [ ] Only received Incomes contribute to money in
- [ ] Navigation between months, any month, past or current
- [ ] The calendar month is the reporting unit; arbitrary date ranges are out of scope
- [ ] All month reads go through one path, so later tickets have a single place to hook into
- [ ] Tests use a fixed clock and never read the wall clock

## Comments

Handoff from ticket 09 (Income), which is done. One thing this ticket inherits, and it is the assertion 09 could not make:

- **Ticket 09's last box is open and this ticket closes it.** It reads "tests cover an unpaid Income being excluded from totals and then included once paid" — and there was no total to be excluded from, because this is the ticket that builds it. Ticket 09 asserts the *state* those sums filter on (an unpaid Income round-trips with an empty `payment_date`, a payment date can be added later and taken back again); what is still owed is the *arithmetic*: **an unpaid Income absent from a month's money-in, and the same Income present once it has a payment date.** That test belongs here, and it is what the second box above amounts to in practice.
- **Unpaid has exactly one spelling: `payment_date IS NULL`.** Both of an Income's dates cross the API as `""` and are stored NULL, and the columns `CHECK (payment_date <> '')` so an empty string cannot get in by another route. So the filter every sum needs is `payment_date IS NOT NULL` and nothing more elaborate — no `!= ''`, no `COALESCE`. ADR-0003 calls omitting it the single most likely bug in this codebase; the column shape is what makes forgetting it fail loudly rather than quietly overcount.
- **`income` has no date of its own for a month to filter on except the payment date.** A received Income belongs to the month its money arrived — `payment_date` — not the month its invoice went out. `invoice_sent_date` is for "how long has this been sitting", which is ticket 14's question, not this one's.

