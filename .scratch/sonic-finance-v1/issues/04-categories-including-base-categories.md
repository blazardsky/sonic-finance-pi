# 04: Categories, including base categories

**What to build:** The household can shape its own Category list — add, rename, hide — while the two Categories the code actually depends on are protected from being renamed or deleted out from under the tax summary.

**Blocked by:** 03

**Status:** ready-for-agent

- [ ] A Category has a name and applies to Expenses, Incomes, or both — the expense picker never offers an income-only Category
- [ ] Categories are a flat list; no nesting (deferred deliberately)
- [ ] Renaming a Category changes what every past entry displays, because entries reference it rather than copying its name
- [ ] Hiding a Category removes it from pickers while old entries still resolve
- [ ] `freelance` (income) and `taxes` (expense) are seeded as Base categories: rename and delete return 409, hide is allowed — see ADR-0008
- [ ] Other convenience Categories are seeded but fully editable, since nothing in the code resolves them
- [ ] A management screen for all of the above
