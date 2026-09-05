# 12: Anno (Year.tsx) redesign + real chart

**What to build:** Full-width layout for the Year page, plus replacing its hand-rolled div-based bar chart with a real shadcn `Chart`.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] Root container uses `max-w-(--content-max-width)` instead of the current `max-w-md` literal.
- [x] The totals `<dl>` and the tax summary section reflow via `flex flex-wrap`/grid `auto-fit` instead of one forced narrow column.
- [x] The 12-month total is currently a custom `<div>`-based bar chart (`Year.tsx`, custom `Bar` divs) — replace it with a real shadcn `Chart` (the already-installed `chart.tsx` wrapper around Recharts, a `BarChart`), using the existing `--chart-1..5` CSS tokens, consistent with how `Dashboard.tsx`/`Month.tsx` already chart their data. The numbers must remain visible (via labels and/or `ChartTooltip` on hover) — this isn't a decorative-only replacement.
- [x] Do **not** touch `YearlyReport.tsx` ("Report annuale", the separate `yearReport` screen reached from within Anno) — explicitly out of scope this round. (Confirmed untouched — `git status` shows no changes to it.)
- [x] `tsc`/build passes; manually verified at narrow, laptop, and wide viewport widths.

## Comments

Implemented in `web/src/pages/Year.tsx` only — `YearlyReport.tsx` was never opened for editing and `git status` confirms no changes to it. Root container is `mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6`. Following ticket 11's (Month) precedent exactly: all three sections — Totals, the tax summary, and the by-month chart — are `Card`s inside one `flex flex-wrap items-start gap-6` row, sized by content rather than a fixed grid (`min-w-64 flex-1` for the two compact stat cards, `min-w-full flex-[2] lg:min-w-[32rem]` for the chart card, same values ticket 11 used for its own equivalents).

The 12-month chart: replaced the custom `<div>`-based `Bar` helper (width set inline as a percentage of the year's peak month) with a real `BarChart` from `chart.tsx`, following `CategoryTrendChart.tsx`'s established convention (`ChartContainer` + `ChartConfig` + `XAxis` + `ChartTooltip`/`ChartTooltipContent` with a custom `formatter` for € values + `ChartLegend`) rather than inventing a second pattern. One deliberate deviation from "use `--chart-1..5` for both series": income uses `var(--chart-2)` (the same token mechanism `categoryColor()` uses elsewhere), but expense uses `var(--destructive)` — the app's existing, consistent color for "money out" (a negative `Row` already renders in it). All five `--chart-N` tokens are variations on the same blue, so two of them side by side would have lost the red/blue income-vs-expense distinction the original hand-rolled bars had (`bg-primary`/`bg-destructive`); keeping `--destructive` for expense preserves that while still routing income through the shared chart-token mechanism the ticket asked for. The numbers are reachable via `ChartTooltip` on hover (verified live — hovering a bar shows the month, "Entrate €X,XX" and "Spese €X,XX" with matching colour swatches), same as `CategoryTrendChart`'s own charts elsewhere; nothing is print laid permanently over the bars, matching how the app's other charts already work.

No raw-HTML-to-shadcn swap was otherwise needed: the year stepper is already a `Button`-flanked `<h1>` (there's no native year picker to replace, per the code's own comment), and the `<dl>`/`<dt>`/`<dd>` label-value pairs stay as genuinely appropriate semantic HTML, same reasoning as ticket 11.

Verified live: reused the dev password from tickets 03–11, `go run ./cmd` + `pnpm --dir web dev`, scripted headless-Chromium (Playwright). Confirmed at 1280px, 1920px and 390px: the three cards fit one row at 1280px and 1920px and stack at 390px; the chart renders real bars (not decorative placeholders) with a working legend and hover tooltip showing the exact € figures; year navigation (prev/next) still works after the layout change (clicking "previous year" correctly moved 2026 → 2025 and reloaded the totals). `npm run build` (`tsc -b && vite build`) passes clean throughout.
