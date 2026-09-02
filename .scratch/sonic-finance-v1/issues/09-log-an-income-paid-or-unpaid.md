# 09: Log an Income, paid or unpaid

**What to build:** An invoice sent is recorded the day it goes out, not the day it is paid. Money that has not arrived is visible as owed, and counts toward nothing.

**Blocked by:** 04, 08

**Status:** ready-for-human

- [x] An Income has an amount, a Category ("income reason" in the UI — it is not a second field), an optional Client, a Payer, and a note
- [x] An Income can be created with no payment date: it exists, and it is unpaid — see ADR-0003
- [x] An invoice-sent date can be recorded, and later a payment date added
- [x] **Unpaid Incomes count toward no total anywhere.** Every query that sums Income filters on payment-date-not-empty — this is the single most likely bug in the codebase
- [x] The Payer list is the same one Expenses use, in both directions
- [x] An Income stores one number, the amount that arrived: no gross/net pair, no tax percentage — see ADR-0004
- [x] Incomes can be edited and deleted
- [ ] Tests cover an unpaid Income being excluded from totals and then included once paid — **owed by ticket 10**, see below

## Comments

Handoff from ticket 08 (Clients), which is done. Three things this ticket inherits:

- **`client_id` must be a plain `INTEGER REFERENCES client(id)` — not `ON DELETE SET NULL`, and not `ON DELETE CASCADE`.** The spec calls the column nullable, which is about an Income that names nobody, *not* about what happens when a Client is deleted. A `SET NULL` would silently strip the name off every past Income the moment someone deletes a Client, which breaks ticket 08's "old Incomes still resolve" with no error anywhere; a `CASCADE` would delete the money along with the name. A plain reference refuses instead, and `writeError` already turns that refusal into a 409 — so a Client with Incomes behind it becomes undeletable, and hiding becomes the way to retire one, with no new code in this ticket. This was raised and deliberately settled while reviewing 08: `DELETE /api/clients/{id}` stays a real delete, guarded by the foreign key, rather than becoming a second spelling of hide.
- **Two of ticket 08's boxes finish here.** "Renaming updates what every past Income displays" and "hiding removes a Client from pickers while old Incomes still resolve" are structurally satisfied by referencing `client_id` rather than copying the name, and 08's tests assert only the id-stability half. Assert both through a real Income: rename a Client and read an Income back, hide a Client and confirm the Income still resolves it. Same standard ticket 05 was held to for Categories.
- **The Client picker is the first one that exists, and it filters on `hidden`.** `GET /api/clients` deliberately returns hidden Clients — the management screen is the only place one can be brought back from — so filtering is the picker's job, exactly as ticket 05 does for Categories. The picker is also optional: an Income may name no Client at all.

---

Implemented. `income.go` holds the table and the four handlers; `web/src/Incomes.tsx` is the screen.

