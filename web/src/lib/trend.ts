// The plain date math behind both trend charts: turning a flat list of daily,
// per-Category rows into the buckets a chart draws — one bucket per day for
// the Dashboard's rolling window, one per Monday-start week for the Month
// page. Not a library: it is a handful of date offsets, and this is the only
// place in the app that cares about weeks.

import type { DailyCategoryTotal } from "@/types"

// One column a trend chart draws: a label the axis shows, and each Category's
// cents under it, keyed by Category id — never by name, so a rename does not
// split one Category's bar into two.
export type TrendBucket = {
  key: string
  label: string
  amounts: Record<number, number>
}

// addDays walks a YYYY-MM-DD by whole days, in either direction. Date does
// the carrying across months and years, the same bargain shiftMonth makes for
// whole months.
export function addDays(date: string, days: number): string {
  const d = new Date(date + "T00:00:00")
  d.setDate(d.getDate() + days)
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// shortDate is a chart axis's spelling of a day: no year, because every trend
// chart here spans at most a few months.
export function shortDate(date: string): string {
  const [, m, d] = date.split("-")
  return `${d}/${m}`
}

// mondayOf is the Monday on or before date, as YYYY-MM-DD — Monday-start per
// the ticket. getDay() counts Sunday as 0, so it is rotated to count Monday
// as 0 before subtracting.
function mondayOf(date: string): string {
  const d = new Date(date + "T00:00:00")
  const sinceMonday = (d.getDay() + 6) % 7
  return addDays(date, -sinceMonday)
}

// categoriesIn is every Category a daily report mentions, biggest total
// first — the order both the legend and the by_category list already use.
// The name and colour travel with the id straight off the rows themselves,
// so a trend chart never has to hold the whole Category list just to label
// or colour its legend.
export function categoriesIn(
  daily: DailyCategoryTotal[]
): { id: number; name: string; color: string }[] {
  const totals = new Map<number, { name: string; color: string; total: number }>()
  for (const row of daily) {
    const c = totals.get(row.category_id) ?? {
      name: row.category,
      color: row.category_color,
      total: 0,
    }
    c.total += row.amount_cents
    totals.set(row.category_id, c)
  }
  return [...totals.entries()]
    .sort(([, a], [, b]) => b.total - a.total)
    .map(([id, c]) => ({ id, name: c.name, color: c.color }))
}

// dailyBuckets is one bucket per day in days, in that order, whether or not
// the API returned anything for it — a quiet day is a real gap in the trend,
// not a day to skip drawing.
export function dailyBuckets(
  daily: DailyCategoryTotal[],
  days: string[]
): TrendBucket[] {
  const byDay = new Map<string, Record<number, number>>()
  for (const row of daily) {
    const amounts = byDay.get(row.day) ?? {}
    amounts[row.category_id] = row.amount_cents
    byDay.set(row.day, amounts)
  }
  return days.map((day) => ({
    key: day,
    label: shortDate(day),
    amounts: byDay.get(day) ?? {},
  }))
}

// weeklyBuckets groups a month's daily rows into Monday-start weeks. A week
// is labelled with its real date range — Monday to the Sunday six days later
// — even when that range runs past either end of the month: the ticket asks
// for the edge week shown as what it really is, not clipped or merged into
// its neighbour. Only weeks the month actually has data for appear, the same
// "nothing to show" rule the daily rows already apply day by day.
export function weeklyBuckets(daily: DailyCategoryTotal[]): TrendBucket[] {
  const byWeek = new Map<string, Record<number, number>>()
  for (const row of daily) {
    const monday = mondayOf(row.day)
    const amounts = byWeek.get(monday) ?? {}
    amounts[row.category_id] =
      (amounts[row.category_id] ?? 0) + row.amount_cents
    byWeek.set(monday, amounts)
  }
  return [...byWeek.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([monday, amounts]) => ({
      key: monday,
      label: `${shortDate(monday)}–${shortDate(addDays(monday, 6))}`,
      amounts,
    }))
}

