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
  formatMonth,
  shiftMonth,
  thisMonth,
  toCents,
  toTyped,
} from "@/lib/money"
import { nameOf, pickableCategories, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Lists, Recurring } from "@/types"

// What the form holds: the amount and the day as they were typed, everything
// else as the API's own field names, so submitting is one spread.
type Draft = Omit<
  Recurring,
  "id" | "amount_cents" | "category_id" | "day_of_month"
> & {
  amount: string
  // "" is the unchosen picker, which the required select refuses to submit.
  category_id: number | ""
  // A string so the field can be emptied while it is being retyped; "" means
  // "say nothing", which the server answers with the day it was created.
  day_of_month: string
}

const blankDraft = (): Draft => ({
  amount: "",
  category_id: "",
  store: "",
  payer: "",
  payment_method: "",
  note: "",
  day_of_month: "",
  // The current month, which is also what the server would default to — put
  // in the field so it is visible and can be set back before saving.
  start_month: thisMonth(),
  end_month: "",
})

const draftOf = (r: Recurring): Draft => ({
  amount: toTyped(r.amount_cents),
  category_id: r.category_id,
  store: r.store,
  payer: r.payer,
  payment_method: r.payment_method,
  note: r.note,
  day_of_month: String(r.day_of_month),
  start_month: r.start_month,
  end_month: r.end_month,
})

// Still running is the window against the current month, not a stored flag —
// ADR-0005. The month comes from the phone rather than the Pi, which has no
// RTC; both are zero-padded, so this is a string comparison.
//
// Strictly after, not "this month or later": ending one sets its end month to
// the current one, and a definition that still read "in corso" the moment it
// was ended — still offering to end it — would be the screen disagreeing with
// the tap that just happened. It still covers this month, which is what the
// window line says; it is no longer running, which is what the label says.
// Not a component, so fast refresh objects to it being exported alongside
// RecurringExpenses below — same shape as buttonVariants in ui/button.tsx.
// eslint-disable-next-line react-refresh/only-export-components
export const running = (r: Recurring) =>
  !r.end_month || r.end_month > thisMonth()

// Whichever template details were filled in, on one quiet line — empty when
// there are none, which is the same question as whether to show the line.
const details = (r: Recurring) =>
  [r.store, r.payer, r.payment_method, r.note].filter(Boolean).join(" · ")

// The window on one line, which is what "is this still running" is read from.
const windowOf = (r: Recurring) =>
  `${formatMonth(r.start_month)} → ${
    r.end_month ? formatMonth(r.end_month) : "…"
  }`

