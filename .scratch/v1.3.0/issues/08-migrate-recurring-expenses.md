# 08: Migrate RecurringExpenses to DataTable

**What to build:** `RecurringExpenses.tsx` renders through `DataTable`, using `renderSubRow` (ticket 02) for its existing expandable Details.

**Blocked by:** 01, 02, 04

**Status:** ready-for-agent

- [ ] Columns match today's exactly — Category (`select`), Status (`select` — ongoing/ended, PAC/expense badges), Note (`text`, keep the existing `hidden md:table-cell` treatment), Amount (`range`), Actions (no filter, `display` column). The current manual "Details" column (the expand trigger) is replaced by ticket 02's auto-injected expand-toggle column, not kept as a second column
- [ ] `renderSubRow` renders the existing Details content exactly as today, Note included: the Note column is `hidden md:table-cell` (mobile-hidden), and the current expand panel already repeats `r.note` for that reason (see `RecurringExpenses.tsx`'s own comment: "Negozio, Pagato da and Metodo di pagamento have no column of their own; Status and Note do" — Note still needs the expand panel as its mobile fallback despite having a desktop column)
- [ ] Existing per-row dropdown actions (Edit, End, Replace/new amount, Delete) are preserved in the Actions column
- [ ] Manual check: expand/collapse still works and shows the right Details; filter by Category and Status (alone and combined); confirm PAC badge and ongoing/ended badge rendering is unchanged; mobile filter sheet parity
