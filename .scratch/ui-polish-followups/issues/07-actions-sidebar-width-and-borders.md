# 07: Actions sidebar — width, borders, submit bar position

**What to build:** The shared `FormSidebar`/`Sidebar` components (used by Spese, Entrate, Ricorrenti, Risparmi, Categorie, Clienti, Investimenti) get a wider desktop panel, a bordered submit bar, a consistently full-height panel border, and a submit bar that tracks the bottom of the viewport until the true page end approaches.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] The form sidebar's desktop width is 20rem (up from its current value).
- [x] The submit/cancel footer bar has a visible border separating it from the page content behind it.
- [x] The panel's border spans the full viewport height even when the panel's own content is shorter than the viewport, instead of looking "short".
- [x] The submit bar sits at `bottom-0` of the viewport while the page still has more than 3rem below the fold, and eases to `bottom-3` once the true page bottom is within 3rem (position: sticky is an acceptable implementation if it satisfies this without overlapping page content past the list's end).
- [x] Verified across at least two of the seven pages sharing this component, with both a short-content and a long-content panel.

## Comments

Implemented in `web/src/components/form-sidebar.tsx` and `web/src/components/ui/sidebar.tsx`:

- Width: `FormSidebar`'s own `SidebarProvider` overrides `--sidebar-width` to 20rem, scoped to the form panel only (the main nav sidebar's 16rem is untouched).
- Panel border height: the `contained` `sidebar-container` div gets `h-full`, and each of the 7 pages' outer wrapper div gets `md:min-h-full`, so the row holding the list and the panel has a real height to stretch into — the panel border now matches the taller of the two columns, or the viewport when both are short.
- Submit bar: tried `position: sticky` first per the ticket's suggestion, but verified live (Playwright against a running dev server) that it doesn't stick — this app's shell is sized with `min-h-svh` (a floor), so `main`'s `overflow-auto` never actually clips into its own scrollport; the window scrolls the whole page instead, and sticky only holds against an ancestor that actually scrolls. Kept the bar `fixed` (as before) but now computes its own `bottom` inset in JS: 0 while the panel's measured bottom is more than 3rem below the viewport bottom, linearly easing to a 0.75rem rest gap as that gap closes to 0. The 4 pages that previously reserved `md:pb-32` in their form to dodge the old always-`bottom-3` overlay no longer need it — the dynamic easing already stops short of the panel's true end.
- Verified with a live Playwright session (fresh dev DB) against Expenses (empty form + list, then a 30-row list to force scrolling) and Categories (short form and list) at 1400×900: panel width 320px in both, panel border height matches the row's height exactly in every case (short-viewport-filled, and long-content-stretched), and the footer sits flush at `bottom-0` while scrolling through the middle of a long list, easing to a small rest gap at the true bottom without covering the last row.
