# 05: Client contracts

**What to build:** Recording a Contract (a total and a date range) against a Client, linking Incomes to it, and the computed should-have/got/invoice-this-month figures.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] A Contract can be created for a Client: a total amount and a start/end month.
- [ ] A Client can have more than one Contract over time; two Contracts for the same Client are refused if their date ranges overlap.
- [ ] An Income can optionally link to one of that Client's Contracts; unlinked Income from that Client counts as Extra and doesn't affect any Contract's figures (but still counts toward the Client's total-earned, from ticket 04).
- [ ] Each Contract shows: how much should have arrived by now (straight-line share of time elapsed, capped at the total once the end month has passed), how much has actually been received (paid Incomes linked to it), and how much to invoice this month, recomputed as the remaining total (net of everything already received or already invoiced under it) divided by the months remaining.
- [ ] A Contract past its end month shows the remaining shortfall as overdue rather than dividing by zero or a negative month count.
- [ ] Backend tests cover the should-have/received/invoice-target math at several points in a contract's span, the overlap refusal, the overdue case, and Extra income being excluded from contract figures but included in total-earned.
