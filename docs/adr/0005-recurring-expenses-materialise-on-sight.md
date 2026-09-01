# Recurring expenses materialise on sight, within a start/end window

There is no scheduler. A recurring expense carries a start month and an end month (empty = ongoing), and any view or report covering a month first creates that month's missing expenses, then reads. Generation is idempotent and never runs past the current month.

A cron or timer was rejected because the Pi may be powered off when it would fire. The window, not a boolean flag, is what makes past periods reproducible: a report over a month inside the window always generates the same rows, whenever it is run. Deactivating sets the end month; reactivating creates a *new* recurring rather than backfilling the gap, since the gap months genuinely had no payment. Deleting a generated Expense records a skip for that month so it does not reappear.
