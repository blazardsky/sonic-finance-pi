# 01 — Auto-open the form sidebar on mobile when editing

Status: ready-for-agent

## Problem

On mobile, tapping "modifica" (edit) on a row opens the edit form on desktop
but the mobile sheet stays closed, so nothing visibly happens.

## Scope

`components/form-sidebar.tsx`, `components/ui/sidebar.tsx`,
`hooks/use-mobile.ts`. Make opening a form that is already populated (edit)
also open the mobile sheet, without breaking the create flow or desktop.

## Done when

Editing any record on a narrow viewport opens the sheet automatically.
