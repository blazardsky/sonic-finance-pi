# 04: DataTable mobile filter sheet

**What to build:** A mobile-appropriate way to reach the same per-column filters ticket 01 wires into desktop headers — a `Sheet` (shadcn) triggered by a filter icon, listing every filterable column's control as a stacked form, reading and writing the exact same `columnFilters` state the desktop inline headers use.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `DataTable` renders a filter-icon trigger button whenever at least one column has a `meta.filterVariant` set
- [ ] Tapping it opens a `Sheet` listing every filterable column, each rendered with the same widget type/logic as ticket 01's desktop inline filters (`text`/`select`/`range`/`date-range`) — same `columnFilters` state, no separate mobile-only filter state
- [ ] Changing a filter in the sheet updates the table immediately (or on an explicit "Apply" action inside the sheet — pick whichever matches this app's existing form-dialog conventions, e.g. how other `Sheet`/`Dialog` usages in this repo commit changes) and is reflected back on desktop's inline headers if the viewport is ever resized without a remount
- [ ] The sheet shows an indicator (e.g. a badge count) of how many filters are currently active, and a "clear all" action
- [ ] Manual check: on a narrow viewport, confirm the desktop inline header filters are hidden/inaccessible in favor of the sheet trigger, and that filtering via the sheet produces the same filtered rows filtering via desktop headers would
