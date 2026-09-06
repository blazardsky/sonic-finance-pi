import { useCallback, useEffect, useState } from "react"
import {
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiDeleteBinLine,
  RiEditLine,
  RiMoreLine,
} from "@remixicon/react"

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
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
import { DatePicker } from "@/components/date-picker"
import { FormSidebar } from "@/components/form-sidebar"
import { SpoilerAmount } from "@/components/SpoilerAmount"
import { toast } from "@/lib/toast"
import { formatCents, formatDate, toCents, today, toTyped } from "@/lib/money"
import { isGiftCategory, nameOf, pickableCategories, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Expense, Lists } from "@/types"

// What the payment-method Select needs that a plain string can't say: "no
// method chosen" — Radix Select refuses an empty-string item value, so the
// unset state gets its own sentinel, mapped back to "" on the way out.
const NO_PAYMENT_METHOD = "__none__"

// What the form holds: the amount as it was typed, and everything else as the
// API's own field names, so submitting is one spread rather than a mapping.
// holding_id is left out of the draft entirely: this form never shows a
// Holding picker (ticket 03's Buy/Sell form on the Savings page does), and
// omitting the key from the submitted body — rather than sending null — is
// what leaves a buy's holding_id untouched when it is later edited here for
// some other reason, e.g. its amount.
type Draft = Omit<
  Expense,
  "id" | "amount_cents" | "category_id" | "items" | "holding_id"
> & {
  amount: string
  // "" is the unchosen picker, which the required select refuses to submit.
  category_id: number | ""
  items: ItemDraft[]
}

// An Item mid-typing: same shape, same reasons — the amount as it was typed,
// and "" for a Category not chosen yet. A row is added empty and filled in,
// so it has to be able to hold nothing.
type ItemDraft = {
  name: string
  amount: string
  category_id: number | ""
}

const blankItem = (): ItemDraft => ({ name: "", amount: "", category_id: "" })

const blankDraft = (): Draft => ({
  amount: "",
  occurred_on: today(),
  category_id: "",
  store: "",
  payer: "",
  payment_method: "",
  note: "",
  tax_year: 0,
  items: [],
})

// Field by field rather than a spread, so the draft carries what the form
// owns and nothing else: an id in a PATCH body is not something the server
// should have to defend against.
const draftOf = (e: Expense): Draft => ({
  amount: toTyped(e.amount_cents),
  occurred_on: e.occurred_on,
  category_id: e.category_id,
  store: e.store,
  payer: e.payer,
  payment_method: e.payment_method,
  note: e.note,
  tax_year: e.tax_year,
  items: e.items.map((it) => ({
    name: it.name,
    amount: toTyped(it.amount_cents),
    category_id: it.category_id,
  })),
})

// itemsCents sums a draft breakdown, skipping what is not yet an amount. The
// Expense's own total is never computed from this — ADR-0002 — it is only what
// the form checks the total against and shows the remainder from.
const itemsCents = (items: ItemDraft[]) =>
  items.reduce((sum, it) => sum + (toCents(it.amount) ?? 0), 0)

