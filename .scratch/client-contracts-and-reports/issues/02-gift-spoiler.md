# 02: Gift spoiler

**What to build:** Categorizing an Expense as Gift, and having its amount blur by default wherever it appears in a list or grid, revealed by hover or tap.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] The Gift category is selectable on the Expense form like any other category.
- [x] A Gift Expense's amount is blurred by default in every list/grid it appears in (Recent entries, Month's list, etc.) — date, store, and category stay legible.
- [x] Hovering (desktop) or tapping (mobile) a blurred amount reveals it; it blurs again once you move on or leave the screen — nothing is persisted across navigation.
- [x] The Gift category behaves like any other protected base category: renaming and deleting are refused, hiding is allowed.
- [x] Backend/frontend tests cover the category protection and (where feasible) the reveal behavior.

## Comments

Verified the Expense form and Categories screen already worked generically (applies_to=both, no name-specific logic); added `category.gift` and `recentEntry.is_gift` (both resolved by `code`, not name) so the frontend could identify a Gift row, a shared `<SpoilerAmount>` CSS hover/focus component wired into Dashboard, Month, and the Expenses table, and backend tests for both the exposed `gift` flag and `is_gift` (Expense-only, not Income) plus the previously-missing Regali case in the base-category-protection test; no frontend test runner exists in this repo (pre-existing, unchanged), so the reveal interaction itself is untested per the spec's own testing decision.
