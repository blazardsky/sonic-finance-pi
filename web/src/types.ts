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

export type Expense = {
  id: number
  occurred_on: string
  amount_cents: number
  category_id: number
}
