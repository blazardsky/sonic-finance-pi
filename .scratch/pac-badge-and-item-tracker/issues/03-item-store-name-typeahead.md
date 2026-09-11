# 03: Item & store name typeahead (FTS5)

**What to build:** Typing an Item name or a Store gets typo-tolerant suggestions drawn from previously-entered names, so "melanzana"/"melanzane" and "esselunga"/"Esselunga" converge in practice without a hard-enforced canonical table (ADR-0013, ADR-0015). Backed by SQLite FTS5, already compiled into the driver in use — no new dependency, no new process.

**Blocked by:** None (can start immediately)

**Status:** ready-for-human

- [x] An FTS5 virtual table (or tables) indexes historical Item names and Store values, kept in sync as Items/Expenses are written
- [x] A read endpoint returns ranked, typo-tolerant suggestions for Item names given a query string
- [x] A read endpoint returns ranked, typo-tolerant suggestions for Store values given a query string
- [x] `schemaVersion` is bumped with a new `migrateStep` creating the FTS5 table(s)
- [x] Covered by tests through the existing `testApp` HTTP seam: seed some Item/Store names, query with a prefix or a small typo, assert the expected suggestions come back

## Comments

Implemented: two append-only FTS5 vocabularies, `item_name_fts` and
`store_fts` (`tokenize='trigram'`), schema step 13→14 in `cmd/migrate.go`
(`migrateItemStoreFTS` in `cmd/typeahead.go`), backfilled from existing rows.
Synced via a plain `INSERT` beside the existing writes (`insertItems` in
`cmd/expense.go` for item names, `syncStoreFTS` called from
`handleCreateExpense`/`handlePatchExpense` for store) — no triggers, no
external-content table, since names are a historical vocabulary, not a live
index that needs deleting from. Two new endpoints,
`GET /api/items/suggest?q=` and `GET /api/stores/suggest?q=`, ranked via
FTS5's `rank` column over trigram OR-queries built from the input; a query
under 3 runes falls back to a plain prefix `LIKE` (no trigram can be
extracted). Tests in `cmd/typeahead_test.go` through `testApp`. Left at
`ready-for-human` rather than inventing a "done" state — no such label exists
in `docs/agents/triage-labels.md`.
