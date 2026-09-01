# Sonic Finance

A tracker for the money moving through a single household — what came in, what went out, and what it went on. It serves exactly one household: there is nothing to select, switch, or join.

## Language

**Household**:
The people sharing the space whose money this app tracks. There is only ever one, so it is never named or chosen.
_Avoid_: family, tenant, account, group

**Expense**:
Money leaving the household, on the date it left.
_Avoid_: spesa, cost, outgoing, purchase, payment, transaction

**Income**:
Money owed to or received by the household. An Income exists from the moment it is expected — an invoice sent, a payment promised — and becomes received when it has a payment date. Totals only ever count received Incomes.
_Avoid_: earning, revenue, incoming, deposit, transaction, invoice

**Client**:
Who an Income comes from — a freelance customer, an employer, or the relative who sent a gift. Named once and referenced, so the household can ask whether a particular one has paid.
_Avoid_: customer, source, payer, counterparty

**Item**:
Part of an Expense that belongs under a different Category than the Expense as a whole — a book bought during the grocery shop. Items are exceptions, always optional, and never have to account for the whole Expense: whatever they don't cover stays under the Expense's own Category.
_Avoid_: line, line item, detail, product, row

**Recurring expense**:
A template that produces one ordinary Expense per calendar month. Once produced, the Expense is no different from a typed one. Incomes do not recur — this household's income varies month to month.
_Avoid_: subscription, schedule, standing order, repeat

**Category**:
A named bucket money is counted under when the household looks at where it went. Every Expense and Income has one; an Item may override an Expense's for part of the amount. A Category applies to Expenses, to Incomes, or to both — "Food" is never an Income and "Freelance" is never an Expense. Categories are a flat list. On Income screens the same concept is worded "income reason"; it is not a second field.
_Avoid_: tag, label, type, kind, type of payment

**Base category**:
A Category the app itself depends on and therefore protects from renaming and deletion — "Freelance" and "Taxes" are what the yearly tax summary is computed from. They can be hidden from the pickers once you stop using them, but never renamed or removed. Only categories the code actually resolves are protected; every other Category is yours to add, rename, and archive freely.
_Avoid_: system category, built-in, default, reserved

**Pending payment**:
Money the household is waiting for: an Income with no payment date, or a freelance Client who is usually billed by now and has not been. It is a view over Incomes, never a record of its own.
_Avoid_: receivable, debtor, outstanding invoice, arrears

**Tax year**:
The year a tax payment relates to, which is usually not the year it was paid — tax on 2026's income is paid during 2027. Recorded on tax Expenses so the yearly summary can attribute them to the income that caused them.
_Avoid_: fiscal year, accounting period

**Store**:
Where an Expense happened, remembered as free text rather than chosen from a fixed list.
_Avoid_: shop, merchant, vendor, place

**Payer**:
Which household member's money it is — whose account an Expense left, or whose pocket an Income landed in. Chosen from one short configured list shared by both, including options like "Both" and "Someone else". It is a label for recall: the app never computes who owes whom from it.
_Avoid_: person, user, owner, account holder, recipient, earner

**Payment method**:
How an Expense was paid — cash, credit card, debit card. Recorded only so the household can recognise the entry later. Applies to Expenses alone, and is unrelated to why an Income arrived.
_Avoid_: tender, payment type, type of payment, source
