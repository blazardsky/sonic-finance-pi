# 03: Yearly Report's stat tiles get the Dashboard's Card treatment

**What to build:** `Figure` (defined in `YearlyReport.tsx`, reused by
`Month.tsx` for its own Entrate/Spese/Differenza tiles) currently renders as
a bare `<div className="rounded-md border border-border p-3">` — plainer
than every stat tile on the Dashboard, which uses `Card`/`CardHeader`/
`CardTitle`/`CardContent`. Restyle `Figure` to match that pattern instead.

Since `Figure` is shared, this changes Month.tsx's three tiles too, not just
the Yearly Report's six — intentional: there's no reason for the shared
component to look plain in one place and prominent in the other, and forking
a second, near-identical component just to scope the change to one page
would be the wrong kind of caution here.

**Blocked by:** None

**Status:** resolved

- [x] `Figure` renders its label/value inside `Card`/`CardHeader`/
      `CardTitle`/`CardContent` (or the closest fit), matching the visual
      weight of Dashboard's own stat cards.
- [x] Negative-value destructive-red styling (`cents < 0`) still applies.
- [x] Checked live on both Month.tsx and YearlyReport.tsx — six tiles on one
      page, three on the other, neither should look broken or overly tall.

## Comments

Verified live on both pages. Month.tsx's Entrate/Spese/Differenza and
YearlyReport's six tiles now match Dashboard's card style; the negative
Differenza figure still renders in destructive red.
