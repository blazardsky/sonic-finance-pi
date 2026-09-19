# 07 — Auto-close the action bar after a record is entered (mobile)

Status: ready-for-agent

## Problem

After successfully submitting a record, the mobile action bar/sidebar stays
open. It should close automatically on mobile.

## Scope

Submit handlers in `pages/Expenses.tsx` and `pages/Incomes.tsx`. Follow the
existing pattern in `pages/Clients.tsx` (`useSidebar().setOpenMobile(false)`
after a successful submit).

## Done when

Submitting on a narrow viewport closes the sheet.
