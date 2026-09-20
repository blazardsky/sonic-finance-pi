# 03: DataTable row grouping

**What to build:** Generic section grouping on `DataTable`, for Categories/Subcategories (ticket 09) to build on — an optional `groupBy` prop that partitions filtered/sorted rows into ordered sections with a header row between them, matching `Categories.tsx`'s current `bySection()` shape.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `DataTable` accepts an optional `groupBy?: { key: keyof TData; order: TData[typeof key][] }` (or equivalent shape) — `order` fixes the section sequence (e.g. Spese/Entrate/Entrambi) rather than sorting sections alphabetically or by first appearance
- [ ] Rows are filtered and sorted first (within the whole data set), then partitioned into sections by `groupBy.key`'s value — a filter narrows rows inside each section, it never merges sections together or reorders them
- [ ] A section header `<TableRow>` (matching the current `hover:bg-transparent` style already used) renders between sections, labeled with the section's value
- [ ] An empty section (no rows match the current filters) either renders no header for that section or an explicit "no rows" state — decide by checking how `Categories.tsx` currently handles an empty `applies_to` group today and match it
- [ ] Pagination, if enabled alongside grouping, paginates within the full grouped set in a way that doesn't split a section confusingly across pages — if this proves awkward, disabling pagination when `groupBy` is set is an acceptable simplification (Categories/Subcategories are small, fixed-size lists; confirm this against ticket 09's actual row counts before shipping either way)
- [ ] Manual check: verify grouping renders correctly against Categories' real data shape (can be done together with ticket 09) before that page migration ships
