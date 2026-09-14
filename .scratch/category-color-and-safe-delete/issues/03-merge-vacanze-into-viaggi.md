# 03: Merge Vacanze into Viaggi

**What to build:** `Vacanze` and `Viaggi` exist as two near-duplicate travel Categories only because there was previously no safe way to consolidate them. Using ticket 01's mechanism on the real database: every Expense currently under `Vacanze` moves to `Viaggi`, and `Vacanze` is deleted. This is a data action, not new code.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `Vacanze`'s Expenses (18 rows as of the original import, may have grown since) all now reference `Viaggi`
- [ ] `Vacanze` no longer exists in the Category list
- [ ] `Viaggi`'s total (sum of its Expenses) equals the pre-merge sum of both Categories combined — verified, not assumed
- [ ] No other Category, Client, Item, Income, or Recurring expense was affected
