# 06: Expense details

**What to build:** An Expense gains the fields that make it recognisable months later — where it happened, whose money it was, how it was paid, and any note. Plus editing and deleting, so a mistake is a correction rather than a retype.

**Blocked by:** 05

**Status:** ready-for-human

- [x] Store is free text on the Expense, and the field suggests Stores used before, so consistency comes without a list to maintain
- [x] Payer is chosen from a short seeded list including "Both" and "Someone else"
- [x] Payment method is chosen from a short seeded list (cash, credit card, debit card)
- [x] Payer and Payment method are stored **as text on the Expense**, not as references: renaming a label later must not rewrite what old entries say
- [x] A free-text note
- [x] Any Expense can be edited after saving, and deleted
- [x] Tests cover that renaming a Payer in settings leaves existing Expenses reading as they did

## Comments

Implemented. The four columns were already on the table from ticket 05, so this
was handlers and screen: `expense.go` grew `PATCH` and `DELETE /api/expenses/{id}`,
`setting.go` grew the two configured lists behind `GET/PUT /api/settings`, and
`web/src/Expenses.tsx` grew the details and the edit flow.

Decisions worth knowing about:

- **The lists are seeded in a migration** (schema step 3, `user_version` 4),
  not defaulted in code, so that they are the household's own from first boot:
  a Payer edited away never comes back. They are JSON arrays in `setting`,
  because a label is free-form once renamed and every separator worth choosing
  is a character somebody could type. Payers seed as `Persona 1`, `Persona 2`,
  `Entrambi`, `Qualcun altro` — ADR-0001's own wording, and placeholders the
  settings screen of ticket 16 is where you replace.
- **`GET/PUT /api/settings` arrives here rather than in 16**, because this
  ticket's own test has to rename a Payer to prove old Expenses keep reading as
  they did. Only the API: the screen that edits the lists is still 16's.
- **Payer and Payment method are not validated against the lists**, on create
  or on edit. They are labels, and an Expense saved under a label since renamed
  away still has to be editable — checking membership would mean correcting its
  amount cost it what it says.
- **A `PATCH` is partial by decoding onto the stored row.** No pointer fields:
  `json.Decode` leaves an absent key alone, which is exactly the semantics
  wanted, and an explicit `"note": ""` still clears the note. `handlePutLists`
  works the same way, so a body carrying one list cannot blank the other.
- **The Store suggestions are a native `<datalist>` filled from the list
  already on the screen.** No endpoint: Store is free text, and the whole point
  is that there is no list to maintain.
- The details sit in a native `<details>` disclosure, closed by default, so the
  three-tap Expense of ticket 05 stays three taps. An edit opens it.

Review fixes applied before commit: the Category `<select>` dropped hidden
Categories, so editing an Expense filed under one would have silently moved it
— both pickers now keep whatever the Expense was saved with, via one
`withSaved` helper, which is the same rule the Payer select needed for a
renamed-away label; the lists moved from an invented `/api/settings/lists` to
the `/api/settings` the spec's API table already fixes; a failed delete said
"not saved"; the note became a `<textarea>`, since it is the field that catches
what the structured ones miss; and two comments cited ADR-0008 for the
Payer-as-label rule that is actually ADR-0001's.
