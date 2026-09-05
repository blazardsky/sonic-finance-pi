# 01: Width token + missing shadcn components

**What to build:** The shared foundation every other ticket in this effort depends on — the width token and the shadcn components that don't exist in `web/src/components/ui/` yet.

**Blocked by:** none

**Status:** ready-for-agent

- [x] `web/src/index.css` gets a new `--content-max-width: 1800px` token, defined alongside the existing tokens (`--sidebar`, `--shell`, `--chart-1..5`, etc.).
- [x] Install via `npx shadcn add <name>` from `web/` (style `radix-nova`): `dropdown-menu`, `hover-card`, `accordion`, `checkbox`, `combobox`, `field`, `input-group`, `label`, `radio-group`, `select`, `toggle`, `popover`. Do not re-add `chart`, `sidebar`, `sheet`, `table`, `tooltip`, `badge`, `input`, `separator`, or `textarea` — already present.
- [x] Build a Date Picker as the standard shadcn recipe: `Popover` trigger + the already-installed `Calendar` inside the popover content. This isn't a standalone component — confirm the recipe lives somewhere reusable (e.g. `web/src/components/date-picker.tsx`) rather than copy-pasted per form.
- [x] `npx tsc`/the project's build step still passes after the installs (new components alone shouldn't break anything, but confirm).
- [x] Do not touch any page file in this ticket — this is foundation only, consumed by every other ticket.

## Comments

Implemented: `--content-max-width: 1800px` added to `:root` in `web/src/index.css`, next to the other design tokens (used directly as `max-w-(--content-max-width)`, no `@theme inline` mapping needed since it isn't a Tailwind color/utility name). Installed via `npx shadcn add` from `web/`: `dropdown-menu.tsx`, `hover-card.tsx`, `accordion.tsx`, `checkbox.tsx`, `combobox.tsx`, `field.tsx`, `input-group.tsx`, `label.tsx`, `radio-group.tsx`, `select.tsx`, `toggle.tsx`, `popover.tsx` — all new files under `web/src/components/ui/`. Installing `field`/`input-group`/`combobox` also pulled in registry updates to four already-present shared files (`button.tsx`, `input.tsx`, `textarea.tsx`, `separator.tsx`): each diff only swapped their `cn` import from `@/lib/utils` to the `cn` package (the codebase already uses both forms interchangeably — 12 files on `cn`, 14 on `@/lib/utils`, pre-existing) plus minor upstream style tweaks (aria-invalid/disabled states); no behavioural change, accepted rather than left half-updated. `@base-ui/react` was added to `package.json`/`pnpm-lock.yaml` as a dependency of `combobox`. New `web/src/components/date-picker.tsx` (`DatePicker` component) built on `Popover` + the existing `Calendar`: takes/returns a plain `YYYY-MM-DD` string (`value`/`onValueChange`) to match the native `type="date"` inputs it's meant to replace in later tickets, displays the chosen date formatted in Italian via `date-fns`/`date-fns/locale/it` (already a dependency), closes itself on selection. Added one new string, `t.chooseDate` ("Scegli una data"), to `web/src/lib/strings.ts` for the picker's empty-state placeholder. No page file was touched. `npm run build` (`tsc -b && vite build`) passes clean with no errors.
