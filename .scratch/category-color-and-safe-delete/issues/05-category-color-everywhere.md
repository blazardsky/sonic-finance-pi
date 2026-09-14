# 05: Category color in every picker and chart

**What to build:** The color a Category or Subcategory carries (ticket 04) shows up everywhere that Category is chosen or charted, not just in the management tables — so the same color means the same thing wherever a household member sees it.

**Blocked by:** 04

**Status:** ready-for-agent

- [ ] Every Category/Subcategory `<SelectItem>` in the Expense, Income, and Recurring-expense forms shows the option's color swatch
- [ ] `categoryColor(id)` (the retired id-hash generator) is deleted, not left dead in the codebase
- [ ] `CategoryTrendChart`, Dashboard's upcoming-list dot, and `YearlyReport`'s pie/bar all read each Category's own stored `color` slot (light/dark variant as appropriate) instead of computing a hue
- [ ] Two Categories sharing a color render identically wherever they appear — this is expected (ADR-0016), not a bug to work around with a secondary tiebreaker
