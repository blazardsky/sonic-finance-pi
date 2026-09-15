import { useCallback, useEffect, useState } from "react"
import {
  RiCheckLine,
  RiDeleteBinLine,
  RiEditLine,
  RiEyeLine,
  RiMoreLine,
} from "@remixicon/react"

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldLabel } from "@/components/ui/field"
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
import { Textarea } from "@/components/ui/textarea"
import { api, apiJSON } from "@/lib/api"
import { ColorDot } from "@/components/ColorDot"
import { DatePicker } from "@/components/date-picker"
import { FormSidebar } from "@/components/form-sidebar"
import { PeriodLabel, PeriodStepper } from "@/components/PeriodStepper"
import { ViewRow } from "@/components/ViewRow"
import { toast } from "@/lib/toast"
import {
  formatCents,
  formatDate,
  formatMonth,
  thisMonth,
  thisYear,
  toCents,
  today,
  toTyped,
} from "@/lib/money"
import { nameOf, pickableCategories, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Client, Contract, Income, Lists } from "@/types"

// How many Incomes a page (the default recent view, or one page of a
// selected year) holds — same ceiling Expenses.tsx uses on GET /api/expenses,
// mirrored here on GET /api/incomes.
const PAGE_SIZE = 50

// The Contract picker's default: "Extra" is a real, common choice (unlinked
// income), never an unset placeholder — so, like the payment-method sentinel
// on the Expenses form, it needs a stand-in value Radix Select will accept.
const EXTRA_CONTRACT = "__extra__"

// Same bargain, for the Category Select: `undefined` for "unset" makes the
// Select treat itself as no longer controlled, so it keeps showing whatever
// was last picked instead of clearing — even once `category_id` has
// genuinely reset to "" after a submit.
const NO_CATEGORY = "__none__"

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
  bollo_fattura: true,
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
  bollo_fattura: income.bollo_fattura,
})

