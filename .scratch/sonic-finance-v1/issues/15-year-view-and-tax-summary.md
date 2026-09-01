# 15: Year view and tax summary

**What to build:** The actual figures, as against the forecasts the invoicing software already gives: for a given year, what was really received in freelance income and what was really paid in tax.

**Blocked by:** 10

**Status:** ready-for-agent

- [ ] A year view with year navigation, showing the shape of the whole year
- [ ] A Tax year can be recorded on tax Expenses, defaulting to the year of the expense date and editable — because tax on 2026's income is paid during 2027
- [ ] The Tax year field is surfaced only for Expenses in a tax Category
- [ ] The summary reads "received X, paid Y in tax, net Z%" — see ADR-0008
- [ ] X is received Income in the `freelance` Base category only: employment income arrives already taxed and gifts are not income, and including either makes the percentage meaningless
- [ ] Y is Expenses in the `taxes` Base category, attributed by **Tax year, not payment date**
- [ ] One total; no per-tax-type breakdown
- [ ] Tests cover tax paid in one year attributing to the previous year's summary
