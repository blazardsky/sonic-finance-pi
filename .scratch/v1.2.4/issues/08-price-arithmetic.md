# 08 — Arithmetic in the item price field

Status: ready-for-agent

## Problem

When entering an expense price, allow expressions like `x*y`, `x-y`, `x+y`
and compute the result automatically, so two identical units or an
addition/discount can be entered without manual math.

## Scope

`lib/money.ts` (a parser/evaluator that returns cents, kept integer-safe) and
the item amount input in `pages/Expenses.tsx`. Show the computed value.

## Done when

`12*3` yields 36, `10-2` yields 8, `5+5` yields 10, and a plain number is
unchanged.
