# Yearly tax summary from base categories, attributed by tax year

The app reports, per year, "received X, paid Y in tax, net Z%" from the household's own records — the actual figures, as against the forecasts the user's invoicing software already provides.

**X** is received Income in the `Freelance` base category, and only that. Employment income arrives already taxed and gifts and reimbursements are not income at all; including any of them makes the percentage meaningless. **Y** is Expenses in the `Taxes` base category, attributed by their Tax year rather than their payment date, because tax on one year's income is paid during the next — a cash-basis figure would look like a tax rate while being one.

Both categories are protected from rename and deletion (hiding is allowed) because the report resolves them by identity. A boolean flag on Category was considered and rejected: a report whose correctness depends on a Category existing should protect that Category rather than add a field to work around its absence. Protection is reserved for categories the code genuinely reads — a seeded-but-editable category nothing resolves is not protected.

There is no per-tax-type breakdown (IVA / INPS / IRPEF): with a flat category list they would need either protected categories per Italian tax name or the two-level categories deferred in the design. One total was what was asked for.

This builds on ADR-0004: Income still stores one number, and tax is still an ordinary Expense.
