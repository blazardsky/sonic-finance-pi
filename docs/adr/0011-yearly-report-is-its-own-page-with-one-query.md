# The full yearly report is its own page, deliberately doing what Year.tsx avoids

`readYearReport` deliberately has no per-month Category breakdown — the comment next to it calls twelve month-breakdown queries the heaviest query in the app, unaffordable on a Pi Zero W for a screen loaded by default. The new full yearly report wants exactly that: expenses by Category, for every month of a year.

We're building it anyway, as a **separate, dedicated report page** the household opens on purpose rather than folding it into `Year.tsx`, and as **one query** grouped by `(month, category)` rather than twelve separate per-month breakdown calls — the same cost as any single month's existing breakdown, not twelve of them. The two decisions together are what make this affordable: a report nobody is forced to load every time they check the year, computed in one pass instead of the shape the original comment was warning against.

## Consequences

`Year.tsx` is untouched and stays exactly as cheap as it is today. Anyone extending the full report later should keep it to one grouped query per figure it needs, not one query per month.
