# 02: DataTable row expansion

**What to build:** Generic expandable sub-rows on `DataTable`, for Clients' Contracts and RecurringExpenses' Details (tickets 07/08) to build on — an optional `renderSubRow` prop that, when set, auto-injects an expand-toggle column and renders the sub-content as an extra table row directly beneath an expanded one.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `DataTable` accepts an optional `renderSubRow?: (row: Row<TData>) => React.ReactNode` prop
- [ ] When `renderSubRow` is set, a `display` column with an expand/collapse toggle (chevron icon, matching the existing Clients/RecurringExpenses toggle visuals) is auto-injected as the first column
- [ ] Expansion state is TanStack's own (`rowExpandingFeature`, `row.getIsExpanded()`/`row.toggleExpanded()`), not a component-local `Set`/array the page manages itself
- [ ] An expanded row renders `renderSubRow(row)`'s output as one additional `<TableRow>` (or however many the caller returns) immediately below the row it belongs to, matching the current visual shape (a `<Fragment>` injecting extra rows) Clients/RecurringExpenses already use
- [ ] Filtering/sorting operate on top-level rows only — a sub-row's content is never itself a filter/sort target, matching today's behavior where Contracts/Details aren't independently searchable
- [ ] Manual check: build a throwaway column set + `renderSubRow` against one of the in-scope pages' real data shape (Clients or RecurringExpenses) to confirm expand/collapse renders correctly before ticket 07/08 wire the real pages onto it — this can be the same work as ticket 07 or 08 if done together, but the expansion mechanism itself must work before either page migration ships
