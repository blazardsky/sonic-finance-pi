# 08: Payer required

**What to build:** Payer becomes a required field on new Expenses and Incomes.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Creating an Expense or Income with an empty or whitespace-only Payer is rejected.
- [ ] Updating an Expense or Income to set an empty Payer is rejected the same way.
- [ ] Existing rows with an empty Payer are left untouched — no backfill, no migration.
- [ ] The Expense and Income forms mark Payer as required and can't be submitted without it.
- [ ] Backend tests cover the rejection on create and update.
