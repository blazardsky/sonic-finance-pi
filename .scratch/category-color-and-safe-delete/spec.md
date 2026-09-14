Status: ready-for-agent

# Category Color Coding, Safe Delete, and Applies-To Grouping

## Problem Statement

Deleting a Category or Subcategory still in use is simply refused today, with no way forward except hiding it — there's no path to actually retire one and move its Expenses/Incomes somewhere else. The household also has no visual way to tell related Categories apart at a glance in a growing list (~18 Categories, 16 Subcategories today, with no ceiling), and the Categories management page lists everything in one flat, unsorted table regardless of whether a row applies to Expenses, Incomes, or both. Finally, two near-duplicate travel Categories (`Viaggi` and `Vacanze`) exist purely because there was no safe way to consolidate them once both had Expenses attached.

## Solution

1. Deleting a Category or Subcategory still in use offers a way to actually finish the job: pick another one on the same side to reassign everything to (mandatory for Category, since every reference to it is required; optional for Subcategory, which can also just be cleared).
2. Every Category and Subcategory can carry a color — one of 9 fixed, WCAG AA-validated palette entries — rendered in the management tables, every picker, and the existing chart views. Colors are expected to be shared across related Categories; the point is grouping, not unique identity.
3. The Categories and Subcategories management tables split into sections by `applies_to` (Spese / Entrate / Entrambi), sorted by color group and then name within each section.
4. `Vacanze` is merged into `Viaggi` using the new safe-delete mechanism as its first real use.

## User Stories

**Safe delete — Category**

1. As a household member, I want deleting a Category still in use to offer me a replacement Category on the same side, so I can actually retire one instead of only ever hiding it.
2. As a household member, I want every Expense, Item, Income, Recurring expense, and Client default currently pointing at the deleted Category to point at the replacement instead, so nothing is left dangling or silently orphaned.
3. As a household member, I want the replacement picker to only offer Categories that accept the same side (Expense/Income/both) as the one I'm deleting, so I can never end up with an Expense pointing at an Income-only Category.
4. As a household member, I want a Base Category to still refuse deletion outright, replacement or not, so the reports that resolve it by identity never break.
5. As a household member, I want deleting a Category with nothing pointing at it to work exactly as it does today — no picker, no extra step — so the common case stays just as fast.

**Safe delete — Subcategory**

6. As a household member, I want deleting a Subcategory still in use to offer me a choice: replace it with another Subcategory on the same side, or just remove the tag from everything using it, so I'm not forced into inventing a replacement when I really just want it gone.
7. As a household member, I want "just remove the tag" to leave the Category on every affected Expense untouched, so clearing a Subcategory never touches anything else about the entry.

**Category and Subcategory color**

8. As a household member, I want to pick a color for a Category or Subcategory from a small fixed set, so I can visually group related ones together.
9. As a household member, I want to leave a Category or Subcategory uncolored and have it read as a neutral default, so picking a color is never a required chore.
10. As a household member, I want two unrelated Categories to be able to share the same color without it being treated as a mistake, so I can use a limited palette to mean "these belong together" across as many Categories as I want.
11. As a household member, I want a Category's color to show up in the management table, in every picker where I choose a Category or Subcategory, and in the existing charts (trend chart, Dashboard's upcoming list, the yearly report's pie/bar), so the same color means the same thing everywhere I see it.
12. As a household member, I want my existing Categories to keep reading as close to their current color as possible once this ships, so the switch to a fixed palette doesn't visually scramble everything I'm used to.
13. As a household member, I want each color's text-on-background pairing to be legible (not just distinct hues), so a colored chip is still readable, not just colorful.

**Applies-to grouping**

14. As a household member, I want the Categories table split into Spese / Entrate / Entrambi sections, so I'm not scanning a flat list mixing sides that were never interchangeable anyway.
15. As a household member, I want the Subcategories table split the same way, for the same reason.
16. As a household member, I want Categories sharing a color to sit next to each other within their section, so the color grouping is visible in the list itself, not just in charts.

**Viaggi/Vacanze merge**

17. As a household member, I want `Vacanze`'s Expenses moved onto `Viaggi` and `Vacanze` itself gone, so travel spending isn't split across two near-identical Categories going forward.

## Implementation Decisions

### Schema (schemaVersion 16 → 17, one migration step)

- `category` gains `color TEXT NOT NULL DEFAULT 'blue-gray' CHECK (color IN ('blue','orange','aqua','yellow','magenta','green','violet','red','blue-gray'))`.
- `subcategory` gains the identical column and CHECK.
- Existing `category` rows are backfilled by the migration itself (not left to the application default) to the nearest of the 8 vivid slots by hue distance from each row's legacy generated color — see "Migration backfill" below. Existing `subcategory` rows simply take the column default (`blue-gray`): the old hash-based color never applied to Subcategory, since Subcategory didn't exist before this session.

