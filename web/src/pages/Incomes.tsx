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
import { Textarea } from "@/components/ui/textarea"
import { api, apiJSON } from "@/lib/api"
import {
  formatCents,
  formatDate,
  formatMonth,
  toCents,
  toTyped,
} from "@/lib/money"
import { nameOf, pickableCategories, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Client, Contract, Income, Lists } from "@/types"

// What the form holds: the amount as it was typed, and everything else as the
// API's own field names, so submitting is one spread rather than a mapping.
// "" is the unchosen picker for both references — a Client is optional, so ""
// is a real answer there and travels as null, and so is a Contract (ticket
// 05): unlinked ("Extra") is a real, common answer, never guessed.
// holding_id is left out of the draft entirely: this form never shows a
// Holding picker (ticket 03's Buy/Sell form on the Savings page does), and
// omitting the key from the submitted body — rather than sending null — is
// what leaves a sell's holding_id untouched when it is later edited here for
// some other reason, e.g. recording its payment date.
type Draft = Omit<
  Income,
  | "id"
  | "amount_cents"
  | "category_id"
  | "client_id"
  | "holding_id"
  | "contract_id"
> & {
  amount: string
  category_id: number | ""
  client_id: number | ""
  contract_id: number | ""
}

// Neither date is defaulted. An invoice sent today and a gift that arrived
// last week are equally common, and there is no story asking for either one to
// be filled in — unlike an Expense, which defaults to today because logging at
// the till is what the app is for.
const blankDraft = (): Draft => ({
  amount: "",
  category_id: "",
  client_id: "",
  contract_id: "",
  payer: "",
  payment_date: "",
  invoice_sent_date: "",
  note: "",
})

// Field by field rather than a spread, so the draft carries what the form
// owns and nothing else: an id in a PATCH body is not something the server
// should have to defend against.
const draftOf = (income: Income): Draft => ({
  amount: toTyped(income.amount_cents),
  category_id: income.category_id,
  client_id: income.client_id ?? "",
  contract_id: income.contract_id ?? "",
  payer: income.payer,
  payment_date: income.payment_date,
  invoice_sent_date: income.invoice_sent_date,
  note: income.note,
})

