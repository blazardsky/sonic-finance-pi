import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api, apiJSON } from "@/api"
import { formatCents, formatDate, toCents, toTyped } from "@/money"
import { t } from "@/strings"
import type { Category, Expense, Lists } from "@/types"

// today is read here rather than from the server: the phone in the hand at the
// till has a correct clock, and the Pi has no RTC. Local date parts, not
// toISOString, which would hand back yesterday for most of an Italian evening.
function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// What the form holds: the amount as it was typed, and everything else as the
// API's own field names, so submitting is one spread rather than a mapping.
type Draft = Omit<Expense, "id" | "amount_cents" | "category_id"> & {
  amount: string
  // "" is the unchosen picker, which the required select refuses to submit.
  category_id: number | ""
}

const blankDraft = (): Draft => ({
  amount: "",
  occurred_on: today(),
  category_id: "",
  store: "",
  payer: "",
  payment_method: "",
  note: "",
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
})

// A picker keeps whatever the Expense being edited was saved with, even when
// the list no longer offers it: a hidden Category and a Payer renamed out of
// the list are both still what that entry says, and a select with no matching
// option would quietly rewrite it on the first correction.
function withSaved<T>(offered: T[], saved: T | undefined): T[] {
  return saved !== undefined && !offered.includes(saved)
    ? [...offered, saved]
    : offered
}

// The details an Expense actually carries, on one line, for the list — empty
// when it has none, which is the same question as whether to show the line.
const details = (e: Expense) =>
  [e.store, e.payer, e.payment_method, e.note].filter(Boolean).join(" · ")

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
        ? { ...d, amount: "", store: "", note: "" }
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

  // A name for the id an Expense carries. Every Category is looked up, hidden
  // ones included: hiding takes a Category out of the picker, not out of the
  // entries already filed under it, and an old Expense showing a blank label
  // is the bug that would make hiding unsafe.
  const categoryName = (id: number) =>
    categories.find((c) => c.id === id)?.name ?? ""

  // What the picker offers, which is the narrower list: no hidden Category,
  // and nothing income-only — "Freelance" is never an Expense. Plus the one
  // the Expense being edited is already in, whatever it is.
  const pickable = withSaved(
    categories.filter((c) => !c.hidden && c.applies_to !== "income"),
    categories.find((c) => c.id === draft.category_id)
  )

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
              {pickable.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </NativeSelect>
          </label>
        </div>

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
                  className="h-10"
                >
                  <option value="">{t.notSet}</option>
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

      <ul className="flex flex-col divide-y divide-border">
        {expenses?.map((e) => (
          <li key={e.id}>
            <button
              type="button"
              aria-label={t.editExpense}
              onClick={() => {
                setEditing(e.id)
                setDraft(draftOf(e))
                scrollTo({ top: 0, behavior: "smooth" })
              }}
              className={`flex w-full flex-col gap-0.5 py-2.5 text-left ${
                editing === e.id ? "opacity-50" : ""
              }`}
            >
              <span className="flex items-baseline justify-between gap-2">
                <span>{categoryName(e.category_id)}</span>
                <span className="flex items-baseline gap-3">
                  <span className="text-xs text-muted-foreground">
                    {formatDate(e.occurred_on)}
                  </span>
                  <span className="font-medium tabular-nums">
                    € {formatCents(e.amount_cents)}
                  </span>
                </span>
              </span>
              {/* Whichever details were filled in, on one quiet line: it is
                  what the household reads the list for. */}
              {details(e) && (
                <span className="truncate text-xs text-muted-foreground">
                  {details(e)}
                </span>
              )}
            </button>
          </li>
        ))}
      </ul>
      {expenses?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noExpensesYet}</p>
      )}
    </div>
  )
}
