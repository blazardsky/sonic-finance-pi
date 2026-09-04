import { useCallback, useEffect, useState, type ReactNode } from "react"
import {
  RiCheckboxCircleLine,
  RiErrorWarningLine,
  RiScales3Line,
  RiShoppingBag3Line,
  RiWallet3Line,
} from "@remixicon/react"
import { Area, AreaChart } from "recharts"

import { running } from "@/pages/RecurringExpenses"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Calendar } from "@/components/ui/calendar"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
} from "@/components/ui/chart"
import { Progress } from "@/components/ui/progress"
import { apiJSON } from "@/lib/api"
import { formatCents, formatDate, thisMonth, thisYear } from "@/lib/money"
import { nameOf } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type {
  Category,
  MonthRow,
  MonthTotals,
  Pending,
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
  const ms =
    new Date(date + "T00:00:00").getTime() - new Date().setHours(0, 0, 0, 0)
  return Math.round(ms / 86_400_000)
}

// ponytail: the two progress bars' targets, fixed at the figure the ticket
// asked for rather than a household setting. Wire these up to a real yearly
// target (Settings, presumably) if the household ever wants to change them.
const ESTIMATE_INCOME_CENTS = 50_000 * 100
const ESTIMATE_EXPENSE_CENTS = 50_000 * 100

// The home screen: the year's three totals against a target, what is coming
// due, what was just typed in either direction, and whether the freelance
// side of things looks handled this month. Every number here is either a
// totals report the app already computes (Year.tsx, Month.tsx read the same
// endpoints) or a plain read of one — nothing is invented that the server
// does not already know.
export function Dashboard() {
  const [recent, setRecent] = useState<RecentEntry[]>([])
  const [recurring, setRecurring] = useState<Recurring[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [year, setYear] = useState<YearTotals | null>(null)
  const [tax, setTax] = useState<TaxSummary | null>(null)
  const [month, setMonth] = useState<MonthTotals | null>(null)
  const [pending, setPending] = useState<Pending | null>(null)
  const [error, setError] = useState("")

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<RecentEntry[]>("/api/reports/recent"),
        apiJSON<Recurring[]>("/api/recurring"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<YearTotals>(`/api/reports/year/${thisYear()}`),
        apiJSON<TaxSummary>(`/api/reports/tax/${thisYear()}`),
        apiJSON<MonthTotals>(`/api/reports/month/${thisMonth()}`),
        apiJSON<Pending>("/api/pending-payments"),
      ])
        .then(([r, rec, cat, y, s, m, p]) => {
          setRecent(r)
          setRecurring(rec)
          setCategories(cat)
          setYear(y)
          setTax(s)
          setMonth(m)
          setPending(p)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // Unsliced: recentLimit (cmd/reports.go) already caps how many come back
  // combined, and RecentList scrolls its own overflow rather than losing rows
  // to a second arbitrary cap here.
  const latestExpenses = recent.filter((e) => e.direction === "expense")
  const latestIncomes = recent.filter((e) => e.direction === "income")

  const upcoming = recurring
    .filter(running)
    .map((r) => ({ r, date: nextOccurrence(r) }))
    .sort((a, b) => a.date.localeCompare(b.date))
    .slice(0, 3)

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6">
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {year && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <TotalsCard
            icon={RiWallet3Line}
            label={t.yearIncome}
            cents={year.income_cents}
            estimateCents={ESTIMATE_INCOME_CENTS}
            footnote={t.ofWhichExtra(formatCents(year.extra_income_cents || 0))}
          />
          <TotalsCard
            icon={RiShoppingBag3Line}
            label={t.yearExpenses}
            cents={year.expense_cents}
            estimateCents={ESTIMATE_EXPENSE_CENTS}
            footnote={
              tax ? t.ofWhichTax(formatCents(tax.tax_paid_cents)) : undefined
            }
          />
          <TotalsCard
            icon={RiScales3Line}
            label={t.difference}
            cents={year.net_cents}
          >
            <NetSparkline months={year.months} />
          </TotalsCard>
        </div>
      )}

      {/* items-start: without it the grid stretches the calendar column to
          match whichever side ends up taller, leaving Prossime scadenze
          padded with empty space under a card that is already as tall as it
          needs to be. */}
      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4">
          <UpcomingCard upcoming={upcoming} categories={categories} />
          {month && pending && <ClientsCard month={month} pending={pending} />}
        </div>

        {/* The two recent lists stacked in the two columns Prossime scadenze
            does not use, rather than each squeezed into one column of three:
            more width per row, and stacking (instead of a third column) is
            what actually uses the extra height next to a calendar that is
            shorter than either list could be. */}
        <div className="flex flex-col gap-4 lg:col-span-2">
          <RecentList
            title={t.latestExpenses}
            empty={t.noRecentExpenses}
            entries={latestExpenses}
          />
          <RecentList
            title={t.recentIncomes}
            empty={t.noRecentIncomes}
            entries={latestIncomes}
          />
        </div>
      </div>
    </div>
  )
}

