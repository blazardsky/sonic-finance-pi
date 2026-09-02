# 05: Log an Expense

**What to build:** The core act of the whole app: standing at a till, logging what was just spent, in about three taps. Amount, date, Category — and it appears in a list.

Deliberately minimal. The remaining fields arrive in 06.

**Blocked by:** 04

**Status:** ready-for-human

- [x] An Expense is created with an amount, a date, and a Category
- [x] Money is stored as integer cents throughout; floats never touch it
- [x] The date defaults to today in the browser and stays editable
- [x] Saved Expenses appear in a list, most recent first
- [x] The add form is the most prominent thing on the screen
- [x] Tests drive the HTTP seam: create, then read back and confirm what a later request sees

## Comments

Implemented. `expense.go` holds schema step 2, the list handler and the create
handler; `web/src/Expenses.tsx` is the screen, with the add form first and
biggest and the list under it as confirmation.

Money is integer cents end to end: `web/src/money.ts` builds cents out of the
typed digits with integer arithmetic, the API decodes into `int64`, and the
column is `INTEGER` in a `STRICT` table. A client sending a fractional
`amount_cents` is refused with 400 rather than rounded — the decoder does it,
and a test pins it.

Two deliberate calls worth knowing about:

- The `expense` table is created with the columns ticket 06 fills in (`store`,
  `payer`, `payment_method`, `note`) already present as empty defaults, because
  a migration case can never be edited once a database has run it and those
  fields need nothing that does not already exist. The spec's `recurring_id` is
  the one column left out: it points at a table ticket 12 creates, and adding it
  there keeps it a real foreign key.
- `App.tsx` gained a two-screen toggle, because Expenses opening as the default
  would otherwise leave Categories unreachable. It is not the real nav.

Review fixes applied before commit: an Expense in a hidden Category was listing
with a blank Category name (the picker's filtered list was being used for name
lookup too); a database failure during validation answered 400 instead of 500;
the shared `<select>` chrome became `components/ui/native-select.tsx` on its
second use; amounts and dates now render Italian-style (`1.234,56`,
`03/09/2026`) per spec story 70 — `Intl.NumberFormat("it-IT")` needed
`useGrouping: "always"`, or a four-figure month printed as `1234,56`.
