# 07: Items on an Expense

**What to build:** A book bought during the grocery shop stops counting as Food. Part of an Expense can be broken out under a different Category, without ever having to itemise the whole receipt.

**Blocked by:** 06

**Status:** ready-for-human

- [x] An Item has a name, an amount, and its own Category
- [x] Items are optional — a €7 coffee stays a single record with no Items
- [x] Items are partial: they never have to account for the whole Expense
- [x] The Expense's own amount stays authoritative and is never derived from its Items — see ADR-0002
- [x] Items summing to more than the Expense total are rejected on save with a clear message
- [x] An Item sharing the Expense's Category is permitted; no validation against it
- [x] Items are saved with the Expense in one request; they have no endpoints of their own

## Comments

Implemented. Schema step 4 (`user_version` 5) adds the `item` table; `expense.go`
grew the nested payload, the transaction that saves both, and the arithmetic;
`web/src/Expenses.tsx` grew the rows that break a part out and the lines that
show it on the list.

Decisions worth knowing about:

- **An Item carries no id across the API.** Nothing addresses one — they have
  no endpoints, and they are saved as a set with their Expense — so an id would
  be a handle to nothing. The column exists; the JSON does not. It follows that
  a save replaces the whole breakdown rather than matching rows up, which is
  also why an edit is impossible to get subtly wrong.
- **`PATCH` empties the breakdown before decoding and puts it back if the body
  never mentioned it.** Ticket 06's rule is decode-onto-the-stored-row with no
  pointer fields, and Items are the one field that rule cannot carry: with no
  id to match on, a two-Item body over a three-Item row would leave the third
  showing through, and a shorter list's old names would bleed into the new
  ones. So `"items": [...]` replaces, `"items": []` clears, and an omitted key
  keeps — the same three semantics the other fields have, reached differently.
- **`ON DELETE CASCADE`, not application code.** An Item has no life outside
  its Expense, so the schema says so; without it, deleting an itemised Expense
  would fail on its own foreign key. `foreign_keys(ON)` is already in the DSN,
  and a test deletes an itemised Expense to prove the cascade is live rather
  than merely declared.
- **The sum is checked in the handler, not as a `CHECK`.** A row constraint
  cannot see its siblings' amounts. It is compared inside the loop, so the
  running total never climbs more than one Item above the Expense and an int64
  has no chance to overflow.
- **Exactly covering the Expense is allowed; exceeding it is not.** ADR-0002
  rejects a *negative* remainder, and a fully itemised receipt leaves zero.
  Both boundaries have a test.
- **An Item's Category is held to the same rule as its Expense's**: it must be
  one an Expense can go in, so "Freelance" cannot hold part of a shop. This is
  more than the ticket asked for, and is the rule CONTEXT.md already states.
  Sharing the Expense's own Category stays deliberately unchecked.
- **Refusals now carry their sentence.** `writeInvalid` sits beside
  `writeError`: the messages this codebase writes name the broken rule and
  nothing else, so there is nothing in them to withhold, and "rejected with a
  clear message" is not satisfied by "Bad Request". **This also changed ticket
  06's refusal bodies** — a bad amount, date or Category now says which. The
  screens do not read it: they refuse the same cases themselves in Italian
  before sending, and the form shows the remainder going negative as it is
  typed rather than waiting for a submit.

Deferred, deliberately: the spec's Testing Decisions name three Item cases, and
**"the remainder falls to the Expense's Category"** is the one with no test —
there is no endpoint that attributes anything to a Category yet. It belongs to
ticket 11, with the breakdown it reports on.

Review fixes applied before commit: an Item's Category select coerced its blank
option with `Number("")`, which is `0` — it slipped past the unchosen-Category
check and travelled as a Category no row has, and only the Expense's own select
was saved from it by being `required`; `itemsByExpense` took an expense id where
`0` meant "all of them", a sentinel replaced by loading the table whole, which
also collapsed the grouping lookup at both call sites to one line; `writeInvalid`
took a status it was only ever passed one of, and claimed the frontend read its
messages, which it does not; the migration comment said step 5 where this repo
numbers a step by its `case`; and a comment in `strings.ts` used "spesa", which
CONTEXT.md lists under Expense's own _Avoid_.
