# 02: High-contrast CSS tokens

**What to build:** The actual visual effect of the `"high-contrast"` class from ticket 01: starker structural tokens (background, foreground, border, muted text, focus ring) for both light and dark bases, aimed at ~5.5:1 text contrast and outdoor readability — not thicker borders, not a repaint of category/chart colors, not full WCAG AAA.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] Audit `web/src/index.css`'s `:root` and `.dark` blocks and list every structural color token driving text/background/border/focus (`--background`, `--foreground`, `--card`, `--border`, `--input`, `--muted`, `--muted-foreground`, `--ring`, plus any others in the same category — not `--chart-N` or category-color tokens)
- [ ] Compute today's actual contrast ratios for the tokens above, in both light and dark, as a baseline to improve on
- [ ] **Stop and ask before finalizing colors**: propose two candidate high-contrast palettes (each targeting ~5.5:1 text-on-background, borders/muted text a bit darker/lighter than today but not maximally saturated, a focus ring that stays visible without being the loudest thing on screen), and render both as a small static HTML mock — a card, a table row, a form field, a focused button — for light+HC and dark+HC side by side. Get a choice back before writing the final values into `index.css`.
- [ ] Add `.high-contrast { ... }` (light base) and `.dark.high-contrast { ... }` (dark base) rule blocks to `web/src/index.css`, redefining only the audited tokens with the chosen values
- [ ] Confirm no `border-width`, spacing, font-size, `--chart-N`, or category-color token is touched by either new block
- [ ] Manual check: with ticket 01's toggle on, walk a handful of representative screens (Dashboard, an Expense list row, a form with a focused input, a chart) in each of light+HC and dark+HC, confirming text/background/border read clearly and the focus ring is visible but not overpowering
