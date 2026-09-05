Status: ready-for-agent

# Full-width pages, shadcn redesign, and a right-side form sidebar for Registrazioni

## Problem Statement

Every page under Mese, Anno, Spese, Entrate, Ricorrenti, Risparmi, Categorie, Clienti, Investimenti, and Impostazioni pins its content to a narrow, literal `max-w-md` (or similar) className on its own root div — nine of the eleven page files repeat the exact same string. On a laptop or desktop this wastes most of the main content area, which is especially bad for the pages whose main content is a table. Several pages also hand-roll UI (a plain `<select>`, a bare `<ul>`, raw `<label>` elements, custom div-based bars for a chart) that shadcn/ui already has a real component for, and the "Registrazioni" data-entry pages (plus a few shape-alike "Gestione" pages) mix a form and a list in one unstructured column with no way to get the form out of the way.

## Solution

1. Replace every page's own hardcoded `max-w-*` with one shared token so the whole app's width ceiling is a single source of truth, and let content — especially tables — actually fill the space up to that ceiling.
2. Redesign the pages using currently-missing shadcn/ui components (`DropdownMenu`, `HoverCard`, `Accordion`, `Checkbox`, `Combobox`, `Field`, `Input Group`, `Label`, `Radio Group`, `Select`, `Toggle`, a Popover+Calendar Date Picker) instead of native elements or ad hoc markup, with a dynamic (not fixed-column) grid/flex layout so cards and sections reflow to fill exactly what's available.
3. Give every page whose content is "a form to add something + a list of what's already there" (Spese, Entrate, Ricorrenti, Risparmi, Categorie, Clienti, Investimenti) a shared layout: the list fills the remaining space on the left, the form lives in a fixed right sidebar built on shadcn's `Sidebar` component, hideable on desktop and an auto-collapsing full-width drawer on mobile with an always-reachable bottom trigger.
4. Small, page-specific pieces: a `DropdownMenu` "actions" affordance on every list row that has 2+ actions (not just Categorie); a `HoverCard` on the Dashboard's "Clienti questo mese" entries; Anno's hand-rolled div bar chart replaced with a real shadcn `Chart`.

## Out of Scope

- `Login.tsx` and the `App.tsx` clock-warning banner — neither is one of the named pages.
- `YearlyReport.tsx` ("Report annuale", the `yearReport` screen reached from within Anno) — explicitly left untouched this round, at the user's call, despite being a real width offender (`max-w-2xl` around the app's biggest table). Revisit in a future pass.
- `Dashboard.tsx` ("Panoramica") beyond one targeted enhancement — it already has a proper redesigned layout (see project memory `project_web_dashboard_redesign`); it does **not** get the width-token change or a general redesign pass, only the `HoverCard` addition described in ticket 13.
- The shadcn `Questionnaire` component — real and current, but nothing in these forms is naturally multi-step; not adopted here.
- `@tanstack/react-table` / the shadcn Data Table pattern — no page has enough rows today to justify it (no sort/filter/pagination exists anywhere, backend returns full unpaginated collections). Tables stay the plain shadcn `Table`, optionally with a simple client-side text-search input. Revisit if a specific page's row count actually becomes a problem.
- Any backend/API change. Every capability referenced here (Categorie's Rename/Hide/Delete, Holdings' rename/change-type-only, etc.) already exists; this is frontend-only.

## Implementation Decisions

### Width token

Add one CSS custom property to `web/src/index.css`, alongside the existing design tokens (`--sidebar`, `--shell`, `--chart-1..5`, etc.):

```css
--content-max-width: 1800px;
```

Every in-scope page's root container becomes `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6` (or the page's existing gap value) in place of its own `max-w-md`/`max-w-6xl` literal. Tailwind v4's `max-w-(--content-max-width)` shorthand expands to `max-w-[var(--content-max-width)]` — confirmed against the current Tailwind docs, no `tailwind.config` changes needed (this repo has no JS config file; it's the CSS-first v4 setup already in use).

### Layout philosophy

"Dynamic, not forced": prefer `flex flex-wrap` or `grid` with `auto-fit`/`minmax` over a fixed N-column grid, so stat cards and sections reflow based on available width rather than a hardcoded column count. A page whose content is just one table skips the grid/wrap wrapper entirely — the table alone fills the row. Forms live inside a bounded-width `Card` (for readable line length) that itself participates in the same wrap/reflow layout, rather than stretching edge-to-edge.

### New shadcn components to install

Via `npx shadcn add <name>` from `web/` (style `radix-nova`, per `web/components.json`): `dropdown-menu`, `hover-card`, `accordion`, `checkbox`, `combobox`, `field`, `input-group`, `label`, `radio-group`, `select`, `toggle`. Date Picker is not a standalone component — build it as the standard shadcn recipe (`Popover` + the already-installed `Calendar`); `Popover` itself needs adding too. `chart` is already installed (`web/src/components/ui/chart.tsx`) and does not need re-adding.

