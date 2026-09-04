# 03: Record a Holding buy/sell

**What to build:** A dedicated Buy/Sell form, on a new Savings page, that lets the household record money moving into or out of a Holding — reusing the existing Expense/Income endpoints rather than inventing a new one.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] The Savings page has a Buy/Sell form: pick a Holding, an amount, a date, and a Payer.
- [ ] A "buy" posts to the existing Expense endpoint with the Investments category and the chosen `holding_id`; a "sell" posts to the existing Income endpoint the same way.
- [ ] A recorded buy/sell shows up in the ordinary Month, Year, Dashboard, and Recent-entries totals exactly like any other Expense/Income — no exclusion anywhere except where later tickets (Budget/Target, Estimate) explicitly carve it out.
- [ ] Backend tests cover `holding_id` round-tripping through Expense and Income create/read.
