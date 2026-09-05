# 09: Investimenti (Holdings.tsx) redesign

**What to build:** Same layout treatment as ticket 03, applied to Holdings — plus moving it off a plain `<ul>` onto the shared `Table` component, since every other list page in this effort now uses `Table`.

**Blocked by:** 02

**Status:** ready-for-agent

- [x] Root container uses `max-w-(--content-max-width)` instead of the current `max-w-md` literal.
- [x] The add/edit-holding form moves into the shared `form-sidebar` component.
- [x] The current plain `<ul>` (`Holdings.tsx:124-164`) becomes the shared `Table` component, for visual consistency with every other list page.
- [x] Row action is a single **Edit** affordance (a plain button/icon is fine — the Holdings API only supports rename/change-type via `PATCH /api/holdings/{id}`, no delete, so a `DropdownMenu` isn't needed for one action). Do not add a delete/hide action; the backend doesn't support it.
- [x] Type field (`etf`/`crypto`/`stock`/`bond`/`other`) uses `Select` or `RadioGroup` — implementer's call given it's a small fixed enum. (`Select` — see Comments for why.)
- [x] After a successful submit, the sidebar stays open and the form clears.
- [x] `tsc`/build passes; manually verified at narrow, laptop, and wide viewport widths, including the mobile drawer + bottom trigger.

## Comments

Implemented in `web/src/pages/Holdings.tsx`. Reused the shared `FormSidebar`/`Sidebar` unchanged. Root container is `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6`; a `flex flex-wrap gap-6` row holds the `Table` (`flex-1 min-w-0`, replacing the old `<ul>`/`<li>` list one-for-one: Name, Type, a single Actions column) on the left and `<FormSidebar>` on the right.

**The add/edit split changed shape, not just container.** The original page had name and type edited two different ways: a separate "Rinomina" button that put the name into an inline cell input (blur/Enter to commit), and an always-visible per-row Type `NativeSelect` that PATCHed immediately on change — two mechanisms for the one `PATCH /api/holdings/{id}` endpoint, which already takes both fields together. Ticket 07/08's Categories/Clients keep their inline-rename pattern because those pages only ever change a name from the row; this ticket's own wording — "the **add/edit**-holding form", "a single **Edit** affordance" (not "Rename") — pointed at the same unified pattern tickets 03–06 use instead: one `editing` state, the row's Edit button (`RiEditLine`, no `DropdownMenu` — exactly one action) loads both `name` and `type` into the sidebar's form together, the title switches to `t.editHolding` ("Modifica investimento", new string), and submit does `POST` or `PATCH` with both fields depending on `editing`. This reads more consistently with the rest of the app and removes a row-level control (the per-row type select) in favour of everything living in one place. Submitting a fresh add only clears the name (adding several similar Holdings in a row — a handful of ETFs set up at once — is the common case, same convenience Expenses/Incomes give); finishing an edit resets fully, type included.

Since there's no more inline-cell autoFocus input anywhere on this page (Edit only opens the sidebar, same non-racy pattern Expenses/Incomes/Recurring already use), ticket 07/08's `onCloseAutoFocus`/deferred-`setTimeout` fix doesn't apply here — checked for the pattern that provoked it (a `DropdownMenu` item triggering an inline autoFocus input) and confirmed this page doesn't have it, since there's no `DropdownMenu` at all.

Type field uses `Select`, not `RadioGroup`: with five options (ETF/Crypto/Azione/Obbligazione/Altro) — more than Categories' three — a `RadioGroup` would take five stacked rows in the 16rem sidebar for little benefit, where `Select`'s single compact trigger reads better. No `Combobox` — Holding names are typed once at creation and never picked from a list elsewhere on this page.

Verified live: reused the dev password from tickets 03–08, `go run ./cmd` + `pnpm --dir web dev`, scripted headless-Chromium (Playwright). Confirmed at 1280px, 1920px and 390px: full-width layout, a real add (`POST /api/holdings` 201) with the `Select` type choice, the row's Edit button loading both fields into the sidebar (`Modifica investimento` title, populated Name + Type), changing both together and saving (`PATCH /api/holdings/:id` 200, the row showing the new name and type, the form resetting to add-mode with the default type), the sidebar toggle open/closed, and the mobile FAB opening a genuinely full-width drawer. `npm run build` (`tsc -b && vite build`) passes clean throughout.