// The Income screen: record an invoice the day it goes out, and add the
// payment date when the money lands. The form is first, the list under it is
// both confirmation and the way back into an entry — tapping one loads it into
// the same form, which is why there is only ever one form on the screen.
export function Incomes() {
  const [incomes, setIncomes] = useState<Income[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [clients, setClients] = useState<Client[]>([])
  const [lists, setLists] = useState<Lists>({ payers: [], payment_methods: [] })
  const [error, setError] = useState("")

  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)

  // The picked Client's own Contracts, scoped by re-fetching whenever
  // client_id changes — never all Clients' Contracts at once, since only one
  // Client's are ever relevant to the form open at a time.
  const [clientContracts, setClientContracts] = useState<Contract[]>([])

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  useEffect(() => {
    if (draft.client_id === "") {
      setClientContracts([])
      return
    }
    let cancelled = false
    apiJSON<Contract[]>(`/api/clients/${draft.client_id}/contracts`)
      .then((cs) => {
        if (!cancelled) setClientContracts(cs)
      })
      .catch(() => {
        if (!cancelled) setClientContracts([])
      })
    return () => {
      cancelled = true
    }
  }, [draft.client_id])

  // Tapping (or, from the keyboard, activating) a row is how an entry gets
  // corrected — shared by the row's click and its Enter/Space handling below,
  // which is the table's replacement for the list's own <button>.
  const selectIncome = (income: Income) => {
    setEditing(income.id)
    setDraft(draftOf(income))
    scrollTo({ top: 0, behavior: "smooth" })
  }

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Income[]>("/api/incomes"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Client[]>("/api/clients"),
        apiJSON<Lists>("/api/settings"),
      ])
        .then(([i, c, cl, l]) => {
          setIncomes(i)
          setCategories(c)
          setClients(cl)
          setLists(l)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setError("")

    // Cents are computed here and sent as an integer: the API refuses a
    // fractional amount_cents outright, so a bad parse cannot become a
    // silently rounded Income.
    const cents = toCents(draft.amount)
    if (cents === null || cents <= 0) {
      setError(t.invalidAmount)
      return
    }

    try {
      await api(editing === null ? "/api/incomes" : `/api/incomes/${editing}`, {
        method: editing === null ? "POST" : "PATCH",
        // The draft's field names are the API's, so the body is the draft
        // with the typed amount swapped for whole cents. An unchosen Client
        // travels as null, which is how one is cleared.
        body: JSON.stringify({
          ...draft,
          amount: undefined,
          amount_cents: cents,
          client_id: draft.client_id === "" ? null : draft.client_id,
          contract_id: draft.contract_id === "" ? null : draft.contract_id,
        }),
      })
    } catch {
      setError(t.incomeNotSaved)
      return
    }
    // A correction is finished. A new Income is often one of several invoices
    // in a sitting, so the reason, Client and Payer stay where they are.
    setEditing(null)
    setDraft((d) =>
      editing === null
        ? {
            ...d,
            amount: "",
            note: "",
            payment_date: "",
            invoice_sent_date: "",
          }
        : blankDraft()
    )
    await load()
  }

  async function remove() {
    if (editing === null) return
    const income = incomes?.find((x) => x.id === editing)
    if (
      income &&
      !confirm(t.confirmDeleteIncome(formatCents(income.amount_cents)))
    )
      return
    setError("")
    try {
      await api(`/api/incomes/${editing}`, { method: "DELETE" })
    } catch {
      setError(t.incomeNotDeleted)
      return
    }
    setEditing(null)
    setDraft(blankDraft())
    await load()
  }

  // What the reason picker offers, which is the narrower list: no hidden
  // Category, and nothing expense-only — "Alimentari" is never an Income.
  // Plus whichever one the entry being edited already carries.
  const pickable = pickableCategories(
    categories,
    "income",
    categories.find((c) => c.id === draft.category_id)
  )

  // The Client picker is the first one in the app, and filtering hidden ones
  // out is its job: /api/clients deliberately returns them, because the
  // management screen is the only place a hidden Client can be brought back
  // from.
  const pickableClients = withSaved(
    clients.filter((c) => !c.hidden),
    clients.find((c) => c.id === draft.client_id)
  )

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <form onSubmit={submit} className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5 text-sm">
          {t.amount}
          <div className="flex items-center gap-2">
            <span className="text-xl text-muted-foreground">€</span>
            <Input
              // text with a decimal keypad, not type=number: a comma is what
              // the Italian keypad offers and toCents accepts it.
              type="text"
              inputMode="decimal"
              value={draft.amount}
              onChange={(e) => set("amount", e.target.value)}
              placeholder="0,00"
              autoFocus
              required
              className="h-12 text-2xl"
            />
          </div>
        </label>

        <div className="flex gap-2">
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.incomeReason}
            <NativeSelect
              value={draft.category_id}
              onChange={(e) => set("category_id", Number(e.target.value))}
              required
              className="h-10"
            >
              <option value="">{t.chooseCategory}</option>
              {pickable.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </NativeSelect>
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.client}
            <NativeSelect
              value={draft.client_id}
              // Back to the blank option is "" and not Number("") — 0, which
              // no Client has, would travel as a Client the server refuses.
              onChange={(e) => {
                const clientId =
                  e.target.value === "" ? "" : Number(e.target.value)
                set("client_id", clientId)
                // A different Client means a different Contract picker, and
                // the one just picked would otherwise dangle: unlinked is the
                // only safe carry-over, never a guess at the new Client's own.
                set("contract_id", "")
                // A one-time prefill, not a lock (ticket 04): picking a
                // Client with a default Income Category loads it into the
                // reason picker, which stays freely editable from here.
                const defaultCategoryId = clients.find(
                  (c) => c.id === clientId
                )?.default_category_id
                if (defaultCategoryId != null)
                  set("category_id", defaultCategoryId)
              }}
              className="h-10"
            >
              <option value="">{t.notSet}</option>
              {pickableClients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </NativeSelect>
          </label>
        </div>

        {/* Only shown once a Client with at least one Contract is picked:
            "Extra" (unlinked) is always the default, never guessed, so a
            Client with no Contracts has nothing to offer here at all. */}
        {clientContracts.length > 0 && (
          <label className="flex flex-col gap-1.5 text-sm">
            {t.contract}
            <NativeSelect
              value={draft.contract_id}
              onChange={(e) =>
                set(
                  "contract_id",
                  e.target.value === "" ? "" : Number(e.target.value)
                )
              }
              className="h-10"
            >
              <option value="">{t.extraIncome}</option>
              {clientContracts.map((ct) => (
                <option key={ct.id} value={ct.id}>
                  {formatMonth(ct.start_month)} – {formatMonth(ct.end_month)}
                </option>
              ))}
            </NativeSelect>
          </label>
        )}

        {/* The two dates, side by side, because the whole point of this screen
            is that they are different questions: when the invoice went out,
            and whether the money has arrived. An empty payment date is not a
            missing field — it is the unpaid state. */}
        <div className="flex gap-2">
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.invoiceSentDate}
            <Input
              type="date"
              value={draft.invoice_sent_date}
              onChange={(e) => set("invoice_sent_date", e.target.value)}
              className="h-10"
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.paymentDate}
            <Input
              type="date"
              value={draft.payment_date}
              onChange={(e) => set("payment_date", e.target.value)}
              className="h-10"
            />
          </label>
        </div>

        <details open={editing !== null} className="flex flex-col gap-3">
          <summary className="cursor-pointer py-1 text-sm text-muted-foreground">
            {t.moreDetails}
          </summary>
          <div className="flex flex-col gap-3 pt-2">
            <label className="flex flex-col gap-1.5 text-sm">
              {t.incomePayer}
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

            <label className="flex flex-col gap-1.5 text-sm">
              {t.note}
              <Textarea
                value={draft.note}
                onChange={(e) => set("note", e.target.value)}
                rows={2}
              />
            </label>
          </div>
        </details>

        <Button type="submit" size="lg" className="h-12 text-base">
          {editing === null ? t.addIncome : t.save}
        </Button>
        {editing !== null && (
          <div className="flex gap-2">
            <Button
              type="button"
              variant="ghost"
              className="flex-1"
              onClick={() => {
                setEditing(null)
                setDraft(blankDraft())
              }}
            >
              {t.cancel}
            </Button>
            <Button
              type="button"
              variant="destructive"
              className="flex-1"
              onClick={() => void remove()}
            >
              {t.delete}
            </Button>
          </div>
        )}
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t.incomeReason}</TableHead>
            <TableHead>{t.details}</TableHead>
            <TableHead>{t.date}</TableHead>
            <TableHead className="text-right">{t.amount}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {incomes?.map((income) => (
            <TableRow
              key={income.id}
              role="button"
              tabIndex={0}
              aria-label={t.editIncome}
              onClick={() => selectIncome(income)}
              onKeyDown={(ev) => {
                if (ev.target !== ev.currentTarget) return
                if (ev.key !== "Enter" && ev.key !== " ") return
                ev.preventDefault()
                selectIncome(income)
              }}
              className={`cursor-pointer ${
                editing === income.id ? "opacity-50" : ""
              }`}
            >
              <TableCell className="truncate">
                {[
                  nameOf(categories, income.category_id),
                  nameOf(clients, income.client_id),
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </TableCell>
              <TableCell className="text-xs whitespace-normal text-muted-foreground">
                {[income.payer, income.note].filter(Boolean).join(" · ")}
              </TableCell>
              <TableCell className="whitespace-normal">
                <div className="flex flex-col gap-0.5">
                  <span className="text-xs text-muted-foreground">
                    {income.payment_date ? formatDate(income.payment_date) : ""}
                  </span>
                  {/* An unpaid Income says so, because that is the one thing
                      the amount does not tell you: this money has not
                      arrived and counts toward nothing. Ticket 14 turns
                      these into a list of their own; here they only have to
                      be recognisable. */}
                  {!income.payment_date && (
                    <span className="text-xs text-destructive">
                      {t.notPaidYet}
                      {income.invoice_sent_date &&
                        ` · ${t.waitingSince(
                          formatDate(income.invoice_sent_date)
                        )}`}
                    </span>
                  )}
                </div>
              </TableCell>
              <TableCell className="text-right font-medium tabular-nums">
                € {formatCents(income.amount_cents)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {incomes?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noIncomesYet}</p>
      )}
    </div>
  )
}
