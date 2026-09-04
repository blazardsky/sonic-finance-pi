# 03: Record a Holding buy/sell

**What to build:** A dedicated Buy/Sell form, on a new Savings page, that lets the household record money moving into or out of a Holding — reusing the existing Expense/Income endpoints rather than inventing a new one.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] The Savings page has a Buy/Sell form: pick a Holding, an amount, a date, and a Payer.
- [x] A "buy" posts to the existing Expense endpoint with the Investments category and the chosen `holding_id`; a "sell" posts to the existing Income endpoint the same way.
- [x] A recorded buy/sell shows up in the ordinary Month, Year, Dashboard, and Recent-entries totals exactly like any other Expense/Income — no exclusion anywhere except where later tickets (Budget/Target, Estimate) explicitly carve it out.
- [x] Backend tests cover `holding_id` round-tripping through Expense and Income create/read.

## Comments

Implemented: `holding_id` (nullable, `*int64`) added to the Expense/Income structs and their create/read/patch paths, following the same pointer-nullable shape `income.client_id` already uses (not `tax_year`'s int/0-sentinel shape, since `holding_id` has none of `tax_year`'s category-driven defaulting/clearing — it's a plain pass-through, per spec); new `web/src/pages/Savings.tsx` with a Buy/Sell form (Holding/amount/date/Payer) posting straight to `/api/expenses`/`/api/incomes` with the Investments category pre-resolved client-side as "the Base category that applies to both sides" (`code` stays unpublished); a sell sets `payment_date` immediately so it counts per ADR-0003; wired into `App.tsx`/`app-sidebar.tsx` next to Expenses/Incomes/Recurring. In passing, fixed a latent bug in `Expenses.tsx`'s tax-year-field check (`c.base` alone stopped uniquely identifying Taxes once Investments also became a Base category), since the same "resolve the base category" logic is what the new page relies on. Verified live via a scratch instance (login, create Holding, buy Expense, sell Income, `GET /api/reports/month/...`) that `holding_id` round-trips and the buy amount lands in the ordinary month total with no exclusion.
