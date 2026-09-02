# 08: Clients

**What to build:** Who money comes from becomes something the app knows, rather than three spellings of the same name. This is what later makes "has this Client paid me" answerable.

**Blocked by:** 03

**Status:** ready-for-human

- [x] A Client has a name and can be created, renamed, and hidden
- [x] Renaming updates what every past Income displays
- [x] Hiding removes a Client from pickers while old Incomes still resolve
- [x] A Client works for any kind of Income, not only freelance — "Mum" for a birthday gift is a valid Client
- [x] A management screen for the list

## Comments

Implemented. `clients.go` holds the table and the four handlers; `web/src/Clients.tsx` is the management screen.

- **Three columns is the whole table** — id, name, `hidden` — per the spec's Schema section. Deliberately no scope column, which is what "works for any kind of Income" amounts to in practice: nothing on the way in asks what the Client is for, so "Mum" and "Studio Rossi" are the same kind of row. `TestAnyNameIsAValidClient` is what fails if a scope ever appears.
- **Nothing is seeded.** Unlike Categories, where thirteen guesses save an evening of typing, who the household is paid by is not something the code can guess — an empty list is the honest starting point, and `TestAFreshDatabaseHasNoClients` holds it there.
- **Nothing is protected either.** No report resolves a Client by identity the way the tax summary resolves the two Base categories, so every Client is the household's to rename. That is the whole reason this file is half the size of `categories.go`.
- **Schema step 5** (`schemaVersion` 5 → 6), a new `case 5` in `migrateStep`, no earlier case touched.
- **The list endpoint returns hidden Clients**, `COLLATE NOCASE`, for the same two reasons as Categories: the management screen is the only place a hidden Client can be brought back from, and SQLite's default binary collation would sort "zia Carla" above "Banca".
- **Names are trimmed on the way in**, which is the point of the whole ticket — "Rossi " and "Rossi" are exactly the two spellings a Client exists to collapse. No uniqueness constraint: two real Clients can share a name, and the household is the one who knows.
- **`POST` reads only `name`.** Reading the whole struct would let a create ask for an already-hidden Client, which is not a state anything wants; `PATCH` decodes onto the stored row, so hiding a Client cannot blank its name (`TestPatchingOneFieldLeavesTheOtherAlone`).
- **`DELETE` exists** because the spec's API table has `PATCH/DELETE /api/clients/{id}`, for the Client typed twice by mistake. A Client with Incomes behind it is retired by hiding instead, and the delete now refuses rather than taking the name off every Income — see the next point.
- **A foreign key refusing is a 409, not a 500** — fixed in `writeError`, which is where every `db.Exec` error in the app already passes through. This closes a bug ticket 04 handed to ticket 05 and 05 did not take: deleting a Category with Expenses in it returned a raw 500 until now, verified against the shipped binary before and after. Doing it in the one shared place rather than per-delete is what makes ticket 09's `income.client_id` foreign key correct on arrival with no further work. `TestDeletingACategoryInUseIsRefused` covers it through the Category side, which is the only side with a foreign key today.
- **Both screens tell the two 409s apart.** A conflict now has two readings — the tax summary resolves this Category, or entries still point at it — so `write` takes the sentence from the caller. Without that, deleting Alimentari would have read "questa categoria è di base", which is not what happened.
- **Tests** (`clients_test.go`, through the one HTTP seam): a fresh database is empty; create puts it in the list visible; rename keeps the id; hide leaves the row resolvable and unhide brings it back; a one-field PATCH leaves the other alone; delete, then delete again → 404; a validation table (no/empty/blank name, rename to nothing, unknown and non-numeric id, delete of an unknown id) with a follow-up read proving none of it landed; trimming on both create and rename; any name is a valid Client; case-insensitive ordering. Clean under `-race`.
- **Frontend**: `Clients.tsx` — add form, inline rename on the row, hide/unhide, delete behind a native `confirm()`, and an empty-state line, since an empty list is where every household starts. No `applies_to` select and no disabled buttons, because a Client has neither a scope nor a protected variant. `App.tsx`'s two-way toggle became a three-link row; strings are in `strings.ts`, with `nascosto`/`nascosta` split for Italian gender agreement.
- **Verified against the real shipped binary**: unauthenticated 401, an empty list, create with a trimmed name → 201, `hidden` in a create body ignored, case-insensitive ordering, blank name → 400 with the rule as its sentence, rename, hide leaving the name intact, delete → 204, delete twice → 404, unknown and non-numeric id → 404, a 100KB body → 400, a restart preserving a hidden renamed Client, the Italian strings in the built bundle, and — for the `writeError` change — Category delete-in-use → 409, Base delete still 409, unused Category delete still 204.
- **Deferred to ticket 09, which owns the other half.** Two boxes are structurally satisfied but only half observable until Incomes exist, exactly as ticket 04's were until 05: "renaming updates what every past Income displays" and "old Incomes still resolve" after hiding both rest on an Income referencing `client_id` rather than copying the name, and today's tests assert the id-stability half of that. **Ticket 09 should assert both through a real Income, filter the Client picker on `hidden`, and add `client_id INTEGER REFERENCES client(id)`** — the 409 guard is already in place waiting for it. There is no Client picker anywhere yet, because there is nothing yet to pick one for.
- **Not built, and not asked for by this ticket**: the "not yet invoiced" list of story 40 is ticket 14's, and reads Clients through Incomes rather than needing anything more on the Client itself.
