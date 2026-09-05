import { useEffect, useState } from "react"
import { Bar, BarChart, XAxis } from "recharts"

import { Row } from "@/pages/Month"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { apiJSON } from "@/lib/api"
import { formatCents, thisYear } from "@/lib/money"
import { t } from "@/lib/strings"
import type { TaxSummary, YearTotals } from "@/types"

// Income/expense keep the app's own red-for-money-out convention (the same
// --destructive a negative Row already renders in) rather than two of the
// --chart-N blues, which would read as the same colour twice; income still
// takes one of the shared chart tokens, same mechanism Dashboard/Month use.
const chartConfig = {
  income: { label: t.incomes, color: "var(--chart-2)" },
  expense: { label: t.expenses, color: "var(--destructive)" },
} satisfies ChartConfig

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

  const step = (by: number) => setYear(String(Number(year) + by))

  // What the chart draws: one row per month, income and expense as the two
  // series the shared chart tokens above are keyed to.
  const chartData =
    totals?.months.map((m, i) => ({
      month: t.monthsShort[i],
      income: m.income_cents,
      expense: m.expense_cents,
    })) ?? []

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
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

      {/* Free to sit next to whatever fits beside it — same reflow pattern
          Month.tsx uses, nothing pinned to a fixed column count. */}
      <div className="flex flex-wrap items-start gap-6">
        {totals && (
          <Card className="min-w-64 flex-1">
            <CardContent>
              <dl className="flex flex-col divide-y divide-border">
                <Row label={t.incomes} cents={totals.income_cents} />
                <Row label={t.expenses} cents={totals.expense_cents} />
                <Row label={t.difference} cents={totals.net_cents} big />
              </dl>
            </CardContent>
          </Card>
        )}

        {/* The figure the invoicing software cannot give: what was really
            received and what was really paid. Freelance income only —
            employment income arrives already taxed and a gift is not income,
            so counting either would make the percentage meaningless. */}
        {tax && (
          <Card className="min-w-64 flex-1">
            <CardHeader>
              <CardTitle>{t.taxSummary}</CardTitle>
            </CardHeader>
            <CardContent>
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
                      nothing was received: a percentage of nothing is not 0%,
                      and the server sends null rather than let the screen
                      divide. */}
                  {tax.net_percent !== null && (
                    <p className="pt-2 text-sm">
                      {t.netPercent(
                        tax.net_percent.toFixed(1).replace(".", ",")
                      )}
                    </p>
                  )}
                  <p className="pt-2 text-xs text-muted-foreground">
                    {t.taxSummaryHint}
                  </p>
                </>
              )}
            </CardContent>
          </Card>
        )}

        {/* The shape of the year: twelve months, each with what came in and
            what went out. A real Chart in place of the old div-based bars —
            the numbers stay reachable on hover (ChartTooltip), same as every
            other chart in the app, rather than printed permanently over
            twelve narrow bars. */}
        {totals && (
          <Card className="min-w-full flex-[2] lg:min-w-[32rem]">
            <CardHeader>
              <CardTitle>{t.byMonth}</CardTitle>
            </CardHeader>
            <CardContent>
              <ChartContainer config={chartConfig} className="aspect-auto h-72 w-full">
                <BarChart
                  data={chartData}
                  margin={{ top: 4, right: 8, left: 8, bottom: 0 }}
                >
                  <XAxis
                    dataKey="month"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                  />
                  <ChartTooltip
                    cursor={false}
                    content={
                      <ChartTooltipContent
                        formatter={(value, name, item) => (
                          <>
                            <div
                              className="h-2.5 w-2.5 shrink-0 rounded-[2px]"
                              style={{ backgroundColor: item.color }}
                            />
                            <div className="flex flex-1 justify-between gap-2 leading-none">
                              <span className="text-muted-foreground">
                                {name === "income" ? t.incomes : t.expenses}
                              </span>
                              <span className="font-mono font-medium tabular-nums">
                                € {formatCents(Number(value))}
                              </span>
                            </div>
                          </>
                        )}
                      />
                    }
                  />
                  <ChartLegend content={<ChartLegendContent />} />
                  <Bar dataKey="income" fill="var(--color-income)" radius={2} />
                  <Bar dataKey="expense" fill="var(--color-expense)" radius={2} />
                </BarChart>
              </ChartContainer>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
