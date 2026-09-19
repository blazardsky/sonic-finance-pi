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
  // Only reachable from a bug — a response missing the field the caller read,
  // most often a server that predates it. Named in the console rather than
  // swallowed into a 0, which would read as a real amount of nothing; the
  // "NaN,NaN" below is left visible on purpose as the matching signal.
  if (!Number.isFinite(cents)) {
    console.error("formatCents: not a finite number:", cents)
  }
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

// toCentsExpr is toCents, but first checks for a simple two-operand
// expression — "12*3", "10-2", "5+5" — so an item price at the till can be
// typed as the math that produced it rather than done by hand first. Each
// operand is read with toCents (so it stays whole cents, not a float), and
// + / - combine those cents directly, exact by construction. * doesn't stay
// exact in general (quantity × unit price, e.g. "1,5*3,33"), so it rounds to
// the nearest cent rather than carry a fractional one nothing else expects.
export function toCentsExpr(typed: string): number | null {
  const m = typed
    .trim()
    .match(/^(\d+(?:[.,]\d{1,2})?)\s*([*+-])\s*(\d+(?:[.,]\d{1,2})?)$/)
  if (!m) return toCents(typed)
  const a = toCents(m[1])
  const b = toCents(m[3])
  if (a === null || b === null) return null
  if (m[2] === "+") return a + b
  if (m[2] === "-") return a - b
  return Math.round((a * b) / 100)
}

// toQuantity reads what was typed for an Item's Quantity (ticket 01) — the
// same comma an Italian keypad offers, accepted the way toCents accepts it,
// but with no cents-style scaling or digit cap: "0,5" is half a kilo. ""
// (and anything not a positive number) is "no quantity", which is a valid
// answer since the field is optional — that is why this returns null rather
// than refusing, the way toCents does for an amount that must exist.
export function toQuantity(typed: string): number | null {
  const n = Number(typed.trim().replace(",", "."))
  return typed.trim() !== "" && Number.isFinite(n) && n > 0 ? n : null
}

// quantityTyped is toQuantity backwards, for an Item being edited: null (no
// quantity) becomes "", and a number becomes what the field would have been
// typed as, comma decimal.
export function quantityTyped(quantity: number | null): string {
  return quantity == null ? "" : String(quantity).replace(".", ",")
}

// formatPricePerUnit renders an Item's server-computed price_per_unit — cents
// per unit, ADR-0014 — as euros, comma decimal: the one place amount_cents
// and Quantity meet is the server, so this only ever formats what it already
// divided, never re-derives it from a draft still mid-typing.
export function formatPricePerUnit(pricePerUnitCents: number): string {
  return (pricePerUnitCents / 100).toFixed(2).replace(".", ",")
}

// formatDate turns the API's YYYY-MM-DD into the 01/09/2026 the household
// reads. The parts are swapped rather than parsed into a Date, which would
// drag a timezone into a value that has none.
export function formatDate(occurredOn: string): string {
  const [y, m, d] = occurredOn.split("-")
  return `${d}/${m}/${y}`
}

// formatMonth turns the API's YYYY-MM into the 03/2026 the household reads.
// Swapped rather than parsed into a Date, for the same reason formatDate is.
export function formatMonth(month: string): string {
  return month.split("-").reverse().join("/")
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

// thisYear is the year today falls in, read from the browser for the same
// reason today() is.
export function thisYear(): string {
  return today().slice(0, 4)
}

// shiftMonth walks a YYYY-MM by whole months, in either direction. Date does
// the carrying: month -1 of January 2026 is December 2025, without this having
// to know it.
export function shiftMonth(month: string, by: number): string {
  const [y, m] = month.split("-").map(Number)
  const d = new Date(y, m - 1 + by, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`
}