### Migration backfill (existing Categories only)

The retired frontend color function was `hue = (categoryId * 137.508) % 360`. The migration reproduces that formula in the migration step itself (or a one-time Go port of it) to get each existing Category id's legacy hue, then assigns whichever of the 8 vivid slots' own hue angle is angularly closest (circular distance on the 0–360 wheel):

| slot | hue° |
|---|---|
| red | 0.4 |
| orange | 17.0 |
| yellow | 40.8 |
| green | 120.0 |
| aqua | 158.5 |
| blue | 212.8 |
| violet | 248.8 |
| magenta | 337.4 |

`blue-gray` is never a snap target — it's the neutral default for categories with no meaningful legacy color to preserve, not a hue on the wheel.

### The 9-slot palette

Computed, not hand-picked: the 8 vivid slots reuse the dataviz standard's own CVD-validated categorical hues as `primary` (both light and dark chart variants, unchanged from that standard); `background` and `foreground` per slot are newly derived from the same hue and verified to clear WCAG AA (≥4.5:1, background vs. foreground) — this is new work this palette needed that the dataviz standard's own reference doesn't define, since that standard only specifies one chart-mark color per hue, not a background/foreground chip pair. `blue-gray` is a ninth, desaturated entry outside the dataviz set, computed the same way.

Light mode:

| slot | primary | background | foreground | contrast |
|---|---|---|---|---|
| blue | `#2a78d6` | `#e7eff8` | `#195fb3` | 5.44 |
| orange | `#eb6834` | `#f8ece7` | `#b34519` | 4.80 |
| aqua | `#1baf7a` | `#e7f8f2` | `#127d57` | 4.66 |
| yellow | `#eda100` | `#f8f3e7` | `#8f6814` | 4.56 |
| magenta | `#e87ba4` | `#f8e7ee` | `#b31953` | 5.55 |
| green | `#008300` | `#e7f8e7` | `#128112` | 4.55 |
| violet | `#4a3aa7` | `#eae7f8` | `#3019b3` | 9.02 |
| red | `#e34948` | `#f8e7e7` | `#b31a19` | 5.71 |
| blue-gray | `#7b8b9d` | `#eef0f1` | `#52647a` | 5.31 |

Dark mode (own surface, own computation — not a mechanical flip of the light values):

| slot | primary | background | foreground | contrast |
|---|---|---|---|---|
| blue | `#3987e5` | `#1e2c3e` | `#6596d2` | 4.61 |
| orange | `#d95926` | `#3e271e` | `#d18161` | 4.64 |
| aqua | `#199e70` | `#1e3e33` | `#61d1aa` | 6.25 |
| yellow | `#c98500` | `#3e331e` | `#d1ab61` | 5.72 |
| magenta | `#d55181` | `#3e1e2a` | `#d56d92` | 4.52 |
| green | `#008300` | `#1e3e1e` | `#61d161` | 6.13 |
| violet | `#9085e9` | `#211e3e` | `#877dd9` | 4.52 |
| red | `#e66767` | `#3e1e1e` | `#d67171` | 4.58 |
| blue-gray | `#9da5af` | `#2a2e32` | `#8d98a5` | 4.67 |

Every pairing above clears 4.5:1. This table is the palette; it lives once as a shared frontend constant (mirroring how `--chart-N` tokens were replaced by `categoryColor` before), keyed by the same 9 strings the `color` column stores — nothing computes a color at render time anymore.

### Color usage

