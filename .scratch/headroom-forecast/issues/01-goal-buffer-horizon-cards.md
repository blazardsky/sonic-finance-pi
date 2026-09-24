# 01: Goal as a buffer, horizon to the end of the year, new cards

**What to build:** Every forecast month's Headroom follows the Goal-buffer rule: with leftover L and Goal G, L ≥ G keeps L − G, −G ≤ L < G reads 0, L < −G reads L + G. The horizon becomes the remaining months of the current year, at least one and at most six (December forecasts January). The page shows two cards, "Margine mensile previsto" (the first horizon month's Headroom) and the Headroom accumulated to the end of the horizon; the "Entrate tipiche" card and its hint go. Placement and the missing amount keep working over the new running total.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] Goal rule (Goal 500): +800 → 300, +501 → 1, +200 → 0, −300 → 0, −500 → 0, −600 → −100
- [x] A deficit carries in the running total until later months recover it
- [x] Horizon by clock: March → 6 months, September → 3 (Oct–Dec), November → 1 (Dec), December → 1 (January)
- [x] Page: "Margine mensile previsto" and the accumulated Headroom cards; no "Entrate tipiche" card, no hint text
- [x] Placement and missing amount unchanged in behaviour over the new running total
