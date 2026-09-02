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
export type MonthTotals = {
  month: string
  income_cents: number
  expense_cents: number
  net_cents: number
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
export type RecentEntry = {
  direction: "expense" | "income"
  id: number
  date: string
  amount_cents: number
  category: string
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
