// What the two entry screens share about pickers: how a select offers a list,
// and how an id is turned back into the name it stands for. Both were written
// for Expenses and both are exactly as true of Incomes, so they live here
// rather than in two copies that can drift apart.

import type { Applies, Category } from "@/types"

// What a Category picker offers: no hidden Category, and nothing that
// belongs to the other side — "Freelance" is never an Expense, "Alimentari"
// is never an Income — plus whichever one the entry being edited already
// carries, whatever it is.
export function pickableCategories(
  categories: Category[],
  side: Exclude<Applies, "both">,
  chosen: Category | undefined
): Category[] {
  return withSaved(
    categories.filter((c) => !c.hidden && c.applies_to !== opposite(side)),
    chosen
  )
}

function opposite(side: Exclude<Applies, "both">): Applies {
  return side === "expense" ? "income" : "expense"
}

// A picker keeps whatever the entry being edited was saved with, even when the
// list no longer offers it: a hidden Category, a Client since hidden, and a
// Payer renamed out of the list are all still what that entry says, and a
// select with no matching option would quietly rewrite it on the first
// correction.
export function withSaved<T>(offered: T[], saved: T | undefined): T[] {
  return saved !== undefined && !offered.includes(saved)
    ? [...offered, saved]
    : offered
}

// A name for an id an entry carries. Every row is looked up, hidden ones
// included: hiding takes a Category or a Client out of the picker, not out of
// the entries already filed under it, and an old entry showing a blank label
// is the bug that would make hiding unsafe. An id of null is an entry that
// names nobody — an Income without a Client — and has no name to show.
export function nameOf(
  rows: { id: number; name: string }[],
  id: number | null | ""
): string {
  return id === null || id === ""
    ? ""
    : (rows.find((r) => r.id === id)?.name ?? "")
}

// A Category's colour for an id an entry carries — same lookup as nameOf,
// for the random per-Category colour (assigned once, at creation) rather
// than the old --chart-N hue-by-id computation.
export function colorOf(
  rows: { id: number; color: string }[],
  id: number | null | ""
): string {
  return id === null || id === ""
    ? ""
    : (rows.find((r) => r.id === id)?.color ?? "")
}

// Whether an Expense's Category is the Gift one — what a list of full Expense
// rows (which only carry category_id) needs to know to spoiler an amount.
// Resolved against the loaded Category list's own `gift` flag, never by name.
export function isGiftCategory(categories: Category[], id: number): boolean {
  return categories.find((c) => c.id === id)?.gift ?? false
}
