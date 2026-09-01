# Income exists before it is paid; no separate Invoice entity

An Income is created when the money is expected — an invoice sent — and becomes received when it gets a payment date. Unpaid Incomes are rows with an empty payment date. There is no Invoice entity; a separate Invoice/Income pair would double the model and force a join on every screen.

## Consequences

Every total, report, and chart **must** filter on payment-date-not-empty. Omitting that filter in one place counts money that has not arrived — the single most likely bug in this codebase.
