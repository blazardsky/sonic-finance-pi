import { useEffect, useState } from "react"

import { Row } from "@/pages/Month"
import { Button } from "@/components/ui/button"
import { apiJSON } from "@/lib/api"
import { formatCents, thisYear } from "@/lib/money"
import { t } from "@/lib/strings"
import type { TaxSummary, YearTotals } from "@/types"

// The year screen: the shape of a whole year, and the year in tax terms. Two
// questions on one screen because they are asked together — how did the year
// go, and what did it cost in tax — and two endpoints behind it, because the
// second one is not a sum of the first: tax paid is attributed by the year it
// relates to, not the year the money left (ADR-0008).
//
// Which year it is is the browser's answer, like which month is on the month
// screen: the phone has a correct clock and the Pi has no RTC. Stepping
// forward stops at the current year, because a year that has not happened has
// no totals — its months would be twelve rows of zeros.
export function Year() {
  const [year, setYear] = useState(thisYear)
  const [totals, setTotals] = useState<YearTotals | null>(null)
  const [tax, setTax] = useState<TaxSummary | null>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    let current = true
    Promise.all([
      apiJSON<YearTotals>(`/api/reports/year/${year}`),
      apiJSON<TaxSummary>(`/api/reports/tax/${year}`),
    ])
      // A slow year arriving after the household has already tapped on to the
      // next one must not overwrite it.
      .then(([y, s]) => {
        if (!current) return
        setTotals(y)
        setTax(s)
        setError("")
      })
      .catch(() => {
        if (!current) return
        // The numbers go with the year: last year's totals left sitting under
        // this year's heading would be a wrong answer rather than a missing
        // one.
        setTotals(null)
        setTax(null)
        setError(t.serverUnreachable)
      })
    return () => {
      current = false
    }
  }, [year])

  // What the bars are drawn against: the biggest single month, in either
  // direction, so the tallest bar of the year is full width and every other
  // month is read against it. Zero when the year is empty, which draws
  // nothing rather than dividing by it.
  const peak = Math.max(
    0,
    ...(totals?.months.flatMap((m) => [m.income_cents, m.expense_cents]) ?? [])
  )

  const step = (by: number) => setYear(String(Number(year) + by))

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <div className="flex items-center justify-between gap-2">
        <Button
          variant="ghost"
          size="sm"
          aria-label={t.previousYear}
          onClick={() => step(-1)}
        >
          ‹
        </Button>
        {/* The year itself, as a heading rather than an input: there is no
            native year picker, and a household comparing against 2019 taps
            seven times rather than learning a control. */}
        <h1 className="text-2xl font-medium tabular-nums">{year}</h1>
        <Button
          variant="ghost"
          size="sm"
          aria-label={t.nextYear}
          // String comparison is a date comparison here too: four digits sort
          // correctly, which is why a year is text everywhere but the column.
          disabled={year >= thisYear()}
          onClick={() => step(1)}
        >
          ›
        </Button>
      </div>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {totals && (
        <dl className="flex flex-col divide-y divide-border">
          <Row label={t.incomes} cents={totals.income_cents} />
          <Row label={t.expenses} cents={totals.expense_cents} />
          <Row label={t.difference} cents={totals.net_cents} big />
        </dl>
      )}

      {/* The figure the invoicing software cannot give: what was really
          received and what was really paid. Freelance income only — employment
          income arrives already taxed and a gift is not income, so counting
          either would make the percentage meaningless. */}
      {tax && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.taxSummary}
          </h2>
          {tax.received_cents === 0 && tax.tax_paid_cents === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t.nothingReceived(year)}
            </p>
          ) : (
            <>
              <dl className="flex flex-col divide-y divide-border">
                <Row label={t.received} cents={tax.received_cents} />
                <Row label={t.taxPaid} cents={tax.tax_paid_cents} />
                <Row label={t.net} cents={tax.net_cents} big />
              </dl>
              {/* The percentage the freelancer actually keeps. Absent when
                  nothing was received: a percentage of nothing is not 0%, and
                  the server sends null rather than let the screen divide. */}
              {tax.net_percent !== null && (
                <p className="text-sm">
                  {t.netPercent(tax.net_percent.toFixed(1).replace(".", ","))}
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                {t.taxSummaryHint}
              </p>
            </>
          )}
        </section>
      )}

      {/* The shape of the year: twelve months, each with what came in and what
          went out, and a pair of bars scaled to the biggest month so the shape
          is visible without reading twenty-four numbers. */}
      {totals && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.byMonth}
          </h2>
          <ul className="flex flex-col divide-y divide-border">
            {totals.months.map((m, i) => (
              <li key={m.month} className="flex flex-col gap-1 py-2">
                <div className="flex items-baseline justify-between gap-2 text-xs">
                  <span className="text-muted-foreground">
                    {t.monthsShort[i]}
                  </span>
                  <span className="tabular-nums">
                    + {formatCents(m.income_cents)} · −{" "}
                    {formatCents(m.expense_cents)}
                  </span>
                </div>
                <Bar
                  cents={m.income_cents}
                  peak={peak}
                  className="bg-primary"
                />
                <Bar
                  cents={m.expense_cents}
                  peak={peak}
                  className="bg-destructive"
                />
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}

// One month's bar, as a share of the biggest month of the year. A month with
// nothing in it draws a hairline rather than nothing at all, so the twelve
// rows stay the same height and the year keeps its shape.
function Bar({
  cents,
  peak,
  className,
}: {
  cents: number
  peak: number
  className: string
}) {
  return (
    <div
      className={`h-1 rounded-full ${className}`}
      style={{
        width: peak > 0 ? `${Math.max(1, (cents / peak) * 100)}%` : "1%",
      }}
    />
  )
}
