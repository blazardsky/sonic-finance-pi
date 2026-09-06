import { useCallback, useEffect, useState, type ReactNode } from "react"
import {
  RiBellFill,
  RiBellLine,
  RiCheckboxCircleLine,
  RiErrorWarningLine,
  RiScales3Line,
  RiShoppingBag3Line,
  RiWallet3Line,
} from "@remixicon/react"
import { Area, AreaChart } from "recharts"

import { running } from "@/pages/RecurringExpenses"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
} from "@/components/ui/chart"
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card"
import { Input } from "@/components/ui/input"
import { Toggle } from "@/components/ui/toggle"
import { CategoryTrendChart } from "@/components/CategoryTrendChart"
import { SpoilerAmount } from "@/components/SpoilerAmount"
import { Progress } from "@/components/ui/progress"
import { api, apiJSON } from "@/lib/api"
import { formatCents, formatDate, thisMonth, thisYear, today } from "@/lib/money"
import { nameOf, colorOf } from "@/lib/pickers"
import { t } from "@/lib/strings"
import { addDays, categoriesIn, dailyBuckets } from "@/lib/trend"
import type {
  Category,
  DailyCategoryTotal,
  EstimateReport,
  MonthRow,
  Pending,
  RecentEntry,
  Recurring,
  Reminder,
  TaxSummary,
  YearTotals,
} from "@/types"

