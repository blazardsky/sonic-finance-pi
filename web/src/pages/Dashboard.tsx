import { useCallback, useEffect, useState, type ReactNode } from "react"
import {
  RiAddLine,
  RiCheckboxBlankCircleLine,
  RiCheckboxCircleFill,
  RiCheckboxCircleLine,
  RiCheckLine,
  RiDeleteBinLine,
  RiEditLine,
  RiErrorWarningLine,
  RiMoreLine,
  RiScales3Line,
  RiShoppingBag3Line,
  RiWallet3Line,
} from "@remixicon/react"
import { Area, AreaChart } from "recharts"

import { Row } from "@/pages/Month"
import { running } from "@/pages/RecurringExpenses"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
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
import { colorOf, nameOf } from "@/lib/pickers"
import { paletteVar } from "@/lib/palette"
import { t } from "@/lib/strings"
import { addDays, categoriesIn, dailyBuckets } from "@/lib/trend"
import type {
  BudgetReport,
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
export function Dashboard({
  clockOK,
  onEditIncome,
}: {
  clockOK: boolean
  onEditIncome: (id: number) => void
}) {
  const [recent, setRecent] = useState<RecentEntry[]>([])
  const [recurring, setRecurring] = useState<Recurring[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [year, setYear] = useState<YearTotals | null>(null)
  const [tax, setTax] = useState<TaxSummary | null>(null)
  const [estimate, setEstimate] = useState<EstimateReport | null>(null)
  const [pending, setPending] = useState<Pending | null>(null)
  const [daily, setDaily] = useState<DailyCategoryTotal[]>([])
  const [error, setError] = useState("")
  // True only until the first load settles — every list here starts empty
  // regardless of whether the household has no data yet or the request is
  // simply still in flight, and a Skeleton is how the two stop looking the
  // same for the second or so that tells them apart.
  const [loading, setLoading] = useState(true)

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
        .catch(() => setError(t.serverUnreachable))
        .finally(() => setLoading(false)),
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

      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Skeleton className="h-44 w-full" />
          <Skeleton className="h-44 w-full" />
          <Skeleton className="h-44 w-full" />
        </div>
      ) : year && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <TotalsCard
            icon={RiWallet3Line}
            label={t.yearIncome}
            cents={year.income_cents}
            estimateCents={estimate?.income_estimate_cents ?? undefined}
            footnotes={[
              { label: t.ofWhichExtra, cents: year.extra_income_cents || 0 },
              // "Lordo" is the freelance slice of the same total: invoiced
              // work arrives before tax, and tax leaves again as an Expense
              // (ADR-0004). received_cents is already that sum — freelance
              // only, cash-basis — so this reads the tax summary the card
              // below it reads rather than asking for a second year total.
              ...(tax ? [{ label: t.ofWhichGross, cents: tax.received_cents }] : []),
            ]}
          />
          <TotalsCard
            icon={RiShoppingBag3Line}
            label={t.yearExpenses}
            cents={year.expense_cents}
            estimateCents={estimate?.expense_estimate_cents ?? undefined}
            footnotes={[
              ...(tax ? [{ label: t.ofWhichTax, cents: tax.tax_paid_cents }] : []),
              { label: t.ofWhichInvestments, cents: year.investments_cents },
            ]}
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

      <DailyTrendCard loading={loading} daily={daily} categories={categories} />

      {/* items-start: without it the grid stretches the calendar column to
          match whichever side ends up taller, leaving Prossime scadenze
          padded with empty space under a card that is already as tall as it
          needs to be. */}
      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4">
          <UpcomingCard loading={loading} upcoming={upcoming} categories={categories} />
          <BudgetCard />
          {loading ? (
            <Skeleton className="h-32 w-full" />
          ) : (
            pending && (
              <ClientsCard
                pending={pending}
                clockOK={clockOK}
                onEditIncome={onEditIncome}
                onPaid={load}
              />
            )
          )}
          <RemindersCard loading={loading} />
        </div>

        {/* The two recent lists stacked in the two columns Prossime scadenze
            does not use, rather than each squeezed into one column of three:
            more width per row, and stacking (instead of a third column) is
            what actually uses the extra height next to a calendar that is
            shorter than either list could be. */}
        <div className="flex flex-col gap-4 lg:col-span-2">
          <RecentList
            loading={loading}
            title={t.latestExpenses}
            empty={t.noRecentExpenses}
            entries={latestExpenses}
          />
          <RecentList
            loading={loading}
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
  footnotes,
  children,
}: {
  icon: typeof RiWallet3Line
  label: string
  cents: number
  estimateCents?: number
  footnotes?: { label: string; cents: number }[]
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
        {footnotes != null && footnotes.length > 0 && (
          <>
            <Separator />
            {/* flex-wrap: side by side when the card is wide enough for
                both, each its own row (stacked) the moment it is not —
                no breakpoint to pick, the row just wraps like text does. */}
            <div className="flex flex-wrap gap-x-4 gap-y-1.5">
              {footnotes.map((footnote) => (
                <p
                  key={footnote.label}
                  className="flex items-center gap-1.5 text-xs text-muted-foreground"
                >
                  {footnote.label}
                  <Badge variant="secondary" className="tabular-nums">
                    € {formatCents(footnote.cents)}
                  </Badge>
                </p>
              ))}
            </div>
          </>
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
function DailyTrendCard({
  loading,
  daily,
  categories,
}: {
  loading: boolean
  daily: DailyCategoryTotal[]
  categories: Category[]
}) {
  const days = Array.from({ length: dailyWindowDays }, (_, i) =>
    addDays(today(), -(dailyWindowDays - 1 - i))
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t.dailyTrend}</CardTitle>
      </CardHeader>
      <CardContent elevated={!loading && daily.length === 0}>
        {loading ? (
          <Skeleton className="h-64 w-full" />
        ) : daily.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.noDailyTrend}</p>
        ) : (
          <CategoryTrendChart
            buckets={dailyBuckets(daily, days)}
            categories={categoriesIn(daily, categories)}
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
  loading,
  upcoming,
  categories,
}: {
  loading: boolean
  upcoming: { r: Recurring; date: string }[]
  categories: Category[]
}) {
  const [selected, setSelected] = useState<Date | undefined>(new Date())
  const dayColors = Object.fromEntries(
    upcoming.map((u) => [
      u.date,
      paletteVar(colorOf(categories, u.r.category_id), "primary"),
    ])
  )

  return (
    <Card className="@container">
      <CardHeader>
        <CardTitle>{t.upcomingRecurring}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 @[28rem]:flex-row @[28rem]:items-start">
        {loading ? (
          <Skeleton className="h-64 w-full" />
        ) : (
          <>
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
                        className="size-1.5 shrink-0 -translate-y-px rounded-full"
                        style={{
                          backgroundColor: paletteVar(
                            colorOf(categories, r.category_id),
                            "primary"
                          ),
                        }}
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
          </>
        )}
      </CardContent>
    </Card>
  )
}

// Unpaid invoiced Incomes (any month), and Clients with an active Contract
// not yet fully invoiced — the two lists pendingPayments already computes
// (cmd/pending.go), as names rather than the full rows Month.tsx shows. Each
// name gets a HoverCard with the detail the badge itself has no room for —
// amount_cents/days_waiting are already in this same response (ticket 13);
// total_earned_cents is deliberately left out, since it lives on /api/clients,
// a request this card has never made and shouldn't start making just for a
// hover detail.
//
// The second list used to be readNotYetInvoiced's own billing-recency guess
// (habitually billed lately, nothing yet this month) — too easy to both miss
// a Client who invoices irregularly and nudge one who is simply ahead of
// schedule. contracts_due_this_month is a plain fact instead: an active
// Contract whose total isn't fully accounted for yet. Month.tsx's own
// PendingPayments section is untouched — the old heuristic still answers
// there, since nothing asked to change it.
function ClientsCard({
  pending,
  clockOK,
  onEditIncome,
  onPaid,
}: {
  pending: Pending
  clockOK: boolean
  onEditIncome: (id: number) => void
  onPaid: () => Promise<void>
}) {
  // Every unpaid Income, same list readOutstanding answers — an Income with
  // no Client and one with no invoice date are both money genuinely owed
  // (ADR-0003), so neither is a reason to leave it off the one screen that
  // says what is owed. category fills in for a name when there is no Client,
  // matching how PendingPayments.tsx (Month.tsx) already reads this list.
  const waiting = pending.outstanding
  const dueThisMonth = pending.contracts_due_this_month
  const totalDueCents = dueThisMonth.reduce((sum, c) => sum + c.due_cents, 0)
  const invoicesToDo = dueThisMonth
  const [error, setError] = useState("")

  // clockOK false means the Pi's own clock cannot be trusted, and today()
  // reads the browser regardless — but a badge tapped from across the room
  // is not the moment to gamble a payment date the household never chose.
  // The edit form is where it types the date itself instead.
  async function markPaid(id: number) {
    if (!clockOK) {
      onEditIncome(id)
      return
    }
    setError("")
    try {
      await api(`/api/incomes/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ payment_date: today() }),
      })
      await onPaid()
    } catch {
      setError(t.incomeNotSaved)
    }
  }

  return (
    <Card className="@container">
      <CardHeader>
        <CardTitle>{t.clientsThisMonth}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        {/* Only shown once a Contract is actually active — empty here means
            no Contract currently covers this month, not that nothing is
            owed (Contracts and pending Incomes are unrelated numbers). The
            per-Client breakdown is folded into Fatture da fare below. */}
        {totalDueCents > 0 && (
          <p className="text-sm">
            <span className="text-muted-foreground">{t.dueThisMonth}:</span>{" "}
            <span className="font-medium tabular-nums">
              € {formatCents(totalDueCents)}
            </span>
          </p>
        )}
        <div className="flex flex-col gap-3 @[28rem]:flex-row @[28rem]:items-start">
          <ClientAlert
            title={t.pendingPayments}
            okText={t.noPendingClients}
            clients={waiting.map((o) => ({
              key: o.id,
              name: `${o.client || o.category}: € ${formatCents(o.amount_cents)}`,
              date: formatDate(o.waiting_since),
              amountCents: o.amount_cents,
              daysWaiting: o.days_waiting,
            }))}
            // The amount is inline in the name itself now — this HoverCard's
            // only job left is the one thing that name has no room for.
            hoverContent={(c) => (
              <p className="text-xs text-muted-foreground">
                {t.waitingDays(c.daysWaiting)}
              </p>
            )}
            action={(c) => (
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t.markPaidLabel(c.name)}
                onClick={() => void markPaid(c.key as number)}
              >
                <RiCheckLine />
              </Button>
            )}
          />
          <ClientAlert
            title={t.invoicesSentTitle}
            okText={t.invoicesAllSent}
            clients={invoicesToDo.map((c) => ({
              key: c.client_id,
              name: `${c.client}: € ${formatCents(c.due_cents)}`,
            }))}
            // Nothing else travels with this list — the HoverCard surfaces
            // why the Client is listed at all, the same explanation
            // Month.tsx's own pending-payments section gives once for the
            // whole list.
            hoverContent={() => (
              <p className="text-xs text-muted-foreground">
                {t.contractsDueHint}
              </p>
            )}
          />
        </div>
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
  action,
}: {
  title: string
  okText: string
  clients: T[]
  hoverContent: (client: T) => ReactNode
  action?: (client: T) => ReactNode
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
                {action?.(c)}
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

// The card's three management actions, each a mode rather than a command:
// picking one reveals what it needs (the create form, or a control on every
// row), picking it again puts it away. Only one at a time — renaming and
// deleting the same row in one gesture is not a thing to offer.
type ReminderMode = "add" | "rename" | "delete"

const reminderModes = [
  { mode: "add", icon: RiAddLine, label: t.addReminder, badge: t.reminderModeAdd },
  { mode: "rename", icon: RiEditLine, label: t.rename, badge: t.rename },
  { mode: "delete", icon: RiDeleteBinLine, label: t.delete, badge: t.delete },
] as const satisfies {
  mode: ReminderMode
  icon: typeof RiAddLine
  label: string
  badge: string
}[]

// A household-managed list of on/off toggles for a manual action the app
// doesn't automate (ticket 06) — created, toggled, renamed and deleted right
// here. Nothing else in the app references a Reminder, so there is no separate
// management screen behind this card, and enabled already comes back
// collapsed from the server: this component never compares months itself.
//
// By default the card is only the list: the toggles are what gets read every
// day, and the managing controls hide behind the header menu so they are not
// three chances to mistap next to them.
function RemindersCard({ loading }: { loading: boolean }) {
  const [reminders, setReminders] = useState<Reminder[]>([])
  const [mode, setMode] = useState<ReminderMode | null>(null)
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

  // Sends only the label, so the toggle the row is currently in keeps
  // whatever state it had — the PATCH treats an absent field as untouched.
  async function rename(reminder: Reminder, to: string) {
    if (to.trim() === "" || to.trim() === reminder.label) return
    await write(`/api/reminders/${reminder.id}`, {
      method: "PATCH",
      body: JSON.stringify({ label: to }),
    })
  }

  const active = reminderModes.find((m) => m.mode === mode)

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t.reminders}</CardTitle>
        {/* The checkmark inside the menu is only visible once it is open, so
            the active mode says so out here too — otherwise the row inputs
            (or the Elimina buttons) are the only clue anything is on. */}
        <CardAction className="flex items-center gap-1.5">
          {active && (
            <Badge variant="secondary">
              <active.icon />
              {active.badge}
            </Badge>
          )}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t.actions}
              >
                <RiMoreLine />
              </Button>
            </DropdownMenuTrigger>
            {/* Same reason Categories.tsx prevents it: Radix hands focus back
                to the "..." trigger as it closes, which would pull it straight
                off the row inputs Rinomina just rendered. */}
            <DropdownMenuContent
              align="end"
              onCloseAutoFocus={(event) => event.preventDefault()}
            >
              {reminderModes.map(({ mode: item, icon: Icon, label: text }) => (
                <DropdownMenuCheckboxItem
                  key={item}
                  checked={mode === item}
                  onCheckedChange={(on) => setMode(on ? item : null)}
                >
                  <Icon /> {text}
                </DropdownMenuCheckboxItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </CardAction>
      </CardHeader>
      <CardContent
        elevated={!loading && reminders.length === 0}
        className="flex flex-col gap-3"
      >
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        {loading ? (
          <Skeleton className="h-32 w-full" />
        ) : reminders.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.noRemindersYet}</p>
        ) : (
          <ul className="flex flex-col divide-y divide-border">
            {reminders.map((r) => (
              <li
                key={r.id}
                className="flex items-center justify-between gap-3 py-2"
              >
                {mode === "rename" ? (
                  // Keyed on the stored label so a rename the server trimmed
                  // or refused remounts the field on what is actually saved,
                  // rather than leaving it showing what was typed.
                  <Input
                    key={r.label}
                    defaultValue={r.label}
                    className="min-w-0 flex-1"
                    aria-label={t.rename}
                    onBlur={(event) => void rename(r, event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") event.currentTarget.blur()
                      if (event.key === "Escape") {
                        event.currentTarget.value = r.label
                        event.currentTarget.blur()
                      }
                    }}
                  />
                ) : (
                  /* A single independent on/off per row, not a segmented
                     set — any number of Reminders can be enabled at once —
                     so Toggle, not ToggleGroup (ticket 14). Frozen while
                     Elimina is showing: reaching for a row that is about to
                     be deleted should not also flip it. */
                  <Toggle
                    size="sm"
                    className="min-w-0 flex-1 justify-start gap-2 px-2.5 data-[state=on]:bg-credit/10 data-[state=on]:text-credit dark:data-[state=on]:bg-credit/20"
                    pressed={r.enabled}
                    disabled={mode === "delete"}
                    onPressedChange={(pressed) =>
                      void write(`/api/reminders/${r.id}`, {
                        method: "PATCH",
                        body: JSON.stringify({ enabled: pressed }),
                      })
                    }
                    aria-label={t.reminderToggleLabel(r.label, r.enabled)}
                  >
                    {r.enabled ? <RiCheckboxCircleFill /> : <RiCheckboxBlankCircleLine />}
                    <span className="truncate text-sm font-normal">
                      {r.label}
                    </span>
                  </Toggle>
                )}
                {mode === "delete" && (
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
                )}
              </li>
            ))}
          </ul>
        )}
        {mode === "add" && (
          <form
            onSubmit={(event) => void add(event)}
            className="flex flex-col gap-2"
          >
            <Input
              autoFocus
              value={label}
              onChange={(event) => setLabel(event.target.value)}
              placeholder={t.reminderLabel}
            />
            <Button type="submit" size="sm">
              {t.addReminder}
            </Button>
          </form>
        )}
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
  loading,
  title,
  empty,
  entries,
}: {
  loading: boolean
  title: string
  empty: string
  entries: RecentEntry[]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent elevated={!loading && entries.length === 0}>
        {loading ? (
          <Skeleton className="h-48 w-full" />
        ) : entries.length === 0 ? (
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

// Budget (computed), Target and Goal (household-set) — ticket 04. They
// describe the current month, so they live here with the rest of "now"
// rather than on the Month page, where a past month has no target to show.
// Own fetch, outside Dashboard's Promise.all: a budget that fails to load
// just leaves the card out instead of blanking the whole overview.
function BudgetCard() {
  const [budget, setBudget] = useState<BudgetReport | null>(null)
  useEffect(() => {
    apiJSON<BudgetReport>("/api/reports/budget")
      .then(setBudget)
      .catch(() => setBudget(null))
  }, [])
  if (!budget) return null
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t.budgetTargetGoal}</CardTitle>
      </CardHeader>
      <CardContent>
        {/* Target/Goal show even when Budget has too little history: they
            are just settings, only Budget's own row waits for data. */}
        <dl className="flex flex-col divide-y divide-border">
          {budget.available ? (
            <Row label={t.budget} cents={budget.budget_cents} />
          ) : (
            <p className="py-3 text-sm text-muted-foreground">
              {t.budgetUnavailable}
            </p>
          )}
          <Row label={t.target} cents={budget.target_cents} />
          <Row label={t.savingsGoal} cents={budget.goal_cents} />
        </dl>
      </CardContent>
    </Card>
  )
}
