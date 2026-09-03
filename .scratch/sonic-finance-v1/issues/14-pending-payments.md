# 14: Pending payments

**What to build:** The freelancer stops reading a spreadsheet carefully to work out who owes them. Two lists: money genuinely owed, and invoices that look forgotten.

Note the vocabulary: **Pending payment**, never "receivable" — this app does not speak accounting.

**Blocked by:** 08, 09

**Status:** ready-for-human

- [x] A list of every Income with no payment date, oldest first, showing how many days it has been waiting
- [x] A second list of Clients with at least one freelance Income in the previous three calendar months and nothing recorded this month
- [x] The second list is limited to freelance-category income, so a one-off gift is never treated as a missing invoice
- [x] The three-month window is a code constant, not a setting
- [x] Everything surfaces as an in-app notice — no email, no push, nothing that depends on the Pi being awake
- [x] Tests cover a Client falling in and out of the second list as months pass on a fixed clock

## Notes

- **No schema change.** Both lists are views over `income` as it already stands — ADR-0003's `payment_date IS NULL` is the whole of the first one, and the second groups the same rows by Client and month. `schemaVersion` stays at 9.

- **One endpoint, two lists, `pending.go`.** `GET /api/pending-payments` answers `{outstanding, not_yet_invoiced}`. They are read together on one screen, so one round trip; they are separate arrays because they are separate questions, and the screen shows either, both, or neither.

- **"Billed on" is one expression, and it takes the table alias.** `billedOn(alias)` is `COALESCE(alias.invoice_sent_date, substr(alias.created_at, 1, 10))`: an unpaid Income has no payment date by definition, so the invoice date is the only date it has, and the day it was typed is the fallback. Sliced to ten characters rather than parsed, because every date here is compared as text. It takes the alias rather than reading a bare column name because the nudge asks it of two Incomes in one statement — the outer one and the one inside its `NOT EXISTS` — and which of them a bare name binds to is not something to work out at 3am.

- **The window is `billingWindowMonths = 3`, and the month arithmetic goes through the month string.** `time.Parse(monthLayout, now())` gives the 1st, and `AddDate(0, -3, 0)` off the 1st cannot slip — off the 31st of May it would land in March. The window is `>= from AND < thisMonth`, so "the previous three calendar months" excludes the current one, and the `NOT EXISTS` on the current month is what "nothing recorded this month" means.

- **Freelance is resolved by code, not by name.** The nudge joins `category.code = 'freelance'` (ADR-0008), so renaming the Base category leaves it working, and a gift from a relative is never a missing invoice. The *first* list is deliberately not filtered that way: a reimbursement that never arrived is money waiting as much as an invoice is.

- **Hidden Clients are not chased.** Hiding is how a Client is retired, and the app has no business asking for an invoice for someone the household stopped billing. This is the opposite of what `checkIncome` does with `hidden` — recording an Income for a hidden Client is still allowed — and deliberately: one is a correction to history, this is a prompt to act.

- **The timezone bug the suite could not see.** `days_waiting` subtracts the stored date from the clock's date. `time.Parse` anchors a stored date in UTC; anchoring the clock's date in the clock's own zone made the two sides differ by the offset, and the real binary on Italian time reported an invoice sent on 1 August as waiting 32 days on 3 September. The clock's calendar date is now anchored in UTC too, where there is no DST either, so every subtraction is a whole number of days. The suite ran green through all of it — its clock is UTC — so `TestDaysWaitingDoNotDependOnTheServersTimezone` runs on a `+02:00` clock and was confirmed to fail (32, want 33) without the fix.

- **No clock guard, and the honest version of why.** An unset Pi reads 1970: the window then contains no Income, so the nudge is empty, and `daysSince` floors at zero so nothing waits a negative number of days. Reading is still not refused — what is unpaid is unpaid whatever the clock says, and unlike generation there is nothing to lose by answering. The cost is real and worth naming: every "days waiting" reads as zero, so a three-month-old invoice says it was sent today. That is what `/api/health`'s `clock_ok` and `App.tsx`'s warning above every screen are for, and `TestAnUnbelievableClockStillAnswersWhatIsOwed` pins all three halves of it. An earlier draft of this note claimed both lists "degrade to saying nothing rather than something wrong"; the review caught that the first one does not, and the comment in `pending.go` now says what actually happens.

- **The notice is a section on the month screen** (`PendingPayments.tsx`), above the Category breakdown and fetched once — what is owed is owed whichever month the screen is showing. It renders nothing at all when both lists are empty, so a household owed nothing sees no heading about it. The amounts are muted and carry no `+`: this money has not arrived and is in none of the totals above it (ADR-0003). Nothing here is an email or a push (story 42).

- **Tests** (`pending_test.go`, through the one HTTP seam): three unpaid Incomes oldest first with their days and names, a paid one absent; recording the payment removing one; an Income with no invoice date waiting from the day it was typed; a Client falling in and out of the nudge across four positions of a fixed clock, including June as the window's far edge; a gift never starting a nudge and never clearing one; a hidden Client not chased; an invoice sent this month being outstanding without also being missing; both lists as `[]` when nothing is pending; the 1970 clock; and the timezone regression. Clean under `-race`.

- **Verified against the real shipped binary** (clock 2026-09-03, Italian time): an unpaid August invoice reading 33 days and a clientless gift reading 0 from the day it was typed; Studio Rossi nudged from a July invoice with nothing in September; recording the payment taking the invoice off the owed list and leaving the nudge alone; a September invoice for the same Client clearing the nudge and appearing as owed at 0 days; the SPA shell at 200 with the bundle carrying the notice.

## Spec deviation

- **"Nothing recorded this month" is read as nothing *freelance* recorded this month.** The spec qualifies the history side explicitly — "at least one `freelance` Income in the previous three calendar months and nothing recorded this month" — and leaves the current month's side open. The ticket's own wording settles it the other way: "the second list is limited to freelance-category income". Taken as one rule covering both halves, a gift is not an invoice in either direction — it must not start a nudge, and it must not clear one, because a relative sending money in March is no evidence that a Client was billed. Pinned by `TestAGiftThisMonthDoesNotClearTheNudge`, so the reading is a test rather than an assumption.

- **"Recorded this month" means billed, not typed.** `billedOn` answers it, so an Income typed today carrying last month's invoice date counts against last month. That is the same definition the first list is ordered by — one notion of when an Income was billed rather than two that could drift — and it matches what the household did: the invoice went out when it went out.

The spec's "oldest first by invoice-sent date (falling back to creation)" is that same `billedOn`.

## Beyond the letter of the ticket

- **Hidden Clients are not chased.** Neither the ticket nor the spec mentions them. Hiding is how a Client is retired (`clients.go`), and a notice that keeps asking for an invoice for someone the household stopped billing is a notice they learn to ignore. Tested, and one line to drop if it turns out to hide a Client who is merely quiet.

## Not built, and not asked for by this ticket

- **A screen of its own.** The ticket asks for a notice, and the month screen is the screen the app opens on. A dedicated view earns its place when the lists are long enough to scroll past everything else.
- **The month of the last invoice on a nudged Client.** One `MAX()` away if the household finds the bare name too thin, and speculation until they say so.
- **Chasing actions** — no "mark as chased", no reminder count. The Income and its dates are the record; a second state for how often it has been asked about is a feature nobody has asked for.
