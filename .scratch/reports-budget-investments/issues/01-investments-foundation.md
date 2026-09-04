# 01: Investments foundation

**What to build:** The schema and API foundation for tracking investments: a Holding entity, the columns that let an Expense/Income/Recurring expense reference one, and the existing dormant "Investimenti" category promoted to a real protected base category. Delivers a working Holding list a household can maintain (create, rename, change type) even though nothing yet lets you record a buy or sell against one — that's ticket 03.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] `schemaVersion` bumps from 9 to 10 in one migration step: new `holding` table (id, name, type — one of etf/crypto/stock/bond/other), a nullable `holding_id` added to `expense`, `income`, and `recurring_expense`.
- [x] The same migration step updates the existing `Investimenti` category in place — `applies_to` becomes `both`, and it gets a new protected `code` (`investments`) — rather than inserting a new category row.
- [x] The Investments base category is protected exactly like Freelance and Taxes: renaming or deleting it is refused; hiding it from pickers is allowed.
- [x] `GET /api/holdings` lists every Holding; `POST /api/holdings` creates one with a name and type; `PATCH /api/holdings/{id}` renames it or changes its type.
- [x] A screen exists (e.g. on Settings) to view, add, and rename Holdings.
- [x] A freshly migrated database has exactly one category with `code = 'investments'` — never a second, separately-named "Investimenti" row.
- [x] Backend tests cover the migration (category promotion, new table/columns) and Holding CRUD, following the existing `testApp` black-box pattern.

## Comments

Implemented: schema step 9 (holding table + holding_id columns + Investimenti promotion), Holding CRUD (`cmd/holding.go`), and a Holdings management screen (`web/src/pages/Holdings.tsx`) reachable from the sidebar's management group; buy/sell wiring and the portfolio breakdown are deliberately left to ticket 03.
