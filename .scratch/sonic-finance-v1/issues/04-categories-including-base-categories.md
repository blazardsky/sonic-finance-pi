# 04: Categories, including base categories

**What to build:** The household can shape its own Category list — add, rename, hide — while the two Categories the code actually depends on are protected from being renamed or deleted out from under the tax summary.

**Blocked by:** 03

**Status:** ready-for-human

- [x] A Category has a name and applies to Expenses, Incomes, or both — the expense picker never offers an income-only Category
- [x] Categories are a flat list; no nesting (deferred deliberately)
- [x] Renaming a Category changes what every past entry displays, because entries reference it rather than copying its name
- [x] Hiding a Category removes it from pickers while old entries still resolve
- [x] `freelance` (income) and `taxes` (expense) are seeded as Base categories: rename and delete return 409, hide is allowed — see ADR-0008
- [x] Other convenience Categories are seeded but fully editable, since nothing in the code resolves them
- [x] A management screen for all of the above

## Comments

Implemented. `categories.go` holds the table, the seed, and the four handlers; `web/src/Categories.tsx` is the management screen.

- **A non-null `code` is the whole of "Base category."** `code` is `TEXT UNIQUE` and null on every editable row, so protection is a property of the row rather than a list of ids in the code or a boolean flag to keep in sync — ADR-0008 rejected the flag for that reason. The two codes are `freelance` and `taxes`; the reports of ticket 15 resolve by them, which is why the household may hide either one and still get a correct summary. The code itself is **not** published by the API: `SELECT … code IS NOT NULL` reduces it to the `base` boolean the UI needs, and the household has no use for the string.
- **Rename and delete return 409, hide returns 200**, per the ticket and story 48. Note that the spec's API block reads "`PATCH` or `DELETE` on a Base category returns 409" without qualification — taken literally that would forbid hiding, which ADR-0008 and story 48 both require, so the per-field rule is what is implemented. Worth amending that line in the spec so a strict reader does not "fix" it back.
- **`applies_to` is refused on a Base category too**, which ADR-0008 does not name explicitly. The reasoning is one step further out than rename: the summary reads Tasse on the Expense side, so re-scoping it to income-only leaves no Expense able to land in it — the report keeps resolving the Category and keeps finding nothing. Documented at `handlePatchCategory`. A base PATCH carrying the row's *current* name also 409s; a refused no-op is cosmetic, and the screen never offers it.
- **The seed is inside migration step 1**, so it runs exactly once: a Category the household deletes stays deleted across restarts, verified against the real binary rather than only asserted. Fifteen rows — the two Base ones plus Alimentari, Casa, Bollette, Trasporti, Salute, Svago, Ristoranti, Abbigliamento, Stipendio, Regali, Rimborsi, Investimenti, Altro — so first use is not an evening of typing. None of the thirteen is protected, and a test fails if one ever becomes so.
- **The list endpoint returns hidden Categories.** Pickers filter on `hidden` themselves, because the management screen is the only place a hidden Category can be brought back from, and a report over an old month has to resolve one. Ordered `COLLATE NOCASE`: SQLite's default collation is binary, which would sort a lowercase name above every capitalised one.
- **Names are trimmed on the way in**, so "Bici " and "Bici" are not two rows that look identical in a picker. No uniqueness constraint on name: not asked for, and it would collide awkwardly with hiding.
- **An unparseable id and a missing row are both 404**, since from outside they are the same thing — that Category is not there. `findCategory` writes that response itself, which is what keeps the three handlers that need a row down to their actual work.
- **Tests** (`categories_test.go`, through the one HTTP seam): the two Base ones are seeded with the right `applies_to` and are the *only* protected rows; every seeded row declares where it applies and both pickers are covered; create, rename under a stable id, hide and unhide, delete, delete twice → 404; rename/re-scope/delete of a Base category → 409 while hide → 200, with a follow-up read proving the refused edits did not land anyway; a validation table (empty name, missing and unknown `applies_to`, rename to nothing, unknown and non-numeric id); trimming. Clean under `-race` over repeated runs.
- **`decodeJSON` in `app.go`** caps request bodies at 64KB. `/api/login` keeps its own tighter 4KB cap: it is the one route reachable without a session.
- **Frontend**: `Categories.tsx` — add form, inline rename on the row itself, hide/unhide, delete behind a native `confirm()`. Rename and delete are `disabled` on a Base category, so the household is not offered a refusal, and a 409 arriving anyway renders the one sentence explaining why. `applies_to` is a native `<select>` rather than a shadcn one, so a phone gives its own picker wheel; its classes are `Input`'s chrome. Strings are in `strings.ts`. `App.tsx` now renders this screen where the placeholder shell was, and `t.connected` went with it.
- **Verified against the real shipped binary**: unauthenticated 401, the fifteen seeded rows with `base` true on exactly two, create with a trimmed name → 201, Base rename → 409, Base hide → 200, Base delete → 409, edit and delete of an ordinary row, delete twice → 404, unknown `applies_to` → 400, a 100KB body → 400, the Italian strings present in the built bundle, and a restart that neither re-seeds a deleted Category nor un-hides a hidden one.
- **Deferred to the tickets that own the other half.** Two of this ticket's boxes are structurally satisfied but only half observable until Expenses exist: "renaming changes what every past entry displays" (entries reference `category_id`, and the test asserts the id survives a rename) and "old entries still resolve" after hiding. Ticket 05 should assert both through an Expense, and must also decide what `DELETE` does to a Category already in use — the FK arrives with the `expense` table, and today's delete is unconditional because there is nothing to point at it yet. No picker filters on `applies_to`/`hidden` yet either; that lands with the first picker, in 05.
