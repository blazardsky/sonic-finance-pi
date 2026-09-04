# A Contract's "invoice this month" figure recomputes from the shortfall, not a fixed schedule

A Contract states a total and a date range, not a payment schedule. The obvious naive reading — divide the total evenly across the months and always ask for that fixed amount — was rejected: it would keep suggesting the same number even after a month ran short or over, silently drifting from the total actually owed.

Instead, "how much to invoice this month" is `(contract total − everything already received or already invoiced under this Contract) ÷ months remaining`, recomputed every time it's shown. It adapts automatically: invoice less than expected one month, and the following months' figure rises to compensate; invoice more, and it falls.

## Consequences

The figure can swing between months if invoicing is uneven — that's the point, not a bug. There is deliberately no record of what any single month "should" have been in isolation; only the running total and the remaining months matter.
