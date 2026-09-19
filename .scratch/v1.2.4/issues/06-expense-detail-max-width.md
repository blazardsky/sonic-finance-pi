# 06 — Expense detail max width so notes wrap

Status: ready-for-agent

## Problem

In the expense detail dialog, notes render on a single line. Set a max width
(less than 100% on mobile, accounting for padding) so long text wraps.

## Scope

`components/ViewRow.tsx` (the `truncate` value cell) and the dialog content
in `pages/Expenses.tsx`. Fix at the shared row so Incomes benefits too.

## Done when

Long note values wrap instead of truncating on all viewports.
