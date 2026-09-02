# 09: Log an Income, paid or unpaid

**What to build:** An invoice sent is recorded the day it goes out, not the day it is paid. Money that has not arrived is visible as owed, and counts toward nothing.

**Blocked by:** 04, 08

**Status:** ready-for-agent

- [ ] An Income has an amount, a Category ("income reason" in the UI — it is not a second field), an optional Client, a Payer, and a note
- [ ] An Income can be created with no payment date: it exists, and it is unpaid — see ADR-0003
- [ ] An invoice-sent date can be recorded, and later a payment date added
- [ ] **Unpaid Incomes count toward no total anywhere.** Every query that sums Income filters on payment-date-not-empty — this is the single most likely bug in the codebase
- [ ] The Payer list is the same one Expenses use, in both directions
- [ ] An Income stores one number, the amount that arrived: no gross/net pair, no tax percentage — see ADR-0004
- [ ] Incomes can be edited and deleted
- [ ] Tests cover an unpaid Income being excluded from totals and then included once paid

## Comments

Handoff from ticket 08 (Clients), which is done. Three things this ticket inherits:

- **`client_id` must be a plain `INTEGER REFERENCES client(id)` — not `ON DELETE SET NULL`, and not `ON DELETE CASCADE`.** The spec calls the column nullable, which is about an Income that names nobody, *not* about what happens when a Client is deleted. A `SET NULL` would silently strip the name off every past Income the moment someone deletes a Client, which breaks ticket 08's "old Incomes still resolve" with no error anywhere; a `CASCADE` would delete the money along with the name. A plain reference refuses instead, and `writeError` already turns that refusal into a 409 — so a Client with Incomes behind it becomes undeletable, and hiding becomes the way to retire one, with no new code in this ticket. This was raised and deliberately settled while reviewing 08: `DELETE /api/clients/{id}` stays a real delete, guarded by the foreign key, rather than becoming a second spelling of hide.
- **Two of ticket 08's boxes finish here.** "Renaming updates what every past Income displays" and "hiding removes a Client from pickers while old Incomes still resolve" are structurally satisfied by referencing `client_id` rather than copying the name, and 08's tests assert only the id-stability half. Assert both through a real Income: rename a Client and read an Income back, hide a Client and confirm the Income still resolves it. Same standard ticket 05 was held to for Categories.
- **The Client picker is the first one that exists, and it filters on `hidden`.** `GET /api/clients` deliberately returns hidden Clients — the management screen is the only place one can be brought back from — so filtering is the picker's job, exactly as ticket 05 does for Categories. The picker is also optional: an Income may name no Client at all.
