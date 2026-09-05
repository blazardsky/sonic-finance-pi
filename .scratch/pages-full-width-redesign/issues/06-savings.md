# 06: Risparmi (Savings.tsx) redesign

**What to build:** Same treatment as ticket 03, applied to Savings — this is the Buy/Sell + holdings-breakdown page.

**Blocked by:** 02

**Status:** ready-for-agent

- [x] Root container uses `max-w-(--content-max-width)` instead of the current `max-w-md` literal.
- [x] The Buy/Sell form (`Savings.tsx:232`) moves into the shared `form-sidebar` component; the holdings-breakdown table (`:265`) fills the remaining width.
- [x] Holding/Payer pickers use the new `Select`/`Combobox` instead of `native-select.tsx`.
- [x] The date field uses the new Date Picker recipe.
- [x] If the breakdown table has any row actions, consolidate them into a `DropdownMenu`; if it's read-only, leave it as a plain table. (Both tables are read-only — left as plain tables.)
- [x] After a successful submit, the sidebar stays open and the form clears.
- [x] `tsc`/build passes; manually verified at narrow, laptop, and wide viewport widths, including the mobile drawer + bottom trigger.

## Comments

Implemented in `web/src/pages/Savings.tsx`, adapted from tickets 03–05's pattern to this page's different shape (no "one form + one list" — a Buy/Sell form, a read-only portfolio breakdown, a read-only transaction history, plus a starting-balance setting, all sharing one screen). Reused the shared `FormSidebar`/`Sidebar` unchanged — no fixes needed, ticket 03 already covers the trigger z-index, `themed`, and `mobileWidth` issues at the shared-component level. Root container is `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6`; below the page's own `<h1>`, a `flex flex-wrap gap-6` row holds the list column (`flex-1 min-w-0`) on the left and `<FormSidebar>` (`title={kind === "buy" ? t.recordBuy : t.recordSell}`) on the right, holding only the Buy/Sell form — the ticket's checklist names just this form for the move, not the starting-balance setting or either read-only table, which stay in the list column as page-level content, now simply benefiting from the full width. Unlike tickets 03–05 there's no "editing" concept here (the transaction history has no click-to-correct-a-past-entry flow, so no `selectX`/reopen-on-load wiring was needed) — `sidebarOpen` exists only so the panel can be toggled, defaulting open.

The savings-total figure and the starting-balance form (previously two bare blocks) are now a `flex flex-wrap gap-4` row of two shadcn `Card`s, matching the stat-card convention already used on `Dashboard.tsx` (out of scope for this spec, but the same primitives) — a small bonus consistency touch, not asked for by the checklist but cheap given the form was already being rewrapped. The starting-balance amount input also picked up a `Field`+`InputGroup` € adornment for the same reason. The `error` state (Buy/Sell submit failures only — neither table has row actions that could error) stays inside the sidebar, next to the submit button, matching tickets 03/04's placement (unlike ticket 05, which kept it in the list column because that page's errors come from row actions too).

Holding and Payer pickers use the new `Select` (Holding: required + `disabled` with a hint when there are no Holdings yet, same pattern as ticket 05's recurring-investment Holding picker; Payer: required, no sentinel needed since it always starts unset). The date field uses ticket 01's `DatePicker` — a real day-granularity date here (a buy/sell's `occurred_on`/`payment_date`), unlike ticket 05's month-only fields, so it applies cleanly. Date + Payer sit side by side in one row, same pairing (DatePicker + a short-labelled Select) that worked fine on tickets 03/04 — verified live this still fits the 16rem sidebar without overflow, unlike ticket 04's two-DatePicker row.

Both tables (portfolio breakdown, transaction history) are read-only — neither had row actions before, so both stayed plain `Table`s with no `DropdownMenu`.

Verified live: reused the dev password from tickets 03–05, `go run ./cmd` + `pnpm --dir web dev`, scripted headless-Chromium (Playwright). Seeded one Holding via a direct `POST /api/holdings` call (this dev db had none) to exercise the form past its disabled state. Confirmed at 1280px, 1920px and 390px: full-width layout with the stat-card row reflowing correctly at each width, a real Buy (`POST /api/expenses` 201) and Sell (`POST /api/incomes` 201) both landing in the transaction history and portfolio breakdown with the correct net/percentage and +/− sign, the starting-balance save working independently of the Buy/Sell form, the sidebar toggle open/closed via a real DOM click, and the mobile FAB opening a genuinely full-width drawer with Date+Payer fitting side by side. `npm run build` (`tsc -b && vite build`) passes clean throughout.
