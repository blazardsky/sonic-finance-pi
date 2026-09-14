import { useCallback, useEffect, useState } from "react"
import {
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiDeleteBinLine,
  RiEditLine,
  RiMoreLine,
  RiRefreshLine,
  RiStopCircleLine,
} from "@remixicon/react"

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Badge } from "@/components/ui/badge"
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
import { ColorDot } from "@/components/ColorDot"
import { FormSidebar } from "@/components/form-sidebar"
import { toast } from "@/lib/toast"
import {
  formatCents,
  formatMonth,
  shiftMonth,
  thisMonth,
  toCents,
  toTyped,
} from "@/lib/money"
import {
  nameOf,
  pickableCategories,
  pickableSubcategories,
  withSaved,
} from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Holding, Lists, Recurring, Subcategory } from "@/types"

// Payer and Payment method are both optional here (unlike Expenses/Incomes'
// required Payer) and default to "—" — Radix Select refuses an empty-string
// item value, so the unset state gets this sentinel, mapped back to "" on
// the way out.
const NONE = "__none__"

// What the form holds: the amount and the day as they were typed, everything
// else as the API's own field names, so submitting is one spread.
type Draft = Omit<
  Recurring,
  "id" | "amount_cents" | "category_id" | "day_of_month" | "holding_id"
