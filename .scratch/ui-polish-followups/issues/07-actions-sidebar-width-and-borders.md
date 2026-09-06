# 07: Actions sidebar — width, borders, submit bar position

**What to build:** The shared `FormSidebar`/`Sidebar` components (used by Spese, Entrate, Ricorrenti, Risparmi, Categorie, Clienti, Investimenti) get a wider desktop panel, a bordered submit bar, a consistently full-height panel border, and a submit bar that tracks the bottom of the viewport until the true page end approaches.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] The form sidebar's desktop width is 20rem (up from its current value).
- [ ] The submit/cancel footer bar has a visible border separating it from the page content behind it.
- [ ] The panel's border spans the full viewport height even when the panel's own content is shorter than the viewport, instead of looking "short".
- [ ] The submit bar sits at `bottom-0` of the viewport while the page still has more than 3rem below the fold, and eases to `bottom-3` once the true page bottom is within 3rem (position: sticky is an acceptable implementation if it satisfies this without overlapping page content past the list's end).
- [ ] Verified across at least two of the seven pages sharing this component, with both a short-content and a long-content panel.
