# 02: Category default Spending intent

**What to build:** A household member can set a default Spending intent on a Category (Necessity, Desire, Desire+Wise, Desire+Bullshit, or none), so Categories that are always spent the same way don't need re-classifying on every Expense. This ticket only covers the Category side — nothing reads this default yet.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `POST`/`PATCH /api/categories` accept and return `spending_intent` (nullable enum, same four values as ticket 01's CHECK)
- [ ] An unrecognized `spending_intent` value is rejected with 400, not a 500 — same convention as `holding.validate()`
- [ ] A Category with no `spending_intent` set reads back `null`
- [ ] Categories management page gets a two-pair badge control on the Category edit form: Necessity/Desire badges (tapping one dims/disables the other), and — only once Desire is selected — a second Wise/Bullshit pair appears beneath it, same mutual-exclusivity behavior. Switching back to Necessity clears and hides the Wise/Bullshit pair.
- [ ] The whole control is optional — a Category can be saved with nothing picked
- [ ] Control is only rendered when `spending_intent_enabled` (ticket 01) is on
- [ ] Test: `spending_intent` round-trips through Category create/update; an invalid value is rejected with 400; a Category with it unset reads back `null`
