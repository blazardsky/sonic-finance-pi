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
Part of an Expense that belongs under a different Category than the Expense as a whole — a book bought during the grocery shop. Items are exceptions, always optional, and never have to account for the whole Expense: whatever they don't cover stays under the Expense's own Category. An Item may also record a quantity and a kg/lt/piece unit, and whether it was discounted — a passing markdown rather than its usual price. The Tracker derives a price per unit from these (amount ÷ quantity); it is never stored, so it can never drift from what was actually paid.
_Avoid_: line, line item, detail, product, row

**Recurring expense**:
A template that produces one ordinary Expense per calendar month. Once produced, the Expense is no different from a typed one. Incomes do not recur — this household's income varies month to month.
_Avoid_: subscription, schedule, standing order, repeat

**Category**:
A named bucket money is counted under when the household looks at where it went. Every Expense and Income has one; an Item may override an Expense's for part of the amount. A Category applies to Expenses, to Incomes, or to both — "Food" is never an Income and "Freelance" is never an Expense. Categories are a flat list. On Income screens the same concept is worded "income reason"; it is not a second field.
_Avoid_: tag, label, type, kind, type of payment

**Base category**:
A Category the app itself depends on and therefore protects from renaming and deletion — "Freelance" and "Taxes" are what the yearly tax summary is computed from, "Investments" is what Budget, Target, and the yearly Estimate exclude, and "Gift" is what the spoiler blur resolves. They can be hidden from the pickers once you stop using them, but never renamed or removed. Only categories the code actually resolves are protected; every other Category is yours to add, rename, and archive freely.
_Avoid_: system category, built-in, default, reserved

**Gift**:
A Category, applying to both Expenses and Incomes, for money spent buying something for someone else or money received as one. An Expense under it is blurred by default wherever it appears in a list or grid, revealed by hovering or tapping it, so the amount isn't visible at a glance.
_Avoid_: present

**Holding**:
A specific stock, ETF, crypto asset, bond, or other investment vehicle the household buys and sells, named and typed from a short fixed list rather than free text — unlike Store, it must match exactly, because the portfolio percentage breakdown groups by it. Money moving in or out of one is recorded as an ordinary Expense or Income under the Investments base category; a Holding is never priced or revalued by the app.
_Avoid_: asset, position, security, ticker, investment

**PAC**:
A Recurring expense whose Category is Investments and which also names a Holding — a fixed amount moved into the same investment on the same schedule every month. Both are required: naming Investments without a Holding, or a Holding without Investments, is not a PAC. It does not track how many units or shares that money bought, consistent with a Holding never being priced or revalued — a PAC is a cash-flow habit, not a position size.
_Avoid_: recurring investment, DCA, dollar-cost averaging, subscription

**Tracker**:
A view over Items across every Expense, grouping by Store and Category to show how much the same Item has cost over time and where it was cheaper. Never a record of its own — always computed from ordinary Expenses and Items, the same way Pending payment is a view over Incomes.
_Avoid_: price tracker, history, analytics

**Pending payment**:
Money the household is waiting for: an Income with no payment date, or a freelance Client who is usually billed by now and has not been. It is a view over Incomes, never a record of its own.
_Avoid_: receivable, debtor, outstanding invoice, arrears

**Tax year**:
The year a tax payment relates to, which is usually not the year it was paid — tax on 2026's income is paid during 2027. Recorded on tax Expenses so the yearly summary can attribute them to the income that caused them.
_Avoid_: fiscal year, accounting period

**Store**:
Where an Expense happened, remembered as free text rather than chosen from a fixed list. The Tracker normalizes it (trimmed, lowercased) only when grouping for its own view — the stored value is untouched, and two spellings of the same shop can still end up as separate rows if they differ enough.
_Avoid_: shop, merchant, vendor, place

**Payer**:
Which household member's money it is — whose account an Expense left, or whose pocket an Income landed in. Chosen from one short configured list shared by both, including options like "Both" and "Someone else". It is a label for recall: the app never computes who owes whom from it.
_Avoid_: person, user, owner, account holder, recipient, earner

**Payment method**:
How an Expense was paid — cash, credit card, debit card. Recorded only so the household can recognise the entry later. Applies to Expenses alone, and is unrelated to why an Income arrived.
_Avoid_: tender, payment type, type of payment, source

**Savings**:
The household's uninvested surplus: cumulative Income minus Expense, excluding the Investments category, since the household started using the app, plus a one-time starting balance for money saved before then. Always computed, never stored as a record of its own.
_Avoid_: balance, savings account, cash

**Budget**:
The trailing 12 completed months' median monthly Expense, excluding Investments — what the household typically spends. Always computed; never edited by hand. Distinct from the median Expense, Income, and Net the yearly report shows, which are scoped to that report's own calendar year rather than a rolling 12 months.
_Avoid_: average, spending limit

**Target**:
The Expense ceiling the household is aiming to stay under this month. Starts equal to Budget and stays there until the household deliberately changes it.
_Avoid_: budget, limit, cap

**Goal**:
How much the household wants to save each month, chosen by hand rather than computed. Distinct from Budget and Target, which are about Expenses, not savings.
_Avoid_: target, budget

**Net worth target**:
The total the household wants Savings plus its portfolio to reach, chosen by hand. Unlike Goal it names a destination rather than a monthly rate, and the years between here and it are projected from the median month's saving over the trailing year — or from Goal, until there is enough history to median. A projection, never a promise, on the same terms as Estimate.
_Avoid_: goal, savings goal, milestone

**Estimate**:
A projected figure for Income, Expenses, or tax for the rest of the year — this year's year-to-date growth against last year's at the same point (by whole completed months), applied to last year's full-year totals. The only number in the app that is not a cash-basis fact, and always labeled as a projection rather than shown alongside actuals unmarked.
_Avoid_: projection, forecast, prediction

**Contract**:
An amount the household expects to receive from a Client between two given months, used to compute how much should have arrived by now and how much is left to invoice. An Income counts toward a Contract only if it references one; an Income from that Client with no Contract is Extra — real money from them outside the agreed total.
_Avoid_: retainer, agreement, invoice schedule

**Reminder**:
A short household-labeled toggle for a manual action done outside the app — a bank transfer, a payment made by hand — and not tied to any Expense or Recurring expense. Turned on once it's done; reads as off again from the first of the next calendar month.
_Avoid_: alert, notification, task