// The fixed monthly costs in one place. Nothing is generated here — this
// screen only defines what ticket 13 will produce, month by month.
export function RecurringExpenses() {
  const [recurring, setRecurring] = useState<Recurring[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [lists, setLists] = useState<Lists>({ payers: [], payment_methods: [] })
  const [error, setError] = useState("")

  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  // Tapping (or, from the keyboard, activating) a row is how a definition
  // gets corrected — shared by the row's click and its Enter/Space handling
  // below, which is the table's replacement for the list's own <button>.
  const selectRecurring = (r: Recurring) => {
    setEditing(r.id)
    setDraft(draftOf(r))
    scrollTo({ top: 0, behavior: "smooth" })
  }

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Recurring[]>("/api/recurring"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Lists>("/api/settings"),
      ])
        .then(([r, c, l]) => {
          setRecurring(r)
          setCategories(c)
          setLists(l)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // Every write goes through here so the reload and the error wording are in
  // one place. onConflict is the one sentence a caller supplies itself: a 409
  // means Expenses already point at this definition.
  async function write(path: string, init: RequestInit, onConflict?: string) {
    setError("")
    try {
      await api(path, init)
      await load()
      return true
    } catch (res) {
      if (!(res instanceof Response)) setError(t.serverUnreachable)
      else
        setError(
          res.status === 409 && onConflict ? onConflict : t.recurringNotSaved
        )
      return false
    }
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setError("")

    const cents = toCents(draft.amount)
    if (cents === null || cents <= 0) {
      setError(t.invalidAmount)
      return
    }
    // "" is left to the server, which fills in the day it was set up. Anything
    // typed has to be a real day of a month — the server refuses this too, in
    // English; checking here is what gets the household an Italian sentence.
    const day =
      draft.day_of_month === "" ? undefined : Number(draft.day_of_month)
    if (day !== undefined && (!Number.isInteger(day) || day < 1 || day > 31)) {
      setError(t.invalidDayOfMonth)
      return
    }
    if (draft.end_month && draft.end_month < draft.start_month) {
      setError(t.endBeforeStart)
      return
    }
    // The server refuses this too: an ended definition is restarted by
    // creating a new one, never by reopening the window, or generation would
    // fill in the gap months that genuinely had no payment (ADR-0005).
    // Checking here is what gets the household an Italian sentence.
    if (
      editing !== null &&
      !draft.end_month &&
      recurring?.find((r) => r.id === editing)?.end_month
    ) {
      setError(t.noReactivating)
      return
    }

    const ok = await write(
      editing === null ? "/api/recurring" : `/api/recurring/${editing}`,
      {
        method: editing === null ? "POST" : "PATCH",
        body: JSON.stringify({
          ...draft,
          amount: undefined,
          amount_cents: cents,
          day_of_month: day,
        }),
      }
    )
    if (!ok) return
    setEditing(null)
    setDraft(blankDraft())
  }

  // Deactivating: the end month becomes the current one, and that is the whole
  // of it. The months it already covered stay exactly as they were, which is
  // why there is no flag and nothing is deleted.
  async function end(r: Recurring) {
    if (!confirm(t.confirmEndRecurring(formatMonth(thisMonth())))) return false
    return write(`/api/recurring/${r.id}`, {
      method: "PATCH",
      body: JSON.stringify({ end_month: thisMonth() }),
    })
  }

  // The rent goes up. The old definition ends this month and its fields are
  // loaded into the form as a new one starting the month after, so the months
  // before the change keep the amount that was actually true then.
  //
  // The month after, not this one: a window covers its end month, so two
  // definitions both naming the current month would produce two rents for it.
  async function replace(r: Recurring) {
    if (!(await end(r))) return
    setEditing(null)
    setDraft({
      ...draftOf(r),
      // The one field being changed is left empty, because typing it is the
      // whole reason this is a new definition rather than an edit.
      amount: "",
      start_month: shiftMonth(thisMonth(), 1),
      end_month: "",
    })
    scrollTo({ top: 0, behavior: "smooth" })
  }

  // What the picker offers: no hidden Category, and nothing income-only — plus
  // whichever one the definition being edited already carries.
  const pickable = pickableCategories(
    categories,
    "expense",
    categories.find((c) => c.id === draft.category_id)
  )

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <h1 className="font-medium">{t.recurring}</h1>

      <form onSubmit={submit} className="flex flex-col gap-3">
        <div className="flex gap-2">
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
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
                className="h-10"
              />
            </div>
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

        <label className="flex flex-col gap-1.5 text-sm">
          {t.dayOfMonth}
          <Input
            type="number"
            inputMode="numeric"
            min={1}
            max={31}
            value={draft.day_of_month}
            onChange={(e) => set("day_of_month", e.target.value)}
            className="h-10"
          />
          <span className="text-xs text-muted-foreground">{t.dayClamped}</span>
        </label>

        {/* Native month inputs: the phone gives its own picker and the value
            is already the YYYY-MM the API wants. The end is left empty for
            one that is still running — that is the whole of ADR-0005's
            state, so it is a field rather than a switch. */}
        <div className="flex gap-2">
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.startMonth}
            <Input
              type="month"
              value={draft.start_month}
              onChange={(e) => set("start_month", e.target.value)}
              required
              className="h-10"
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm">
            {t.endMonth}
            <Input
              type="month"
              value={draft.end_month}
              onChange={(e) => set("end_month", e.target.value)}
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
              {t.store}
              <Input
                value={draft.store}
                onChange={(e) => set("store", e.target.value)}
                className="h-10"
              />
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
          {editing === null ? t.addRecurring : t.save}
        </Button>
        {editing !== null && (
          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              setEditing(null)
              setDraft(blankDraft())
            }}
          >
            {t.cancel}
          </Button>
        )}
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {recurring?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noRecurringYet}</p>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t.category}</TableHead>
            <TableHead>{t.details}</TableHead>
            <TableHead className="text-right">{t.amount}</TableHead>
            <TableHead>{t.actions}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {recurring?.map((r) => (
            <TableRow
              key={r.id}
              role="button"
              tabIndex={0}
              aria-label={t.editRecurring}
              onClick={() => selectRecurring(r)}
              onKeyDown={(ev) => {
                if (ev.target !== ev.currentTarget) return
                if (ev.key !== "Enter" && ev.key !== " ") return
                ev.preventDefault()
                selectRecurring(r)
              }}
              className={`cursor-pointer ${
                editing === r.id ? "opacity-50" : ""
              }`}
            >
              <TableCell
                className={running(r) ? "" : "text-muted-foreground"}
              >
                {nameOf(categories, r.category_id)}
              </TableCell>
              <TableCell className="whitespace-normal">
                <div className="flex flex-col gap-0.5">
                  {/* Whether it is still running, said in words rather than
                      by a colour alone, next to the window it is read
                      from. */}
                  <span className="text-xs text-muted-foreground">
                    {r.day_of_month} · {windowOf(r)} ·{" "}
                    {running(r)
                      ? t.ongoing
                      : t.endedIn(formatMonth(r.end_month))}
                  </span>
                  {details(r) && (
                    <span className="truncate text-xs text-muted-foreground">
                      {details(r)}
                    </span>
                  )}
                </div>
              </TableCell>
              <TableCell className="text-right font-medium tabular-nums">
                € {formatCents(r.amount_cents)}
              </TableCell>
              {/* Stopped here, at both the click and the key that would
                  otherwise bubble up to the row's own edit-select — these
                  buttons are a second action on the row, not a way into it. */}
              <TableCell
                onClick={(ev) => ev.stopPropagation()}
                onKeyDown={(ev) => ev.stopPropagation()}
              >
                <div className="flex gap-1">
                  {running(r) && (
                    <>
                      <Button
                        size="xs"
                        variant="ghost"
                        onClick={() => void end(r)}
                      >
                        {t.end}
                      </Button>
                      <Button
                        size="xs"
                        variant="ghost"
                        onClick={() => void replace(r)}
                      >
                        {t.newAmount}
                      </Button>
                    </>
                  )}
                  <Button
                    size="xs"
                    variant="destructive"
                    onClick={() => {
                      if (confirm(t.confirmDeleteRecurring))
                        void write(
                          `/api/recurring/${r.id}`,
                          { method: "DELETE" },
                          t.recurringInUse
                        )
                    }}
                  >
                    {t.delete}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
