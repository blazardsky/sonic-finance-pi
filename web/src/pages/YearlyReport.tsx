import { useEffect, useMemo, useState } from "react"
import { Line, LineChart, XAxis } from "recharts"

import { Button } from "@/components/ui/button"
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { apiJSON } from "@/lib/api"
import { formatCents, thisMonth, thisYear } from "@/lib/money"
import { t } from "@/lib/strings"
import type { FullYearReport, MonthRow, YearTotals } from "@/types"

// ADR-0011: this page, and only this page, is allowed to ask for a whole
// year's Category breakdown — a query the ticket that added Year.tsx's own
// report named the heaviest in the app and deliberately left out of a screen
// loaded by default. It is affordable here because the household opens it on
// purpose, and because the backend answers it as one (month, category)
// -grouped query rather than twelve per-month calls (cmd/yearreport.go).
export function YearlyReport() {
  const [year, setYear] = useState(thisYear)
  const [totals, setTotals] = useState<YearTotals | null>(null)
  const [full, setFull] = useState<FullYearReport | null>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    let current = true
    Promise.all([
      apiJSON<YearTotals>(`/api/reports/year/${year}`),
      apiJSON<FullYearReport>(`/api/reports/year/${year}/full`),
    ])
      .then(([y, f]) => {
        if (!current) return
        setTotals(y)
        setFull(f)
        setError("")
      })
      .catch(() => {
        if (!current) return
        setTotals(null)
        setFull(null)
        setError(t.serverUnreachable)
      })
    return () => {
      current = false
    }
  }, [year])

  const grid = useMemo(() => pivotByMonth(full?.by_month ?? [], year), [full, year])
  const path = useMemo(() => cumulativePath(totals?.months ?? []), [totals])
  const step = (by: number) => setYear(String(Number(year) + by))

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 p-6">
      <div className="flex items-center justify-between gap-2">
        <Button
          variant="ghost"
          size="sm"
          aria-label={t.previousYear}
          onClick={() => step(-1)}
        >
          ‹
        </Button>
        <h1 className="text-2xl font-medium tabular-nums">
          {t.yearReport} · {year}
        </h1>
        <Button
          variant="ghost"
          size="sm"
          aria-label={t.nextYear}
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

      {full && totals && (
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <Figure label={t.expenseExcludingTax} cents={full.expense_excluding_tax_cents} />
          {/* Calendar-year tax (occurred_on): the amount the neighbouring figure left out, not ADR-0008's tax-year attribution. */}
          <Figure label={t.taxes} cents={totals.expense_cents - full.expense_excluding_tax_cents} />
          <Figure
            label={t.savingsAtStartOfYear(year)}
            cents={full.savings_at_start_cents}
          />
          <Figure label={t.medianExpense} cents={full.median_expense_cents} />
          <Figure label={t.medianIncome} cents={full.median_income_cents} />
          <Figure label={t.medianNet} cents={full.median_net_cents} />
        </dl>
      )}
      {full && <p className="text-xs text-muted-foreground">{t.medianHint(year)}</p>}

      {totals && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.financesPath}
          </h2>
          <FinancesPathChart path={path} />
        </section>
      )}

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium text-muted-foreground">
          {t.fullYearBreakdown}
        </h2>
        {grid.categories.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.noSpendThisYear}</p>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t.month}</TableHead>
                  {grid.categories.map((c) => (
                    <TableHead key={c.id} className="text-right">
                      {c.name}
                    </TableHead>
                  ))}
                  <TableHead className="text-right">{t.total}</TableHead>
                  <TableHead className="text-right">{t.runningTotal}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {grid.rows.map((row) => (
                  <TableRow key={row.month}>
                    <TableCell>{row.label}</TableCell>
                    {grid.categories.map((c) => (
                      <TableCell key={c.id} className="text-right tabular-nums">
                        {formatCents(row.amounts[c.id] ?? 0)}
                      </TableCell>
                    ))}
                    <TableCell className="text-right font-medium tabular-nums">
                      {formatCents(row.total)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums text-muted-foreground">
                      {formatCents(row.running)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </section>
    </div>
  )
}

function Figure({ label, cents }: { label: string; cents: number }) {
  return (
    <div className="rounded-md border border-border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-medium tabular-nums">€ {formatCents(cents)}</p>
    </div>
  )
}

// One row of the breakdown grid: a month, each Category's share of it (0 when
// that Category had nothing that month), the month's own total, and the
// running total through it.
type GridRow = {
  month: string
  label: string
  amounts: Record<number, number>
  total: number
  running: number
}

// pivotByMonth turns the flat (month, category) list the grouped query
// answers with into the grid the table draws: Categories ordered by their
// share of the year (biggest first, the same order readBreakdown's own
// per-month list uses), and all twelve months present whether or not
// anything was spent in them — a report over a whole year is a shape, same
// as Year.tsx's own view.
function pivotByMonth(byMonth: FullYearReport["by_month"], year: string) {
  const categoryTotals = new Map<number, { name: string; total: number }>()
  const perMonth = new Map<string, Record<number, number>>()

  for (const line of byMonth) {
    const known = categoryTotals.get(line.category_id)
    categoryTotals.set(line.category_id, {
      name: line.category,
      total: (known?.total ?? 0) + line.amount_cents,
    })
    const row = perMonth.get(line.month) ?? {}
    row[line.category_id] = line.amount_cents
    perMonth.set(line.month, row)
  }

  const categories = [...categoryTotals.entries()]
    .sort((a, b) => b[1].total - a[1].total)
    .map(([id, c]) => ({ id, name: c.name }))

  let running = 0
  const rows: GridRow[] = t.monthsShort.map((label, i) => {
    const month = `${year}-${String(i + 1).padStart(2, "0")}`
    const amounts = perMonth.get(month) ?? {}
    const total = Object.values(amounts).reduce((sum, v) => sum + v, 0)
    running += total
    return { month, label, amounts, total, running }
  })

  return { categories, rows }
}

type PathPoint = { label: string; income: number; expense: number; net: number }

// cumulativePath is Dashboard.tsx's netPath (cmd/reports.go's per-month Net,
// run forward through the year) widened to keep Income and Expense alongside
// Net — the "finances path" chart the ticket asks for, built entirely from
// figures /api/reports/year/{year} already returns. Months after "now" are
// dropped for the same reason Dashboard's sparkline drops them: they are
// still zeros, and the right edge would flatten for no reason.
function cumulativePath(months: MonthRow[]): PathPoint[] {
  const now = thisMonth()
  let income = 0
  let expense = 0
  return months
    .filter((m) => m.month <= now)
    .map((m) => {
      income += m.income_cents
      expense += m.expense_cents
      const monthIndex = Number(m.month.slice(5, 7)) - 1
      return { label: t.monthsShort[monthIndex], income, expense, net: income - expense }
    })
}

const pathChartConfig = {
  income: { label: t.incomes, color: "var(--primary)" },
  expense: { label: t.expenses, color: "var(--destructive)" },
  net: { label: t.difference, color: "var(--credit)" },
} satisfies ChartConfig

function FinancesPathChart({ path }: { path: PathPoint[] }) {
  return (
    <ChartContainer config={pathChartConfig} className="aspect-auto h-64 w-full">
      <LineChart data={path} margin={{ top: 4, right: 8, left: 8, bottom: 0 }}>
        <XAxis
          dataKey="label"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          interval="preserveStartEnd"
        />
        <ChartTooltip
          cursor={false}
          content={
            <ChartTooltipContent
              formatter={(value, name) => (
                <div className="flex flex-1 justify-between gap-2 leading-none">
                  <span className="text-muted-foreground">
                    {pathChartConfig[name as keyof typeof pathChartConfig]?.label ?? name}
                  </span>
                  <span className="font-mono font-medium tabular-nums">
                    € {formatCents(Number(value))}
                  </span>
                </div>
              )}
            />
          }
        />
        <ChartLegend content={<ChartLegendContent />} />
        <Line dataKey="income" type="monotone" stroke="var(--color-income)" strokeWidth={2} dot={false} />
        <Line dataKey="expense" type="monotone" stroke="var(--color-expense)" strokeWidth={2} dot={false} />
        <Line dataKey="net" type="monotone" stroke="var(--color-net)" strokeWidth={2} dot={false} />
      </LineChart>
    </ChartContainer>
  )
}
