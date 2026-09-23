import { useCallback, useEffect, useState, type ReactNode } from "react"

import { RiEditLine } from "@remixicon/react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  createDataTableColumnHelper,
  DataTable,
  type DataTableColumnDef,
} from "@/components/data-table"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Progress } from "@/components/ui/progress"
import { apiJSON } from "@/lib/api"
import { formatCents, thisYear, toCents, toTyped } from "@/lib/money"
import { t } from "@/lib/strings"
import type { HoldingBreakdown, HoldingType, Lists, SavingsReport } from "@/types"

// Same labels Holdings.tsx uses for the fixed type list — each screen keeps
// its own copy rather than sharing one, the existing convention here.
const typeLabels: Record<HoldingType, string> = {
  etf: t.holdingTypeETF,
  crypto: t.holdingTypeCrypto,
  stock: t.holdingTypeStock,
  bond: t.holdingTypeBond,
  other: t.holdingTypeOther,
}

// The Risparmi page: a read-only recap, no entries added here except the net
// worth target, set by hand where its progress is read (the starting balance
// is set once, in Impostazioni) — the total Savings figure and the portfolio breakdown (ticket 07),
// both off the one /api/reports/savings request. Buying and selling a Holding
// happens on the Investimenti page instead.
export function Savings() {
  const [error, setError] = useState("")

  const [savings, setSavings] = useState<SavingsReport | null>(null)

  const load = useCallback(
    () =>
      apiJSON<SavingsReport>("/api/reports/savings")
        .then(setSavings)
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // Today's portfolio value and a flat-5%/year projection of it — a
  // projection, not a promise, same spirit as the yearly Estimate
  // (ADR-0010). Derived from the holdings breakdown already on hand, not a
  // second endpoint: the figure is a sum recharted client-side, nothing this
  // page needs the server's help to compute.
  const investmentsTotalCents = (savings?.holdings ?? []).reduce(
    (sum, h) => sum + h.net_cents,
    0
  )
  const estimatedIn20YearsCents = Math.round(
    investmentsTotalCents * Math.pow(1.05, 20)
  )

  // The target is aimed at everything the household holds — Savings plus the
  // portfolio, the same combined figure the first card already shows — not at
  // the uninvested part alone.
  const netWorthCents = savings?.combined_cents ?? 0
  const targetCents = savings?.net_worth_target_cents ?? 0
  const yearlySavingsCents = savings?.yearly_savings_cents ?? 0
  const yearsToTarget =
    targetCents > netWorthCents && yearlySavingsCents > 0
      ? Math.ceil((targetCents - netWorthCents) / yearlySavingsCents)
      : null

  return (
    <div className="@container mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.savings}</h1>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {/* One card per row when there is only room for one, two by two once
          there is, four across once there is room for that — measured
          against this column's own width rather than the window's, which
          says nothing about how wide the column is with a sidebar open. */}
      <div className="grid grid-cols-1 gap-4 @lg:grid-cols-2 @5xl:grid-cols-4">
        <Card>
          <CardHeader>
            <CardTitle>{t.savings}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-1">
            <p className="text-2xl font-medium tabular-nums">
              € {formatCents(savings?.savings_cents ?? 0)}
            </p>
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              {t.savingsPlusPortfolio}
              <Badge variant="secondary" className="tabular-nums">
                € {formatCents(savings?.combined_cents ?? 0)}
              </Badge>
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t.investments}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-1">
            <p className="text-2xl font-medium tabular-nums">
              € {formatCents(investmentsTotalCents)}
            </p>
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              {t.estimatedIn20Years}
              <Badge variant="secondary" className="tabular-nums">
                € {formatCents(estimatedIn20YearsCents)}
              </Badge>
            </p>
          </CardContent>
        </Card>

        <AmountSettingCard
          title={t.netWorthTarget}
          cents={targetCents}
          field="net_worth_target_cents"
          hint={t.netWorthTargetHint}
          editLabel={t.editNetWorthTarget}
          errorText={t.netWorthTargetNotSaved}
          onSaved={load}
        >
          <p className="text-sm text-muted-foreground">
            {targetCents === 0
              ? t.noNetWorthTarget
              : yearsToTarget !== null
                ? t.willGetThereIn(yearsToTarget)
                : netWorthCents >= targetCents
                  ? t.netWorthTargetReached
                  : t.netWorthTargetNoPace}
          </p>
          {targetCents > 0 && (
            <>
              <div className="relative">
                <Progress
                  aria-label={t.netWorthTarget}
                  value={percentOf(netWorthCents, targetCents)}
                />
                {yearMilestones(netWorthCents, targetCents, yearlySavingsCents).map(
                  (m) => (
                    <span
                      key={m.year}
                      aria-hidden
                      title={t.netWorthMilestone(m.year, formatCents(m.cents))}
                      className="absolute top-0 h-1 w-0.5 -translate-x-1/2 bg-card"
                      style={{ left: `${percentOf(m.cents, targetCents)}%` }}
                    />
                  )
                )}
              </div>
              <p className="text-xs text-muted-foreground tabular-nums">
                {t.yearlySavingsPace(formatCents(yearlySavingsCents))}
              </p>
            </>
          )}
        </AmountSettingCard>
      </div>

      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-medium">{t.portfolio}</h2>
        <DataTable
          columns={portfolioColumns}
          data={savings?.holdings ?? []}
          getRowId={(h) => String(h.holding_id)}
        />
      </div>
    </div>
  )
}