// One of the three headline cards: the actual figure, big. Income and
// expenses also draw a bar against a fixed target; the difference does
// not — it is not an estimate — and draws the running leftover instead.
// Income names the non-work slice; expenses name the tax paid against
// this year (ADR-0008: by tax year, not payment date).
function TotalsCard({
  icon: Icon,
  label,
  cents,
  estimateCents,
  footnote,
  children,
}: {
  icon: typeof RiWallet3Line
  label: string
  cents: number
  estimateCents?: number
  footnote?: string
  children?: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm text-ink">
          <Icon className="size-4" />
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent elevated className="flex flex-col gap-3">
        <p
          className={`text-2xl font-medium tabular-nums ${
            cents < 0 ? "text-destructive" : ""
          }`}
        >
          € {formatCents(cents)}
        </p>
        {estimateCents != null && (
          <>
            <Progress
              value={Math.max(0, Math.min(100, (cents / estimateCents) * 100))}
            />
            <p className="text-xs text-muted-foreground">
              {t.ofEstimate(formatCents(estimateCents))}
            </p>
          </>
        )}
        {footnote != null && (
          <p className="text-xs text-muted-foreground">{footnote}</p>
        )}
        {children}
      </CardContent>
    </Card>
  )
}

// Running leftover from January through this month. Future months are
// dropped so the line ends at now — later months in the year report are
// still zeros, and including them would flatten the right edge for no reason.
function netPath(months: MonthRow[]) {
  const now = thisMonth()
  let run = 0
  return months
    .filter((m) => m.month <= now)
    .map((m, i) => ({ month: t.monthsShort[i], net: (run += m.net_cents) }))
}

function NetSparkline({ months }: { months: MonthRow[] }) {
  const data = netPath(months)
  const color =
    (data.at(-1)?.net ?? 0) >= 0 ? "var(--credit)" : "var(--destructive)"
  const chartConfig = {
    net: { label: t.difference, color },
  } satisfies ChartConfig

  return (
    <ChartContainer
      config={chartConfig}
      className="aspect-auto h-16 w-full"
      aria-label={t.difference}
    >
      <AreaChart
        accessibilityLayer
        data={data}
        margin={{ top: 4, right: 2, left: 2, bottom: 0 }}
      >
        <defs>
          <linearGradient id="netFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-net)" stopOpacity={0.4} />
            <stop offset="100%" stopColor="var(--color-net)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <ChartTooltip
          cursor={false}
          content={({ active, payload }) => {
            const row = payload?.[0]?.payload as
              { month: string; net: number } | undefined
            if (!active || !row) return null
            return (
              <div className="rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl">
                {row.month}: € {formatCents(row.net)}
              </div>
            )
          }}
        />
        <Area
          dataKey="net"
          type="monotone"
          stroke="var(--color-net)"
          strokeWidth={2}
          fill="url(#netFill)"
          dot={false}
        />
      </AreaChart>
    </ChartContainer>
  )
}