- `Category`/`Subcategory` JSON gains `color`, always one of the 9 keys, never empty.
- Rendered as a small swatch (the slot's `primary`) next to the name in: the Categories table, the Subcategories table, and every `<SelectItem>` in the Expense/Income/Recurring category and subcategory pickers.
- Chart call sites (`categoryColor` in `web/src/lib/trend.ts`, its callers in `CategoryTrendChart.tsx`, `Dashboard.tsx`'s upcoming-list dot, `YearlyReport.tsx`'s pie/bar) switch from computing a hue by id to looking up the Category's own stored `color` slot's `primary` (light) / dark variant. `categoryColor(id)` is deleted, not deprecated-in-place.
- The add form for both Category and Subcategory gains a color picker (9 swatches, `blue-gray` visually distinguishable as "no color chosen" without being a separate null state). `PATCH` on either accepts an optional `color` to change it later, exactly like `hidden` — including on a Base Category, since color is not one of the attributes Base protection covers.

### Safe delete

- `DELETE /api/categories/{id}` behavior is unchanged when the Category is unused (204) or Base (409, unconditionally — this guard runs before anything else). When in use and not Base, it now accepts an optional `replace_with` query parameter (a Category id):
  - Absent: refuses with 409, exactly as today (no behavior change for an unaware caller).
  - Present: validated the same way `category_id` is validated on an Expense/Income write (must exist, must accept the same side as the Category being deleted) — 400 if not. On success, in one transaction: every `expense.category_id`, `item.category_id`, `income.category_id`, `recurring_expense.category_id`, and `client.default_category_id` pointing at the old id is updated to `replace_with`, then the old Category row is deleted.
- `DELETE /api/subcategories/{id}` gains the same `replace_with` param (validated the same way, against Subcategory this time) plus a `clear=true` alternative:
  - Neither present: refuses with 409, unchanged from today.
  - `replace_with` present: same reassignment pattern, over `expense.subcategory_id` and `recurring_expense.subcategory_id`.
  - `clear=true` present: the same two columns are set to `NULL` instead, then the Subcategory is deleted. Requiring an explicit `clear=true` (rather than treating a bare DELETE as "clear by default") keeps the conservative default: no existing caller can silently wipe every reference just by retrying an unparameterized delete.
- Frontend: the existing delete flow (confirm dialog, then `DELETE`) is unchanged for the common unused case. On a 409, instead of just showing an error, a second dialog opens offering the replacement picker (Category: required; Subcategory: the picker plus a "remove the tag instead" option) and re-issues the `DELETE` with the appropriate parameter once confirmed.

### Applies-to grouping

- Both management tables split into three sections (Spese / Entrate / Entrambi) instead of one flat table, matching the existing three `applies_to` values — no new grouping concept.
- Within each section, sort by `color` (grouped, `blue-gray`/uncolored last) then `name COLLATE NOCASE` — replacing today's plain `ORDER BY name COLLATE NOCASE`.

### Viaggi/Vacanze merge

- Performed via the new mechanism once it ships: `DELETE /api/categories/{vacanze_id}?replace_with={viaggi_id}`. No separate data migration — this is the feature's own first real use, on real data.

## Testing Decisions

- Single seam, matching every existing spec in this repo: black-box HTTP tests via the shared `testApp` harness (`a.post`/`a.patch`/`a.delete`/`a.get`), no mocking, no new seam introduced.
- `categories_test.go` / a new `subcategories_test.go` section: `replace_with` reassigns every referencing row and removes the old Category/Subcategory (mirroring the existing `TestDeletingACategoryInUseIsRefused` shape, extended); `replace_with` pointing at a wrong-side or nonexistent target is refused with 400; a Base Category still refuses deletion even with `replace_with` set; `clear=true` on a Subcategory nulls out every reference without touching the Category on the same rows.
- Same files: `color` defaults to `blue-gray` on create when omitted, round-trips on `PATCH`, and is rejected outside the 9-value CHECK the same way an unknown `applies_to` already is.
- `migrate_test.go`: the schemaVersion 16→17 step backfills existing Categories' `color` deterministically from their id (same fixture style as the other migration-step tests, e.g. `TestItemPricingMigrationAddsColumnsWithSafeDefaults`) — assert a couple of known ids land on the expected slot, not just that the column exists.
- No automated test for the color swatch rendering, the picker UI, or the two-choice delete dialog — frontend-only visual behavior, consistent with this repo's established practice (no frontend test runner) already stated in the `client-contracts-and-reports` spec's own Testing Decisions.

## Out of Scope

- A free color picker (hex/RGB) — the palette is exactly 9 fixed entries, no custom colors.
- Any change to which Categories are Base-protected, or to what Base protection covers — color is simply not one of the guarded attributes.
- Reassigning `Item.category_id` independently of its parent Expense during a Category delete — it's included in the same reassignment pass as every other `category_id` column, not a separate decision.
- A history or audit trail of past reassignments (which Category something used to belong to before a merge).
- Extending safe-delete's replacement mechanism to any entity other than Category and Subcategory (Client, Holding, etc.).
- Round-robin or automatic color assignment on create — a new Category/Subcategory either gets an explicit pick or the `blue-gray` default, never an algorithmic guess.

## Further Notes

- Builds on ADR-0008 (Base category protection — unchanged, still runs first), ADR-0002 (Item's own Category, relevant to the reassignment's column list), and the new ADR-0016 recorded this session (color is a shared 9-slot palette, not a unique identity — read it before touching anything here, since sharing colors is the intended behavior, not a bug to fix).
- `CONTEXT.md` already has a `Subcategory` entry (added this session); no new glossary terms are needed for this spec — "color" and "palette" are UI/implementation vocabulary, not domain concepts the household reasons about.
- The dataviz standard's 8 primary hues (light and dark) are reused verbatim as this palette's `primary` role; only `background`/`foreground` are new computation this feature needed.
- This whole spec was shaped by an interactive grilling session in this repo's conversation history rather than a written brief — if anything here reads ambiguous, that transcript is the tie-breaker, not a guess.