> & {
  amount: string
  // "" is the unchosen picker, which the required select refuses to submit.
  category_id: number | ""
  // A string so the field can be emptied while it is being retyped; "" means
  // "say nothing", which the server answers with the day it was created.
  day_of_month: string
  // "" is the unchosen Holding, the same bargain category_id makes — shown
  // only for a recurring investment (ticket 06).
  holding_id: number | ""
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
  holding_id: "",
  subcategory_id: null,
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
  holding_id: r.holding_id ?? "",
  subcategory_id: r.subcategory_id,
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
  const [subcategories, setSubcategories] = useState<Subcategory[]>([])
  const [holdings, setHoldings] = useState<Holding[]>([])
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
  // Which rows have their Dettagli disclosure open — Negozio, Pagato da and
  // Metodo di pagamento have no column of their own; Status and Note do, and
  // are never in here.
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set())
  const toggleExpanded = (id: number) =>
    setExpandedIds((s) => {
      const next = new Set(s)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  // Loads a definition into the form and makes sure the panel holding it is
  // actually visible — the form updates whether it is on screen or not, and a
  // silently updated but hidden form is indistinguishable from a broken one.
  const selectRecurring = (r: Recurring) => {
    setEditing(r.id)
    setDraft(draftOf(r))
    setDetailsOpen(true)
    setSidebarOpen(true)
  }

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Recurring[]>("/api/recurring"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Holding[]>("/api/holdings"),
        apiJSON<Lists>("/api/settings"),
        apiJSON<Subcategory[]>("/api/subcategories"),
      ])
        .then(([r, c, h, l, s]) => {
          setRecurring(r)
          setCategories(c)
          setHoldings(h)
          setLists(l)
          setSubcategories(s)
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
          holding_id: draft.holding_id === "" ? null : draft.holding_id,
        }),
      }
    )
    if (!ok) return
    if (editing === null) toast(t.added)
    setEditing(null)
    setDraft(blankDraft())
    setDetailsOpen(false)
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
    setDetailsOpen(false)
    setSidebarOpen(true)
  }

  async function removeRecurring(r: Recurring) {
    if (!confirm(t.confirmDeleteRecurring)) return
    await write(`/api/recurring/${r.id}`, { method: "DELETE" }, t.recurringInUse)
  }

  // What the picker offers: no hidden Category, and nothing income-only — plus
  // whichever one the definition being edited already carries.
  const pickable = pickableCategories(
    categories,
    "expense",
    categories.find((c) => c.id === draft.category_id)
  )

  // A second, independent tag alongside Category — never narrowed by which
  // Category is chosen, the same as Expenses.tsx.
  const pickableSub = pickableSubcategories(
    subcategories,
    "expense",
    subcategories.find((s) => s.id === draft.subcategory_id)
  )

  // Whether a category_id names the Investments Base category specifically —
  // Gift is also a Base category with applies_to "both", so that pair alone
  // can't tell them apart; `investments` is published by the API for exactly
  // this, the same way `gift`/`freelance` are (categories.go).
  const categoryIsInvestments = (categoryId: number | "") =>
    categories.some((c) => c.id === categoryId && c.investments)

  // Whether the Holding picker belongs on screen: only for a recurring
  // investment (PAC), the same way Expenses.tsx gates the Tax year field on
  // isTaxExpense.
  const isInvestmentRecurring = categoryIsInvestments(draft.category_id)

  // A PAC (CONTEXT.md glossary): Investments category AND a Holding, both
  // required — read off the row itself rather than the open form's draft.
  const isPAC = (r: Recurring) =>
    r.holding_id !== null && categoryIsInvestments(r.category_id)

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <h1 className="font-medium">{t.recurring}</h1>

          {/* Kept in the list column, not inside the form: unlike Expenses/
              Incomes, this screen's errors come from row actions (end,
              replace, delete) just as often as from the form's own submit,
              and a row action can fire while the sidebar is closed. */}
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
                <TableHead>{t.status}</TableHead>
                <TableHead className="hidden md:table-cell">
                  {t.note}
                </TableHead>
                <TableHead>{t.details}</TableHead>
                <TableHead className="text-right">{t.amount}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recurring?.map((r) => (
                <TableRow
                  key={r.id}
                  className={editing === r.id ? "opacity-50" : ""}
                >
                  <TableCell
                    className={running(r) ? "" : "text-muted-foreground"}
                  >
                    {nameOf(categories, r.category_id)}
                  </TableCell>
                  <TableCell>
                    {/* The window, the day and the rest move to Dettagli
                        below — this is just running or not. */}
                    <div className="flex flex-wrap gap-1">
                      <Badge variant={running(r) ? "secondary" : "outline"}>
                        {running(r) ? t.ongoing : t.endedIn(formatMonth(r.end_month))}
                      </Badge>
                      <Badge
                        className={
                          isPAC(r)
                            ? "bg-green-500/10 text-green-600 dark:text-green-400"
                            : "bg-orange-500/10 text-orange-600 dark:text-orange-400"
                        }
                      >
                        {isPAC(r) ? t.pacBadge : t.expenseBadge}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell className="hidden truncate text-xs text-muted-foreground md:table-cell">
                    {r.note}
                  </TableCell>
                  <TableCell className="whitespace-normal">
                    <div className="flex w-48 flex-col gap-0.5">
                      <button
                        type="button"
                        className="flex items-center gap-1 text-xs text-muted-foreground"
                        onClick={() => toggleExpanded(r.id)}
                        aria-expanded={expandedIds.has(r.id)}
                        aria-label={t.details}
                      >
                        {expandedIds.has(r.id) ? (
                          <RiArrowUpSLine className="size-3.5" />
                        ) : (
                          <RiArrowDownSLine className="size-3.5" />
                        )}
                        {t.details}
                      </button>
                      {expandedIds.has(r.id) && (
                        <div className="flex flex-col gap-0.5">
                          <span className="truncate text-xs text-muted-foreground">
                            {t.dayOfMonth}: {r.day_of_month} · {windowOf(r)}
                          </span>
                          {r.store && (
                            <span className="truncate text-xs text-muted-foreground">
                              {t.store}: {r.store}
                            </span>
                          )}
                          {r.payer && (
                            <span className="truncate text-xs text-muted-foreground">
                              {t.payer}: {r.payer}
                            </span>
                          )}
                          {r.payment_method && (
                            <span className="truncate text-xs text-muted-foreground">
                              {t.paymentMethod}: {r.payment_method}
                            </span>
                          )}
                          {r.note && (
                            <span className="truncate text-xs text-muted-foreground md:hidden">
                              {t.note}: {r.note}
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-medium tabular-nums">
                    € {formatCents(r.amount_cents)}
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
                        <DropdownMenuItem onClick={() => selectRecurring(r)}>
                          <RiEditLine /> {t.editRecurring}
                        </DropdownMenuItem>
                        {running(r) && (
                          <>
                            <DropdownMenuItem onClick={() => void end(r)}>
                              <RiStopCircleLine /> {t.end}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => void replace(r)}>
                              <RiRefreshLine /> {t.newAmount}
                            </DropdownMenuItem>
                          </>
                        )}
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => void removeRecurring(r)}
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
        </div>

        <FormSidebar
          title={editing === null ? t.addRecurring : t.editRecurring}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          footer={
            <div className="flex flex-col gap-2">
              <Button
                type="submit"
                form="recurring-form"
                size="lg"
                className="h-12 text-base"
              >
                {editing === null ? t.addRecurring : t.save}
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
            id="recurring-form"
            onSubmit={submit}
            className="flex flex-col gap-3"
          >
            <Field>
              <FieldLabel htmlFor="amount">{t.amount}</FieldLabel>
              <InputGroup className="h-10">
                <InputGroupAddon className="text-xl">€</InputGroupAddon>
                <InputGroupInput
                  id="amount"
                  type="text"
                  inputMode="decimal"
                  value={draft.amount}
                  onChange={(e) => set("amount", e.target.value)}
                  placeholder="0,00"
                  required
                />
              </InputGroup>
            </Field>

            <Field>
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

            <Field>
              <FieldLabel htmlFor="subcategory">{t.subcategory}</FieldLabel>
              <Select
                value={
                  draft.subcategory_id === null
                    ? "none"
                    : String(draft.subcategory_id)
                }
                onValueChange={(v) =>
                  set("subcategory_id", v === "none" ? null : Number(v))
                }
              >
                <SelectTrigger id="subcategory" className="h-10 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t.chooseSubcategory}</SelectItem>
                  {pickableSub.map((s) => (
                    <SelectItem key={s.id} value={String(s.id)}>
                      <span className="flex items-center gap-2">
                        <ColorDot color={s.color} />
                        {s.name}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel htmlFor="day_of_month">{t.dayOfMonth}</FieldLabel>
              <Input
                id="day_of_month"
                type="number"
                inputMode="numeric"
                min={1}
                max={31}
                value={draft.day_of_month}
                onChange={(e) => set("day_of_month", e.target.value)}
                className="h-10"
              />
              <FieldDescription>{t.dayClamped}</FieldDescription>
            </Field>

            {/* Only for a recurring investment: which Holding it buys is a
                template property, not a detail, so it sits outside the
                details disclosure — the same placement Expenses.tsx gives
                Tax year. */}
            {isInvestmentRecurring && (
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
            )}

            {/* Native month inputs, one above the other rather than side by
                side: the sidebar's fixed 16rem width has no headroom for two
                fields with visible labels next to it, and — unlike a plain
                date — there is no shadcn Date Picker recipe for month-only
                granularity, so the phone's own month picker (the whole reason
                type="date" was native before ticket 01) stays the better fit
                here. The end is left empty for one that is still running —
                that is the whole of ADR-0005's state, so it is a field rather
                than a switch. */}
            <Field>
              <FieldLabel htmlFor="start_month">{t.startMonth}</FieldLabel>
              <Input
                id="start_month"
                type="month"
                value={draft.start_month}
                onChange={(e) => set("start_month", e.target.value)}
                required
                className="h-10"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="end_month">{t.endMonth}</FieldLabel>
              <Input
                id="end_month"
                type="month"
                value={draft.end_month}
                onChange={(e) => set("end_month", e.target.value)}
                className="h-10"
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
                    <FieldLabel htmlFor="store">{t.store}</FieldLabel>
                    <Input
                      id="store"
                      value={draft.store}
                      onChange={(e) => set("store", e.target.value)}
                      className="h-10"
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="payer">{t.payer}</FieldLabel>
                    <Select
                      value={draft.payer || NONE}
                      onValueChange={(v) =>
                        set("payer", v === NONE ? "" : v)
                      }
                    >
                      <SelectTrigger id="payer" className="h-10 w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={NONE}>{t.notSet}</SelectItem>
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
                    <FieldLabel htmlFor="payment_method">
                      {t.paymentMethod}
                    </FieldLabel>
                    <Select
                      value={draft.payment_method || NONE}
                      onValueChange={(v) =>
                        set("payment_method", v === NONE ? "" : v)
                      }
                    >
                      <SelectTrigger id="payment_method" className="h-10 w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={NONE}>{t.notSet}</SelectItem>
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

          </form>
        </FormSidebar>
      </div>
    </div>
  )
}
