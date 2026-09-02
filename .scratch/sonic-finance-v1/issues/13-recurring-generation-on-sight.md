# 13: Recurring generation on sight

**What to build:** The hardest ticket here. Recurring expenses turn into real Expenses by themselves, with no scheduler anywhere — because the Pi may be switched off when any timer would have fired. Anything that reads a month generates that month's missing Expenses first, then reads.

**Blocked by:** 10, 12

**Status:** ready-for-human

- [x] Any read covering a month materialises that month's missing Expenses before reading — see ADR-0005
- [x] Generation is idempotent: opening a month twice does not produce two rents
- [x] An Expense is generated only for months inside the Recurring expense's start/end window
- [x] A report over an old month still generates that month's Expenses, so a month nobody opened at the time is not permanently empty
- [x] Nothing is ever generated beyond the current month
- [x] The generated date is the day of month, clamped to the month's length — a 31 becomes the 30th in April
- [x] A generated Expense is indistinguishable in use from a typed one: editable and deletable with no special rules
- [x] Deleting a generated Expense records a skip for that Recurring expense and month, so it does not come back
- [x] **Clock guard:** the Pi has no real-time clock. If the clock reads earlier than a build-time constant, the app has booted without network time — it logs loudly, refuses to generate, and exposes the condition so the UI can warn. Generating nothing is recoverable; generating against a wrong clock is not
- [x] Tests cover every bullet above with a fixed clock, including reading the same month twice and reading a month long past

## Notes

- **Schema step 8, `schemaVersion` 9.** `recurring_skip (recurring_id, month)` as the spec's Schema section shapes it, primary key on both — so re-skipping a month is a no-op rather than a second row. `ON DELETE CASCADE` for the reason an Item's is: a skip has no life outside its definition, and without the cascade a definition whose generated Expenses had all been deleted would be undeletable by its own skip rows. `recurring_id` on `expense` was already there, added by ticket 12's step as a real foreign key; this ticket is the first thing to write it.

- **One hook, and it is `readMonth`.** Ticket 11 left it as "the single path every month read goes through", and that turned out to be the whole of the integration: `materialise(db, now, month)` on the first line, before the three queries under it. `handleMonthReport` now takes `now` for that, which is the one clock any report has ever needed. Ticket 15's year view reads twelve months through the same function and gets generation for free.

- **Generation is one SQL statement, and that is what makes it idempotent under load.** `INSERT INTO expense … SELECT … FROM recurring_expense r WHERE window AND NOT EXISTS(expense for that pair) AND NOT EXISTS(skip)`. A check-then-insert in Go would let two phones loading the same month both decide the rent is missing; a single statement evaluates its own `NOT EXISTS` inside the write. No unique index backs it up deliberately — an index on `(recurring_id, month)` would make editing a generated Expense's date into an already-covered month fail, which is exactly the "second set of rules" the ticket forbids.

- **The clamp is Go arithmetic and a SQL `min()`.** The month's last day is `start.AddDate(0,1,0).AddDate(0,0,-1).Day()`, passed as a parameter, so the SQL is `printf('%s-%02d', ?, min(r.day_of_month, ?))` rather than a `date(…,'+1 month','-1 day')` expression nobody wants to decode at 3am. Verified on a 30-day month, a 31-day month, February, and a leap February.

- **The clock floor is `2026-09-01`, and it is a `var` so the suite can move it.** The spec asks for a build-time constant; the honest one is the month the binary was built, which sits *after* the suite's fixed `testClock` of 2026-03-15 — every existing test would otherwise have run as a Pi with no network time. `TestMain` lowers it to 2020, the same bargain `bcryptCost` already makes, which leaves a test that sets the clock to 1970 still obviously wrong. Bumping the floor on a real deploy tightens the guard and costs one line.

- **Refusing to generate is not refusing to read.** A bad clock logs `CLOCK UNSET` (once at startup, and at every read that would have generated), returns `nil`, and the month still reports what was typed. `/api/health` gained `clock_ok`, and `App.tsx` warns on it in Italian above whatever screen is open. The frontend treats a missing field as fine, so a stale bundle against a new server warns about nothing rather than warning always.

