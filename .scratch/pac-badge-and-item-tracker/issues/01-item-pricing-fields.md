# 01: Item pricing fields (quantity, unit, discounted)

**What to build:** An Item can optionally record how much of it was bought (a quantity and a kg/lt/piece unit) and whether it was discounted. The API computes and returns a price per unit from these; it is never stored and never accepted as input, so it can never drift from what was actually paid (ADR-0014).

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `item` gains an optional `quantity` (a positive number) and `unit` (one of `kg`/`lt`/`piece`) — both present or both absent, rejected otherwise
- [ ] `item` gains a `discounted` boolean flag, defaulting to false, purely informational (no effect on any calculation)
- [ ] The API returns a computed `price_per_unit` (amount ÷ quantity) whenever quantity is present; it is read-only, never accepted on write
- [ ] `schemaVersion` is bumped with a new `migrateStep`; existing Items get `quantity`/`unit` NULL and `discounted` false
- [ ] Invalid `unit` values, or `quantity` without `unit` (or vice versa), are refused
- [ ] Covered by tests through the existing `testApp` HTTP seam: create/update an Expense with an Item carrying quantity/unit/discounted, read it back, assert the derived price per unit
