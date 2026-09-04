# 04: Client default Category and total earned

**What to build:** A default Income Category per Client, and a computed all-time "total earned" figure on the Client screen.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] A Client can have a default Income Category set.
- [ ] Selecting that Client on the Income form prefills the Category picker with it; the picker remains freely editable.
- [ ] The Client screen shows a "total earned" figure: the sum of that Client's received Incomes (`payment_date` set), computed at read time, not stored.
- [ ] Backend tests cover the default-category prefill data and the total-earned computation, including a Client with no received Incomes.
