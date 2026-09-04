// The shapes the API hands out, shared by the screens that read them.

export type Applies = "expense" | "income" | "both"

export type Category = {
  id: number
  name: string
  applies_to: Applies
  hidden: boolean
  // A Base category is one the tax summary resolves by identity: it can be
  // hidden, never renamed or deleted, and the server enforces that with a 409.
  base: boolean
}

// Who money comes from, as one row rather than three spellings. Not a
// freelance-only idea: "Mum" for a birthday gift is a valid Client. Hidden
// Clients stay out of the Income picker but still resolve for old Incomes.
export type Client = {
  id: number
  name: string
  hidden: boolean
}

export type HoldingType = "etf" | "crypto" | "stock" | "bond" | "other"

// A specific stock, ETF, crypto asset, bond or other investment vehicle the
// household buys and sells — never priced or revalued by the app. Named and
// typed from this fixed list rather than free text, because the portfolio
// percentage breakdown (ticket 03) groups by it exactly (CONTEXT.md).
export type Holding = {
  id: number
  name: string
  type: HoldingType
}

export type Expense = {
  id: number
  occurred_on: string
  amount_cents: number
  category_id: number
  // The four details, always strings and never null: they go straight into
  // inputs. Payer and payment method are the label text the Expense was saved
  // with, not a reference — renaming the list leaves them as they read.
  store: string
  payer: string
  payment_method: string
  note: string
  // The optional partial breakdown: the parts of this Expense that belong
  // under a different Category. Always an array, empty for the €7 coffee.
  // Never the source of amount_cents — see ADR-0002.
  items: Item[]
  // The year this payment's tax relates to, and 0 on every Expense that is not
  // a tax one: tax on 2026's income is paid during 2027, so the yearly summary
  // attributes it by this rather than by occurred_on (ADR-0008). The server
  // derives it — 0 sent for a tax Expense comes back as the payment's year, and
  // anything sent for any other Expense comes back as 0.
  tax_year: number
}

// A part of an Expense under its own Category. No id: an Item is saved as one
// of a set with its Expense and addressed by nothing, so what the form sends
// back is the whole breakdown.
export type Item = {
  name: string
  amount_cents: number
  category_id: number
}

// The two short configured lists the Expense form picks from.
export type Lists = {
  payers: string[]
  payment_methods: string[]
}

// Money owed to or received by the household. It exists from the moment it is
// expected — an invoice sent — and is received only once it has a payment
// date: an empty payment_date is the unpaid state, and unpaid Income counts
// toward no total anywhere (ADR-0003).
//
// One number, the amount that arrived. No gross/net pair and no tax
// percentage: tax is an Expense on the day it is paid (ADR-0004).
export type Income = {
  id: number
  amount_cents: number
  // The "income reason" the screens word differently — the same Category
  // field an Expense carries, referenced so a rename fixes every past Income.
  category_id: number
  // Who it came from, or null: a reimbursement names nobody.
  client_id: number | null
  // The label text the Income was saved with, not a reference — the same
  // bargain an Expense makes.
  payer: string
  // Both dates are "" when absent rather than null: they go straight into
  // date inputs, and "" is how one is cleared on the way back.
  payment_date: string
  invoice_sent_date: string
  note: string
}

// Where a calendar month stands: what came in, what went out, and the
// difference. Money in counts received Incomes only — an unpaid one is money
// that has not arrived and is in none of these numbers (ADR-0003). net_cents
// is negative in a month that spent more than it received.
//
// A year is twelve of these and nothing more — the shape of the year is drawn
// from the three numbers, and no Category breakdown comes with them.
export type MonthRow = {
  month: string
  income_cents: number
  expense_cents: number
  net_cents: number
}

// A month with what the money went on: what the month screen reads.
export type MonthTotals = MonthRow & {
  // Where the money went, biggest share first, always summing to
  // expense_cents. An Item's amount is under the Item's Category and the
  // remainder under the Expense's own, so nothing is ever "uncategorised"
  // (ADR-0002).
  by_category: CategoryTotal[]
}

