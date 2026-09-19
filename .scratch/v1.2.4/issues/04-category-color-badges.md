# 04 — Color-coded category badges in expense/income lists

Status: ready-for-agent

## Problem

Category names are plain text in the lists; they should render as a badge
using the category's palette color so grouping is visible at a glance.

## Scope

Expense list in `pages/Expenses.tsx`; income list in `pages/Incomes.tsx`.
`components/ui/badge.tsx` and `components/ColorDot.tsx` may be extended.
Use the existing `paletteVar` from `lib/palette.ts` (read-only).

## Done when

Both lists show a colored badge per row consistent with the palette.
