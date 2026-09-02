import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api, apiJSON } from "@/api"
import { formatCents, formatDate, toCents } from "@/money"
import { t } from "@/strings"
import type { Category, Expense } from "@/types"

// today is read here rather than from the server: the phone in the hand at the
// till has a correct clock, and the Pi has no RTC. Local date parts, not
// toISOString, which would hand back yesterday for most of an Italian evening.
function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// The main screen: log what was just spent in about three taps, and see it
// land. The form is first and biggest because it is what the app is for; the
// list under it is confirmation, not a workspace.
export function Expenses() {
  const [expenses, setExpenses] = useState<Expense[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [error, setError] = useState("")

  const [amount, setAmount] = useState("")
  const [occurredOn, setOccurredOn] = useState(today)
  const [categoryID, setCategoryID] = useState("")

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Expense[]>("/api/expenses"),
        apiJSON<Category[]>("/api/categories"),
      ])
        .then(([e, c]) => {
          setExpenses(e)
          setCategories(c)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  async function add(event: React.FormEvent) {
    event.preventDefault()
    setError("")

    // Cents are computed here and sent as an integer: the API refuses a
    // fractional amount_cents outright, so a bad parse cannot become a
    // silently rounded Expense.
    const cents = toCents(amount)
    if (cents === null || cents <= 0) {
      setError(t.invalidAmount)
      return
    }

    try {
      await api("/api/expenses", {
        method: "POST",
        body: JSON.stringify({
          occurred_on: occurredOn,
          amount_cents: cents,
          category_id: Number(categoryID),
        }),
      })
    } catch {
      setError(t.expenseNotSaved)
      return
    }
    // The date stays where it was: a shop run is several Expenses on one day.
    setAmount("")
    await load()
  }

  // A name for the id an Expense carries. Every Category is looked up, hidden
  // ones included: hiding takes a Category out of the picker, not out of the
  // entries already filed under it, and an old Expense showing a blank label
  // is the bug that would make hiding unsafe.
  const categoryName = (id: number) =>
    categories.find((c) => c.id === id)?.name ?? ""

  // What the picker offers, which is the narrower list: no hidden Category,
  // and nothing income-only — "Freelance" is never an Expense.
  const pickable = categories.filter(
    (c) => !c.hidden && c.applies_to !== "income"
  )

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <form onSubmit={add} className="flex flex-col gap-3">
        <label className="flex flex-col gap-1.5 text-sm">
          {t.amount}
          <div className="flex items-center gap-2">
            <span className="text-xl text-muted-foreground">€</span>
            <Input
              // text with a decimal keypad, not type=number: a comma is what
              // the Italian keypad offers and toCents accepts it.
              type="text"
              inputMode="decimal"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
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
              value={occurredOn}
              onChange={(e) => setOccurredOn(e.target.value)}
              required
              className="h-10"
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.category}
            <NativeSelect
              value={categoryID}
              onChange={(e) => setCategoryID(e.target.value)}
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

        <Button type="submit" size="lg" className="h-12 text-base">
          {t.addExpense}
        </Button>
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      <ul className="flex flex-col divide-y divide-border">
        {expenses?.map((e) => (
          <li
            key={e.id}
            className="flex items-baseline justify-between gap-2 py-2.5"
          >
            <span>{categoryName(e.category_id)}</span>
            <span className="flex items-baseline gap-3">
              <span className="text-xs text-muted-foreground">
                {formatDate(e.occurred_on)}
              </span>
              <span className="font-medium tabular-nums">
                € {formatCents(e.amount_cents)}
              </span>
            </span>
          </li>
        ))}
      </ul>
      {expenses?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noExpensesYet}</p>
      )}
    </div>
  )
}
