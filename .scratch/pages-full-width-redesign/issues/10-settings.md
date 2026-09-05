# 10: Impostazioni (Settings.tsx) redesign

**What to build:** Full-width layout and component polish for Settings — no form-sidebar pattern here, since this page has no list.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] Root container uses `max-w-(--content-max-width)` instead of the current `max-w-md` literal (note: current gap is `gap-8`, keep it).
- [x] The two forms (`Settings.tsx:94`, `:189`) each live in their own bounded-width `Card` (readable line length, not edge-to-edge) inside a `flex flex-wrap` (or grid `auto-fit`) container, so they sit side-by-side when there's room and stack when there isn't. (`BackupLinks` also got the same `Card` treatment — see Comments.)
- [x] Payer/payment-method management (add/rename/remove entries) adopts `Field`/`Label`/`Input`/`Badge` (for listing existing payers/methods) as fits, replacing any raw `<label>`/ad hoc list markup.
- [x] Any boolean settings use `Checkbox`/`Toggle`/`Switch` (the app already has `Switch` — don't introduce a second boolean control for no reason; pick whichever this page already leans toward, or `Toggle` where it reads as more of an action than a state). (No boolean setting exists on this page — nothing to convert, see Comments.)
- [x] `tsc`/build passes; manually verified at narrow, laptop, and wide viewport widths.

## Comments

Implemented in `web/src/pages/Settings.tsx`. No `FormSidebar` involved, per the ticket — this page has no list, just three independent sections. Root container is `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-8 p-6` (the `gap-8` kept as instructed). Below the page's own `<h1>`, a `flex flex-wrap items-start gap-6` row holds three `Card`s, each `w-full max-w-md` so a lone or wrapped-alone card doesn't stretch edge-to-edge: the lists+Target+Goal form (no `CardTitle` — nothing in the original had a heading for this specific section, and inventing one felt like unrequested scope beyond "adopt Card"), the Password form (`CardTitle` reusing `t.changePassword`, the same string the original's own `<h2>` and submit button already shared), and **Backup ed esportazione** — not literally one of the ticket's named "two forms" (it's links, not a form), but given the same `Card` treatment anyway: leaving the third section as a bare, unbordered block while its two siblings became bordered `Card`s in the same wrapped row would have read as an inconsistency, not a deliberate choice, so it picked up `CardTitle` (reusing `t.backupAndExport`, again already the original's own `<h2>` text) too. The old `border-t border-border pt-8` separators between sections are gone — each `Card`'s own ring/padding does that job now.

Payer/payment-method management: the deliberate one-line-per-entry `Textarea` editing is unchanged — the code's own comment explains why (`PUT` replaces the whole list; a row-per-entry add/rename/remove UI "would be three times the code to do less"), and nothing in this ticket asked to replace that with a granular per-entry editor, only to adopt the newer field/list components "as fits". Concretely: each `<label>` became `Field`/`FieldLabel`, and a row of read-only `Badge`s (`variant="secondary"`) was added under each Textarea, live-deriving from the box's current text (`toList(payers)`/`toList(paymentMethods)`) — a glanceable "here's what's currently listed" preview, not a second way to edit (no per-Badge remove button, which would have reintroduced exactly the "three times the code" problem the original comment warns about). `t.onePerLine`/`t.listsHistorySafe` became `FieldDescription`s at the group level (they describe both lists together, not one field specifically); `t.targetHint`/`t.changePasswordHint` became `FieldDescription` scoped to their own single field.

No boolean setting exists anywhere on this page (checked `Lists` in `types.ts` and the rest of the file) — checklist item on `Checkbox`/`Toggle`/`Switch` is vacuously satisfied, nothing to convert.

Verified live: reused the dev password from tickets 03–09, `go run ./cmd` + `pnpm --dir web dev`, scripted headless-Chromium (Playwright). Confirmed at 1280px, 1920px and 390px: the three `Card`s sit side by side at 1920px, two-per-row at 1280px, and stack fully at 390px; the Badge preview strips render the current payers/payment-methods correctly; a real lists save (`PUT /api/settings` 200) round-trips correctly and shows "Elenchi salvati." `npm run build` (`tsc -b && vite build`) passes clean throughout.
