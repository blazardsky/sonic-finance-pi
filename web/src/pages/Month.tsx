import { useEffect, useState } from "react"

import { PendingPayments } from "@/pages/PendingPayments"
import { Figure } from "@/pages/YearlyReport"
import { CategoryTrendChart } from "@/components/CategoryTrendChart"
import { SpoilerAmount } from "@/components/SpoilerAmount"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { apiJSON } from "@/lib/api"
import { formatCents, formatDate, shiftMonth, thisMonth } from "@/lib/money"
import { t } from "@/lib/strings"
import { categoriesIn, weeklyBuckets } from "@/lib/trend"
import type {
  BudgetReport,
  DailyCategoryTotal,
  MonthTotals,
  RecentEntry,
} from "@/types"

// The screen the app opens on: where does this month stand. Three numbers,
// where the money went, and the last few things typed — the totals answer the
// question, the breakdown says what made it, and the strip at the bottom
// confirms what was just entered.
//
// The two arrows step through the months either side, and the month itself is
// a native picker: a comparison against a month three years back should not be
// thirty-six taps of an arrow. Both ways of moving stop at the current month,
// because there is no such thing as next month's total.
//
// Which month it is is the browser's answer, not the server's: the phone has a
// correct clock and the Pi has no RTC. The server is never asked which month
// this is — it reports whatever month the path names.
export function Month() {
  const [month, setMonth] = useState(thisMonth)
  const [totals, setTotals] = useState<MonthTotals | null>(null)
  const [recent, setRecent] = useState<RecentEntry[]>([])
  const [daily, setDaily] = useState<DailyCategoryTotal[]>([])
  const [budget, setBudget] = useState<BudgetReport | null>(null)
  const [error, setError] = useState("")

  // The last few entries are not this month's — they are the last few typed,
  // whatever month they landed in — so this fetch does not depend on the
  // month and is not repeated when the household steps through the months.
  // A failure here leaves the list empty: the three totals are the screen,
  // and losing the confirmation strip is not worth blanking them for.
  useEffect(() => {
    apiJSON<RecentEntry[]>("/api/reports/recent")
      .then(setRecent)
      .catch(() => setRecent([]))
  }, [])

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

  // The weekly chart's own fetch, independent of the totals above: a slow
  // month arriving after the household has stepped on must not repaint the
  // chart under the next month's heading either.
  useEffect(() => {
    let current = true
    apiJSON<DailyCategoryTotal[]>(`/api/reports/month/${month}/daily`)
      .then((d) => current && setDaily(d))
      .catch(() => current && setDaily([]))
    return () => {
      current = false
    }
  }, [month])

  // Budget/Target/Goal describe "now," not an arbitrary month (ticket 04) —
  // fetched only while the current real month is on screen, and cleared the
  // moment the household steps away from it so a past month never shows this
  // month's figures under its own heading.
  useEffect(() => {
    let current = true
    if (month === thisMonth()) {
      apiJSON<BudgetReport>("/api/reports/budget")
        .then((b) => current && setBudget(b))
        .catch(() => current && setBudget(null))
    } else {
      // setState belongs in a callback, not the effect body itself — deferred
      // the same way the fetch above defers it, just with nothing to await.
      Promise.resolve().then(() => current && setBudget(null))
    }
    return () => {
      current = false
    }
  }, [month])

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
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
          className="h-10 w-auto text-center"
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

      {/* Every card below is free to sit next to whatever fits beside it —
          nothing here is pinned to a fixed column count, so a wide screen
          fills with stat-shaped cards side by side while a narrow one just
          stacks them, same content either way. */}
      <div className="flex flex-wrap items-start gap-6">
        {totals && (
          <Card className="min-w-72 flex-1">
            <CardContent>
              {/* Three small boxes, the same reading YearlyReport's own
                  figures use, rather than the plain label/value rows every
                  other card here still is — the totals get to look like the
                  answer the screen opens on. */}
              <dl className="grid grid-cols-3 gap-3">
                <Figure label={t.incomes} cents={totals.income_cents} />
                <Figure label={t.expenses} cents={totals.expense_cents} />
                <Figure label={t.difference} cents={totals.net_cents} />
              </dl>
            </CardContent>
          </Card>
        )}

        {/* Budget (computed), Target and Goal (household-set) — ticket 04.
            Only for the current real month: a past or future month has no
            "this month's target" to speak of. Target/Goal show even when
            Budget itself has too little history, since they are just
            settings; only Budget's own row waits for enough data. */}
        {budget && (
          <Card className="min-w-64 flex-1">
            <CardHeader>
              <CardTitle>{t.budgetTargetGoal}</CardTitle>
            </CardHeader>
            <CardContent>
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
        )}

        {/* What is owed and what looks unbilled: it is a notice rather than a
            report, and a forgotten invoice is worth more than knowing which
            Category the shopping went under. It renders nothing when nothing
            is pending. */}
        <PendingPayments className="min-w-72 flex-1" />

        {/* Where the money went. Every line is real spend — the breakdown
            adds up to the expense total above it, with an itemised shop split
            across its Categories and no "uncategorised" line to absorb a
            remainder. */}
        {totals && (
          <Card className="min-w-72 flex-1">
            <CardHeader>
              <CardTitle>{t.byCategory}</CardTitle>
            </CardHeader>
            <CardContent>
              {totals.by_category.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {t.nothingSpent}
                </p>
              ) : (
                <dl className="flex flex-col divide-y divide-border">
                  {totals.by_category.map((line) => (
                    <div
                      key={line.category_id}
                      className="flex items-baseline justify-between gap-3 py-2"
                    >
                      <dt className="truncate">{line.category}</dt>
                      <dd className="whitespace-nowrap tabular-nums">
                        € {formatCents(line.amount_cents)}
                      </dd>
                    </div>
                  ))}
                </dl>
              )}
            </CardContent>
          </Card>
        )}

        {/* Weekly, not daily: a month's worth of daily bars would be too thin
            to read on a phone. Bucketed here rather than by a second
            endpoint — a week is a client-side grouping of the same daily rows
            the by_category breakdown above is drawn from (cmd/reports.go).
            An edge week that only partly falls in this month keeps its real
            date range, per the ticket, rather than being merged into a
            neighbour. A wide min-width so it claims a full row on its own at
            most screen sizes — a bar chart wants the room a stat card doesn't
            need. */}
        {daily.length > 0 && (
          <Card className="min-w-full flex-[2] lg:min-w-[32rem]">
            <CardHeader>
              <CardTitle>{t.weeklyTrend}</CardTitle>
            </CardHeader>
            <CardContent>
              <CategoryTrendChart
                buckets={weeklyBuckets(daily)}
                categories={categoriesIn(daily)}
              />
            </CardContent>
          </Card>
        )}

        {/* The confirmation strip: what was just typed, newest first. An
            Expense reads as money out and an Income as money in, which is the
            one thing about an entry this list has to get across.

            An Income with no date is one that has not been paid, and it gets
            neither a sign nor full-strength text: it sits directly under a
            money in total that excludes it, and a "+ € 800,00" here would be
            ADR-0003's named bug rendered on screen. */}
        <Card className="min-w-72 flex-1">
          <CardHeader>
            <CardTitle>{t.recentEntries}</CardTitle>
          </CardHeader>
          <CardContent>
            {recent.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t.noEntriesYet}</p>
            ) : (
              <ul className="flex flex-col divide-y divide-border">
                {recent.map((entry) => (
                  <li
                    key={`${entry.direction}-${entry.id}`}
                    className="flex items-baseline justify-between gap-3 py-2"
                  >
                    <span className="min-w-0">
                      <span className="truncate">{entry.category}</span>{" "}
                      <span className="text-xs text-muted-foreground">
                        {entry.date ? formatDate(entry.date) : t.notPaidYet}
                      </span>
                    </span>
                    <span
                      className={`whitespace-nowrap tabular-nums ${
                        !entry.date
                          ? "text-muted-foreground"
                          : entry.direction === "expense"
                            ? "text-destructive"
                            : "text-credit"
                      }`}
                    >
                      {entry.date &&
                        (entry.direction === "expense" ? "−" : "+")}{" "}
                      <SpoilerAmount
                        cents={entry.amount_cents}
                        gift={entry.is_gift}
                      />
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

// One label and one amount, the way both report screens read a total: the big
// one is the answer the screen exists to give, and the only number that can be
// negative — which the sign and the colour both say.
export function Row({
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
