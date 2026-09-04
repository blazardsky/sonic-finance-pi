# 04: Client default Category and total earned

**What to build:** A default Income Category per Client, and a computed all-time "total earned" figure on the Client screen.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] A Client can have a default Income Category set.
- [x] Selecting that Client on the Income form prefills the Category picker with it; the picker remains freely editable.
- [x] The Client screen shows a "total earned" figure: the sum of that Client's received Incomes (`payment_date` set), computed at read time, not stored.
- [x] Backend tests cover the default-category prefill data and the total-earned computation, including a Client with no received Incomes.

## Comments

Implemented via `client.default_category_id` (plain, unvalidated reference — same treatment as `income.holding_id`) and a `total_earned_cents` field computed by a `SUM(...) WHERE payment_date IS NOT NULL` subquery in `clientSelect`; `handlePatchClient` restores the pre-decode value so a PATCH body can't forge it. Frontend: `Clients.tsx` gained a default-Category picker and a read-only total-earned column, and `Incomes.tsx`'s Client picker now copies the chosen Client's default Category into the reason picker on change only (existing entries load unaffected).
