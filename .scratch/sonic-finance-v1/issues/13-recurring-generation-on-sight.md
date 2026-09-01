# 13: Recurring generation on sight

**What to build:** The hardest ticket here. Recurring expenses turn into real Expenses by themselves, with no scheduler anywhere — because the Pi may be switched off when any timer would have fired. Anything that reads a month generates that month's missing Expenses first, then reads.

**Blocked by:** 10, 12

**Status:** ready-for-agent

- [ ] Any read covering a month materialises that month's missing Expenses before reading — see ADR-0005
- [ ] Generation is idempotent: opening a month twice does not produce two rents
- [ ] An Expense is generated only for months inside the Recurring expense's start/end window
- [ ] A report over an old month still generates that month's Expenses, so a month nobody opened at the time is not permanently empty
- [ ] Nothing is ever generated beyond the current month
- [ ] The generated date is the day of month, clamped to the month's length — a 31 becomes the 30th in April
- [ ] A generated Expense is indistinguishable in use from a typed one: editable and deletable with no special rules
- [ ] Deleting a generated Expense records a skip for that Recurring expense and month, so it does not come back
- [ ] **Clock guard:** the Pi has no real-time clock. If the clock reads earlier than a build-time constant, the app has booted without network time — it logs loudly, refuses to generate, and exposes the condition so the UI can warn. Generating nothing is recoverable; generating against a wrong clock is not
- [ ] Tests cover every bullet above with a fixed clock, including reading the same month twice and reading a month long past