- **The three handoffs from 08 land as specified.** `client_id INTEGER REFERENCES client(id)` — plain, no `SET NULL`, no `CASCADE` — so a Client with Incomes behind it refuses deletion with 409 through the existing `writeError`, with no new code. `TestAClientWithIncomesCannotBeDeleted` holds it. Ticket 08's two half-observable boxes are now asserted end to end through a real Income: `TestRenamingAClientChangesWhatAPastIncomeDisplays` and `TestHidingAClientLeavesItsIncomesResolvable`. The Client picker filters `!c.hidden` itself and stays optional, exactly as ticket 05 does for Categories.
- **Unpaid has exactly one spelling: `payment_date IS NULL`.** Both dates cross the API as `""` and are stored as NULL, converted by `nullDate` on the two write paths — and the columns `CHECK (payment_date <> '')` so that is structural rather than conventional. ADR-0003 calls the missing filter the single most likely bug in this codebase; a stray `''` would be an Income that reads as paid and sums as nothing, so the column refuses it and a query that forgets `nullDate` fails loudly instead of being quietly wrong. Verified directly against the shipped database.
- **`""` at the seam is also what makes the partial edit work.** A `PATCH` decodes onto the stored Income, so an omitted field keeps what it had and an empty one clears it — the same bargain the Expense note makes. That is how a payment date is added weeks later (`{"payment_date": "2026-03-14"}`) and how one typed onto the wrong Income is taken back (`{"payment_date": ""}`). A `null` `client_id` clears the Client the same way.
- **Schema step 6** (`schemaVersion` 6 → 7), a new `case 6` in `migrateStep`, no earlier case touched.
- **Both references are checked on the way in, so a bad id is a 400.** Left to the foreign keys they would arrive as 409s, which is the right answer for a delete blocked by rows still pointing at a row and the wrong one for a write naming something that was never there. The Category check also catches what no FK can see: an expense-only Category is a real row and still not somewhere income can land. `acceptsExpense` generalised into `acceptsCategory(w, db, id, applies, refusal)` in `categories.go`, so an Expense, its Items and an Income all pass the same gate asking for their own direction.
- **Neither date is defaulted.** An Expense defaults to today because story 2 asks for it; nothing asks for either Income date, and an invoice sent today and a gift that arrived last week are equally common. A stray invoice-sent date on a birthday gift is wrong data created silently, which is worse than one tap on a native date picker.
- **The list is newest first by id.** An Income has no single date and an unpaid one has none at all, so insertion order is the honest ordering; ticket 11 can sort it if a home screen needs something else. Unpaginated, like the Expense list.
- **Tests** (`income_test.go`, through the one HTTP seam): the round trip; created with no payment date and unpaid; a payment date added later, the invoice-sent date surviving it, and the payment taken back again; every detail kept; the Client optional and clearable; a create validation table (no/negative amount, no/unknown/expense-only Category, unknown Client, malformed and impossible dates) and an edit table, each with a follow-up read proving nothing landed; partial PATCH leaving omitted fields alone; delete, then delete again → 404; the two Client boxes above; Category-with-Income delete → 409; a hidden Category still accepting an Income; newest-first ordering; trimming; a Payer renamed in settings leaving past Incomes reading as before. Clean under `-race`.
- **Frontend**: `Incomes.tsx` — amount, reason, optional Client, both dates side by side (which is the whole point of the screen), Payer and note behind the same disclosure the Expense form uses, and edit/delete by tapping a row. An unpaid row says *Da incassare* and how long it has waited, because that is the one thing the amount does not tell you. `withSaved` and a new `nameOf` moved to `web/src/pickers.ts` rather than being copied from `Expenses.tsx`, which now imports them too. The Payer label is `incomePayer` — one concept, worded for the direction money moved, since "Pagato da" is wrong for an Income.
- **Verified against the real shipped binary**: unauthenticated 401, an empty list, an unpaid invoice → 201, a paid gift with no Client, payment recorded then taken back, the Client cleared and re-attached, nine refusals each answering with the rule it broke, a fractional amount → 400, a 100KB body → 400, unknown and non-numeric ids → 404, a Client rename and hide leaving the Income resolving it, Client-with-Income delete → 409, Category-with-Income delete → 409 and → 204 once the Income is gone, a restart preserving everything, the Italian strings in the built bundle, and — for the shared `acceptsCategory` — Expenses with Items still saving and still refusing an income Category and an over-total breakdown.

## Owed by ticket 10

**The last box is genuinely open, not merely half-observable.** There is no total anywhere to exclude an unpaid Income from — `/api/reports/*` is ticket 10's — so no test in this ticket asserts the arithmetic. What `TestAPaymentDateIsAddedLaterAndCanBeTakenBack` asserts is the state those sums will filter on, in both directions. **Ticket 10 must add the assertion this box names**: an unpaid Income absent from the month's income total, the same Income present once it has a payment date. Everything on this side is in place for it — one spelling of unpaid, enforced by the column — but the sum that has to filter on it does not exist yet, and nothing else in the repo records that the assertion is owed.

## Not built, and not asked for by this ticket

- **`GET /api/incomes/{id}`** is not routed, matching Expenses since ticket 05: the list carries every field, so nothing needs it, and `findIncome` is already there for the two-line addition if a screen ever wants one. Deliberate for both entities rather than an oversight in one.
- **Days waiting as a number** (story 39) and the whole Pending payments view are ticket 14's. The screen shows the invoice-sent date on an unpaid row, which is recognition, not the chasing list.