// The current date, and the next few payments a Recurring definition owes —
// same preview Dashboard always drew, just fewer of them and next to a real
// calendar instead of only a heading.
function UpcomingCard({
  upcoming,
  categories,
}: {
  upcoming: { r: Recurring; date: string }[]
  categories: Category[]
}) {
  const [selected, setSelected] = useState<Date | undefined>(new Date())

  return (
    <Card className="@container">
      <CardHeader>
        <CardTitle>{t.upcomingRecurring}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 @[28rem]:flex-row @[28rem]:items-start">
        <Calendar
          mode="single"
          selected={selected}
          onSelect={setSelected}
          className="mx-auto shrink-0 @[28rem]:mx-0"
        />
        {upcoming.length === 0 ? (
          <p className="text-sm text-muted-foreground @[28rem]:mt-2 @[28rem]:flex-1">
            {t.noUpcomingRecurring}
          </p>
        ) : (
          <ul className="flex flex-col divide-y divide-border @[28rem]:mt-2 @[28rem]:min-w-0 @[28rem]:flex-1">
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
      </CardContent>
    </Card>
  )
}

// Whether anything has come in this month, and whether any regular Client
// looks forgotten — the two questions pendingPayments already answers
// (cmd/pending.go), condensed to a basic alert rather than the full lists
// Month.tsx shows.
function ClientsCard({
  month,
  pending,
}: {
  month: MonthTotals
  pending: Pending
}) {
  const paid = month.income_cents > 0
  const invoicesOK = pending.not_yet_invoiced.length === 0

  return (
    <Card className="@container">
      <CardHeader>
        <CardTitle>{t.clientsThisMonth}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 @[28rem]:flex-row @[28rem]:items-start">
        <Alert
          variant={paid ? "default" : "destructive"}
          className="@[28rem]:min-w-0 @[28rem]:flex-1"
        >
          {paid ? <RiCheckboxCircleLine /> : <RiErrorWarningLine />}
          <AlertTitle>{t.clientsPaidTitle}</AlertTitle>
          <AlertDescription>
            {paid
              ? t.clientsPaid(formatCents(month.income_cents))
              : t.clientsNotPaid}
          </AlertDescription>
        </Alert>
        <Alert
          variant={invoicesOK ? "default" : "destructive"}
          className="@[28rem]:min-w-0 @[28rem]:flex-1"
        >
          {invoicesOK ? <RiCheckboxCircleLine /> : <RiErrorWarningLine />}
          <AlertTitle>{t.invoicesSentTitle}</AlertTitle>
          <AlertDescription>
            {invoicesOK
              ? t.invoicesAllSent
              : t.invoicesMissing(pending.not_yet_invoiced.length)}
          </AlertDescription>
        </Alert>
      </CardContent>
    </Card>
  )
}

// One direction's recent entries. Shared by both cards: an Income with no
// payment date is the unpaid state (ADR-0003) and stays muted, exactly as
// Month.tsx's own recent list reads it — there is only one way this list
// is allowed to read.
//
// The list scrolls its own overflow past a generous height rather than
// growing the card without limit — recentLimit (cmd/reports.go) already
// caps this at a couple dozen rows combined, nowhere near where a windowed
// list (TanStack Virtual) would start paying for itself over a plain
// scrolling <ul>.
function RecentList({
  title,
  empty,
  entries,
}: {
  title: string
  empty: string
  entries: RecentEntry[]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent elevated={entries.length === 0}>
        {entries.length === 0 ? (
          <p className="text-sm text-muted-foreground">{empty}</p>
        ) : (
          <ul className="flex max-h-96 flex-col divide-y divide-border overflow-y-auto">
            {entries.map((entry) => {
              const who =
                entry.direction === "income" ? entry.client : entry.payer
              return (
                <li
                  key={entry.id}
                  className="flex items-center justify-between gap-3 py-3"
                >
                  <span className="min-w-0">
                    <span className="block truncate font-medium">
                      {entry.category}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {[entry.date ? formatDate(entry.date) : t.notPaidYet, who]
                        .filter(Boolean)
                        .join(" · ")}
                    </span>
                  </span>
                  <span
                    className={`font-medium whitespace-nowrap tabular-nums ${
                      !entry.date
                        ? "text-muted-foreground"
                        : entry.direction === "income"
                          ? "text-credit"
                          : ""
                    }`}
                  >
                    € {formatCents(entry.amount_cents)}
                  </span>
                </li>
              )
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
