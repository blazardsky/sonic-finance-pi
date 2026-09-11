# 02: PAC badge on the recurring list

**What to build:** Each row in the recurring-expense list shows a colored badge distinguishing a PAC (recurring investment) from an ordinary recurring expense — green for the former, orange for the latter. A recurring entry is a PAC only when it is categorized under Investments AND has a Holding linked; either alone is not enough (see the PAC glossary entry in CONTEXT.md).

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A green "Investment" badge renders when the row's category is the Investments category AND its `holding_id` is set
- [ ] An orange "Expense" badge renders otherwise
- [ ] The badge sits beside/inside the row's existing status badge cell
- [ ] No backend change — this reuses the already-existing `holding_id` link and Investments category, both already tested at the API level
