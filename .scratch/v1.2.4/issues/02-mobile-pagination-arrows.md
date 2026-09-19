# 02 — Page prev/next become arrows on mobile

Status: ready-for-agent

## Problem

The "pagina precedente" / "pagina successiva" buttons show text labels; on
mobile they should collapse to arrow icons (with accessible labels).

## Scope

`pages/Expenses.tsx` and `pages/Incomes.tsx` pagination controls.

## Done when

On a narrow viewport the controls show only arrows with proper `aria-label`s;
on desktop the text labels remain.
