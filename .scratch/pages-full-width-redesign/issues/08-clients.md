# 08: Clienti (Clients.tsx) redesign

**What to build:** Same treatment as ticket 03, applied to Clients — preserve the expandable-contract-rows behavior, it just moves into the wider layout.

**Blocked by:** 02

**Status:** ready-for-agent

- [x] Root container uses `max-w-(--content-max-width)` instead of the current `max-w-md` literal.
- [x] The add-client form moves into the shared `form-sidebar` component; the table with expandable contract rows (`Clients.tsx:188-395`) fills the remaining width — the expand/collapse behavior is unchanged, just has more room to breathe.
- [x] Row actions consolidate into a `DropdownMenu` "..." button per row. (Rinomina/Nascondi-Mostra/Elimina — Contratti stays its own visible toggle, see Comments.)
- [x] Default-category picker uses the new `Select`; any long free-text fields consider `Combobox` only if they're genuinely a pick-from-list case, not just for the sake of it. (Client name stays free text — no Combobox candidate here.)
- [x] After a successful submit, the sidebar stays open and the form clears.
- [x] `tsc`/build passes; manually verified at narrow, laptop, and wide viewport widths, including the mobile drawer + bottom trigger, and that expanding a contract row still works correctly in the new layout.

## Comments

Implemented in `web/src/pages/Clients.tsx`, following ticket 07's (Categories) shape closely — no whole-entry edit flow, a name changed in place, plus this page's own expandable-contracts wrinkle. Reused the shared `FormSidebar`/`Sidebar` unchanged. Root container is `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6`; a `flex flex-wrap gap-6` row holds the table (`flex-1 min-w-0`) on the left and `<FormSidebar title={t.addClient}>` on the right, holding only the add-client form (always "add" mode, same as Categories). `error` stays in the list column since it can come from any row action (rename/hide/delete/default-category/add-contract), not just the sidebar's own form.

Row actions: **Contratti stays its own visible toggle, not folded into the dropdown** — it opens a disclosure right under the row rather than firing a one-shot action, so it doesn't fit the "invoke and the menu closes" model the other three do, and hiding the page's main interactive feature behind a "..." would hurt discoverability for no benefit. It picked up a small chevron (`RiArrowDownSLine`/`RiArrowUpSLine`, matching the `Accordion` trigger convention elsewhere) and a `secondary` variant while expanded, as a cheap visual cue — the expand/collapse logic itself (`toggleExpanded`, `refreshContracts`) is untouched. Rinomina/Nascondi-Mostra/Elimina became a `DropdownMenu` (`RiMoreLine` trigger), same icons and structure as ticket 07. Default-category picker uses the new `Select` (a `NONE` sentinel for "not set", writing straight through on `onValueChange` exactly as the old `NativeSelect`'s `onChange` did — no local draft state, matches the original's "changes save immediately" behaviour). No `Combobox` anywhere: Client name is free text, not a pick-from-list, and the only picker (default category) is short enough that `Select` already reads fine.

Applied ticket 07's exact fix pre-emptively, since this page has the identical "click a DropdownMenu item to enter inline edit mode" pattern for Rinomina: `onCloseAutoFocus={(e) => e.preventDefault()}` on the `DropdownMenuContent`, plus deferring `setEditing(c.id)` by one tick (`setTimeout(..., 0)`) in the menu item's `onClick`. Verified live via the same `document.activeElement` check ticket 07 used — confirmed `INPUT`, not `BODY`, immediately after clicking Rinomina, so this one didn't need re-discovering the bug to know the fix works.

The nested "add a Contract" mini-form inside an expanded row is untouched — not inside the `FormSidebar` (only the top-level add-client form moves there, per the ticket), not inside a 16rem-constrained panel at all (it lives in the wide list column), so neither the Accordion-validation nor the sidebar-width gotchas apply to it; its fields (native `type="month"` × 2, a plain amount `Input`) stay exactly as they were.

Verified live: reused the dev password from tickets 03–07, `go run ./cmd` + `pnpm --dir web dev`, scripted headless-Chromium (Playwright). Confirmed at 1280px, 1920px and 390px: full-width layout, a real add-client submission (`POST /api/clients` 201), setting a default category via the new `Select` (`PATCH` 200), expanding a row's Contratti disclosure and adding a Contract inside it (`POST /api/clients/:id/contracts` 201, the new Contract rendering with correct expected/received/target figures), collapsing it again, the row dropdown's Rinomina (`PATCH` 200, confirmed focus lands on the input) and Nascondi (`PATCH` 200), a delete correctly refused with the existing "usato da entrate registrate" 409 message once the test Client had a Contract linked to it, the sidebar toggle open/closed, and the mobile FAB opening a genuinely full-width drawer — contract-row expansion also re-checked at 390px, scrolling horizontally same as any other wide table at that width. `npm run build` (`tsc -b && vite build`) passes clean throughout.
