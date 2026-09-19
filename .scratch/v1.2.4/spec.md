# v1.2.4 spec

Scope is the `v1.2.4` checklist in `TODO.md`. All work is frontend under
`web/` unless noted; one ticket file per task under `issues/`.

## Verification (run from `web/`)

- `pnpm typecheck`
- `pnpm lint`

## File ownership (parallel agents must not write outside their set)

| Agent | Owns (write) |
| --- | --- |
| expenses (claude) | `pages/Expenses.tsx`, `lib/money.ts`, `components/ViewRow.tsx`, `components/ui/badge.tsx`, `components/ColorDot.tsx`, `lib/strings.ts` |
| incomes (cursor) | `pages/Incomes.tsx`, `types.ts`, `components/form-sidebar.tsx`, `components/ui/sidebar.tsx`, `components/ui/combobox.tsx`, `components/ui/input-group.tsx`, `hooks/use-mobile.ts`, Go backend |

`lib/palette.ts`, `components/ui/*` (other than the two above) are read-only
shared ground: read, do not edit.

Read `CONTEXT.md` and any ADR touching the area first. Do not commit.
