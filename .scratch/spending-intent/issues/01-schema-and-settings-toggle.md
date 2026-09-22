# 01: Schema and settings toggle

**What to build:** The foundation the rest of Spending intent sits on: the database columns that will hold it, and the single on/off switch that gates its UI everywhere else. Nothing user-visible changes yet except the switch itself existing and persisting.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `schemaVersion` bumps from 21 to 22, one migration step
- [ ] `category`, `expense`, and `recurring_expense` each gain a nullable `spending_intent` column with a CHECK constraint restricting it to `'necessity'`, `'desire'`, `'desire_wise'`, `'desire_bullshit'`, or `NULL`
- [ ] A new `setting` key holds the on/off switch, off (`false`) on a fresh database
- [ ] `GET`/`PUT /api/settings` round-trip a new `spending_intent_enabled` boolean on the `Lists` payload, alongside the existing settings, without disturbing any of them
- [ ] Settings page shows a single toggle switch, "Spending intent," wired to `spending_intent_enabled`
- [ ] Test: `spending_intent_enabled` defaults to `false` on a fresh database, round-trips through `PUT`/`GET /api/settings`, and setting it leaves every other setting on the same payload unchanged
