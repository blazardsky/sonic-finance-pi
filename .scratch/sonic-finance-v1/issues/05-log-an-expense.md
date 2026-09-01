# 05: Log an Expense

**What to build:** The core act of the whole app: standing at a till, logging what was just spent, in about three taps. Amount, date, Category — and it appears in a list.

Deliberately minimal. The remaining fields arrive in 06.

**Blocked by:** 04

**Status:** ready-for-agent

- [ ] An Expense is created with an amount, a date, and a Category
- [ ] Money is stored as integer cents throughout; floats never touch it
- [ ] The date defaults to today in the browser and stays editable
- [ ] Saved Expenses appear in a list, most recent first
- [ ] The add form is the most prominent thing on the screen
- [ ] Tests drive the HTTP seam: create, then read back and confirm what a later request sees
