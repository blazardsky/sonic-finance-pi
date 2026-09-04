# 02: Gift spoiler

**What to build:** Categorizing an Expense as Gift, and having its amount blur by default wherever it appears in a list or grid, revealed by hover or tap.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] The Gift category is selectable on the Expense form like any other category.
- [ ] A Gift Expense's amount is blurred by default in every list/grid it appears in (Recent entries, Month's list, etc.) — date, store, and category stay legible.
- [ ] Hovering (desktop) or tapping (mobile) a blurred amount reveals it; it blurs again once you move on or leave the screen — nothing is persisted across navigation.
- [ ] The Gift category behaves like any other protected base category: renaming and deleting are refused, hiding is allowed.
- [ ] Backend/frontend tests cover the category protection and (where feasible) the reveal behavior.
