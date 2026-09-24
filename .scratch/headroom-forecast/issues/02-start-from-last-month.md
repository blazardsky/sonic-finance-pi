# 02: Start from last month's actual leftover

**What to build:** The accumulated Headroom starts from the last completed month's actual leftover: Income received in it (Contract Incomes included, investment sales excluded) minus all its Expenses (Recurring-generated and Investments included), with that month's Recurring expenses generated first, then the Goal-buffer rule. The report exposes it as `start_cents`, and every month's running total includes it.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] The running total of the first horizon month is `start_cents` plus that month's Headroom
- [x] Last month's Recurring expenses count even if no screen had read that month
- [x] Last month's Contract Incomes count; investment sales don't
- [x] The Goal-buffer rule applies to the start too
- [x] Placement uses the running total including the start
