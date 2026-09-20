# 09: Migrate Categories & Subcategories to DataTable

**What to build:** Both tables in `Categories.tsx` (Categories and Subcategories) render through `DataTable`, using `groupBy` (ticket 03) for the existing Spese/Entrate/Entrambi sectioning.

**Blocked by:** 01, 03, 04

**Status:** ready-for-agent

- [ ] Categories table: `groupBy: { key: "applies_to", order: ["expense", "income", "both"] }` (or whatever the real `applies_to` values/order are — confirm against `web/src/types.ts` and the current `bySection()` implementation before assuming these three literal strings); columns Name (`text`), Actions (no filter) — `applies_to` itself is the grouping key, not also a column filter
- [ ] Subcategories table: same `groupBy` shape, same column treatment
- [ ] Existing per-row dropdown actions (edit, replace-and-delete flows, safe-delete picker from the `category-color-and-safe-delete` feature) are preserved in the Actions column
- [ ] Color swatches (from the `category-color-and-safe-delete` feature, ADR-0016) still render next to each Name exactly as today
- [ ] Manual check: sections render in the fixed Spese/Entrate/Entrambi order regardless of filter/sort state; filtering by Name narrows rows within each section without merging or reordering sections; an empty section (all its rows filtered out) matches whatever `Categories.tsx` did before this migration for that case; safe-delete and color-picker flows still work unchanged; mobile filter sheet parity
