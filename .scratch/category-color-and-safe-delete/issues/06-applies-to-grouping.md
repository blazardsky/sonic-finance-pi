# 06: Applies-to grouping with color-sort

**What to build:** The Categories and Subcategories management tables split into three sections by `applies_to` (Spese / Entrate / Entrambi) instead of one flat list, and within each section, rows sharing a color sit next to each other — since color's whole purpose (ADR-0016) is grouping, related Categories should actually appear grouped in the list, not just in charts.

**Blocked by:** 04 (sorts by the `color` column; does not need 05)

**Status:** ready-for-agent

- [ ] Categories table renders as three sections (Spese / Entrate / Entrambi), each containing only Categories with that `applies_to`
- [ ] Subcategories table gets the identical treatment
- [ ] Within each section, rows are ordered by color group first (uncolored/`blue-gray` last), then name (`COLLATE NOCASE`) — replacing the current plain name-only ordering
- [ ] No row-action behavior (rename, hide, delete) changes — this is a display/ordering change only
