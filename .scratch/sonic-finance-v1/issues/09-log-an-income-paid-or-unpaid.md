# 09: Log an Income, paid or unpaid

**What to build:** An invoice sent is recorded the day it goes out, not the day it is paid. Money that has not arrived is visible as owed, and counts toward nothing.

**Blocked by:** 04, 08

**Status:** ready-for-agent

- [ ] An Income has an amount, a Category ("income reason" in the UI — it is not a second field), an optional Client, a Payer, and a note
- [ ] An Income can be created with no payment date: it exists, and it is unpaid — see ADR-0003
- [ ] An invoice-sent date can be recorded, and later a payment date added
- [ ] **Unpaid Incomes count toward no total anywhere.** Every query that sums Income filters on payment-date-not-empty — this is the single most likely bug in the codebase
- [ ] The Payer list is the same one Expenses use, in both directions
- [ ] An Income stores one number, the amount that arrived: no gross/net pair, no tax percentage — see ADR-0004
- [ ] Incomes can be edited and deleted
- [ ] Tests cover an unpaid Income being excluded from totals and then included once paid
