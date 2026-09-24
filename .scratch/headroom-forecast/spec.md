Status: ready-for-agent

# Headroom forecast, rebuilt from last month forward

## Problem Statement

On Acquisti programmati the Headroom figures don't read the way the household thinks about money. "Entrate tipiche" looks like income but the household expects to see what's actually left each month; the projection starts from a statistical "typical month" instead of from what was really left last month; the monthly Goal is subtracted even when it drives a month negative, so a month that merely failed to save reads as a loss; the forecast ignores seasonality (a quiet August, a heavy December); and the horizon runs six months into next year regardless of where the year is. The household ends up with negative numbers it doesn't believe.

## Solution

Headroom (CONTEXT.md) is rebuilt the way the household reasons about it:

- **Start from reality**: the running total begins with the last completed month's actual leftover.
- **Forecast each remaining month** from the same season last year (the month before, the month itself and the month after, a year earlier), blended with this year's own record as it grows, plus what active Contracts still owe and the Recurring expenses due.
- **Goal is a buffer**: a month keeps what it leaves beyond Goal; a month that leaves less than Goal, or runs short by up to Goal, is zero; only a shortfall beyond Goal is negative. A deficit carries forward until later months recover it.
- **Horizon**: the remaining months of the current year, at least one and at most six.
- **The page** shows two cards: "Margine mensile previsto" (next month's forecast Headroom) and the Headroom accumulated to the end of the horizon. No explanatory hint text. Planned purchases are placed exactly as today.

## User Stories

**Reading the page**

1. As a household member, I want to see next month's forecast Headroom labelled "Margine mensile previsto", so I know what the coming month is expected to leave.
2. As a household member, I want to see the Headroom accumulated to the end of the horizon as a second card, so I know how much I can plan to spend in total.
3. As a household member, I want no explanatory hint under the figures, so the page stays clean.
4. As a household member, I want the month-by-month table to stay gone, so the page shows only what I act on.
5. As a household member, I want the forecast still labelled as a projection, so I never mistake it for fact.

**Starting point**

6. As a household member, I want the accumulated Headroom to start from what was actually left last month, so the forecast begins from real money rather than a statistic.
7. As a household member, I want last month's actual leftover to follow the same Goal rule as any forecast month, so real and forecast months read the same way.
8. As a household member, I want last month's Recurring expenses to count in its leftover even if no screen had generated them yet, so the starting point isn't flattered by a missing rent.
9. As a household member, I want last month's Contract Incomes to count in its leftover, because they are money that actually arrived.
10. As a household member, I want investment sales left out of last month's leftover, because a portfolio is never money to count on.

**Goal as a buffer**

11. As a household member, I want a month that leaves more than Goal to keep only the difference (Goal €500, +€800 → €300; +€501 → €1).
12. As a household member, I want a month that leaves less than Goal to read zero, not negative (Goal €500, +€200 → €0), because failing to save is not losing money.
13. As a household member, I want a month that runs short by up to Goal to read zero (Goal €500, −€300 → €0; −€500 → €0), because the money set aside for Goal covers it.
14. As a household member, I want only a shortfall beyond Goal to be negative (Goal €500, −€600 → −€100).
15. As a household member, I want a negative month to carry forward, so the accumulated Headroom can't turn positive until later months recover it.

**Forecasting a month**

16. As a household member, I want each forecast month to use last year's months around it (October uses last September, October and November), so a seasonal month is forecast from its own season.
17. As a household member, I want that seasonal figure blended with this year's own record, last year weighing (12 − n)/12 where n is the months of this year already over, so the forecast leans on last year early in the year and on this year as data is recorded.
18. As a household member without a record for that season last year, I want the forecast to use this year's record alone, so a household that started this year still gets a forecast.
19. As a household member, I want what active Contracts still owe to be added to the month it's due in, so a Contract ending shows up in the right month.
20. As a household member, I want the Recurring expenses due in a month to be subtracted from it, so a Recurring expense ending shows up in the right month.
21. As a household member, I want Investments to count only as money leaving, never as money to spend, in every forecast month.
22. As a household member, I want Contract Incomes left out of the seasonal and yearly medians, because what Contracts still owe already stands in for them.

**Horizon**

23. As a household member, I want the horizon to be the remaining months of the current year, so the forecast doesn't reach into a year it knows nothing about.
24. As a household member, I want at most six months, so the list stays short-term.
25. As a household member in November or December, I want at least one month (December in November; January in December), so the page always forecasts something.

**Placing Planned purchases**

26. As a household member, I want Planned purchases placed exactly as today (priority order, first month the accumulated Headroom covers it for the rest of the horizon, a purchase that doesn't fit skipped without blocking the others), over the new accumulated Headroom.
27. As a household member, I want a purchase that doesn't fit to be short by its amount minus the accumulated Headroom at the end of the horizon, a deficit widening the gap.
28. As a household member, I want the Savings-or-loan answer compared against that gap.

**Availability**

29. As a household member with less than 3 months of history, I want the same "servono almeno 3 mesi di storico" notice as today, and still be able to save Planned purchases.

## Implementation Decisions

- **Headroom report** (the existing reports endpoint Acquisti programmati already reads) is recomputed; its shape changes:
  - `start_cents`: last completed month's actual leftover after the Goal rule.
  - `months`: one entry per horizon month with its Headroom and running total (running includes `start_cents`), plus the forecast Income, forecast spending, Contract share and Recurring total that produced it, so every figure stays auditable through the API.
  - `placements`: unchanged shape.
  - The previous same-every-month breakdown fields (typical Income, typical spending, the blend weight) go away; Goal stays as a field.
- **Goal rule** — one function applied to both the start and every forecast month. With leftover L and Goal G: L ≥ G → L − G; −G ≤ L < G → 0; L < −G → L + G.
- **Start**: actual received Income by payment date in the last completed month (Contract Incomes included, investment sales excluded) minus all its Expenses (Recurring-generated and Investments included), after making sure that month's Recurring expenses are generated. Then the Goal rule.
- **Forecast month M**:
  - Seasonal medians: median of last year's M−1, M, M+1 (only months on or after the first Expense), separately for non-Contract, non-investment-sale Income and for non-recurring spending (Investments included).
  - Yearly medians: the same two figures over this year's completed months.
  - Blend: w × seasonal + (1 − w) × yearly, w = (12 − n)/12, n = completed months of the current year. No seasonal months → yearly alone; no completed month this year (January) → seasonal alone.
  - Leftover = forecast Income + Contract share(M) − forecast spending − Recurring due(M); then the Goal rule.
- **Horizon**: from next month to December of the current year, clamped to at least 1 and at most 6 months (December → January only).
- **Availability**: unchanged rule, at least 3 months of history since the first Expense; otherwise unavailable with no months and no placements.
- **Placement**: unchanged algorithm (suffix minimum of the running total, skip without blocking); the missing amount keeps the deficit (not floored), measured against the running total at the end of the horizon.
- **Page**: two cards, "Margine mensile previsto" (the first horizon month's Headroom) and the accumulated Headroom at the end of the horizon, labelled with the horizon length; the "Entrate tipiche" card and its hint string are removed; the projection disclaimer line stays.
- **Glossary**: CONTEXT.md's Headroom entry already rewritten to this definition.

## Testing Decisions

- Test external behaviour only, through the two existing seams — no new ones:
  - **The Headroom report over HTTP** with the test app's fixed clock (prior art: the current Headroom tests and Budget's tests), seeding Expenses, Incomes, Recurring expenses, Contracts and Goal through the API.
  - **The pure placement function**, as today.
- Cases:
  - The Goal rule table: +800 → 300, +501 → 1, +200 → 0, −300 → 0, −500 → 0, −600 → −100 (Goal 500).
  - The start uses last month's actual figures, with its Recurring expenses generated even if nothing read that month, Contract Incomes counted and investment sales left out.
  - The seasonal window picks last year's M−1..M+1 and ignores other months.
  - The blend weight, and both fallbacks (no seasonal data; January with no completed month this year).
  - Horizon length by clock: September → 3 (Oct–Dec), November → 1 (Dec), December → 1 (January), March → 6.
  - A deficit carries and blocks the running total from turning positive until recovered.
  - Placement over the new running total, and the missing amount including a deficit.
  - Unavailable under 3 months of history.

## Out of Scope

- Investment performance, prices and sales seasons.
- A configurable horizon or Goal-buffer size.
- Recurring incomes (still "da valutare").
- Changing how Contracts count (what they still owe, spread over their remaining months).
- Any Dashboard card.

## Further Notes

- Supersedes the Headroom parts of `.scratch/planned-purchases/spec.md`; the Planned purchase record, priority order and "Comprato" are unchanged.
- The two-card layout already on this branch (typical Income + accumulated Headroom) is the starting point: the first card is replaced, the second relabelled.