- **The skip is written from the row, not from the request.** `skipMonth(tx, id, staying)` runs `INSERT OR IGNORE INTO recurring_skip SELECT recurring_id, substr(occurred_on,1,7) … WHERE id = ? AND recurring_id IS NOT NULL AND substr(occurred_on,1,7) <> ?`, before the write that changes the row and inside the same transaction. A typed Expense inserts nothing, so there is no branch that could fall out of step — and `recurring_id` never has to cross the API to make it work.

- **A cross-month edit was the review's real finding, and it is fixed.** Generation asks whether that (Recurring expense, month) has an Expense, so PATCHing February's rent to March emptied February in generation's eyes and the next read produced a second rent for a payment that happened once — two rents in the ledger, which is exactly the "second set of rules" the ticket forbids. `handlePatchExpense` now calls the same `skipMonth` the delete does, with the month it is moving *to* as `staying`: a correction inside the month skips nothing, because the rent is still in the month it was paid in. That is what the `staying` parameter is for, and a delete passes `""` because it keeps no month at all. Covered by a test that was confirmed to fail without the fix, and verified on the real binary — a rent moved June→July leaves June at €0 with `recurring_skip` holding `1|2026-06`, one Expense in total, and a second same-month correction adding no further skip.

- **`recurring_id` stays off the API entirely.** The ticket's "indistinguishable in use" is easiest to keep by not publishing the distinction: the expense JSON is unchanged, the frontend has no idea which Expenses were generated, and there is no field for a client to send.

- **`logClockUnset` is one wording, said in two places.** The startup line and the per-read refusal had drifted into two sentences with the same fields; this is the line whoever is wondering where the rent went has to be able to find, so there is one of it.

- **Tests** (`recurring_test.go`, through the one HTTP seam): the full round trip with every template field copied onto the generated Expense and no Expense existing until a month is read; the same month read twice, and a third time through the breakdown; the window's three cases including its end month being covered; a month twenty months past; next month generating nothing and then filling in when the clock reaches it; the clamp on four months including a leap February; a generated Expense edited and the edit surviving the next read; deleting one keeping it deleted while the months either side still generate, and a typed delete skipping nothing; a generated one moved to another month and back leaving exactly one rent; the 1970 clock refusing, `clock_ok` false, the read still answering, and everything owed appearing on the next read once the clock is right. Clean under `-race`.

- **Verified against the real shipped binary** (clock 2026-09-02, above the floor, so no `CLOCK UNSET` at startup): `clock_ok:true` and `schema_version:9`; a day-31 rent from `2026-06` generating `2026-06-30` and `2026-07-31`; `2026-05` (before the start) and `2026-10` (future) both empty; the June read repeated with no second rent; the generated one edited to €860,00 on the 28th and the edit surviving the next read; `DELETE` answering 204, June then reporting €0 with `recurring_skip` holding `1|2026-06`, and July untouched; `DELETE /api/recurring/1` now answering 409 for real rather than from a planted row; the SPA shell at 200 and the bundle carrying `clock_ok` and the warning sentence. The ARMv6 binary carries `recurring_skip`.

## Spec deviation

One, in wording rather than behaviour: the spec asks for the clock to be compared "against a build-time constant", and `clockFloor` is a `var` so `TestMain` can lower it below the suite's fixed clock. It is a compile-time literal that nothing but the test binary touches, and it follows the precedent `bcryptCost` already set.

## Not built, and not asked for by this ticket

- **Generation on the Expense list.** The spec names "month views, year views, and reports" as what materialises, and `/api/expenses` is none of those — it is every month at once. The month view is the screen the app opens on, so the rent is generated before anything else is looked at, and it is in the list from then on. A month filter on that endpoint is already noted there as the point to revisit.
- **A unique index on `(recurring_id, month)`.** See above: it would add a rule a generated Expense is not supposed to have.
- **A year view.** Ticket 15 — it reads twelve months through `readMonth` and generates all twelve on the way.
