# 06: Expense details

**What to build:** An Expense gains the fields that make it recognisable months later — where it happened, whose money it was, how it was paid, and any note. Plus editing and deleting, so a mistake is a correction rather than a retype.

**Blocked by:** 05

**Status:** ready-for-agent

- [ ] Store is free text on the Expense, and the field suggests Stores used before, so consistency comes without a list to maintain
- [ ] Payer is chosen from a short seeded list including "Both" and "Someone else"
- [ ] Payment method is chosen from a short seeded list (cash, credit card, debit card)
- [ ] Payer and Payment method are stored **as text on the Expense**, not as references: renaming a label later must not rewrite what old entries say
- [ ] A free-text note
- [ ] Any Expense can be edited after saving, and deleted
- [ ] Tests cover that renaming a Payer in settings leaves existing Expenses reading as they did
