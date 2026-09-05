import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api, apiJSON } from "@/lib/api"
import { DatePicker } from "@/components/date-picker"
import { FormSidebar } from "@/components/form-sidebar"
import { formatCents, formatDate, toCents, toTyped, today } from "@/lib/money"
import { nameOf, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type {
  Category,
  Expense,
  Holding,
  HoldingType,
  Income,
  Lists,
  SavingsReport,
} from "@/types"

// Same labels Holdings.tsx uses for the fixed type list — each screen keeps
// its own copy rather than sharing one, the existing convention here.
const typeLabels: Record<HoldingType, string> = {
  etf: t.holdingTypeETF,
  crypto: t.holdingTypeCrypto,
  stock: t.holdingTypeStock,
  bond: t.holdingTypeBond,
  other: t.holdingTypeOther,
}

type Kind = "buy" | "sell"

// What the form holds while typing: a Holding not yet chosen is "", the same
// bargain an unchosen Category makes on Expenses.tsx.
type Draft = {
  holding_id: number | ""
  amount: string
  date: string
  payer: string
}

const blankDraft = (): Draft => ({
  holding_id: "",
  amount: "",
  date: today(),
  payer: "",
})

// One row of the confirmation list below the form — a buy is an Expense, a
// sell is an Income, and this is the one shape both are read into so they can
// share a table. id is scoped to its own kind, so the two are never the same
// row even when the numbers collide.
type Transaction = {
  key: string
  kind: Kind
  holding_id: number
  amount_cents: number
  date: string
}

// The Savings page: a dedicated Buy/Sell form so nobody has to remember to
// pick Investments and a Holding on the generic Expense/Income screens. It
// posts to those same two endpoints (spec's Implementation Decisions) — there
// is no buy/sell endpoint of its own, and nothing here is excluded from the
// ordinary Month/Year/Dashboard/Recent totals (ADR-0009).
export function Savings() {
  const [holdings, setHoldings] = useState<Holding[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [lists, setLists] = useState<Lists>({
    payers: [],
    payment_methods: [],
    target_cents: 0,
    goal_cents: 0,
    savings_starting_balance_cents: 0,
  })
  const [records, setRecords] = useState<Transaction[]>([])
  const [error, setError] = useState("")

  const [kind, setKind] = useState<Kind>("buy")
  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [sidebarOpen, setSidebarOpen] = useState(true)

  // The computed total and the portfolio breakdown (ticket 07) — one report,
  // read alongside everything else this page already loads.
  const [savings, setSavings] = useState<SavingsReport | null>(null)
  const [startingBalance, setStartingBalance] = useState("")
  const [balanceError, setBalanceError] = useState("")
  const [balanceMessage, setBalanceMessage] = useState("")

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Holding[]>("/api/holdings"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Lists>("/api/settings"),
        apiJSON<Expense[]>("/api/expenses"),
        apiJSON<Income[]>("/api/incomes"),
        apiJSON<SavingsReport>("/api/reports/savings"),
      ])
        .then(([h, c, l, expenses, incomes, s]) => {
          setHoldings(h)
          setCategories(c)
          setLists(l)
          setSavings(s)
          setStartingBalance(toTyped(s.starting_balance_cents))
          setRecords(
            [
              ...expenses
                .filter((e) => e.holding_id !== null)
                .map(
                  (e): Transaction => ({
                    key: `buy-${e.id}`,
                    kind: "buy",
                    holding_id: e.holding_id!,
                    amount_cents: e.amount_cents,
                    date: e.occurred_on,
                  })
                ),
              ...incomes
                .filter((i) => i.holding_id !== null)
                .map(
                  (i): Transaction => ({
                    key: `sell-${i.id}`,
                    kind: "sell",
                    holding_id: i.holding_id!,
                    // A sell recorded here always carries a payment date (see
                    // submit below), so this is never the unpaid "".
                    amount_cents: i.amount_cents,
                    date: i.payment_date,
                  })
                ),
            ].sort((a, b) => b.date.localeCompare(a.date))
          )
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // The only Base category that applies to both sides — Taxes and Freelance
  // and Gift are each one side only (Expenses.tsx makes the same distinction
  // for Taxes). `code` itself is deliberately not published; this is what
  // resolves Investments without it.
  const investments = categories.find(
    (c) => c.base && c.applies_to === "both"
  )

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setError("")

    const cents = toCents(draft.amount)
    if (cents === null || cents <= 0) {
      setError(t.invalidAmount)
      return
    }
    if (draft.holding_id === "" || !investments) return

    try {
      await api(kind === "buy" ? "/api/expenses" : "/api/incomes", {
        method: "POST",
        // A sell's date is its payment_date, not just an invoice_sent_date:
        // ADR-0003 counts an Income toward nothing until it has one, and this
        // form's whole point is to show up like any other transaction (spec's
        // Implementation Decisions), not to sit "in attesa".
        body: JSON.stringify({
          [kind === "buy" ? "occurred_on" : "payment_date"]: draft.date,
          amount_cents: cents,
          category_id: investments.id,
          payer: draft.payer,
          holding_id: draft.holding_id,
        }),
      })
    } catch {
      setError(t.buySellNotSaved)
      return
    }
    // The Holding and Payer are often the same for the next entry — a PAC
    // contribution split across a few lines, say — so only the amount clears.
    setDraft((d) => ({ ...d, amount: "" }))
    await load()
  }

  // The starting balance rides the same /api/settings payload as Target and
  // Goal (cmd/setting.go) — sending just this one field leaves payers,
  // payment methods, Target and Goal exactly as they were read.
  async function saveStartingBalance(event: React.FormEvent) {
    event.preventDefault()
    setBalanceError("")
    setBalanceMessage("")
    const cents = toCents(startingBalance)
    if (cents === null) {
      setBalanceError(t.invalidAmount)
      return
    }
    try {
      const saved = await apiJSON<Lists>("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ savings_starting_balance_cents: cents }),
      })
      setLists(saved)
      setStartingBalance(toTyped(saved.savings_starting_balance_cents))
      setBalanceMessage(t.listsSaved)
      await load()
    } catch {
      setBalanceError(t.startingBalanceNotSaved)
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.savings}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-6">
          <div className="flex flex-wrap gap-4">
            <Card className="min-w-56">
              <CardHeader>
                <CardTitle>{t.savings}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-medium tabular-nums">
                  € {formatCents(savings?.savings_cents ?? 0)}
                </p>
              </CardContent>
            </Card>

            <Card className="min-w-72 flex-1">
              <CardHeader>
                <CardTitle>{t.startingBalance}</CardTitle>
              </CardHeader>
              <CardContent>
                <form
                  onSubmit={saveStartingBalance}
                  className="flex flex-col gap-1.5"
                >
                  <div className="flex gap-2">
                    <InputGroup className="h-9">
                      <InputGroupAddon>€</InputGroupAddon>
                      <InputGroupInput
                        inputMode="decimal"
                        value={startingBalance}
                        onChange={(e) => setStartingBalance(e.target.value)}
                      />
                    </InputGroup>
                    <Button type="submit" variant="outline">
                      {t.save}
                    </Button>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {t.startingBalanceHint}
                  </span>
                  {balanceError && (
                    <p role="alert" className="text-sm text-destructive">
                      {balanceError}
                    </p>
                  )}
                  {balanceMessage && (
                    <p role="status" className="text-sm text-muted-foreground">
                      {balanceMessage}
                    </p>
                  )}
                </form>
              </CardContent>
            </Card>
          </div>

          <div className="flex flex-col gap-2">
            <h2 className="text-sm font-medium">{t.portfolio}</h2>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t.holding}</TableHead>
                  <TableHead>{t.holdingType}</TableHead>
                  <TableHead className="text-right">{t.amount}</TableHead>
                  <TableHead className="text-right">%</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(savings?.holdings ?? []).map((h) => (
                  <TableRow key={h.holding_id}>
                    <TableCell>{h.name}</TableCell>
                    <TableCell>{typeLabels[h.type]}</TableCell>
                    <TableCell className="text-right tabular-nums">
                      € {formatCents(h.net_cents)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {h.percent.toFixed(1)}%
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {(savings?.holdings ?? []).length === 0 && (
              <p className="text-sm text-muted-foreground">
                {t.noHoldingsInPortfolio}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t.holding}</TableHead>
                  <TableHead>{t.date}</TableHead>
                  <TableHead className="text-right">{t.amount}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {records.map((r) => (
                  <TableRow key={r.key}>
                    <TableCell>
                      {nameOf(holdings, r.holding_id)} ·{" "}
                      {r.kind === "buy" ? t.buy : t.sell}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDate(r.date)}
                    </TableCell>
                    <TableCell className="text-right font-medium tabular-nums">
                      {r.kind === "sell" ? "+ " : ""}€ {formatCents(r.amount_cents)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {records.length === 0 && (
              <p className="text-sm text-muted-foreground">{t.noBuysSellsYet}</p>
            )}
          </div>
        </div>

        <FormSidebar
          title={kind === "buy" ? t.recordBuy : t.recordSell}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
        >
          <form onSubmit={submit} className="flex flex-col gap-3">
            <div className="flex gap-2">
              <Button
                type="button"
                variant={kind === "buy" ? "default" : "outline"}
                className="h-9 flex-1"
                onClick={() => setKind("buy")}
              >
                {t.buy}
              </Button>
              <Button
                type="button"
                variant={kind === "sell" ? "default" : "outline"}
                className="h-9 flex-1"
                onClick={() => setKind("sell")}
              >
                {t.sell}
              </Button>
            </div>

            <Field>
              <FieldLabel htmlFor="holding">{t.holding}</FieldLabel>
              <Select
                value={draft.holding_id === "" ? undefined : String(draft.holding_id)}
                onValueChange={(v) => set("holding_id", Number(v))}
                required
                disabled={holdings.length === 0}
              >
                <SelectTrigger id="holding" className="h-10 w-full">
                  <SelectValue placeholder={t.chooseHolding} />
                </SelectTrigger>
                <SelectContent>
                  {holdings.map((h) => (
                    <SelectItem key={h.id} value={String(h.id)}>
                      {h.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {holdings.length === 0 && (
                <FieldDescription>{t.noHoldingsToBuySell}</FieldDescription>
              )}
            </Field>

            <Field>
              <FieldLabel htmlFor="amount">{t.amount}</FieldLabel>
              <InputGroup className="h-12">
                <InputGroupAddon className="text-xl">€</InputGroupAddon>
                <InputGroupInput
                  id="amount"
                  type="text"
                  inputMode="decimal"
                  value={draft.amount}
                  onChange={(e) => set("amount", e.target.value)}
                  placeholder="0,00"
                  required
                  className="text-2xl"
                />
              </InputGroup>
            </Field>

            <div className="flex gap-2">
              <Field className="flex-1">
                <FieldLabel htmlFor="date">{t.date}</FieldLabel>
                <DatePicker
                  id="date"
                  value={draft.date}
                  onValueChange={(v) => set("date", v)}
                  className="w-full"
                />
              </Field>
              <Field className="flex-1">
                <FieldLabel htmlFor="payer">{t.payer}</FieldLabel>
                <Select
                  value={draft.payer || undefined}
                  onValueChange={(v) => set("payer", v)}
                  required
                >
                  <SelectTrigger id="payer" className="h-10 w-full">
                    <SelectValue placeholder={t.chooseCategory} />
                  </SelectTrigger>
                  <SelectContent>
                    {withSaved(lists.payers, draft.payer || undefined).map(
                      (p) => (
                        <SelectItem key={p} value={p}>
                          {p}
                        </SelectItem>
                      )
                    )}
                  </SelectContent>
                </Select>
              </Field>
            </div>

            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}

            <Button
              type="submit"
              size="lg"
              className="h-12 text-base"
              disabled={holdings.length === 0}
            >
              {kind === "buy" ? t.recordBuy : t.recordSell}
            </Button>
          </form>
        </FormSidebar>
      </div>
    </div>
  )
}
