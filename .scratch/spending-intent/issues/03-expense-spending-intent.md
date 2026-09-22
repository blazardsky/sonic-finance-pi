# 03: Expense Spending intent

**What to build:** The core of the feature — a household member can classify an individual Expense's Spending intent, pre-filled from its Category's default (ticket 02) but freely overridable, and can change or clear it later like any other field.

**Blocked by:** 01, 02

**Status:** ready-for-agent

- [ ] `POST`/`PATCH /api/expenses` accept and return `spending_intent` (nullable enum), independently settable regardless of the Category's own default
- [ ] An unrecognized value is rejected with 400
- [ ] All four non-null values (`necessity`, `desire`, `desire_wise`, `desire_bullshit`) are valid to set directly — `desire_wise`/`desire_bullshit` never require having passed through plain `desire` first
- [ ] Expense form gets the same two-pair badge control as ticket 02 (Necessity/Desire, conditional Wise/Bullshit), rendered only when `spending_intent_enabled` is on
- [ ] On create, picking a Category pre-fills the badges from that Category's `spending_intent` default; picking a different Category re-seeds the pre-fill, but once the badges have been touched by hand this session, a further Category change does not silently overwrite that manual choice
- [ ] The whole section is optional — an Expense saves fine with nothing picked, or with just Necessity/Desire and no Wise/Bullshit refinement
- [ ] Editing an existing Expense shows its current Spending intent and allows changing or clearing it
- [ ] Test: `spending_intent` round-trips through Expense create/update; invalid value rejected with 400; each of the four values settable directly without a prerequisite state; unset stays `null`
