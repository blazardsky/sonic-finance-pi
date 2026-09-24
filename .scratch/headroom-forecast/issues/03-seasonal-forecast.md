# 03: Seasonal forecast

**What to build:** Each forecast month M takes its Income and spending from the median of last year's M−1, M and M+1 (only months on or after the first Expense), blended with the median of this year's completed months: last year weighing (12 − n)/12, n being the months of this year already over. With no months of last year's season on record, this year's median alone; with no completed month this year (January), last year's season alone. Contract Incomes and investment sales stay out of the Income medians, Recurring-generated Expenses out of the spending ones (Investments in). Each month in the report carries its forecast Income and spending.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] October's forecast uses last September, October and November and ignores other months of last year
- [x] The blend weight follows the clock ((12 − n)/12)
- [x] No season on record last year → this year's median alone
- [x] January → last year's season alone
- [x] Each month exposes its forecast Income and spending