// The Dashboard's rolling window: the last dailyWindowDays days ending today,
// matching cmd/reports.go's dailyWindowDays exactly — a household comparing
// the chart against "the last 30 days" in its head must see the same 30 days
// the server just answered with.
const dailyWindowDays = 30

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
  const [estimate, setEstimate] = useState<EstimateReport | null>(null)
  const [pending, setPending] = useState<Pending | null>(null)
  const [daily, setDaily] = useState<DailyCategoryTotal[]>([])
  const [error, setError] = useState("")

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<RecentEntry[]>("/api/reports/recent"),
        apiJSON<Recurring[]>("/api/recurring"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<YearTotals>(`/api/reports/year/${thisYear()}`),
        apiJSON<TaxSummary>(`/api/reports/tax/${thisYear()}`),
        apiJSON<EstimateReport>(`/api/reports/estimate/${thisYear()}`),
        apiJSON<Pending>("/api/pending-payments"),
        apiJSON<DailyCategoryTotal[]>("/api/reports/daily"),
      ])
        .then(([r, rec, cat, y, s, e, p, d]) => {
          setRecent(r)
          setRecurring(rec)
          setCategories(cat)
          setYear(y)
          setTax(s)
          setEstimate(e)
          setPending(p)
          setDaily(d)
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
            estimateCents={estimate?.income_estimate_cents ?? undefined}
            footnote={t.ofWhichExtra(formatCents(year.extra_income_cents || 0))}
          />
          <TotalsCard
            icon={RiShoppingBag3Line}
            label={t.yearExpenses}
            cents={year.expense_cents}
            estimateCents={estimate?.expense_estimate_cents ?? undefined}
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

      <DailyTrendCard daily={daily} />

      {/* items-start: without it the grid stretches the calendar column to
          match whichever side ends up taller, leaving Prossime scadenze
          padded with empty space under a card that is already as tall as it
          needs to be. */}
      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4">
          <UpcomingCard upcoming={upcoming} categories={categories} />
          {pending && <ClientsCard pending={pending} />}
          <RemindersCard />
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
// expenses also draw a bar against estimateCents — cmd/estimate.go's
// projected year-end figure, not a fixed target — labeled "stimati" in the
// caption underneath so it never reads as a second actual total (ADR-0010:
// this is the app's only projected, non-cash-basis number). estimateCents is
// undefined rather than 0 when that projection has no prior year to be built
// from, and the bar simply does not render — no Estimate beats a fabricated
// one. The difference does not draw a bar at all — it is not an estimate —
// and draws the running leftover instead. Income names the non-work slice;
// expenses name the tax paid against this year (ADR-0008: by tax year, not
// payment date).
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

// The Dashboard's daily chart by Category — every day of the rolling window,
// zero-spend days included, so a quiet stretch reads as a real gap rather
// than a shorter chart.
function DailyTrendCard({ daily }: { daily: DailyCategoryTotal[] }) {
  const days = Array.from({ length: dailyWindowDays }, (_, i) =>
    addDays(today(), -(dailyWindowDays - 1 - i))
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t.dailyTrend}</CardTitle>
      </CardHeader>
      <CardContent elevated={daily.length === 0}>
        {daily.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.noDailyTrend}</p>
        ) : (
          <CategoryTrendChart
            buckets={dailyBuckets(daily, days)}
            categories={categoriesIn(daily)}
          />
        )}
      </CardContent>
    </Card>
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
  const dayColors = Object.fromEntries(
    upcoming.map((u) => [u.date, colorOf(categories, u.r.category_id)])
  )

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
          dayColors={dayColors}
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
                <span className="flex min-w-0 items-baseline gap-1.5">
                  <span
                    aria-hidden
                    className="size-1.5 shrink-0 translate-y-[-1px] rounded-full"
                    style={{ backgroundColor: colorOf(categories, r.category_id) }}
                  />
                  <span className="min-w-0">
                    <span className="truncate">
                      {nameOf(categories, r.category_id)}
                    </span>{" "}
                    <span className="text-xs text-muted-foreground">
                      {t.dueIn(daysUntil(date))}
                    </span>
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

// Unpaid invoiced Incomes (any month), and habitual Clients with nothing
// billed this month — the two lists pendingPayments already computes
// (cmd/pending.go), as names rather than the full rows Month.tsx shows. Each
// name gets a HoverCard with the detail the badge itself has no room for —
// amount_cents/days_waiting are already in this same response (ticket 13);
// total_earned_cents is deliberately left out, since it lives on /api/clients,
// a request this card has never made and shouldn't start making just for a
// hover detail.
function ClientsCard({ pending }: { pending: Pending }) {
  const waiting = pending.outstanding.filter(
    (o) => o.client !== "" && o.invoice_sent_date !== ""
  )
  const unbilled = pending.not_yet_invoiced

  return (
    <Card className="@container">
      <CardHeader>
        <CardTitle>{t.clientsThisMonth}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 @[28rem]:flex-row @[28rem]:items-start">
        <ClientAlert
          title={t.pendingPayments}
          okText={t.noPendingClients}
          clients={waiting.map((o) => ({
            key: o.id,
            name: o.client,
            date: formatDate(o.invoice_sent_date),
            amountCents: o.amount_cents,
            daysWaiting: o.days_waiting,
          }))}
          hoverContent={(c) => (
            <div className="flex flex-col gap-1">
              <p className="font-medium tabular-nums">
                € {formatCents(c.amountCents)}
              </p>
              <p className="text-xs text-muted-foreground">
                {t.waitingDays(c.daysWaiting)}
              </p>
            </div>
          )}
        />
        <ClientAlert
          title={t.invoicesSentTitle}
          okText={t.invoicesAllSent}
          clients={unbilled.map((c) => ({ key: c.client_id, name: c.client }))}
          // Nothing numeric travels with this list (NotYetInvoicedClient is
          // just an id and a name) — the HoverCard surfaces why the Client is
          // listed at all instead, the same explanation Month.tsx's own
          // pending-payments section gives once for the whole list.
          hoverContent={() => (
            <p className="text-xs text-muted-foreground">
              {t.notYetInvoicedHint}
            </p>
          )}
        />
      </CardContent>
    </Card>
  )
}

function ClientAlert<
  T extends { key: string | number; name: string; date?: string },
>({
  title,
  okText,
  clients,
  hoverContent,
}: {
  title: string
  okText: string
  clients: T[]
  hoverContent: (client: T) => ReactNode
}) {
  const ok = clients.length === 0
  return (
    <Alert
      variant={ok ? "default" : "destructive"}
      className="@[28rem]:min-w-0 @[28rem]:flex-1"
    >
      {ok ? <RiCheckboxCircleLine /> : <RiErrorWarningLine />}
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription className={ok ? undefined : "flex flex-wrap gap-1"}>
        {ok
          ? okText
          : clients.map((c) => (
              <span key={c.key} className="inline-flex items-center gap-1">
                <ClientBadge name={c.name}>{hoverContent(c)}</ClientBadge>
                {c.date != null && (
                  <span className="text-xs tabular-nums">{c.date}</span>
                )}
              </span>
            ))}
      </AlertDescription>
    </Alert>
  )
}

// Radix's HoverCard only ever opens on a real pointer hover — it explicitly
// ignores touch, since there is no such thing as a "hover" on a phone — so a
// household reading this on the one device the app is actually for could tap
// a name all day and see nothing. Controlled open state plus a plain onClick
// covers that: a tap toggles it open, a mouse still opens it on hover exactly
// as before (the same setOpen either way — Radix's own controllable-state
// plumbing doesn't care which one drove it).
function ClientBadge({ name, children }: { name: string; children: ReactNode }) {
  const [open, setOpen] = useState(false)
  return (
    <HoverCard open={open} onOpenChange={setOpen}>
      <HoverCardTrigger asChild>
        <Badge
          variant="outline"
          className="cursor-help"
          onClick={() => setOpen((o) => !o)}
        >
          {name}
        </Badge>
      </HoverCardTrigger>
      <HoverCardContent>{children}</HoverCardContent>
    </HoverCard>
  )
}

// A household-managed list of on/off toggles for a manual action the app
// doesn't automate (ticket 06) — created, toggled and deleted right here.
// Nothing else in the app references a Reminder, so there is no separate
// management screen behind this card, and enabled already comes back
// collapsed from the server: this component never compares months itself.
function RemindersCard() {
  const [reminders, setReminders] = useState<Reminder[]>([])
  const [label, setLabel] = useState("")
  const [error, setError] = useState("")

  const load = useCallback(
    () =>
      apiJSON<Reminder[]>("/api/reminders")
        .then(setReminders)
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  async function write(path: string, init: RequestInit) {
    setError("")
    try {
      await api(path, init)
      await load()
    } catch {
      setError(t.reminderNotSaved)
    }
  }

  async function add(event: React.FormEvent) {
    event.preventDefault()
    if (!label.trim()) return
    await write("/api/reminders", {
      method: "POST",
      body: JSON.stringify({ label }),
    })
    setLabel("")
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t.reminders}</CardTitle>
      </CardHeader>
      <CardContent
        elevated={reminders.length === 0}
        className="flex flex-col gap-3"
      >
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        {reminders.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.noRemindersYet}</p>
        ) : (
          <ul className="flex flex-col divide-y divide-border">
            {reminders.map((r) => (
              <li
                key={r.id}
                className="flex items-center justify-between gap-3 py-2"
              >
                {/* A single independent on/off per row, not a segmented
                    set — any number of Reminders can be enabled at once —
                    so Toggle, not ToggleGroup (ticket 14). */}
                <Toggle
                  size="sm"
                  className="min-w-0 flex-1 justify-start gap-2 px-2.5 data-[state=on]:bg-credit/10 data-[state=on]:text-credit dark:data-[state=on]:bg-credit/20"
                  pressed={r.enabled}
                  onPressedChange={(pressed) =>
                    void write(`/api/reminders/${r.id}`, {
                      method: "PATCH",
                      body: JSON.stringify({ enabled: pressed }),
                    })
                  }
                  aria-label={t.reminderToggleLabel(r.label, r.enabled)}
                >
                  {r.enabled ? <RiBellFill /> : <RiBellLine />}
                  <span className="truncate text-sm font-normal">
                    {r.label}
                  </span>
                </Toggle>
                <Button
                  type="button"
                  size="xs"
                  variant="ghost"
                  onClick={() => {
                    if (confirm(t.confirmDeleteReminder(r.label)))
                      void write(`/api/reminders/${r.id}`, {
                        method: "DELETE",
                      })
                  }}
                >
                  {t.delete}
                </Button>
              </li>
            ))}
          </ul>
        )}
        <form
          onSubmit={(event) => void add(event)}
          className="flex flex-col gap-2"
        >
          <Input
            value={label}
            onChange={(event) => setLabel(event.target.value)}
            placeholder={t.reminderLabel}
          />
          <Button type="submit" size="sm">
            {t.addReminder}
          </Button>
        </form>
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
                    <SpoilerAmount
                      cents={entry.amount_cents}
                      gift={entry.is_gift}
                    />
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
