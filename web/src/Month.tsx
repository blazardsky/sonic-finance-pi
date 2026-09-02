import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { apiJSON } from "@/api"
import { formatCents, shiftMonth, thisMonth } from "@/money"
import { t } from "@/strings"
import type { MonthTotals } from "@/types"

// The screen the app opens on: where does this month stand. Three numbers, the
// two arrows that step through the months either side, and the month itself as
// a native picker — a comparison against a month three years back should not
// be thirty-six taps of an arrow. Both ways of moving stop at the current
// month, because there is no such thing as next month's total.
//
// Which month it is is the browser's answer, not the server's: the phone has a
// correct clock and the Pi has no RTC. The server is never asked which month
// this is — it reports whatever month the path names.
export function Month() {
  const [month, setMonth] = useState(thisMonth)
  const [totals, setTotals] = useState<MonthTotals | null>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    let current = true
    apiJSON<MonthTotals>(`/api/reports/month/${month}`)
      // A slow month arriving after the household has already tapped on to
      // the next one must not overwrite it.
      .then((m) => {
        if (!current) return
        setTotals(m)
        setError("")
      })
      .catch(() => {
        if (!current) return
        // The numbers go with the month name: last month's three totals left
        // sitting under this month's heading would be a wrong answer rather
        // than a missing one.
        setTotals(null)
        setError(t.serverUnreachable)
      })
    return () => {
      current = false
    }
  }, [month])

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          aria-label={t.previousMonth}
          onClick={() => setMonth(shiftMonth(month, -1))}
        >
          ‹
        </Button>
        <Input
          type="month"
          aria-label={t.month}
          value={month}
          // An empty value is what a cleared picker sends, and there is no
          // month it could mean — the one on screen stays.
          onChange={(e) => e.target.value && setMonth(e.target.value)}
          max={thisMonth()}
          className="h-10 text-center"
        />
        <Button
          variant="ghost"
          size="sm"
          aria-label={t.nextMonth}
          // String comparison is a date comparison here: YYYY-MM sorts
          // correctly, which is the whole reason months are stored that way.
          disabled={month >= thisMonth()}
          onClick={() => setMonth(shiftMonth(month, 1))}
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
          {/* The two totals borrow the nav's own words for the screens they
              are the sum of: one concept, one word. */}
          <Row label={t.incomes} cents={totals.income_cents} />
          <Row label={t.expenses} cents={totals.expense_cents} />
          {/* The difference is the answer, so it is the big one — and the only
              number here that can be negative, which the sign and the colour
              both say. */}
          <Row label={t.difference} cents={totals.net_cents} big />
        </dl>
      )}
    </div>
  )
}

function Row({
  label,
  cents,
  big,
}: {
  label: string
  cents: number
  big?: boolean
}) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-3">
      <dt className={big ? "font-medium" : "text-muted-foreground"}>{label}</dt>
      <dd
        className={`tabular-nums ${big ? "text-2xl font-medium" : "text-lg"} ${
          cents < 0 ? "text-destructive" : ""
        }`}
      >
        € {formatCents(cents)}
      </dd>
    </div>
  )
}
