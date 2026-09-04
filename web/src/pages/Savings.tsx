import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api, apiJSON } from "@/lib/api"
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

// The Savings page: a dedicated Buy/Sell form so nobody has to remember to
// pick Investments and a Holding on the generic Expense/Income screens. It
// posts to those same two endpoints (spec's Implementation Decisions) — there
// is no buy/sell endpoint of its own, and nothing here is excluded from the
// ordinary Month/Year/Dashboard/Recent totals (ADR-0009).
export function Savings() {
  const [holdings, setHoldings] = useState<Holding[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [lists, setLists] = useState<Lists>({ payers: [], payment_methods: [] })
  const [records, setRecords] = useState<Transaction[]>([])
  const [error, setError] = useState("")

  const [kind, setKind] = useState<Kind>("buy")
  const [draft, setDraft] = useState<Draft>(blankDraft)

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
    // The Holding and Payer are often the same for the next entry — a PAC
    // contribution split across a few lines, say — so only the amount clears.
    setDraft((d) => ({ ...d, amount: "" }))
    await load()
  }

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <h1 className="font-medium">{t.savings}</h1>

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

        <label className="flex flex-col gap-1.5 text-sm">
          {t.holding}
          <NativeSelect
            value={draft.holding_id}
            // Back to the blank option is "" and not Number("") — 0, which no
            // Holding has, would slip past the unchosen check in submit() and
            // travel as a Holding the server refuses.
            onChange={(e) =>
              set(
                "holding_id",
                e.target.value === "" ? "" : Number(e.target.value)
              )
            }
            required
            className="h-10"
            disabled={holdings.length === 0}
          >
            <option value="">{t.chooseHolding}</option>
            {holdings.map((h) => (
              <option key={h.id} value={h.id}>
                {h.name}
              </option>
            ))}
          </NativeSelect>
          {holdings.length === 0 && (
            <span className="text-xs text-muted-foreground">
              {t.noHoldingsToBuySell}
            </span>
          )}
        </label>

        <label className="flex flex-col gap-1.5 text-sm">
          {t.amount}
          <div className="flex items-center gap-2">
            <span className="text-xl text-muted-foreground">€</span>
            <Input
              type="text"
              inputMode="decimal"
              value={draft.amount}
              onChange={(e) => set("amount", e.target.value)}
              placeholder="0,00"
              required
              className="h-12 text-2xl"
            />
          </div>
        </label>

        <div className="flex gap-2">
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.date}
            <Input
              type="date"
              value={draft.date}
              onChange={(e) => set("date", e.target.value)}
              required
              className="h-10"
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.payer}
            <NativeSelect
              value={draft.payer}
              onChange={(e) => set("payer", e.target.value)}
              required
              className="h-10"
            >
              <option value="">{t.chooseCategory}</option>
              {withSaved(lists.payers, draft.payer || undefined).map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </NativeSelect>
          </label>
        </div>

        <Button
          type="submit"
          size="lg"
          className="h-12 text-base"
          disabled={holdings.length === 0}
        >
          {kind === "buy" ? t.recordBuy : t.recordSell}
        </Button>
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

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
  )
}
