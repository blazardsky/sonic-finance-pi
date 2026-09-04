# 06: Recurring investments (PAC)

**What to build:** Support for a recurring investment (an accumulation plan) on the existing Recurring Expense machinery, so a monthly stock/ETF purchase can be automated the same way rent or a subscription already is.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] The Recurring Expense form shows a Holding picker only when the Investments category is selected, mirroring the existing conditional Tax year field on the Expense form.
- [x] `materialise` copies a template's `holding_id` into every generated Expense, alongside the fields it already copies (category, store, payer, payment method, note).
- [x] A generated recurring investment purchase is indistinguishable from one typed by hand, and behaves normally under skip/edit/delete.
- [x] Backend tests cover `holding_id` generation using the existing clock-controlled `materialise` test pattern.

## Comments

Implemented as a plain field addition to `recurringExpense`/`materialise` (no new abstraction); frontend gates the Holding picker on `c.base && c.applies_to === "both"`, the same resolution `Savings.tsx` already uses to find Investments without a published `code`.
