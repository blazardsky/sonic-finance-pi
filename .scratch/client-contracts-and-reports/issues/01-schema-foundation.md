# 01: Schema: Gift, Contract, and Reminder foundation

**What to build:** The one migration step this whole spec needs, kept as its own ticket so four unrelated features don't race to claim the same schema version independently. No user-facing behavior yet.

**Blocked by:** None (can start immediately) — note that it bumps `schemaVersion` 10→11, stacked on the other spec's 9→10 bump, so land it after that spec's schema ticket even though nothing here reads Investments-specific data.

**Status:** ready-for-agent

- [ ] `schemaVersion` bumps from 10 to 11 in one migration step.
- [ ] The existing `Regali` category is promoted in place — `applies_to = 'both'`, new protected `code = 'gift'` — no new category row inserted.
- [ ] `client.default_category_id` (nullable, references `category`) is added.
- [ ] A new `contract` table is added: `client_id` (required), `start_month`, `end_month` (both `YYYY-MM`), `total_cents`.
- [ ] `income.contract_id` (nullable, references `contract`) is added.
- [ ] A new `reminder` table is added: `label`, `enabled`, `set_for_month` (nullable `YYYY-MM`).
- [ ] A freshly migrated database has exactly one category with `code = 'gift'` — never a second, separately-named `Regali` row.
- [ ] Backend tests cover the migration via the existing `testApp` pattern.
