# 06: Details column — fixed width when expanded

**What to build:** In the Expenses, Incomes, and RecurringExpenses tables, the "details" column no longer changes the table's column widths when a row's details are toggled open.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] The "details" column has a fixed width in all three tables, so toggling a row's expanded state does not reflow other columns.
- [ ] The toggle button's content/icon can still change between open and closed state without affecting layout.
- [ ] Fixed and verified in all three pages (the pattern is duplicated per-page, not shared — same fix needed in each).
