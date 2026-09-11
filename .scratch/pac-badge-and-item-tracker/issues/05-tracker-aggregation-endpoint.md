# 05: Tracker aggregation endpoint

**What to build:** A read endpoint answering "how has this Item's price changed, and where is it cheaper" — grouping Items by normalized name × Store × Category. The Tracker is a view, never a record of its own (see the Tracker glossary entry in CONTEXT.md); nothing is stored, everything is computed from existing Expense/Item rows.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] A read endpoint groups Items by normalized name (trimmed, lowercased) × normalized Store × Category
- [ ] For each group, returns last price, min price, and max price per Store
- [ ] For each Item, returns a yearly-average price series
- [ ] Nothing new is written to the database by this endpoint — purely computed on read
- [ ] Covered by tests through the existing `testApp` HTTP seam: seed Items across several months and Stores, assert the grouping and aggregate figures