// The Income screen: record an invoice the day it goes out, and add the
// payment date when the money lands. The form lives in the right-side panel;
// the list fills the rest of the width. Choosing a row's "Modifica" action is
// how an entry gets corrected, which loads it into the same form and reopens
// the panel if it was closed, since there is only ever one form on the screen.
export function Incomes({
  editIncomeId,
}: {
  // Set by the Dashboard's "mark as paid" quick action when the clock isn't
  // trustworthy enough to guess a date on its own: this Income opens straight
  // into the form instead, for the household to type the date itself.
  editIncomeId?: number | null
}) {
  const [incomes, setIncomes] = useState<Income[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [clients, setClients] = useState<Client[]>([])
  const [lists, setLists] = useState<Lists>({
    payers: [],
    payment_methods: [],
    target_cents: 0,
    goal_cents: 0,
    savings_starting_balance_cents: 0,
    net_worth_target_cents: 0,
  })
  const [error, setError] = useState("")

  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [detailsOpen, setDetailsOpen] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(true)

  // The picked Client's own Contracts, scoped by re-fetching whenever
  // client_id changes — never all Clients' Contracts at once, since only one
  // Client's are ever relevant to the form open at a time.
  const [clientContracts, setClientContracts] = useState<Contract[]>([])

  // The Income the "Visualizza" action opened, or null when the dialog is
  // closed — Motivo, Ricevuto da, Fattura inviata and Note all live only here
  // now, not in the table itself, the same simplification Expenses.tsx
  // already made for its own row.
  const [viewing, setViewing] = useState<Income | null>(null)
  // null is the default view: the most recent PAGE_SIZE Incomes, unfiltered.
  // Picking a year switches to that year's own Incomes, paged PAGE_SIZE at a
  // time — the same "recent / anno" filter Expenses.tsx offers, off the same
  // invoice-date-or-payment-date the list is ordered by.
  const [year, setYear] = useState<string | null>(null)
  const [page, setPage] = useState(0)

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  useEffect(() => {
    let cancelled = false
    Promise.resolve(
      draft.client_id === ""
        ? []
        : apiJSON<Contract[]>(`/api/clients/${draft.client_id}/contracts`).catch(() => []),
    ).then((cs) => {
      if (cancelled) return
      setClientContracts(cs)
      // A new income defaults to whichever Contract is active this month —
      // a one-time prefill like the Client's own defaults above, never a
      // guess on top of what an edited Income actually has recorded.
      if (editing === null) {
        const active = cs.find(
          (c) => c.start_month <= thisMonth() && thisMonth() <= c.end_month,
        )
        if (active) set("contract_id", active.id)
      }
    })
    return () => {
      cancelled = true
    }
  }, [draft.client_id, editing])

  // Loads a row into the form and makes sure the panel holding it is
  // actually visible — the form updates whether it is on screen or not, and a
  // silently updated but hidden form is indistinguishable from a broken one.
  const selectIncome = (income: Income) => {
    setEditing(income.id)
    setDraft(draftOf(income))
    setDetailsOpen(true)
    setSidebarOpen(true)
  }

  // No year selected: the recent view, `limit` alone. A year selected: that
  // year's own Incomes, one PAGE_SIZE page at a time — same shape
  // Expenses.tsx's own expensesPath uses.
  const incomesPath =
    year === null
      ? `/api/incomes?limit=${PAGE_SIZE}`
      : `/api/incomes?year=${year}&limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Income[]>(incomesPath),
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
    [incomesPath]
  )

  // -1: further back is always available, arbitrarily. +1: a step past the
  // current year returns to the recent (unfiltered) view — same stepper
  // Expenses.tsx uses, off the same invoice-date-or-payment-date ordering.
  const stepYear = (delta: number) => {
    setPage(0)
    setYear((y) => {
      if (delta < 0) return y === null ? thisYear() : String(Number(y) - 1)
      return y !== null && y < thisYear() ? String(Number(y) + 1) : null
    })
  }

  useEffect(() => {
    void load()
  }, [load])

  // Adjusted during render (App.tsx's own pattern for this, e.g.
  // quickAddScreen) rather than an effect: applied tracks the last
  // editIncomeId this form has already loaded, so arriving with a new one
  // selects it once incomes has loaded far enough to contain it, and an edit
  // submitted from this same form afterwards doesn't reselect the
  // just-saved Income right back into the form the submit handler just
  // cleared.
  const [appliedEditId, setAppliedEditId] = useState<number | null>(null)
  if (editIncomeId !== appliedEditId) {
    const income =
      editIncomeId != null
        ? incomes?.find((i) => i.id === editIncomeId)
        : undefined
    if (editIncomeId == null || income) {
      setAppliedEditId(editIncomeId ?? null)
      if (income) selectIncome(income)
    }
  }

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

    // Caught here rather than left to the server's opaque failure: a Select
    // stuck showing a stale category after the form reset (the NO_CATEGORY
    // sentinel fix above addresses the visual half of it) would otherwise
    // submit "" for category_id.
    if (draft.category_id === "") {
      setError(t.invalidCategory)
      return
    }

    // Freelance is billed work: "whose money was it" (ticket 08's Payer rule)
    // extends here to "who owes it". A gift or an Investment sell names
    // nobody by design (cmd/income.go), so the requirement is Freelance-only
    // and, unlike Payer, frontend-only — the server still accepts a null
    // Client on any Category.
    if (isFreelance && draft.client_id === "") {
      setError(t.invalidIncomeClient)
      return
    }

    // The server refuses an Income with no Payer, but Payer lives inside the
    // details disclosure — closed, its Select unmounts, so the browser's own
    // required check on it never runs. Checked here instead, opening the
    // disclosure so the field the error is about is what the household sees.
    if (draft.payer.trim() === "") {
      setError(t.invalidIncomePayer)
      setDetailsOpen(true)
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
    const wasEditing = editing !== null
    if (!wasEditing) toast(t.added)
    setEditing(null)
    setDraft((d) =>
      wasEditing
        ? blankDraft()
        : {
            ...d,
            amount: "",
            note: "",
            payment_date: "",
            invoice_sent_date: "",
          }
    )
    if (wasEditing) setDetailsOpen(false)
    await load()
  }

  // A quick way to record that the money landed today, for an invoiced
  // Income still waiting on it — payment_date is the only field this PATCH
  // sends, so nothing else about the Income changes.
  async function markPaid(income: Income) {
    setError("")
    try {
      await api(`/api/incomes/${income.id}`, {
        method: "PATCH",
        body: JSON.stringify({ payment_date: today() }),
      })
    } catch {
      setError(t.incomeNotSaved)
      return
    }
    await load()
  }

  async function removeIncome(income: Income) {
    if (!confirm(t.confirmDeleteIncome(formatCents(income.amount_cents))))
      return
    setError("")
    try {
      await api(`/api/incomes/${income.id}`, { method: "DELETE" })
    } catch {
      setError(t.incomeNotDeleted)
      return
    }
    if (editing === income.id) {
      setEditing(null)
      setDraft(blankDraft())
      setDetailsOpen(false)
    }
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

  // Whether Fattura inviata belongs on screen, which is only for the
  // Freelance reason — a gift or a salary is never invoiced. Resolved
  // against the loaded Category list's own `freelance` flag, never by name.
  const isFreelance = categories.some(
    (c) => c.id === draft.category_id && c.freelance
  )

  // The Client picker is the first one in the app, and filtering hidden ones
  // out is its job: /api/clients deliberately returns them, because the
  // management screen is the only place a hidden Client can be brought back
  // from.
  const pickableClients = withSaved(
    clients.filter((c) => !c.hidden),
    clients.find((c) => c.id === draft.client_id)
  )
  // The Combobox's own item shape — {value, label} is auto-recognised by the
  // primitive for both display text and equality, so no itemToStringLabel or
  // isItemEqualToValue is needed as long as the controlled `value` below is
  // always looked up from this same freshly-built array.
  const clientOptions = pickableClients.map((c) => ({
    value: c.id,
    label: c.name,
  }))
  const selectedClient =
    clientOptions.find((o) => o.value === draft.client_id) ?? null

  const chooseClient = (option: { value: number; label: string } | null) => {
    const clientId = option?.value ?? ""
    set("client_id", clientId)
    // A different Client means a different Contract picker, and the one just
    // picked would otherwise dangle: unlinked is the only safe carry-over,
    // never a guess at the new Client's own.
    set("contract_id", "")
    // A one-time prefill, not a lock (ticket 04): picking a Client with
    // defaults set loads them into the reason and Payer pickers, both of
    // which stay freely editable from here.
    const client = clients.find((c) => c.id === clientId)
    if (client?.default_category_id != null)
      set("category_id", client.default_category_id)
    if (client?.default_payer) set("payer", client.default_payer)
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <div className="flex items-center justify-between gap-2">
            <PeriodStepper
              previousLabel={t.previousYear}
              onPrevious={() => stepYear(-1)}
              nextLabel={t.nextYear}
              nextDisabled={year === null}
              onNext={() => stepYear(1)}
            >
              <PeriodLabel>{year ?? t.incomesYearFilterRecent}</PeriodLabel>
            </PeriodStepper>
            {/* Paging only exists once a year is picked — the recent view is
                always exactly one page (PAGE_SIZE), by design, so there is
                never a second page of it to flip to. */}
            {year !== null && (
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t.previousPage}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  // A full page might not be the last one — cheaper than a
                  // separate count query, at the cost of one possible extra
                  // (empty) page click at the true end.
                  disabled={(incomes?.length ?? 0) < PAGE_SIZE}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t.nextPage}
                </Button>
              </div>
            )}
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.date}</TableHead>
                <TableHead className="text-right">{t.amount}</TableHead>
                <TableHead>{t.client}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {incomes?.map((income) => (
                <TableRow
                  key={income.id}
                  className={editing === income.id ? "opacity-50" : ""}
                >
                  <TableCell className="whitespace-normal">
                    <div className="flex flex-col gap-0.5">
                      <span className="text-xs text-muted-foreground">
                        {income.payment_date
                          ? formatDate(income.payment_date)
                          : ""}
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
                  <TableCell className="truncate text-xs text-muted-foreground">
                    {nameOf(clients, income.client_id)}
                  </TableCell>
                  <TableCell>
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
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => setViewing(income)}>
                          <RiEyeLine /> {t.viewIncome}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => selectIncome(income)}>
                          <RiEditLine /> {t.editIncome}
                        </DropdownMenuItem>
                        {/* Only for an invoiced Income still waiting on its
                            money — nothing to mark paid otherwise, and one
                            already paid has nothing left to set. */}
                        {income.invoice_sent_date && !income.payment_date && (
                          <DropdownMenuItem onClick={() => void markPaid(income)}>
                            <RiCheckLine /> {t.markPaid}
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => void removeIncome(income)}
                        >
                          <RiDeleteBinLine /> {t.delete}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {incomes?.length === 0 && (
            <p className="text-sm text-muted-foreground">{t.noIncomesYet}</p>
          )}
        </div>

        <FormSidebar
          title={editing === null ? t.addIncome : t.editIncome}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          footer={
            <div className="flex flex-col gap-2">
              <Button
                type="submit"
                form="income-form"
                size="lg"
                className="h-12 text-base"
              >
                {editing === null ? t.addIncome : t.save}
              </Button>
              {editing !== null && (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setEditing(null)
                    setDraft(blankDraft())
                    setDetailsOpen(false)
                  }}
                >
                  {t.cancel}
                </Button>
              )}
            </div>
          }
        >
          <form
            id="income-form"
            onSubmit={submit}
            className="flex flex-col gap-3"
          >
            <Field>
              <FieldLabel htmlFor="amount">{t.amount}</FieldLabel>
              <InputGroup className="h-12">
                <InputGroupAddon className="text-xl">€</InputGroupAddon>
                <InputGroupInput
                  id="amount"
                  // text with a decimal keypad, not type=number: a comma is
                  // what the Italian keypad offers and toCents accepts it.
                  type="text"
                  inputMode="decimal"
                  value={draft.amount}
                  onChange={(e) => set("amount", e.target.value)}
                  placeholder="0,00"
                  autoFocus
                  required
                  className="text-2xl"
                />
              </InputGroup>
            </Field>

            <div className="flex gap-2">
              <Field className="flex-1">
                <FieldLabel htmlFor="client">{t.client}</FieldLabel>
                <Combobox
                  items={clientOptions}
                  value={selectedClient}
                  onValueChange={chooseClient}
                >
                  <ComboboxInput
                    id="client"
                    placeholder={t.notSet}
                    showClear
                    className="h-10"
                  />
                  <ComboboxContent>
                    <ComboboxEmpty>{t.noClientsFound}</ComboboxEmpty>
                    <ComboboxList>
                      {(option: { value: number; label: string }) => (
                        <ComboboxItem key={option.value} value={option}>
                          {option.label}
                        </ComboboxItem>
                      )}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </Field>
              <Field className="flex-1">
                <FieldLabel htmlFor="reason">{t.incomeReason}</FieldLabel>
                <Select
                  value={draft.category_id === "" ? NO_CATEGORY : String(draft.category_id)}
                  onValueChange={(v) =>
                    set("category_id", v === NO_CATEGORY ? "" : Number(v))
                  }
                  required
                >
                  <SelectTrigger id="reason" className="h-10 w-full">
                    <SelectValue placeholder={t.chooseCategory} />
                  </SelectTrigger>
                  <SelectContent>
                    {pickable.map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>
                        <span className="flex items-center gap-2">
                          <ColorDot color={c.color} />
                          {c.name}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </div>

            {/* Only shown once a Client with at least one Contract is picked:
                "Extra" (unlinked) is always the default, never guessed, so a
                Client with no Contracts has nothing to offer here at all. */}
            {clientContracts.length > 0 && (
              <Field>
                <FieldLabel htmlFor="contract">{t.contract}</FieldLabel>
                <Select
                  value={
                    draft.contract_id === ""
                      ? EXTRA_CONTRACT
                      : String(draft.contract_id)
                  }
                  onValueChange={(v) =>
                    set("contract_id", v === EXTRA_CONTRACT ? "" : Number(v))
                  }
                >
                  <SelectTrigger id="contract" className="h-10 w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={EXTRA_CONTRACT}>
                      {t.extraIncome}
                    </SelectItem>
                    {clientContracts.map((ct) => (
                      <SelectItem key={ct.id} value={String(ct.id)}>
                        {formatMonth(ct.start_month)} – {formatMonth(ct.end_month)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            )}

            {/* The two dates, one above the other rather than side by side —
                the sidebar's fixed 16rem width has no room for two DatePicker
                buttons showing a full date each — because the whole point of
                this screen is that they are different questions: when the
                invoice went out, and whether the money has arrived. An empty
                payment date is not a missing field — it is the unpaid state.
                Fattura inviata is Freelance-only: a gift or a salary is never
                invoiced, so the question does not belong on screen for one. */}
            {isFreelance && (
              <Field>
                <FieldLabel htmlFor="invoice_sent_date">
                  {t.invoiceSentDate}
                </FieldLabel>
                <DatePicker
                  id="invoice_sent_date"
                  value={draft.invoice_sent_date}
                  onValueChange={(v) => set("invoice_sent_date", v)}
                  className="w-full"
                />
              </Field>
            )}
            {/* Freelance-only, on by default: every invoice over the bollo
                threshold carries a 2€ marca da bollo, which arrives inside
                amount_cents but is not real revenue — unticking it is for
                the rare invoice under the threshold. */}
            {isFreelance && (
              <FieldLabel htmlFor="bollo_fattura" className="font-normal">
                <Checkbox
                  id="bollo_fattura"
                  checked={draft.bollo_fattura}
                  onCheckedChange={(v) => set("bollo_fattura", v === true)}
                />
                {t.bolloFattura}
              </FieldLabel>
            )}
            <Field>
              <FieldLabel htmlFor="payment_date">{t.paymentDate}</FieldLabel>
              <DatePicker
                id="payment_date"
                value={draft.payment_date}
                onValueChange={(v) => set("payment_date", v)}
                className="w-full"
              />
            </Field>

            <Accordion
              type="single"
              collapsible
              value={detailsOpen ? "details" : ""}
              onValueChange={(v) => setDetailsOpen(v === "details")}
            >
              <AccordionItem value="details">
                <AccordionTrigger className="text-muted-foreground hover:no-underline">
                  {t.moreDetails}
                </AccordionTrigger>
                <AccordionContent className="flex flex-col gap-3">
                  <Field>
                    <FieldLabel htmlFor="payer">{t.incomePayer}</FieldLabel>
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

                  <Field>
                    <FieldLabel htmlFor="note">{t.note}</FieldLabel>
                    <Textarea
                      id="note"
                      value={draft.note}
                      onChange={(e) => set("note", e.target.value)}
                      rows={2}
                    />
                  </Field>
                </AccordionContent>
              </AccordionItem>
            </Accordion>

            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}

          </form>
        </FormSidebar>
      </div>

      {/* Everything the table's own columns used to show inline (Motivo,
          Ricevuto da, the Dettagli disclosure) — one place for the whole
          Income now that the table itself only carries what most rows
          actually need at a glance, the same simplification Expenses.tsx's
          own view dialog already made. */}
      <Dialog
        open={viewing !== null}
        onOpenChange={(open) => !open && setViewing(null)}
      >
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
          {viewing && (
            <>
              <DialogHeader>
                <DialogTitle>{t.incomeDetails}</DialogTitle>
              </DialogHeader>
              <dl className="flex flex-col divide-y divide-border text-sm">
                <ViewRow
                  label={t.amount}
                  value={`€ ${formatCents(viewing.amount_cents)}`}
                />
                <ViewRow
                  label={t.incomeReason}
                  value={nameOf(categories, viewing.category_id)}
                />
                {viewing.client_id !== null && (
                  <ViewRow label={t.client} value={nameOf(clients, viewing.client_id)} />
                )}
                {viewing.payer && (
                  <ViewRow label={t.incomePayer} value={viewing.payer} />
                )}
                {viewing.invoice_sent_date && (
                  <ViewRow
                    label={t.invoiceSentDate}
                    value={formatDate(viewing.invoice_sent_date)}
                  />
                )}
                <ViewRow
                  label={t.paymentDate}
                  value={
                    viewing.payment_date
                      ? formatDate(viewing.payment_date)
                      : t.notPaidYet
                  }
                />
                {viewing.note && <ViewRow label={t.note} value={viewing.note} />}
              </dl>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
