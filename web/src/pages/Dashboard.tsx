import { useCallback, useEffect, useState } from "react"

import { Row } from "@/pages/Month"
import { PendingPayments } from "@/pages/PendingPayments"
import { running } from "@/pages/RecurringExpenses"
import { apiJSON } from "@/lib/api"
import { formatCents, formatDate, thisYear } from "@/lib/money"
import { nameOf } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type {
  Category,
  RecentEntry,
  Recurring,
  TaxSummary,
  YearTotals,
} from "@/types"

// The next date a Recurring lands on, from today. Mirrors the clamp
// cmd/recurring.go's materialise() applies when it actually generates the
// Expense — day_of_month or the month's last day, whichever is smaller — so
// this preview never promises a date generation itself would not produce.
function nextOccurrence(r: Recurring): string {
  const now = new Date()
  const clamp = (year: number, month0: number) =>
    Math.min(r.day_of_month, new Date(year, month0 + 1, 0).getDate())

  let year = now.getFullYear()
  let month0 = now.getMonth()
  let day = clamp(year, month0)
  if (day < now.getDate()) {
    month0 += 1
    if (month0 > 11) {
      month0 = 0
      year += 1
    }
    day = clamp(year, month0)
  }

  const pad = (n: number) => String(n).padStart(2, "0")
  return `${year}-${pad(month0 + 1)}-${pad(day)}`
}

const daysUntil = (date: string) => {
  const ms = new Date(date + "T00:00:00").getTime() - new Date().setHours(0, 0, 0, 0)
  return Math.round(ms / 86_400_000)
}

// A simple run-rate projection — YTD ÷ months elapsed × 12 — not the richer
// same-pace-as-last-year comparison tracked in TODO.md for later.
const projectToYear = (cents: number) =>
  Math.round((cents / (new Date().getMonth() + 1)) * 12)

// The home screen: where the year stands, what is coming due, and the last
// few things typed. Every number here is either a totals report the app
// already computes (Year.tsx, Month.tsx read the same endpoints) or a plain
// client-side projection of one — nothing is invented that the server does
// not already know.
export function Dashboard() {
  const [recent, setRecent] = useState<RecentEntry[]>([])
  const [recurring, setRecurring] = useState<Recurring[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [year, setYear] = useState<YearTotals | null>(null)
  const [tax, setTax] = useState<TaxSummary | null>(null)
  const [error, setError] = useState("")

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<RecentEntry[]>("/api/reports/recent"),
        apiJSON<Recurring[]>("/api/recurring"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<YearTotals>(`/api/reports/year/${thisYear()}`),
        apiJSON<TaxSummary>(`/api/reports/tax/${thisYear()}`),
      ])
        .then(([r, rec, cat, y, tx]) => {
          setRecent(r)
          setRecurring(rec)
          setCategories(cat)
          setYear(y)
          setTax(tx)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  const latestExpenses = recent
    .filter((e) => e.direction === "expense")
    .slice(0, 5)

  const upcoming = recurring
    .filter(running)
    .map((r) => ({ r, date: nextOccurrence(r) }))
    .sort((a, b) => a.date.localeCompare(b.date))
    .slice(0, 5)

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {year && (
        <dl className="flex flex-col divide-y divide-border">
          <Row label={t.yearIncome} cents={year.income_cents} />
          <Row label={t.yearExpenses} cents={year.expense_cents} />
          <Row label={t.difference} cents={year.net_cents} big />
        </dl>
      )}

      {year && tax && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.taxEstimate}
          </h2>
          <dl className="flex flex-col divide-y divide-border">
            <Row
              label={t.projectedIncome}
              cents={projectToYear(year.income_cents)}
            />
            <Row
              label={t.projectedExpenses}
              cents={projectToYear(year.expense_cents)}
            />
            <Row
              label={t.estimatedTax}
              cents={projectToYear(tax.tax_paid_cents)}
            />
          </dl>
          <p className="text-xs text-muted-foreground">{t.projectionHint}</p>
        </section>
      )}

      <PendingPayments />

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium text-muted-foreground">
          {t.upcomingRecurring}
        </h2>
        {upcoming.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t.noUpcomingRecurring}
          </p>
        ) : (
          <ul className="flex flex-col divide-y divide-border">
            {upcoming.map(({ r, date }) => (
              <li
                key={r.id}
                className="flex items-baseline justify-between gap-3 py-2"
              >
                <span className="min-w-0">
                  <span className="truncate">
                    {nameOf(categories, r.category_id)}
                  </span>{" "}
                  <span className="text-xs text-muted-foreground">
                    {t.dueIn(daysUntil(date))}
                  </span>
                </span>
                <span className="whitespace-nowrap tabular-nums">
                  € {formatCents(r.amount_cents)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium text-muted-foreground">
          {t.latestExpenses}
        </h2>
        {latestExpenses.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t.noRecentExpenses}
          </p>
        ) : (
          <ul className="flex flex-col divide-y divide-border">
            {latestExpenses.map((entry) => (
              <li
                key={entry.id}
                className="flex items-baseline justify-between gap-3 py-2"
              >
                <span className="min-w-0">
                  <span className="truncate">{entry.category}</span>{" "}
                  <span className="text-xs text-muted-foreground">
                    {formatDate(entry.date)}
                  </span>
                </span>
                <span className="whitespace-nowrap tabular-nums">
                  − € {formatCents(entry.amount_cents)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
