# 02: Tax reserve in the Headroom forecast

**What to build:** With the self-employed switch on, every forecast month and last month's actual leftover set aside a Tax reserve before the Goal buffer. The rate is the median, across completed tax years (Y ≤ current year − 2, with both freelance Income received and tax paid on record), of the tax summary's tax paid ÷ freelance Income received; with no such year, the fallback percentage. A month's reserve is rate × (Contract share + forecast of Freelance Incomes without a Contract, forecast with the same seasonal window and blend as Income); the start's is rate × the Freelance Incomes received last month. Leftover = Income + Contract share − spending − Recurring due − reserve, then the Goal buffer. While the reserve applies, Taxes-category Expenses leave the spending medians and last month's actual Expenses, and Taxes-category Recurring expenses leave the Recurring due. The report exposes `tax_reserve_cents` and `freelance_forecast_cents` per month, and `start_tax_reserve_cents`, `tax_reserve_percent` and `tax_reserve_source` (`history` / `fallback`) at the top. The page shows "accantonato per tasse: € X" (next month's reserve) under "Margine mensile previsto" only when the reserve applies. Switch off: exactly today's behaviour.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] Fallback 33% applied to a month's Contract share plus its freelance Extra forecast, through the Headroom report with the fixed clock
- [x] The start reserves 33% of last month's Freelance Incomes, Contract Incomes included
- [x] A completed tax year's ratio replaces the fallback and the source reads `history`; last year and this year are ignored; several years give their median
- [x] Salary (and any non-Freelance Income) is never reserved against
- [x] Taxes Expenses are left out of the medians and of the start while the reserve applies
- [x] The Goal buffer applies after the reserve
- [x] Switch off → every figure identical to the current results
- [x] The "accantonato per tasse" line shows only when the reserve applies; type-check passes