`native-select.tsx` usage is replaced by the new `Select` component throughout the forms this spec touches; keep `native-select.tsx` itself (don't delete it) and fall back to it per-field only where it's genuinely simpler/faster than `Select` (implementer's call per field, not a blanket rule).

### The right-side form sidebar (shared component)

One new component (e.g. `web/src/components/form-sidebar.tsx`), built on the existing `Sidebar` primitive (`web/src/components/ui/sidebar.tsx`) rather than a bespoke panel — it already provides exactly the wanted behavior:

- `side="right"`, `collapsible="offcanvas"` (fully hides, no icon-rail mode needed).
- Its own `SidebarProvider` scoped to the page (not the app-wide one `App.tsx` already uses for the main nav), with a **distinct** storage key so toggling the form sidebar never touches the main nav's `sidebar_state` cookie. `sidebar.tsx` currently hardcodes `SIDEBAR_COOKIE_NAME` — either parameterize it (a `storageKey`/`cookieName` prop, defaulting to the current literal for backward compatibility with the main nav) or give the form-sidebar wrapper its own small persistence (e.g. `localStorage`, since per-page open/closed state doesn't need to survive server-rendering) if threading a prop through is more invasive than it's worth. Implementer's call; the requirement is just "no collision with the main nav's cookie."
- **Variant**: the default `variant="sidebar"` (flush panel, shares `--background`/`--border`), not `floating` or `inset` — the user explicitly wants it to read as part of the main content area, not a bolted-on floating panel.
- On mobile, `Sidebar` already auto-renders as a full-width `Sheet`; per the user, the trigger for it must be an always-visible, easily-reachable button anchored near the bottom of the screen (a FAB), not just relying on a header icon.
- On desktop, a `SidebarTrigger`-style button toggles it open/closed.
- After a successful form submission, the sidebar/drawer **stays open** and the form clears itself, ready for the next entry — these are quick, repeated data-entry flows and closing every time adds friction.

Apply this component to Spese, Entrate, Ricorrenti, Risparmi, Categorie, Clienti, and Investimenti (tickets 03–09) — all seven currently share the same "form + list" shape, whether or not they're in the nav's literal "Registrazioni" group (which is actually only Spese/Entrate/Ricorrenti/Risparmi — Categorie/Clienti/Investimenti are "Gestione", confirmed in `app-sidebar.tsx:52-62`, but they share the identical shape and get the identical treatment).

### Row actions

Wherever a list row has 2+ actions, consolidate them into a `DropdownMenu` triggered by an "actions" ("...") button, replacing whatever inline buttons exist today. This applies to all seven form+list pages above, not just Categorie. Categorie's menu has exactly the three actions its backend already supports: Rename, Hide/Unhide, Delete (no merge/reorder/color — the API has no such endpoints, confirmed in `cmd/categories.go`). Investimenti's Holdings API only supports rename/change-type (`PATCH /api/holdings/{id}`, no delete) — its row action is a single "Edit" affordance; a `DropdownMenu` isn't needed there unless a second action shows up.

### Charts

Anno (`Year.tsx`) currently renders its 12-month total as a hand-rolled `<div>`-based bar chart. Replace it with a real shadcn `Chart` (the already-installed `chart.tsx` wrapper around Recharts), using the existing `--chart-1..5` CSS tokens, matching how `Dashboard.tsx` and `Month.tsx` already chart their own data.

### Dashboard ("Panoramica") touch-up

Out of scope for width/layout (see Out of Scope), but add a `HoverCard` to each entry in the "Clienti questo mese" section (`Dashboard.tsx`'s `ClientsCard`, `:420-449`) surfacing data that's already fetched but currently dropped from that card: the outstanding amount and days-waiting (`OutstandingIncome.amount_cents`/`days_waiting`, from the same `GET /api/pending-payments` call already powering the card) for the "In attesa" list, and — where available — the client's `total_earned_cents` (already loaded elsewhere on the Clients page, `Clients.tsx:238`) for either list.

## Testing Decisions

No frontend automated test runner exists in this repo (`web/package.json` has no `test` script) and this spec doesn't add one, consistent with prior specs (see `client-contracts-and-reports/spec.md`). Verify each ticket by running the app (`./build.sh` or the frontend dev server per `docs/agents/run.md`/project convention) and exercising the page manually: layout at a few viewport widths (narrow/mobile, laptop, ultrawide), the sidebar's open/closed/mobile-drawer states, form submit-then-stays-open behavior, and the DropdownMenu/HoverCard interactions. Run `tsc`/the project's lint or build step (whatever `web/package.json` defines) after each ticket as the one automatic check that non-trivial markup/logic changes didn't break the build.

## Further Notes

- This spec covers eleven separable pieces of work. Tickets 01 (tokens + component installs) and 02 (the shared form-sidebar component) are the two things everything else depends on; 03–09 (the seven form+list pages) are independent of each other once 02 lands and can be done in any order; 10 (Impostazioni), 11 (Mese), 12 (Anno) depend only on 01; 13 (Dashboard) depends only on 01 (for `HoverCard`).
- `Sidebar`, `Sheet`, `Table`, `Tooltip`, `Badge`, `Input`, `Separator`, and `Textarea` are already installed and don't need re-adding.
- All new/changed UI strings go through the existing `web/src/lib/strings.ts` (`t.`) pattern, Italian-first, matching the household-facing language already seeded.