// v1.3.0: DataTable's own sortable headers and per-column filters. Holding,
// Type, Amount, % plus current value and gain/loss, both null (blank) on a
// Holding with no current_price_cents typed in (computeHoldingFigures).
// Read-only, same as Investments.tsx: no Actions column, this is a derived
// recap.
const portfolioColumns: DataTableColumnDef<HoldingBreakdown>[] = (() => {
  const helper = createDataTableColumnHelper<HoldingBreakdown>()
  return [
    helper.accessor("name", {
      header: t.holding,
      meta: { filterVariant: "select" },
    }),
    helper.accessor((h): string => typeLabels[h.type], {
      id: "type",
      header: t.holdingType,
      meta: { filterVariant: "select" },
    }),
    helper.accessor("net_cents", {
      header: t.amount,
      meta: { filterVariant: "range" },
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          € {formatCents(row.original.net_cents)}
        </div>
      ),
    }),
    helper.accessor("percent", {
      header: "%",
      meta: { filterVariant: "range" },
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {row.original.percent.toFixed(1)}%
        </div>
      ),
    }),
    helper.accessor("value_now_cents", {
      header: t.valueNow,
      meta: { filterVariant: "range" },
      cell: ({ row }) => (
        <div className="text-right tabular-nums">
          {row.original.value_now_cents === null
            ? ""
            : `€ ${formatCents(row.original.value_now_cents)}`}
        </div>
      ),
    }),
    helper.accessor("gain_loss_cents", {
      header: t.gainLoss,
      meta: { filterVariant: "range" },
      cell: ({ row }) => {
        const { gain_loss_cents: cents, gain_loss_percent: percent } = row.original
        return (
          <div className="text-right tabular-nums">
            {cents === null
              ? ""
              : `€ ${formatCents(cents)}${percent === null ? "" : ` (${percent.toFixed(1)}%)`}`}
          </div>
        )
      },
    }),
  ]
})()

// percentOf is a share of the target as a 0–100 Progress value: negative
// Savings is 0% rather than a bar running backwards, and a figure past the
// target is 100% rather than one running off the end.
function percentOf(cents: number, targetCents: number) {
  return Math.max(0, Math.min(100, (cents / targetCents) * 100))
}

// milestoneLimit is how many year markers the bar will draw. Past it they
// would be a solid block rather than milestones — and the years are stated
// in words above the bar either way.
const milestoneLimit = 40

// One marker per year between here and the target, each the net worth the
// current pace reaches by the end of that year; the last one is the year the
// target is met. Empty when there is nothing to project — no target, one
// already reached, or nothing being saved, which never arrives at all.
//
// ponytail: a flat pace, no compounding — this is "keep doing exactly what
// you are doing", not the Investimenti card's 5%/year growth. The upgrade
// path is the same Math.pow that card already uses.
function yearMilestones(
  fromCents: number,
  targetCents: number,
  yearlyCents: number
) {
  if (targetCents <= fromCents || yearlyCents <= 0) return []
  const years = Math.ceil((targetCents - fromCents) / yearlyCents)
  if (years > milestoneLimit) return []
  return Array.from({ length: years }, (_, i) => ({
    year: Number(thisYear()) + i + 1,
    cents: fromCents + yearlyCents * (i + 1),
  }))
}

// A Card whose one figure is a household-set setting, edited where it is
// read: the amount with a pencil beside it, and the same amount form behind
// the pencil. The net worth target is the one setting this page owns.
//
// The typed value is seeded when the form opens rather than kept in step
// with cents as it loads, so there is nothing to re-sync after a save: the
// reload the parent runs is what puts the new figure on screen.
function AmountSettingCard({
  title,
  cents,
  field,
  hint,
  editLabel,
  errorText,
  onSaved,
  children,
}: {
  title: string
  cents: number
  field: "net_worth_target_cents"
  hint: string
  editLabel: string
  errorText: string
  onSaved: () => Promise<void>
  children?: ReactNode
}) {
  const [typed, setTyped] = useState("")
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState("")
  const [message, setMessage] = useState("")

  // Both fields ride the same /api/settings payload as Target and Goal
  // (cmd/setting.go) — sending just this one leaves payers, payment methods,
  // Target and Goal exactly as they were read.
  async function save(event: React.FormEvent) {
    event.preventDefault()
    setError("")
    setMessage("")
    const value = toCents(typed)
    if (value === null) {
      setError(t.invalidAmount)
      return
    }
    try {
      await apiJSON<Lists>("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ [field]: value }),
      })
      setMessage(t.listsSaved)
      setEditing(false)
      await onSaved()
    } catch {
      setError(errorText)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-1.5">
        {editing ? (
          <form onSubmit={save} className="flex flex-col gap-1.5">
            <div className="flex gap-2">
              <InputGroup className="h-9">
                <InputGroupAddon>€</InputGroupAddon>
                <InputGroupInput
                  inputMode="decimal"
                  value={typed}
                  onChange={(e) => setTyped(e.target.value)}
                />
              </InputGroup>
              <Button type="submit" variant="outline">
                {t.save}
              </Button>
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setError("")
                  setMessage("")
                  setEditing(false)
                }}
              >
                {t.cancel}
              </Button>
            </div>
            <span className="text-xs text-muted-foreground">{hint}</span>
          </form>
        ) : (
          <div className="flex items-center gap-2">
            <p className="text-2xl font-medium tabular-nums">
              € {formatCents(cents)}
            </p>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={editLabel}
              onClick={() => {
                setError("")
                setMessage("")
                setTyped(toTyped(cents))
                setEditing(true)
              }}
            >
              <RiEditLine />
            </Button>
          </div>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        {message && (
          <p role="status" className="text-sm text-muted-foreground">
            {message}
          </p>
        )}
        {children}
      </CardContent>
    </Card>
  )
}
