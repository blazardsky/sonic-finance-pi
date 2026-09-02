// What the two entry screens share about pickers: how a select offers a list,
// and how an id is turned back into the name it stands for. Both were written
// for Expenses and both are exactly as true of Incomes, so they live here
// rather than in two copies that can drift apart.

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
