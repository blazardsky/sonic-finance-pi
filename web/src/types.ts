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
