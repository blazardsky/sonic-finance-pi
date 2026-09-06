import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
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
import { toast } from "@/lib/toast"
import { formatCents, formatDate, toCents, today } from "@/lib/money"
import { nameOf, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Expense, Holding, Income, Lists } from "@/types"

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

// The Investimenti page: a dedicated Buy/Sell form so nobody has to remember
// to pick Investments and a Holding on the generic Expense/Income screens. It
// posts to those same two endpoints (spec's Implementation Decisions) — there
// is no buy/sell endpoint of its own, and nothing here is excluded from the
// ordinary Month/Year/Dashboard/Recent totals (ADR-0009).
export function Investments() {
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
      ])
        .then(([h, c, l, expenses, incomes]) => {
          setHoldings(h)
          setCategories(c)
          setLists(l)
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
    toast(t.added)
    // The Holding and Payer are often the same for the next entry — a PAC
    // contribution split across a few lines, say — so only the amount clears.
    setDraft((d) => ({ ...d, amount: "" }))
    await load()
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.investments}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-2">
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
