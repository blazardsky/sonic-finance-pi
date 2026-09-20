Status: ready-for-agent

# High-Visibility / High-Contrast Theme

## Problem Statement

The app is read outdoors (checking a balance or a client's payment status on the phone, in daylight) where the existing light and dark themes' normal contrast levels are hard to read at a glance. `v1.2.5`'s TODO item — "Aggiungere il tema Alta visibilità / Alto contrasto" — asks for a mode that trades subtlety for starker, more legible contrast.

Two other items originally scoped for `v1.2.5` were descoped during planning and are out of this spec entirely (see Further Notes): moving the annual report's cards to the year view (rejected outright — conflicts with ADR-0011), and recurring incomes (deferred to a new "Maybe" section in `TODO.md` — conflicts with `CONTEXT.md`'s "Incomes do not recur").

## Solution

High-contrast is an independent on/off toggle layered over the existing `light`/`dark`/`system` theme choice, not a fourth theme value. Turning it on swaps a set of structural CSS tokens (background, foreground, border, muted text, focus ring) for starker variants of whichever base theme (light or dark) is currently resolved — everything else (category colors, chart palette, spacing, borders' weight) is untouched. It's a per-device preference stored in `localStorage`, exactly like the existing theme choice, with no server or schema involvement.

## User Stories

1. As a household member checking the app outdoors, I want a high-contrast mode that makes text and numbers easier to read in bright light, so I don't have to shade the screen or squint.
2. As a household member, I want high-contrast to be a toggle independent of my light/dark choice, so I can have high-contrast light or high-contrast dark, and switching one doesn't reset the other.
3. As a household member, I want to reach the toggle from the header, next to the existing theme button, so it's as quick to reach as light/dark already is.
4. As a household member, I want my high-contrast choice to persist on this device the same way my theme choice already does, so I don't have to re-enable it every visit.
5. As a household member, I want high-contrast to make text and interactive boundaries easier to read without turning the app visually loud — no thicker borders, no color changes to categories or charts, just starker background/foreground/text contrast and a focus ring I can still see without it glowing.

## Implementation Decisions

### State and persistence

- `web/src/components/theme-provider.tsx`'s `ThemeProvider` gains a second, independent piece of state: `highContrast: boolean` / `setHighContrast(next: boolean)`, exposed on `ThemeProviderContext` alongside the existing `theme`/`setTheme`.
- Stored under its own `localStorage` key (e.g. `"high-contrast"`, sibling to the existing `"theme"` key — reuse the existing `storageKey` prop pattern rather than hardcoding a second literal), values `"true"`/`"false"`. Missing or unparseable defaults to `false`.
- Applied the same way `theme` is applied: an effect toggles a `"high-contrast"` class on `document.documentElement`, additive to whichever of `"light"`/`"dark"` is already there from the existing `applyTheme` effect — never replacing it. Cross-tab sync via the existing `storage` event listener extends to this key too, same as `theme` already does.
- No new keyboard shortcut. The existing `"d"` shortcut stays theme-only; high-contrast is toggle-only, mouse/tap.

### Header control

- `web/src/components/site-header.tsx` gains a second icon button next to the existing `ThemeToggle`, following the same `Button variant="ghost" size="icon-sm"` pattern and `headerButton` class. A single click flips `highContrast`; the icon reflects current state (e.g. two `@remixicon/react` icons for on/off — pick whichever pair in the existing icon set reads clearly as "contrast on" vs "off", consistent with `ThemeToggle`'s own icon-per-state approach). `aria-label` states the resulting action or current state, matching `ThemeToggle`'s convention.

### CSS tokens

- `web/src/index.css` gains two new rule blocks, `.high-contrast` (light base) and `.dark.high-contrast` (dark base), each redefining only the structural tokens that already exist in `:root`/`.dark` today: `--background`, `--foreground`, `--card`, `--border`, `--input`, `--muted`, `--muted-foreground`, `--ring`, and any other token currently driving text/background/border color (audit `:root`/`.dark` for the full list — do not invent new token names). `--chart-N` and any category-color tokens are never touched.
- Values are **not decided by this spec**. Before writing final OKLCH/hex values, the implementing agent must:
  1. Compute the current light/dark tokens' actual contrast ratios as a baseline.
  2. Propose two candidate high-contrast palettes (each hitting roughly 5.5:1 text-on-background — not overshooting for its own sake, not requiring full WCAG AAA), and darken/lighten borders and muted text a bit rather than leaving them at today's low-contrast values.
  3. Render both as a small static HTML mock (a card, a table row, a form field, a focus ring on a button) side by side, for each of light+HC and dark+HC.
  4. Ask which to use before committing — this is a deliberate stop-and-ask step, not a judgment call to make silently.
- Borders/dividers get a touch darker/lighter for definition, not thicker — no `border-width` changes anywhere.
- Focus rings (`--ring`) get a high-contrast variant that stays clearly visible against both the new background and the elements it outlines, without being the brightest thing on screen (i.e., don't just crank `--ring` to a saturated primary color — a slightly darker/lighter but still legible ring, consistent with the "starker, not louder" brief).

### Explicitly not touched

- Category color palette (ADR-0016's 9-slot palette) — same colors in both normal and high-contrast mode.
- Chart series colors (`--chart-N` tokens).
- Layout, spacing, border widths, font sizes.
- Any server-side setting or schema — this is 100% client-side, `localStorage`-only, same as the existing theme mechanism.

## Testing Decisions

- No automated test: this is a CSS-token and small-UI-state change with no branch of application logic to assert on, consistent with this repo's established practice of no frontend test runner (see prior specs, e.g. `category-color-and-safe-delete`'s Testing Decisions).
- Manual check before shipping: toggle high-contrast on top of each of light, dark, and system-resolved-to-either, and confirm the header icon's state and the applied class both agree with `localStorage` after a reload.

## Out of Scope

- Any change to the category color palette or chart colors (see above).
- A dedicated Settings-page control — the header icon is the only entry point (Settings page is left alone).
- A high-contrast variant of the 9-slot category palette.
- Recurring incomes (`v1.2.5`'s third original item) — deferred, see `TODO.md`'s new "Maybe" section.
- Moving the annual report's cards to the year view (`v1.2.5`'s second original item) — rejected outright, conflicts with ADR-0011 (see Further Notes).

## Further Notes

- **Recurring incomes descoped**: `CONTEXT.md`'s Recurring expense glossary entry states "Incomes do not recur — this household's income varies month to month." The household chose, during grilling, to leave this as-is for now rather than reverse the domain decision; the TODO item moved to a new "Maybe" section below the roadmap rather than being planned here. If revisited later, re-run `/domain-modeling` on the reversal itself before touching schema, since it would also require updating `recurring_expense`'s own doc comment ("Incomes do not recur — this household's income varies month to month" is baked into `cmd/recurring.go`'s comments too) and deciding a data shape (a new `recurring_income` table was the leaning during grilling, mirroring the existing Expense/Income table split, carrying only the bare fields a template needs — amount, category, payer, window — not Income's Client/invoice/contract fields).
- **Annual report → year view move rejected**: the six cards on `YearlyReport.tsx` (expense excluding tax, taxes, savings at start of year, median expense/income/net) are served by `/api/reports/year/{year}/full`, the same endpoint ADR-0011 calls "the heaviest query in the app" — bundled together with the per-month category breakdown ADR-0011 deliberately keeps off `Year.tsx`, which the ADR states explicitly stays "untouched... exactly as cheap as it is today." Moving the cards as-is would have reloaded that cost on a default-loaded page; a lightweight split-out endpoint was the fallback option, but the household chose to reject the item outright instead. Marked `RIFIUTATO` (strikethrough) in `TODO.md` rather than deleted, so the decision and its reasoning stay visible.
- This whole spec was shaped by an interactive grilling session in this repo's conversation history rather than a written brief — if anything here reads ambiguous, that transcript is the tie-breaker, not a guess.
