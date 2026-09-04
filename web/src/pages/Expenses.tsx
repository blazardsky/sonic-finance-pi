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
import { formatCents, formatDate, toCents, today, toTyped } from "@/lib/money"
import { nameOf, pickableCategories, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Expense, Lists } from "@/types"

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

// The details an Expense actually carries, on one line, for the list — empty
// when it has none, which is the same question as whether to show the line.
const details = (e: Expense) =>
  [e.store, e.payer, e.payment_method, e.note].filter(Boolean).join(" · ")

// itemsCents sums a draft breakdown, skipping what is not yet an amount. The
// Expense's own total is never computed from this — ADR-0002 — it is only what
// the form checks the total against and shows the remainder from.
const itemsCents = (items: ItemDraft[]) =>
  items.reduce((sum, it) => sum + (toCents(it.amount) ?? 0), 0)

// The main screen: log what was just spent in about three taps, and see it
// land. The form is first and biggest because it is what the app is for; the
// list under it is confirmation, and the way back into an entry that came out
// wrong — tapping one loads it into the same form, which is why there is only
// ever one form on the screen.
export function Expenses() {
  const [expenses, setExpenses] = useState<Expense[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [lists, setLists] = useState<Lists>({ payers: [], payment_methods: [] })
  const [error, setError] = useState("")

  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  // Tapping (or, from the keyboard, activating) a row is how an entry gets
  // corrected — shared by the row's click and its Enter/Space handling below,
  // which is the table's replacement for the list's own <button>.
  const selectExpense = (e: Expense) => {
    setEditing(e.id)
    setDraft(draftOf(e))
    scrollTo({ top: 0, behavior: "smooth" })
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
    setEditing(null)
    setDraft((d) =>
      editing === null
        ? { ...d, amount: "", store: "", note: "", items: [] }
        : blankDraft()
    )
    await load()
  }

  async function remove() {
    if (editing === null) return
    const e = expenses?.find((x) => x.id === editing)
    if (e && !confirm(t.confirmDeleteExpense(formatCents(e.amount_cents))))
      return
    setError("")
    try {
      await api(`/api/expenses/${editing}`, { method: "DELETE" })
    } catch {
      setError(t.expenseNotDeleted)
      return
    }
    setEditing(null)
    setDraft(blankDraft())
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
            {t.date}
            {/* A native date input, so the phone gives its own picker and the
                value is already the YYYY-MM-DD the API wants. */}
            <Input
              type="date"
              value={draft.occurred_on}
              onChange={(e) => set("occurred_on", e.target.value)}
              required
              className="h-10"
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.category}
            <NativeSelect
              value={draft.category_id}
              onChange={(e) => set("category_id", Number(e.target.value))}
              required
              className="h-10"
            >
              <option value="">{t.chooseCategory}</option>
              {pickable(draft.category_id).map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </NativeSelect>
          </label>
        </div>

        {/* Only for a tax payment, and outside the details disclosure: for
            this one Expense the year it relates to is not a detail, it is the
            field the yearly summary is computed from. It defaults to the year
            of the payment and stays editable, because tax on 2026's income is
            paid during 2027. */}
        {isTaxExpense && (
          <label className="flex flex-col gap-1.5 text-sm">
            {t.taxYear}
            {/* The bounds are the browser's to enforce: a number input with
                min and max refuses a half-typed 202 itself, in the phone's own
                language, and the server refuses the same range in English.
                Clearing the field snaps back to the payment's year rather than
                showing empty, so there is nothing to mark required. */}
            <Input
              type="number"
              inputMode="numeric"
              min={1000}
              max={9999}
              value={taxYear}
              onChange={(e) => set("tax_year", Number(e.target.value))}
              className="h-10"
            />
            <span className="text-xs text-muted-foreground">
              {t.taxYearHint}
            </span>
          </label>
        )}

        {/* The details are what make an entry recognisable months later, and
            none of them is worth a tap at the till — so they are a native
            disclosure, closed unless the entry needs them. An edit opens it,
            because that is what a correction is usually about. */}
        <details open={editing !== null} className="flex flex-col gap-3">
          <summary className="cursor-pointer py-1 text-sm text-muted-foreground">
            {t.moreDetails}
          </summary>
          <div className="flex flex-col gap-3 pt-2">
            <label className="flex flex-col gap-1.5 text-sm">
              {t.store}
              {/* A native datalist: free text that suggests what has been
                  typed before, without becoming a list to maintain. */}
              <Input
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
            </label>

            <div className="flex gap-2">
              <label className="flex flex-1 flex-col gap-1.5 text-sm">
                {t.payer}
                <NativeSelect
                  value={draft.payer}
                  onChange={(e) => set("payer", e.target.value)}
                  required
                  className="h-10"
                >
                  <option value="">{t.chooseCategory}</option>
                  {withSaved(lists.payers, draft.payer || undefined).map(
                    (p) => (
                      <option key={p} value={p}>
                        {p}
                      </option>
                    )
                  )}
                </NativeSelect>
              </label>
              <label className="flex flex-1 flex-col gap-1.5 text-sm">
                {t.paymentMethod}
                <NativeSelect
                  value={draft.payment_method}
                  onChange={(e) => set("payment_method", e.target.value)}
                  className="h-10"
                >
                  <option value="">{t.notSet}</option>
                  {withSaved(
                    lists.payment_methods,
                    draft.payment_method || undefined
                  ).map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </NativeSelect>
              </label>
            </div>

            <label className="flex flex-col gap-1.5 text-sm">
              {t.note}
              <Textarea
                value={draft.note}
                onChange={(e) => set("note", e.target.value)}
                rows={2}
              />
            </label>

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
                  <NativeSelect
                    value={it.category_id}
                    // Back to the blank option is "" and not Number("") — 0,
                    // which no Category has, would slip past the unchosen
                    // check and travel as a Category the server refuses.
                    // The Expense's own select is `required`, so it never
                    // reaches this; an Item row has no such guard.
                    onChange={(e) =>
                      setItem(i, {
                        category_id:
                          e.target.value === "" ? "" : Number(e.target.value),
                      })
                    }
                    aria-label={t.category}
                    className="h-10 flex-1"
                  >
                    <option value="">{t.chooseCategory}</option>
                    {pickable(it.category_id).map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </NativeSelect>
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
          </div>
        </details>

        <Button type="submit" size="lg" className="h-12 text-base">
          {editing === null ? t.addExpense : t.save}
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
            <TableHead>{t.category}</TableHead>
            <TableHead>{t.details}</TableHead>
            <TableHead>{t.date}</TableHead>
            <TableHead className="text-right">{t.amount}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {expenses?.map((e) => (
            <TableRow
              key={e.id}
              role="button"
              tabIndex={0}
              aria-label={t.editExpense}
              onClick={() => selectExpense(e)}
              onKeyDown={(ev) => {
                // Only the row itself, not a bubbled key from something
                // inside it — this row has nothing else focusable, but the
                // guard is what keeps this honest as rows change shape.
                if (ev.target !== ev.currentTarget) return
                if (ev.key !== "Enter" && ev.key !== " ") return
                ev.preventDefault()
                selectExpense(e)
              }}
              className={`cursor-pointer ${editing === e.id ? "opacity-50" : ""}`}
            >
              <TableCell>{nameOf(categories, e.category_id)}</TableCell>
              <TableCell className="whitespace-normal">
                <div className="flex flex-col gap-0.5">
                  {/* Whichever details were filled in, on one quiet line: it
                      is what the household reads the list for. */}
                  {details(e) && (
                    <span className="truncate text-xs text-muted-foreground">
                      {details(e)}
                    </span>
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
              <TableCell className="text-xs text-muted-foreground">
                {formatDate(e.occurred_on)}
              </TableCell>
              <TableCell className="text-right font-medium tabular-nums">
                € {formatCents(e.amount_cents)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {expenses?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noExpensesYet}</p>
      )}
    </div>
  )
}
