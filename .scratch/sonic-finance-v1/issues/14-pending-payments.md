# 14: Pending payments

**What to build:** The freelancer stops reading a spreadsheet carefully to work out who owes them. Two lists: money genuinely owed, and invoices that look forgotten.

Note the vocabulary: **Pending payment**, never "receivable" — this app does not speak accounting.

**Blocked by:** 08, 09

**Status:** ready-for-agent

- [ ] A list of every Income with no payment date, oldest first, showing how many days it has been waiting
- [ ] A second list of Clients with at least one freelance Income in the previous three calendar months and nothing recorded this month
- [ ] The second list is limited to freelance-category income, so a one-off gift is never treated as a missing invoice
- [ ] The three-month window is a code constant, not a setting
- [ ] Everything surfaces as an in-app notice — no email, no push, nothing that depends on the Pi being awake
- [ ] Tests cover a Client falling in and out of the second list as months pass on a fixed clock
