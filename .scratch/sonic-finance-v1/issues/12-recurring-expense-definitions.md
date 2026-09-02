# 12: Recurring expense definitions

**What to build:** The rent stops being something to retype. A Recurring expense is defined once, with the months it covers — but nothing is generated yet; that is 13.

**Blocked by:** 06

**Status:** ready-for-human

- [x] A Recurring expense holds the same fields as an Expense, plus a day of the month
- [x] It has a start month and an optional end month; empty end month means ongoing
- [x] Creating one defaults the start to the current month, and the start can be set back deliberately
- [x] "Deactivating" sets the end month to the current one; there is no separate active flag — the window is the record, per ADR-0005
- [x] Changing an amount is done by ending one and creating another, so history keeps what was true at the time
- [x] A screen lists Recurring expenses and shows which are still running

## Notes

- **Schema step 7, `schemaVersion` 8.** `recurring_expense` as the spec's Schema section shapes it, plus the one column ticket 05's migration deliberately deferred: `ALTER TABLE expense ADD COLUMN recurring_id INTEGER REFERENCES recurring_expense(id)`. `migrateExpenses`' own comment handed that here — "it points at a table ticket 12 creates, and adding it there keeps it a real foreign key rather than a bare integer" — and it can only be added once the table exists. Nothing writes it yet; ticket 13 does. `recurring_skip` is ticket 13's, and is not here.
- **The window is the whole of the state.** No active flag, no `hidden`, nothing deleted on deactivation — ADR-0005. `end_month` is NULL for ongoing and flattened to `""` across the API, which is the same bargain an Income's two dates make and is what lets a PATCH merge onto the stored row. Both months are `YYYY-MM` and zero-padded, so ticket 13's `start_month <= M <= coalesce(end_month, M)` is a plain string comparison.
- **Reactivation is refused, and it was the review's real finding.** ADR-0005 answers reactivation with a *new* definition — "the gap months genuinely had no payment" — and the first draft's partial PATCH quietly allowed the opposite: `{"end_month": ""}` on an ended row reopened the window, and ticket 13 would then have generated every month of the gap. `handlePatchRecurring` now refuses exactly that merge, and nothing more: an end month typed wrongly is still moveable, because moving it invents no month. The screen refuses it too, in Italian, before sending.
- **The changeover month was the second real finding.** "Nuovo importo" ends the old definition at the current month and prefills a new one — originally starting the *same* month, which, because a window covers its end month, would have produced two rents for it. It starts the month after.
- **"In corso" is `end_month > thisMonth()`, not `>=`.** A definition ended today still covers today's month, which is what the window line says; it is no longer running, which is what the label says. With `>=`, pressing "Termina" changed nothing visible and still offered to end it again.
- **`day_of_month` defaults to the day it was set up**, per the spec, and 29/30/31 are stored as meant rather than pre-clamped — ticket 13 clamps to the month's length. An explicit `0` is indistinguishable from an omitted field to a JSON decode and is no day anyone means, so it is the sentinel the default fills in rather than a value to refuse.
- **No Items breakdown.** An Item is the exception on a receipt; a fixed monthly cost is one number under one Category, and the spec's `recurring_expense` shape carries no recurring one.
- **Delete answers 409 once it has produced anything**, through the foreign key and `writeError`'s existing correction — the money did leave the account, so a real Recurring expense is stopped by ending its window, not by erasing the months it produced. Verified by planting a row with `recurring_id` set, since nothing writes it until 13; no test covers it for the same reason.
- **`validMonth` is now shared with `handleMonthReport`**, which had the same `time.Parse(monthLayout, …)` inline.
- **Frontend**: `RecurringExpenses.tsx`, a fifth nav screen. Native `<input type="month">`, so the phone gives its own picker and the value is already the `YYYY-MM` the API wants. Whether one is running is computed against the *browser's* month, like every other date on the frontend — the Pi has no RTC.
- **Tests** (`recurring_test.go`, through the one HTTP seam): the full round trip with every template field; the start defaulting to the clock's month and following it when the clock moves; a start set back to January; ongoing surviving as `""`; deactivation leaving the start alone; an end before the start refused on both create and edit; the reactivation refusal, the end month still moveable, and an unrelated edit not reading its omission as a clear; the amount change as an ending plus a new definition; the day defaulting, `31` stored as meant, and `-1`/`32`/`100` refused; four malformed months; an income-only and a nonexistent Category; delete and the 404s. Clean under `-race`.
- **Verified against the real shipped binary**: 401 unauthenticated; a definition created with the real clock defaulting to `2026-09` and day `2`; a start set back to `2026-01` with `day_of_month` 31; deactivation by PATCH; all five refusals answering 400 with their own sentence; `PRAGMA foreign_key_list(expense)` showing `recurring_id → recurring_expense(id)`; a planted generated Expense turning DELETE into 409 while an unused definition deletes 204; the reactivation refusal and the end month still moveable; the month report still refusing `2026-9` and answering `2026-09`; the SPA shell at 200 and the bundle carrying `/api/recurring`, "Nuovo importo" and the reactivation sentence. The ARMv6 binary carries `recurring_expense`.

## Spec deviation

None. The endpoints are the spec's API line exactly, and the table is its Schema shape.

## Flagged, as the spec asked

The spec's Recurring generation section says: *"`day_of_month` defaults to the day the Recurring expense was created. **This was decided while writing the spec rather than during design — flag it if you would rather every generated Expense simply landed on the 1st.**"* Built as specified, and worth keeping: the rent leaves on the 5th and the household recognises it there. But the default is the weak half — a Recurring expense set up on the 23rd because that is when someone got round to it now claims the 23rd is meaningful. The 1st would be a more honest default with the field still editable. Say the word and it is a one-line change.

## Not built, and not asked for by this ticket

- **Generation.** No Expense is produced, no `recurring_skip` table, no clock-sanity guard (spec story 72). All ticket 13.
- **A `running` field on the API.** It would be the Pi's clock answering a question the phone's clock answers correctly, and the window is already on the wire.
- **Refusing to edit an amount.** History is kept by the *window*, and the months already generated carry their own amounts, so an amount typed wrong on the day it was created still has to be fixable. "Nuovo importo" is the offered path, not the only one.
- **Moving a start month backwards is still allowed.** The ticket asks for the start to be settable back deliberately; ADR-0005 names only reactivation, so only that is refused.