// The main screen: log what was just spent in about three taps, and see it
// land. The form lives in the right-side panel — first and biggest on a
// narrow screen, where it opens full-width, because it is what the app is
// for; the list fills the rest of the width on a wider one. Choosing a row's
// "Modifica" action is how an entry gets corrected, which loads it into the
// same form and reopens the panel if it was closed, since there is only ever
// one form on the screen.
export function Expenses({ quickAdd }: { quickAdd?: boolean }) {
  const [expenses, setExpenses] = useState<Expense[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [lists, setLists] = useState<Lists>({
    payers: [],
    payment_methods: [],
    target_cents: 0,
    goal_cents: 0,
    savings_starting_balance_cents: 0,
  })
  const [error, setError] = useState("")

  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [detailsOpen, setDetailsOpen] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  // Negozio and Note get their own columns but are hidden on narrow screens
  // for space — this is what a row's mobile-only toggle in the Dettagli
  // column expands to show instead.
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set())
  const toggleExpanded = (id: number) =>
    setExpandedIds((s) => {
      const next = new Set(s)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  // Captured once at mount, not read live — a mobile quick-add shortcut is
  // meant to open the panel on arrival, not force it back open every time
  // App re-renders with the flag still set (see App.tsx's reset-on-navigate-
  // away).
  const [openMobileOnArrival] = useState(() => quickAdd ?? false)

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  // Loads a row into the form and makes sure the panel holding it is
  // actually visible — the form updates whether it is on screen or not, and a
  // silently updated but hidden form is indistinguishable from a broken one.
  const selectExpense = (e: Expense) => {
    setEditing(e.id)
    setDraft(draftOf(e))
    setDetailsOpen(true)
    setSidebarOpen(true)
  }

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Expense[]>("/api/expenses"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Lists>("/api/settings"),
      ])
        .then(([e, c, l]) => {
          setExpenses(e)
          setCategories(c)
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
    // silently rounded Expense.
    const cents = toCents(draft.amount)
    if (cents === null || cents <= 0) {
      setError(t.invalidAmount)
      return
    }

    // The server refuses an Expense with no Payer, but Payer lives inside the
    // details disclosure — closed, its Select unmounts, so the browser's own
    // required check on it never runs. Checked here instead, opening the
    // disclosure so the field the error is about is what the household sees.
    if (draft.payer.trim() === "") {
      setError(t.invalidPayer)
      setDetailsOpen(true)
      return
    }

    // A row added and then left alone is not an Item, so it is dropped rather
    // than refused — anything half-filled in is a mistake worth stopping on.
    const filled = draft.items.filter(
      (it) => it.name.trim() !== "" || it.amount !== "" || it.category_id !== ""
    )
    const items = filled.map((it) => ({
      name: it.name.trim(),
      amount_cents: toCents(it.amount) ?? 0,
      category_id: it.category_id,
    }))
    if (
      items.some(
        (it) => !it.name || it.amount_cents <= 0 || it.category_id === ""
      )
    ) {
      setError(t.invalidItem)
      return
    }
    // The server refuses this too, and in English: checking here is what gets
    // the household an Italian sentence rather than a bare failed save.
    if (itemsCents(filled) > cents) {
      setError(t.itemsOverTotal)
      return
    }

    try {
      await api(
        editing === null ? "/api/expenses" : `/api/expenses/${editing}`,
        {
          method: editing === null ? "POST" : "PATCH",
          // The draft's field names are the API's, so the body is the draft
          // with the typed amount swapped for whole cents. JSON.stringify
          // drops the undefined, which is why `amount` does not travel.
          body: JSON.stringify({
            ...draft,
            amount: undefined,
            amount_cents: cents,
            items,
          }),
        }
      )
    } catch {
      setError(t.expenseNotSaved)
      return
    }
    // A correction is finished; a new Expense is often one of several from the
    // same trip, so the date, Category, Payer and method stay where they are.
    const wasEditing = editing !== null
    if (!wasEditing) toast(t.added)
    setEditing(null)
    setDraft((d) =>
      wasEditing ? blankDraft() : { ...d, amount: "", store: "", note: "", items: [] }
    )
    if (wasEditing) setDetailsOpen(true)
    await load()
  }

  async function removeExpense(e: Expense) {
    if (!confirm(t.confirmDeleteExpense(formatCents(e.amount_cents)))) return
    setError("")
    try {
      await api(`/api/expenses/${e.id}`, { method: "DELETE" })
    } catch {
      setError(t.expenseNotDeleted)
      return
    }
    if (editing === e.id) {
      setEditing(null)
      setDraft(blankDraft())
      setDetailsOpen(false)
    }
    await load()
  }

  // What a picker offers, which is the narrower list: no hidden Category, and
  // nothing income-only — "Freelance" is never an Expense. Plus whichever one
  // the entry being edited already carries, whatever it is. An Item answers to
  // the same rule as its Expense, so both pickers come from here.
  const pickable = (chosen: number | "") =>
    pickableCategories(
      categories,
      "expense",
      categories.find((c) => c.id === chosen)
    )

  // Whether the Tax year field belongs on screen, which is only for an Expense
  // in a tax Category. Ticket 03: Investments is also a Base category now, and
  // it applies to both sides — so `base` alone no longer picks out Taxes
  // uniquely, and the expense-only half of it is what does. The code behind
  // `base` is not published — it is an implementation detail of the reports —
  // and this is the one question the boolean answers here.
  const isTaxExpense = categories.some(
    (c) => c.id === draft.category_id && c.base && c.applies_to === "expense"
  )

  // What the field shows before anything is typed: the year of the payment,
  // which is the same default the server applies to a 0. The common case
  // needs no typing, and the real case — tax paid in 2027 on 2026's income —
  // is one edit away.
  const taxYear = draft.tax_year || Number(draft.occurred_on.slice(0, 4)) || ""

  const setItem = (i: number, over: Partial<ItemDraft>) =>
    setDraft((d) => ({
      ...d,
      items: d.items.map((it, j) => (j === i ? { ...it, ...over } : it)),
    }))

  // What the Expense's own Category keeps: the total less whatever the Items
  // claim. Shown rather than enforced-by-arithmetic, because the total is the
  // authoritative number and this is derived from it — never the reverse.
  const remainder = (toCents(draft.amount) ?? 0) - itemsCents(draft.items)

  // The Store suggestions are the Stores already used, which the list on this
  // screen already carries — no endpoint and no list to maintain, which is the
  // whole point of Store being free text.
  const stores = [
    ...new Set(expenses?.map((e) => e.store).filter(Boolean)),
  ].sort()

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.date}</TableHead>
                <TableHead className="text-right">{t.amount}</TableHead>
                <TableHead>{t.category}</TableHead>
                <TableHead className="hidden md:table-cell">
                  {t.store}
                </TableHead>
                <TableHead>{t.payer}</TableHead>
                <TableHead>{t.paymentMethod}</TableHead>
                <TableHead className="hidden md:table-cell">
                  {t.note}
                </TableHead>
                <TableHead>{t.details}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {expenses?.map((e) => (
                <TableRow
                  key={e.id}
                  className={editing === e.id ? "opacity-50" : ""}
                >
                  <TableCell className="text-xs text-muted-foreground">
                    {formatDate(e.occurred_on)}
                  </TableCell>
                  <TableCell className="text-right font-medium tabular-nums">
                    <SpoilerAmount
                      cents={e.amount_cents}
                      gift={isGiftCategory(categories, e.category_id)}
                    />
                  </TableCell>
                  <TableCell>{nameOf(categories, e.category_id)}</TableCell>
                  <TableCell className="hidden truncate text-xs text-muted-foreground md:table-cell">
                    {e.store}
                  </TableCell>
                  <TableCell className="truncate text-xs text-muted-foreground">
                    {e.payer}
                  </TableCell>
                  <TableCell className="truncate text-xs text-muted-foreground">
                    {e.payment_method}
                  </TableCell>
                  <TableCell className="hidden truncate text-xs text-muted-foreground md:table-cell">
                    {e.note}
                  </TableCell>
                  <TableCell className="whitespace-normal">
                    <div className="flex flex-col gap-0.5">
                      {/* Negozio and Note have their own columns above, hidden
                          below md — this is the one place left to reach them
                          on a narrow screen. */}
                      {(e.store || e.note) && (
                        <button
                          type="button"
                          className="flex items-center gap-1 text-xs text-muted-foreground md:hidden"
                          onClick={() => toggleExpanded(e.id)}
                          aria-expanded={expandedIds.has(e.id)}
                          aria-label={t.details}
                        >
                          {expandedIds.has(e.id) ? (
                            <RiArrowUpSLine className="size-3.5" />
                          ) : (
                            <RiArrowDownSLine className="size-3.5" />
                          )}
                          {t.details}
                        </button>
                      )}
                      {expandedIds.has(e.id) && (
                        <div className="flex flex-col gap-0.5 md:hidden">
                          {e.store && (
                            <span className="truncate text-xs text-muted-foreground">
                              {t.store}: {e.store}
                            </span>
                          )}
                          {e.note && (
                            <span className="truncate text-xs text-muted-foreground">
                              {t.note}: {e.note}
                            </span>
                          )}
                        </div>
                      )}
                      {/* Each Item on its own line, indented under the Expense it
                          was broken out of: the point of an Item is that this
                          part of the €62 shop counts as something else, so the
                          row has to say so and name the Category it went to. */}
                      {e.items.map((it, i) => (
                        <span
                          key={i}
                          className="flex items-baseline justify-between gap-2 pl-3 text-xs text-muted-foreground"
                        >
                          <span className="truncate">
                            ↳ {it.name} · {nameOf(categories, it.category_id)}
                          </span>
                          <span className="tabular-nums">
                            € {formatCents(it.amount_cents)}
                          </span>
                        </span>
                      ))}
                    </div>
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
                        <DropdownMenuItem onClick={() => selectExpense(e)}>
                          <RiEditLine /> {t.editExpense}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => void removeExpense(e)}
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
          {expenses?.length === 0 && (
            <p className="text-sm text-muted-foreground">{t.noExpensesYet}</p>
          )}
        </div>

        <FormSidebar
          title={editing === null ? t.addExpense : t.editExpense}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          openMobile={openMobileOnArrival}
        >
          <form onSubmit={submit} className="flex flex-col gap-3">
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
                <FieldLabel htmlFor="occurred_on">{t.date}</FieldLabel>
                <DatePicker
                  id="occurred_on"
                  value={draft.occurred_on}
                  onValueChange={(v) => set("occurred_on", v)}
                  className="w-full"
                />
              </Field>
              <Field className="flex-1">
                <FieldLabel htmlFor="category">{t.category}</FieldLabel>
                <Select
                  value={draft.category_id === "" ? undefined : String(draft.category_id)}
                  onValueChange={(v) => set("category_id", Number(v))}
                  required
                >
                  <SelectTrigger id="category" className="h-10 w-full">
                    <SelectValue placeholder={t.chooseCategory} />
                  </SelectTrigger>
                  <SelectContent>
                    {pickable(draft.category_id).map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </div>

            {/* Only for a tax payment, and outside the details disclosure: for
                this one Expense the year it relates to is not a detail, it is the
                field the yearly summary is computed from. It defaults to the year
                of the payment and stays editable, because tax on 2026's income is
                paid during 2027. */}
            {isTaxExpense && (
              <Field>
                <FieldLabel htmlFor="tax_year">{t.taxYear}</FieldLabel>
                {/* The bounds are the browser's to enforce: a number input with
                    min and max refuses a half-typed 202 itself, in the phone's own
                    language, and the server refuses the same range in English.
                    Clearing the field snaps back to the payment's year rather than
                    showing empty, so there is nothing to mark required. */}
                <Input
                  id="tax_year"
                  type="number"
                  inputMode="numeric"
                  min={1000}
                  max={9999}
                  value={taxYear}
                  onChange={(e) => set("tax_year", Number(e.target.value))}
                  className="h-10"
                />
                <FieldDescription>{t.taxYearHint}</FieldDescription>
              </Field>
            )}

            {/* The details are what make an entry recognisable months later, and
                none of them is worth a tap at the till — so they are a
                disclosure, closed unless the entry needs them. An edit opens it,
                because that is what a correction is usually about. */}
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
                    <FieldLabel htmlFor="store">{t.store}</FieldLabel>
                    {/* A native datalist: free text that suggests what has been
                        typed before, without becoming a list to maintain. */}
                    <Input
                      id="store"
                      list="stores"
                      value={draft.store}
                      onChange={(e) => set("store", e.target.value)}
                      className="h-10"
                    />
                    <datalist id="stores">
                      {stores.map((s) => (
                        <option key={s} value={s} />
                      ))}
                    </datalist>
                  </Field>

                  <div className="flex gap-2">
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
                    <Field className="flex-1">
                      <FieldLabel htmlFor="payment_method">
                        {t.paymentMethod}
                      </FieldLabel>
                      <Select
                        value={draft.payment_method || NO_PAYMENT_METHOD}
                        onValueChange={(v) =>
                          set("payment_method", v === NO_PAYMENT_METHOD ? "" : v)
                        }
                      >
                        <SelectTrigger id="payment_method" className="h-10 w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value={NO_PAYMENT_METHOD}>
                            {t.notSet}
                          </SelectItem>
                          {withSaved(
                            lists.payment_methods,
                            draft.payment_method || undefined
                          ).map((m) => (
                            <SelectItem key={m} value={m}>
                              {m}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>

                  <Field>
                    <FieldLabel htmlFor="note">{t.note}</FieldLabel>
                    <Textarea
                      id="note"
                      value={draft.note}
                      onChange={(e) => set("note", e.target.value)}
                      rows={2}
                    />
                  </Field>

                  {/* The breakdown: the book bought during the grocery shop. Rows
                      are added one at a time and the receipt is never itemised in
                      full, so what is on screen is only the exceptions — with the
                      remainder underneath, saying what the Expense's own Category
                      still keeps. */}
                  <div className="flex flex-col gap-2">
                    <span className="text-sm">{t.items}</span>
                    {draft.items.map((it, i) => (
                      <div key={i} className="flex items-end gap-2">
                        <Input
                          value={it.name}
                          onChange={(e) => setItem(i, { name: e.target.value })}
                          placeholder={t.itemName}
                          aria-label={t.itemName}
                          className="h-10 flex-1"
                        />
                        <Input
                          type="text"
                          inputMode="decimal"
                          value={it.amount}
                          onChange={(e) => setItem(i, { amount: e.target.value })}
                          placeholder="0,00"
                          aria-label={t.amount}
                          className="h-10 w-20"
                        />
                        <Select
                          value={it.category_id === "" ? undefined : String(it.category_id)}
                          onValueChange={(v) => setItem(i, { category_id: Number(v) })}
                        >
                          <SelectTrigger aria-label={t.category} className="h-10 flex-1">
                            <SelectValue placeholder={t.chooseCategory} />
                          </SelectTrigger>
                          <SelectContent>
                            {pickable(it.category_id).map((c) => (
                              <SelectItem key={c.id} value={String(c.id)}>
                                {c.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          aria-label={t.removeItem}
                          className="size-10 shrink-0 text-lg"
                          onClick={() =>
                            set(
                              "items",
                              draft.items.filter((_, j) => j !== i)
                            )
                          }
                        >
                          ×
                        </Button>
                      </div>
                    ))}
                    <Button
                      type="button"
                      variant="outline"
                      className="h-10"
                      onClick={() => set("items", [...draft.items, blankItem()])}
                    >
                      {t.addItem}
                    </Button>
                    {/* The remainder, live as the amounts are typed. Once it goes
                        negative there is no remainder to name — that is the refusal
                        ADR-0002 describes, said here rather than on submit, while
                        the number that caused it is still under the thumb. */}
                    {draft.items.length > 0 && draft.category_id !== "" && (
                      <span
                        className={`text-xs ${
                          remainder < 0 ? "text-destructive" : "text-muted-foreground"
                        }`}
                      >
                        {remainder < 0
                          ? t.itemsOverTotal
                          : t.remainderIn(
                              nameOf(categories, draft.category_id),
                              formatCents(remainder)
                            )}
                      </span>
                    )}
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>

            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}

            {/* Sticky rather than in-flow: the form above can run long
                (Items, notes, ...), and the submit button is the one thing
                that must stay reachable without scrolling all the way down. */}
            <div className="sticky bottom-0 -mx-4 -mb-4 flex flex-col gap-2 border-t bg-background p-4">
              <Button type="submit" size="lg" className="h-12 text-base">
                {editing === null ? t.addExpense : t.save}
              </Button>
              {editing !== null && (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setEditing(null)
                    setDraft(blankDraft())
                    setDetailsOpen(true)
                  }}
                >
                  {t.cancel}
                </Button>
              )}
            </div>
          </form>
        </FormSidebar>
      </div>
    </div>
  )
}
