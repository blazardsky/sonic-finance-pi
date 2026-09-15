import { useEffect, useMemo, useState } from "react"
import {
  Line,
  LineChart,
  Pie,
  PieChart,
  PolarAngleAxis,
  PolarGrid,
  Radar,
  RadarChart,
  XAxis,
} from "recharts"

import { PeriodLabel, PeriodStepper } from "@/components/PeriodStepper"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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
import { paletteVar } from "@/lib/palette"
import { colorOf } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, FullYearReport, MonthRow, YearTotals } from "@/types"

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
  const [categories, setCategories] = useState<Category[]>([])
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

  // Independent of which year is on screen — the pie chart below only reads
  // each Category's own stored `color` (ticket 05), which never changes with
  // the year selector.
  useEffect(() => {
    apiJSON<Category[]>("/api/categories")
      .then(setCategories)
      .catch(() => setCategories([]))
  }, [])

  const grid = useMemo(() => pivotByMonth(full?.by_month ?? [], year), [full, year])
  const path = useMemo(() => cumulativePath(totals?.months ?? []), [totals])
  const monthly = useMemo(() => monthlyTotals(totals?.months ?? []), [totals])
  const slices = useMemo(
    () => categorySlices(categoryShares(full?.by_month ?? []), categories),
    [full, categories]
  )
  const sliceConfig = useMemo(() => categorySliceConfig(slices), [slices])
  const step = (by: number) => setYear(String(Number(year) + by))

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <PeriodStepper
        previousLabel={t.previousYear}
        onPrevious={() => step(-1)}
        nextLabel={t.nextYear}
        nextDisabled={year >= thisYear()}
        onNext={() => step(1)}
      >
        <PeriodLabel as="h1">
          {t.yearReport} · {year}
        </PeriodLabel>
      </PeriodStepper>

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

      {/* The two square charts share a row while both can hold their floor
          width, and each takes a row of its own the moment they cannot —
          flex-wrap picks that up from the space it actually has, so there is
          no breakpoint here to keep in step with the sidebar's width. */}
      <div className="flex flex-wrap gap-6 empty:hidden">
        {slices.length > 0 && (
          <section className="flex min-w-72 flex-1 flex-col gap-2">
            <h2 className="text-sm font-medium text-muted-foreground">
              {t.byCategory}
            </h2>
            <CategoryPieChart slices={slices} config={sliceConfig} />
          </section>
        )}

        {monthly.length > 0 && (
          <section className="flex min-w-72 flex-1 flex-col gap-2">
            <h2 className="text-sm font-medium text-muted-foreground">
              {t.byMonth}
            </h2>
            <MonthlyRadarChart months={monthly} />
          </section>
        )}
      </div>

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
// stat-tile reading both report pages already use — restyled to the
// Dashboard's own Card/CardHeader/CardContent pattern (its first stat card,
// e.g.) rather than a plain bordered div, so a figure reads with the same
// weight wherever it shows up.
export function Figure({ label, cents }: { label: string; cents: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{label}</CardTitle>
      </CardHeader>
      <CardContent elevated>
        <p
          className={`text-2xl font-medium tabular-nums ${cents < 0 ? "text-destructive" : ""}`}
        >
          € {formatCents(cents)}
        </p>
      </CardContent>
    </Card>
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

// One slice of the pie chart: a Category's share of the year, with
// the long tail lumped into "Altre categorie" so a year with thirty
// Categories still draws a pie someone can read. Colours come from each
// Category's own stored `color` slot (ticket 05) — the same lookup
// pickers.ts's colorOf does elsewhere — so a Category is the same colour
// here as it is on the Dashboard's trend chart and calendar dots. Two
// Categories sharing a color render as the same wedge colour, which is
// expected (ADR-0016), not a bug this chart works around.
type CategorySlice = { key: string; label: string; amount: number; fill: string }

const MAX_CATEGORY_SLICES = 5

function categorySlices(
  shares: CategoryShare[],
  categories: Category[]
): CategorySlice[] {
  const top = shares.slice(0, MAX_CATEGORY_SLICES)
  const rest = shares.slice(MAX_CATEGORY_SLICES)

  const slices: CategorySlice[] = top.map((c) => ({
    key: `c${c.id}`,
    label: c.name,
    amount: c.amount,
    fill: paletteVar(colorOf(categories, c.id), "primary"),
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
    <ChartContainer config={config} className="mx-auto aspect-square max-h-80 w-full max-w-80">
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

// The path chart draws Income as a 2px line, where --primary reads fine. A
// filled radar in --primary is a black blob in light mode and a white one in
// dark, so it borrows --credit — the green the app already spells received
// money in (RecentList, Dashboard). Expense keeps the path chart's red.
const radarChartConfig = {
  income: { label: t.incomes, color: "var(--credit)" },
  expense: pathChartConfig.expense,
} satisfies ChartConfig

// Both charts built on pathChartConfig name the hovered series and spell its
// cents as money — the default tooltip row, which would otherwise print the
// raw dataKey and a bare integer.
const eurosTooltip = (
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
)

// Per-month Income and Expense, one spoke each: the months cumulativePath
// draws, left un-accumulated. A radar compares months against each other,
// and a running total would only ever grow clockwise.
function monthlyTotals(months: MonthRow[]) {
  const now = thisMonth()
  return months
    .filter((m) => m.month <= now)
    .map((m) => ({
      label: t.monthsShort[Number(m.month.slice(5, 7)) - 1],
      income: m.income_cents,
      expense: m.expense_cents,
    }))
}

// The same two series the path chart runs left to right, closed into a ring:
// a month that spent more than it earned is the red web poking outside the
// green one, which the cumulative lines can never show — they only cross once.
//
// Both webs are outlined and only faintly filled. Two solid fills and no
// outline leaves whichever recharts draws second washing the other out, so
// the smaller month reads as an empty gap rather than as its own shape.
function MonthlyRadarChart({
  months,
}: {
  months: ReturnType<typeof monthlyTotals>
}) {
  return (
    <ChartContainer
      config={radarChartConfig}
      className="mx-auto aspect-square max-h-80 w-full max-w-80"
    >
      <RadarChart data={months} margin={{ top: -40, bottom: -10 }}>
        <ChartTooltip cursor={false} content={eurosTooltip} />
        <PolarAngleAxis dataKey="label" />
        <PolarGrid />
        <Radar
          dataKey="income"
          fill="var(--color-income)"
          fillOpacity={0.2}
          stroke="var(--color-income)"
          strokeWidth={2}
        />
        <Radar
          dataKey="expense"
          fill="var(--color-expense)"
          fillOpacity={0.2}
          stroke="var(--color-expense)"
          strokeWidth={2}
        />
        <ChartLegend className="mt-8" content={<ChartLegendContent />} />
      </RadarChart>
    </ChartContainer>
  )
}

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
        <ChartTooltip cursor={false} content={eurosTooltip} />
        <ChartLegend content={<ChartLegendContent />} />
        <Line dataKey="income" type="monotone" stroke="var(--color-income)" strokeWidth={2} dot={false} />
        <Line dataKey="expense" type="monotone" stroke="var(--color-expense)" strokeWidth={2} dot={false} />
        <Line dataKey="net" type="monotone" stroke="var(--color-net)" strokeWidth={2} dot={false} />
      </LineChart>
    </ChartContainer>
  )
}
