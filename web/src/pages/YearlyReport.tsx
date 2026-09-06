import { useEffect, useMemo, useState } from "react"
import {
  Label,
  Line,
  LineChart,
  Pie,
  PieChart,
  PolarRadiusAxis,
  RadialBar,
  RadialBarChart,
  XAxis,
} from "recharts"

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
  const slices = useMemo(() => categorySlices(categoryShares(full?.by_month ?? [])), [full])
  const sliceConfig = useMemo(() => categorySliceConfig(slices), [slices])
  const step = (by: number) => setYear(String(Number(year) + by))

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

      {slices.length > 0 && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.byCategory}
          </h2>
          <div className="flex flex-wrap gap-6">
            <CategoryPieChart slices={slices} config={sliceConfig} />
            <CategoryRadialChart slices={slices} config={sliceConfig} />
          </div>
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

// Exported for Month's income/expense/difference card (ticket 01), the same
// small-box reading both report pages already use.
export function Figure({ label, cents }: { label: string; cents: number }) {
  return (
    <div className="rounded-md border border-border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p
        className={`text-lg font-medium tabular-nums ${cents < 0 ? "text-destructive" : ""}`}
      >
        € {formatCents(cents)}
      </p>
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
  const perMonth = new Map<string, Record<number, number>>()
  for (const line of byMonth) {
    const row = perMonth.get(line.month) ?? {}
    row[line.category_id] = line.amount_cents
    perMonth.set(line.month, row)
  }

  const categories = categoryShares(byMonth).map((c) => ({ id: c.id, name: c.name }))

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

// A Category's total for the whole year, biggest first — the same shape
// pivotByMonth's column order and the pie/radial charts below all read.
type CategoryShare = { id: number; name: string; amount: number }

function categoryShares(byMonth: FullYearReport["by_month"]): CategoryShare[] {
  const totals = new Map<number, { name: string; amount: number }>()
  for (const line of byMonth) {
    const known = totals.get(line.category_id)
    totals.set(line.category_id, {
      name: line.category,
      amount: (known?.amount ?? 0) + line.amount_cents,
    })
  }
  return [...totals.entries()]
    .sort((a, b) => b[1].amount - a[1].amount)
    .map(([id, c]) => ({ id, name: c.name, amount: c.amount }))
}

// One slice of the pie/radial charts: a Category's share of the year, or
// (past the 5 colors --chart-1..5 give us) the rest lumped into "Altre
// categorie" rather than cycling colors and making two different Categories
// look like the same one.
type CategorySlice = { key: string; label: string; amount: number; fill: string }

const MAX_CATEGORY_SLICES = 5

function categorySlices(shares: CategoryShare[]): CategorySlice[] {
  const top = shares.slice(0, MAX_CATEGORY_SLICES)
  const rest = shares.slice(MAX_CATEGORY_SLICES)

  const slices: CategorySlice[] = top.map((c, i) => ({
    key: `c${c.id}`,
    label: c.name,
    amount: c.amount,
    fill: `var(--chart-${i + 1})`,
  }))

  if (rest.length > 0) {
    slices.push({
      key: "other",
      label: t.otherCategories,
      amount: rest.reduce((sum, c) => sum + c.amount, 0),
      fill: "var(--muted-foreground)",
    })
  }

  return slices
}

function categorySliceConfig(slices: CategorySlice[]): ChartConfig {
  return Object.fromEntries(
    slices.map((s) => [s.key, { label: s.label, color: s.fill }])
  )
}

function CategoryPieChart({
  slices,
  config,
}: {
  slices: CategorySlice[]
  config: ChartConfig
}) {
  return (
    <ChartContainer config={config} className="mx-auto aspect-square max-h-64 w-full max-w-64">
      <PieChart>
        <ChartTooltip
          content={
            <ChartTooltipContent
              hideLabel
              nameKey="key"
              // Pie's own dataKey is always "amount" — the slice a hovered
              // wedge belongs to is on its payload, not its dataKey.
              formatter={(value, _name, item) => (
                <div className="flex flex-1 justify-between gap-2 leading-none">
                  <span className="text-muted-foreground">
                    {config[String(item.payload.key)]?.label}
                  </span>
                  <span className="font-mono font-medium tabular-nums">
                    € {formatCents(Number(value))}
                  </span>
                </div>
              )}
            />
          }
        />
        <Pie data={slices} dataKey="amount" nameKey="key" />
        <ChartLegend content={<ChartLegendContent nameKey="key" />} />
      </PieChart>
    </ChartContainer>
  )
}

function CategoryRadialChart({
  slices,
  config,
}: {
  slices: CategorySlice[]
  config: ChartConfig
}) {
  const total = slices.reduce((sum, s) => sum + s.amount, 0)
  // One stacked row: every slice as a ring of the same bar, its share of the
  // year drawn as an arc length rather than a wedge — the same breakdown as
  // the pie chart, read a second way.
  const row = Object.fromEntries(slices.map((s) => [s.key, s.amount]))

  return (
    <ChartContainer config={config} className="mx-auto aspect-square max-h-64 w-full max-w-64">
      <RadialBarChart data={[row]} innerRadius={30} outerRadius={110}>
        <ChartTooltip
          content={
            <ChartTooltipContent
              hideLabel
              nameKey="key"
              formatter={(value, _name, item) => (
                <div className="flex flex-1 justify-between gap-2 leading-none">
                  <span className="text-muted-foreground">
                    {config[String(item.dataKey)]?.label}
                  </span>
                  <span className="font-mono font-medium tabular-nums">
                    € {formatCents(Number(value))}
                  </span>
                </div>
              )}
            />
          }
        />
        <PolarRadiusAxis tick={false} tickLine={false} axisLine={false}>
          <Label
            content={({ viewBox }) => {
              if (!viewBox || !("cx" in viewBox) || !("cy" in viewBox)) return null
              return (
                <text x={viewBox.cx} y={viewBox.cy} textAnchor="middle" dominantBaseline="middle">
                  <tspan x={viewBox.cx} y={viewBox.cy} className="fill-foreground font-medium tabular-nums">
                    € {formatCents(total)}
                  </tspan>
                </text>
              )
            }}
          />
        </PolarRadiusAxis>
        {slices.map((s) => (
          <RadialBar
            key={s.key}
            dataKey={s.key}
            stackId="a"
            cornerRadius={4}
            fill={`var(--color-${s.key})`}
            className="stroke-transparent stroke-2"
          />
        ))}
        <ChartLegend content={<ChartLegendContent />} />
      </RadialBarChart>
    </ChartContainer>
  )
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
