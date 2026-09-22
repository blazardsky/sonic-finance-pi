# 04: Recurring expense Spending intent

**What to build:** A Recurring expense carries its own Spending intent, set once, and every Expense it generates each month carries that same value — the same way a Recurring expense's Category already propagates to what it generates.

**Blocked by:** 01, 02, 03

**Status:** ready-for-agent

- [ ] `POST`/`PATCH /api/recurring` accept and return `spending_intent` (nullable enum), same validation as ticket 03
- [ ] Recurring expense form gets the same badge control as ticket 03, pre-filled from its Category's default the same way
- [ ] Materialising a month copies the Recurring expense's `spending_intent` onto the generated Expense, the same way Category is already copied
- [ ] Test: `spending_intent` round-trips through Recurring expense create/update; materialising a month produces a generated Expense carrying the Recurring expense's `spending_intent`, mirroring however the existing Category-copy is already tested for materialise
