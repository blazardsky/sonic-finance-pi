# 12: Recurring expense definitions

**What to build:** The rent stops being something to retype. A Recurring expense is defined once, with the months it covers — but nothing is generated yet; that is 13.

**Blocked by:** 06

**Status:** ready-for-agent

- [ ] A Recurring expense holds the same fields as an Expense, plus a day of the month
- [ ] It has a start month and an optional end month; empty end month means ongoing
- [ ] Creating one defaults the start to the current month, and the start can be set back deliberately
- [ ] "Deactivating" sets the end month to the current one; there is no separate active flag — the window is the record, per ADR-0005
- [ ] Changing an amount is done by ending one and creating another, so history keeps what was true at the time
- [ ] A screen lists Recurring expenses and shows which are still running
