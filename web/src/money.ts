// Money is integer cents everywhere, including here. A euro amount is read as
// its two groups of digits and combined with integer arithmetic, never by
// multiplying the typed decimal by 100: 7.99 * 100 is 798.9999999999999 in
// floating point, and one rounding is all it takes for a month's total to stop
// matching the entries under it.

// toCents reads what was typed at the till — "7,99", "7.9", "12" — and
// returns whole cents, or null when it is not an amount. A comma is what an
// Italian keypad offers, so it is accepted as the decimal separator.
export function toCents(typed: string): number | null {
  const m = typed
    .trim()
    .replace(",", ".")
    .match(/^(\d*)(?:\.(\d{0,2}))?$/)
  if (!m || (!m[1] && !m[2])) return null
  return Number(m[1] || "0") * 100 + Number((m[2] ?? "").padEnd(2, "0"))
}

// formatCents renders whole cents Italian-style — 123456 becomes "1.234,56".
// Still no float: the euros and the cents are two integers, the euros grouped
// by Intl and the cents padded, with a comma between them.
// useGrouping "always" because Italian's default groups only from five digits
// up, which would print a €1.234,56 shopping month as 1234,56.
const euroGroups = new Intl.NumberFormat("it-IT", { useGrouping: "always" })

export function formatCents(cents: number): string {
  // The sign is taken off first and put back by hand. Left on, it would be
  // formatted into both halves and print -65,43 as "-65,-43" — the euros and
  // the cents are two integers here, and only one of them wants a sign.
  const abs = Math.abs(cents)
  return `${cents < 0 ? "−" : ""}${euroGroups.format(Math.trunc(abs / 100))},${String(abs % 100).padStart(2, "0")}`
}

// toTyped is toCents backwards: whole cents as the form's own input would
// have them typed, for an Expense being edited. Ungrouped, unlike
// formatCents — a thousands separator is not something toCents accepts back.
export function toTyped(cents: number): string {
  return `${Math.trunc(cents / 100)},${String(cents % 100).padStart(2, "0")}`
}

// formatDate turns the API's YYYY-MM-DD into the 01/09/2026 the household
// reads. The parts are swapped rather than parsed into a Date, which would
// drag a timezone into a value that has none.
export function formatDate(occurredOn: string): string {
  const [y, m, d] = occurredOn.split("-")
  return `${d}/${m}/${y}`
}

// today is read from the browser rather than from the server: the phone in the
// hand at the till has a correct clock, and the Pi has no RTC. Local date
// parts, not toISOString, which would hand back yesterday for most of an
// Italian evening — and last month for the first hours of a new one.
export function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// thisMonth is the month today falls in, read from the browser for the same
// reason today() is.
export function thisMonth(): string {
  return today().slice(0, 7)
}

// shiftMonth walks a YYYY-MM by whole months, in either direction. Date does
// the carrying: month -1 of January 2026 is December 2025, without this having
// to know it.
export function shiftMonth(month: string, by: number): string {
  const [y, m] = month.split("-").map(Number)
  const d = new Date(y, m - 1 + by, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`
}