// One Category's share of a month's spend. The name comes with the id, so the
// month screen has no reason to hold the Category list — including for a
// Category since hidden, which still has to say what it was called.
export type CategoryTotal = {
  category_id: number
  category: string
  amount_cents: number
}

// The shape of a whole year: the same three numbers a month answers with, over
// twelve months, plus the twelve months themselves, plus extra_income_cents —
// received Income that is not work. months is always twelve long, January
// first, whether or not anything happened in any of them — a year view is a
// shape, and a missing month would be a gap in it.
export type YearTotals = {
  year: string
  income_cents: number
  expense_cents: number
  net_cents: number
  // Received Income that is not work (neither Freelance nor Stipendio).
  extra_income_cents: number
  months: MonthRow[]
}

// The year in tax terms: the actual figures, as against the invoicing
// software's forecasts. received_cents is freelance Income only and counts
// received money alone (ADR-0003); tax_paid_cents is the tax Category
// attributed by Tax year rather than by payment date, because tax on one
// year's income is paid during the next (ADR-0008). One total, no per-tax-type
// breakdown.
//
// net_percent is null rather than 0 when nothing was received: a percentage of
// nothing is not zero percent, and the screen has to say nothing instead.
export type TaxSummary = {
  year: string
  received_cents: number
  tax_paid_cents: number
  net_cents: number
  net_percent: number | null
}

// One thing the household typed, in either direction, as the home screen's
// list reads it. Not month-scoped and ordered by when it was typed rather than
// by the date on it: the list exists to confirm an entry landed, including one
// backdated to a month the screen is not showing.
//
// date is the day the money moved — occurred_on for an Expense, the payment
// date for an Income — and "" for an Income that has not been paid, because
// money that has not arrived has no day it arrived on. An empty date is
// therefore the unpaid state, and the screen must show it as one: this list
// sits under a money-in total that excludes it (ADR-0003).
//
// payer is whose money an Expense left ("" on an Income). client is who an
// Income came from ("" on an Expense, and on an Income that names nobody).
export type RecentEntry = {
  direction: "expense" | "income"
  id: number
  date: string
  amount_cents: number
  category: string
  payer: string
  client: string
}

// The rent, defined once. It carries an Expense's template fields — every one
// of them is copied onto the Expenses it produces — plus the day of the month
// it lands on and the window it covers.
//
// There is no active flag: end_month is the whole of it (ADR-0005). Empty
// means still running, and a month is covered when start_month <= it <=
// end_month. Both are YYYY-MM and zero-padded, so that is a string comparison.
//
// Changing an amount is not an edit: the old definition is ended and a new one
// started, so the months before the change keep the amount that was true then.
export type Recurring = {
  id: number
  amount_cents: number
  category_id: number
  store: string
  payer: string
  payment_method: string
  note: string
  day_of_month: number
  start_month: string
  end_month: string
}

// The two lists a freelancer used to get from reading a spreadsheet: money
// genuinely owed, and invoices that look forgotten. Both are views over
// Incomes and never records of their own — the word is "pending payment",
// never "receivable".
export type Pending = {
  outstanding: OutstandingIncome[]
  not_yet_invoiced: NotYetInvoicedClient[]
}

// One Income still waiting for its money, oldest first. days_waiting counts
// calendar days from waiting_since — the invoice date, or the day the Income
// was typed when no invoice was sent — and never counts backwards.
//
// client and category are names rather than ids: this list is read on a screen
// that holds neither list, and a Client since hidden still has to say what it
// is called. client is "" for an Income that names nobody.
export type OutstandingIncome = {
  id: number
  amount_cents: number
  client: string
  category: string
  waiting_since: string
  days_waiting: number
  // "" when no invoice was sent — waiting_since then falls back to the day
  // the Income was typed. The Dashboard's "in attesa" list skips those.
  invoice_sent_date: string
}

// One Client billed in the previous three calendar months with no freelance
// Income recorded this month. Freelance Income on both sides, so a gift from a
// relative neither starts a nudge nor clears one.
//
// The id is what the list keys on: two Clients can share a name.
export type NotYetInvoicedClient = {
  client_id: number
  client: string
}
