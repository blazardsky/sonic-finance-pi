# 05: Client contracts

**What to build:** Recording a Contract (a total and a date range) against a Client, linking Incomes to it, and the computed should-have/got/invoice-this-month figures.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] A Contract can be created for a Client: a total amount and a start/end month.
- [x] A Client can have more than one Contract over time; two Contracts for the same Client are refused if their date ranges overlap.
- [x] An Income can optionally link to one of that Client's Contracts; unlinked Income from that Client counts as Extra and doesn't affect any Contract's figures (but still counts toward the Client's total-earned, from ticket 04).
- [x] Each Contract shows: how much should have arrived by now (straight-line share of time elapsed, capped at the total once the end month has passed), how much has actually been received (paid Incomes linked to it), and how much to invoice this month, recomputed as the remaining total (net of everything already received or already invoiced under it) divided by the months remaining.
- [x] A Contract past its end month shows the remaining shortfall as overdue rather than dividing by zero or a negative month count.
- [x] Backend tests cover the should-have/received/invoice-target math at several points in a contract's span, the overlap refusal, the overdue case, and Extra income being excluded from contract figures but included in total-earned.

## Comments

Implemented in `cmd/contract.go` (GET/POST `/api/clients/{id}/contracts`), wired `income.contract_id` through `cmd/income.go` with client-ownership validation, and added the Contracts view/create form to `web/src/pages/Clients.tsx` plus a scoped Contract picker to `web/src/pages/Incomes.tsx`; `cmd/contract_test.go` pins the should-have/invoice-target math at early/mid/exactly-at-end/overdue points, the overlap boundary case, and Extra-income exclusion — all passing alongside a clean two-axis code-review (standards + spec) with only cosmetic nits, since fixed.
