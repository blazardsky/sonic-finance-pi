# 03: Table swap

**What to build:** Swap the Expenses, Incomes, Categories, Clients, and Recurring Expenses screens from plain lists to shadcn/ui's Table component.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] shadcn's Table component is added to the project.
- [x] Expenses, Incomes, Categories, Clients, and Recurring Expenses render their record lists as a Table.
- [x] Same rows, same columns, same row actions as today — no sorting or pagination added in this pass.
- [x] Month's category breakdown and Dashboard's Recent-entries lists are left as they are (summary widgets, not in scope).

## Comments

Swapped all five screens' `<ul>`/`<li>` lists for `Table`/`TableRow`/`TableCell` (added via `npx shadcn add table`), one row per record as before; the whole-row click-to-edit on Expenses/Incomes/Recurring became `role="button"`/`tabIndex`/`onKeyDown` on `TableRow` (keyboard-equivalent to the old `<button>`), and Recurring's per-row action buttons stop event propagation so they don't also trigger the row's edit-select. Added two small shared header strings (`t.details`, `t.actions`) to `strings.ts`; no other copy, endpoints, or Go files touched.
