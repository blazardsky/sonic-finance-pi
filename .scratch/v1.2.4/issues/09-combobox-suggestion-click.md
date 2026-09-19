# 09 — Combobox: clicking the suggested-text selector does nothing

Status: ready-for-agent

## Problem

In the Combobox, the control that should select the suggested text is not
clickable (hit-testing/pointer-events issue).

## Scope

`components/ui/combobox.tsx` and `components/ui/input-group.tsx`.

## Done when

Clicking the suggested-text selector behaves as expected on desktop and
mobile.
