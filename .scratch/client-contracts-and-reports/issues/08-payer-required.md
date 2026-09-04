# 08: Payer required

**What to build:** Payer becomes a required field on new Expenses and Incomes.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] Creating an Expense or Income with an empty or whitespace-only Payer is rejected.
- [x] Updating an Expense or Income to set an empty Payer is rejected the same way.
- [x] Existing rows with an empty Payer are left untouched — no backfill, no migration.
- [x] The Expense and Income forms mark Payer as required and can't be submitted without it.
- [x] Backend tests cover the rejection on create and update.

## Comments

Added the check to `expense.validate()`/`income.validate()` right after the existing amount check (trim, then reject empty), matched by shadcn `required` on the Payer `NativeSelect` in both forms. Also defaulted a Payer in the `addExpense`/`addIncome`/`rent` test helpers, since the rule now applies uniformly to every write including edits of Recurring-generated Expenses — otherwise dozens of unrelated tests (and real generated Expenses from a Payer-less template) could no longer be saved at all.
