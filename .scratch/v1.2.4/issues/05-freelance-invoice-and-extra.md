# 05 — Freelance invoice number and contract "importo extra"

Status: ready-for-agent

## Problem

Freelance income entries need two optional fields: an invoice number (may be
auto-derived, split per person e.g. nicco/sofi) and, when the entry belongs to
a contract, an "importo extra" (expense reimbursement, late fee, indemnity).
The extra counts toward the amount collected but must NOT count toward the
contract total.

## Scope

Frontend `pages/Incomes.tsx` + `types.ts`; backend Go for the new columns and
the collected-vs-contract-total math (schema bump in `migrate.go`, new
`case`). This is the largest ticket.

## Done when

The new fields persist, show in the view dialog, and the contract total
excludes the extra while the collected amount includes it.
