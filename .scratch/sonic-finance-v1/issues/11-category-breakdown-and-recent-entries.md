# 11: Category breakdown and recent entries

**What to build:** The month view stops being three numbers and starts telling you where the money went — with the Item exceptions landing under their own Categories rather than the Expense's.

**Blocked by:** 07, 10

**Status:** ready-for-human

- [x] The month view breaks Expenses down by Category
- [x] Each Item's amount is attributed to the Item's Category
- [x] Whatever the Items do not cover is attributed to the Expense's own Category — the remainder is never "uncategorised"
- [x] The breakdown's total always equals the month's total spend, whatever the mix of itemised and unitemised Expenses
- [x] The most recent entries are listed on the home screen, so what was just typed is visible
- [x] Tests cover a partially itemised Expense splitting correctly across two Categories

## Notes

- **No schema change.** Both halves read columns that already exist — `item.category_id`, `expense.category_id`, and the `created_at` both entity tables have carried since their own migrations — so `schemaVersion` stays at 7.
- **The breakdown is one query, and it is ADR-0002 as arithmetic.** `readBreakdown` unions the two ways money is attributed: each Item's amount under the Item's Category, and each Expense's amount *minus its Items* under the Expense's own. Because the remainder is a subtraction rather than a redistribution, every Expense contributes exactly its own amount and the breakdown's total is unconditionally `SUM(amount_cents)` — there is no mix of itemised and unitemised Expenses that can make it disagree, which is why the invariant is tested as an identity against the month's own total rather than against a hardcoded number. The remainder is never "uncategorised" because there is nowhere else for it to be: an Expense has a Category whether or not anything was itemised.
- **A remainder of zero is not a line.** An Expense its Items account for entirely spent nothing under its own Category, and a €0,00 row under Casa would read as money that went there. `HAVING SUM(...) > 0` drops it; amounts are all positive, so nothing legitimate sums to zero.
- **Biggest share first**, tie-broken by name, because "where did the money go" is asked in that order. Hidden Categories appear with their names: money spent under a name since hidden did not stop having been spent.
- **`readMonth` stayed the single hook** ticket 10 designed it to be — the breakdown is a third read inside it, so ticket 13 still has one function to materialise Recurring expenses in front of.
- **Recent entries are ordered by when they were typed, not by the date on them.** `created_at`, descending. This is the ticket's own reason for the list — "so what was just typed is visible" — and it is the difference that matters: a receipt found in a coat pocket and backdated three weeks is exactly the entry most worth confirming, and an `occurred_on` ordering would bury it. Tested directly.
- **The list is not month-scoped**, deliberately, for the same reason: the home screen steps through months, and an entry backdated to last month still has to appear the moment it is saved.
- **An unpaid Income carries no date, and the screen shows it as unpaid.** This was the review's real finding and it is ADR-0003's named bug in its newest disguise: the list is not a total, so it rightly includes unpaid Incomes, but a `+ € 800,00` with a plausible date sitting immediately under a *Entrate* total that correctly excludes that €800 is the same lie the ADR exists to prevent. An Income's `date` is its payment date and nothing else — money that has not arrived has no day it arrived on — so an empty date *is* the unpaid state, and the screen gives those entries "Da incassare", no sign, and muted text. No extra flag was needed for it.
- **`invoice_sent_date` is not a fallback for that date**, which was the first draft and was wrong twice over: it dressed an unpaid Income up as a paid one, and "how long has this been sitting" is ticket 14's question, which ticket 10's notes had already fenced off.
- **`direction`, not `kind`.** CONTEXT.md's Category entry lists `kind` and `type` among the words to keep out of the codebase, and the strings file lost "Ultimi movimenti" for the same reason — *movimenti* is what a bank calls these, and this app has Expenses and Incomes. "Ultime registrazioni" borrows the word the app already uses for having recorded one.
- **`category_id` was dropped from a recent entry** after the review pointed out nothing reads it. The name is what the list shows; the id was there for a tap-to-open that no ticket has asked for.
- **Frontend**: `Month.tsx` gains two sections under the three totals. The breakdown re-renders with the month; the recent list is fetched once and not re-fetched when the household steps through the months, because it is not about the month on screen. A failed recent fetch leaves that list empty rather than blanking the totals — the three numbers are the screen, and the confirmation strip is not worth losing them for.
- **Tests** (`reports_test.go`, through the one HTTP seam): the breakdown across two Categories with an Income present to prove money in stays out of it; the ticket's named case, a €62,00 shop with a €14,00 book splitting into €48,00 and €14,00; a fully itemised Expense leaving no line under its own Category; the invariant over the hard mix — unitemised, partly itemised, fully itemised, two Items under one Category, and a neighbouring month — asserted against the month's own total; an empty month as `[]` rather than null; both directions listed newest-typed-first; the backdated entry still at the top; an unpaid Income with no date, gaining one when paid; the cap; and an empty list as `[]`. Clean under `-race`.
- **Verified against the real shipped binary**: unauthenticated 401 on both report routes and 200 on the SPA shell; the ticket's own case — a €62,00 Coop shop with a €14,00 book, alongside an unitemised €33,00, a fully itemised €800,00 rent, and a €50,00 in April — breaking down as Casa €800,00, Alimentari €81,00, Svago €14,00, summing to `expense_cents` exactly with April excluded and the rent producing one line rather than two; the €1.500,00 received Income in `income_cents` and the €800,00 unpaid one absent from it; that unpaid Income present in the recent list with `"date": ""`; taking the book back out (`PATCH {"items":[]}`) moving its €14,00 into Alimentari and dropping Svago, still summing; deleting the rent leaving the breakdown summing to what remained; an empty 2019 month as zeros and `[]`; and a malformed month as 400. The ARMv6 binary carries the route and the breakdown SQL, and the built bundle carries `/api/reports/recent`, `by_category`, and the four new Italian strings.
- **`created_at` is stored to the second and that is fine.** Six entries posted by script inside one second tie, and the tie breaks across two tables' independent ids, so an Income sorted between two Expenses. Nothing cares: `created_at` orders this one list and is read as data nowhere, entries typed in the same second are equally "just typed", and a household types one at a time anyway. The third `ORDER BY` term exists only so two identical requests answer in the same order.

## Spec deviation

`GET /api/reports/recent` is not in the spec's API list, which is otherwise exhaustive. Nothing listed answers "the last few entries across both directions": `/api/expenses` and `/api/incomes` are both unpaginated and newest-first, so a client-side merge was the zero-backend option, but neither exposes `created_at` — so the frontend could not order by when an entry was typed, which is the whole point of the list. It sits under `/api/reports/` because it is a read-only projection over both entities, beside the month report it shares a screen with.

## Not built, and not asked for by this ticket

- **A tap on a recent entry opening it.** The ticket asks for the entries to be listed so they can be confirmed, not edited; both full lists with their edit forms are one tap away in the nav.
- **Pending payments.** An unpaid Income now reads as unpaid on the home screen, which is not the same as ticket 14's two lists, its days-waiting count, or its not-yet-invoiced nudge.
- **A chart of the breakdown.** The spec's story asks to see what the money went on; a sorted list of names and amounts is that, and a pie chart on a Pi Zero W's bundle is a dependency this has not earned.
- **A prominent add button** (story 54) and the **year view** (56) are their own tickets.
