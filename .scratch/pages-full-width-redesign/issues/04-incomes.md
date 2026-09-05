# 04: Entrate (Incomes.tsx) redesign

**What to build:** Same treatment as ticket 03, applied to Incomes.

**Blocked by:** 02

**Status:** ready-for-agent

- [x] Root container uses `max-w-(--content-max-width)` instead of the current `max-w-md` literal.
- [x] The add-income form (`Incomes.tsx:246`) moves into the shared `form-sidebar` component; the table (`:436`) fills the remaining width.
- [x] Table row actions consolidate into a `DropdownMenu` "..." button per row.
- [x] Category/client/payer/payment-method pickers use the new `Select`/`Combobox` instead of `native-select.tsx` (Client picker is a good `Combobox` candidate if the list can get long).
- [x] Date field(s) use the new Date Picker recipe (e.g. payment date, invoice-sent date).
- [x] Relevant fields adopt `Field`/`Label`/`Input Group` where they fit naturally.
- [x] After a successful submit, the sidebar stays open and the form clears.
- [x] `tsc`/build passes; manually verified at narrow, laptop, and wide viewport widths, including the mobile drawer + bottom trigger.

## Comments

Implemented in `web/src/pages/Incomes.tsx`, the same treatment as ticket 03 (Expenses), reusing the shared `FormSidebar`/`Sidebar` as-is — no changes needed there, since ticket 03 already fixed the trigger z-index, `themed`, and `mobileWidth` issues at the shared-component level. Root container is `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6`; layout is a `flex flex-wrap gap-6` row with the table (`flex-1 min-w-0`) on the left and `<FormSidebar>` on the right holding the form, `title={editing === null ? t.addIncome : t.editIncome}`, wired to a `sidebarOpen` state so `selectIncome` (loading a row) reopens a closed panel, same as Expenses. Row actions: a new `Azioni` column with a `DropdownMenu` (`RiMoreLine` trigger, "Modifica entrata"/`RiEditLine` and destructive "Elimina"/`RiDeleteBinLine`), replacing the old whole-row click-to-edit and its `scrollTo` (unneeded now the panel is `fixed`); `remove()` became `removeIncome(income: Income)`, directly callable per row.

Pickers: Motivo (Category) and the Contract picker (`EXTRA_CONTRACT` sentinel, since "Extra"/unlinked is a real value, not an unset placeholder — the same trick as Expenses' payment-method sentinel) use the new `Select`. The Client picker uses the new `Combobox`, per the ticket's suggestion — items are mapped to `{value, label}` objects (`clientOptions`), which the primitive auto-recognises for both display text and equality with no `itemToStringLabel`/`isItemEqualToValue` needed, since the controlled `value` is always looked up fresh from that same render's `clientOptions` array (no memoisation required — new object references each render are fine as long as `value` and `items` come from the same `.map()` call). `showClear` is enabled so an already-picked Client can be cleared back to unset, matching the old native select's blank option. Both date fields (Fattura inviata, Data incasso) use ticket 01's `DatePicker` — stacked vertically rather than side by side as they were natively: two `DatePicker` buttons showing a full date each don't fit side by side in the sidebar's fixed 16rem width (found via live testing — content silently overflowed the panel with no visible scrollbar, since `SidebarContent` is `no-scrollbar`). Amount uses `Field`+`InputGroup` for the € adornment; the rest of the fields use `Field`/`FieldLabel`. The details disclosure (Payer, Note) is a shadcn `Accordion`, mirroring Expenses' `detailsOpen` state (opens on `selectIncome`, closes when an edit finishes/cancels/deletes).

Same bug as ticket 03, checked for and confirmed present: Payer (`incomePayer`, "Ricevuto da") is `required` but lives inside the details `Accordion`, so Radix unmounting it while closed skips the browser's native validation. Added the equivalent explicit check in `submit()` — new string `t.invalidIncomePayer` ("Scegli da chi è arrivato.", worded for the Income direction rather than reusing Expenses' "Scegli chi ha pagato.") — which opens the disclosure and shows the real reason instead of a bare "Entrata non salvata. Riprova." Also added `t.noClientsFound` for the Combobox's empty-search-results state.

Verified live: reused the dev password from ticket 03 (`db/sonic.db` already reset), `go run ./cmd` + `pnpm --dir web dev`, scripted headless-Chromium (Playwright). Confirmed at 1280px, 1920px and 390px: full-width layout, a real add-income submission (Select reason, Combobox-pick a Client, open details, Select payer, submit — `POST /api/incomes` 201), the row dropdown's Modifica (loads the row, Combobox shows the picked Client with its clear button) and Elimina (confirm dialog, DELETE, row count drops) both working end to end, the sidebar toggle open/closed/reopen-on-edit via a real DOM click, the mobile FAB opening a genuinely full-width drawer with both Select/Combobox/DatePicker fields fitting cleanly, and the Payer-validation fix firing correctly (submitted with Payer untouched → `t.invalidIncomePayer` shown, disclosure auto-opened). `npm run build` (`tsc -b && vite build`) passes clean throughout.
