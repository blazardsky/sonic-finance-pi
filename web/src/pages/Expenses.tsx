import { useCallback, useEffect, useMemo, useState } from "react"
import {
  RiDeleteBinLine,
  RiEditLine,
  RiEyeLine,
  RiMoreLine,
} from "@remixicon/react"

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import {
  Autocomplete,
  AutocompleteContent,
  AutocompleteInput,
  AutocompleteItem,
  AutocompleteList,
} from "@/components/ui/autocomplete"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { useSidebar } from "@/components/ui/sidebar"
import { api, apiJSON } from "@/lib/api"
import { CategoryBadge, ColorDot } from "@/components/ColorDot"
import {
  createDataTableColumnHelper,
  DataTable,
  type DataTableColumnDef,
} from "@/components/data-table"
import { DatePicker } from "@/components/date-picker"
import { FormSidebar } from "@/components/form-sidebar"
import { PeriodLabel, PeriodStepper } from "@/components/PeriodStepper"
import { SpoilerAmount } from "@/components/SpoilerAmount"
import { ViewRow } from "@/components/ViewRow"
import { toast } from "@/lib/toast"
import { useDebouncedValue } from "@/hooks/use-debounce"
import {
  formatCents,
  formatPricePerUnit,
  formatDate,
  quantityTyped,
  thisYear,
  toCents,
  toCentsExpr,
  toQuantity,
  today,
  toTyped,
} from "@/lib/money"
import {
  isGiftCategory,
  nameOf,
  pickableCategories,
  pickableSubcategories,
  withSaved,
} from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Expense, Holding, Lists, Subcategory } from "@/types"

// A free-text field that suggests from the household's own history as it
// types (ticket 03's /api/items/suggest and /api/stores/suggest) — built on
// Autocomplete, not Combobox: base-ui's own docs are explicit that Combobox
// is "a filterable Select... does not allow free-form text input", and it
// isn't just a documentation nicety — this codebase hit it directly. Typing
// with Combobox would get silently reset to "" a couple hundred milliseconds
// later, once the debounced fetch below resolved with a fresh suggestion
// list that didn't happen to contain the exact text as a selectable item:
// Combobox reconciles the input against its item list and clears it when it
// finds no match, which anything actually typed rarely will on the first
// try. Autocomplete never does this; `items` is only ever a hint, and typing
// past it is exactly as valid as picking one. `filter={() => true}` turns
// off the primitive's own client-side filtering, because the server already
// ranked what it returned.
function SuggestField({
  id,
  value,
  onValueChange,
  suggestPath,
  placeholder,
  className,
}: {
  id?: string
  value: string
  onValueChange: (v: string) => void
  suggestPath: string
  placeholder?: string
  className?: string
}) {
  const debounced = useDebouncedValue(value, 250)
  const [suggestions, setSuggestions] = useState<string[]>([])

  useEffect(() => {
    let cancelled = false
    const q = debounced.trim()
    // Empty stays a resolved [] rather than an early return, so every branch
    // here sets state from a .then rather than synchronously in the effect
    // body.
    const suggested =
      q === ""
        ? Promise.resolve<string[]>([])
        : apiJSON<string[]>(`${suggestPath}?q=${encodeURIComponent(q)}`).catch(
            () => []
          )
    suggested.then((names) => {
      if (!cancelled) setSuggestions(names)
    })
    return () => {
      cancelled = true
    }
  }, [debounced, suggestPath])

  return (
    <Autocomplete
      items={suggestions}
      value={value}
      onValueChange={onValueChange}
      filter={() => true}
    >
      <AutocompleteInput id={id} placeholder={placeholder} className={className} />
      <AutocompleteContent>
        <AutocompleteList>
          {(name: string) => (
            <AutocompleteItem key={name} value={name}>
              {name}
            </AutocompleteItem>
          )}
        </AutocompleteList>
      </AutocompleteContent>
    </Autocomplete>
  )
}

// The label an Item's unit picks from renders as — Select needs the fixed
// list's own words, kept together with the rest of this vocabulary in
// lib/strings rather than spelled out again here.
const unitLabel = (unit: string) =>
  unit === "kg" ? t.itemUnitKg : unit === "lt" ? t.itemUnitLt : unit === "piece" ? t.itemUnitPiece : ""

// What the payment-method Select needs that a plain string can't say: "no
// method chosen" — Radix Select refuses an empty-string item value, so the
// unset state gets its own sentinel, mapped back to "" on the way out.
const NO_PAYMENT_METHOD = "__none__"

// Same bargain, for an Item's unit Select: "no unit chosen" needs a value
// Radix Select will accept, since "" is reserved for a real item.
const NO_UNIT = "__none__"

// Same bargain again, for the Category Select specifically: passing
// `undefined` for "unset" (rather than a real sentinel) makes the Select
// treat itself as no longer controlled, so it keeps showing whatever was
// last picked instead of clearing — even once `category_id` has genuinely
// reset to "" after a submit.
const NO_CATEGORY = "__none__"

// What the form holds: the amount as it was typed, and everything else as the
// API's own field names, so submitting is one spread rather than a mapping.
// holding_id is left out of the draft entirely: this form never shows a
// Holding picker (ticket 03's Buy/Sell form on the Savings page does), and
// omitting the key from the submitted body — rather than sending null — is
// what leaves a buy's holding_id untouched when it is later edited here for
// some other reason, e.g. its amount.
type Draft = Omit<
  Expense,
  "id" | "amount_cents" | "category_id" | "items" | "holding_id" | "quantity"
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
  // Ticket 01: as typed, like amount — "" is "not set" for all three, since
  // quantity and unit are optional and discounted defaults to false.
  quantity: string
  unit: string
  discounted: boolean
}

const blankItem = (category_id: number | "" = ""): ItemDraft => ({
  name: "",
  amount: "",
  category_id,
  quantity: "",
  unit: "",
  discounted: false,
})

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
  subcategory_id: null,
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
  subcategory_id: e.subcategory_id,
  items: e.items.map((it) => ({
    name: it.name,
    amount: toTyped(it.amount_cents),
    category_id: it.category_id,
    quantity: quantityTyped(it.quantity),
    unit: it.unit,
    discounted: it.discounted,
  })),
})

// Wraps the expense form's own <form> tag only, so it can call useSidebar()
// (ticket 07) — that hook only resolves inside FormSidebar's own
// SidebarProvider, which is an ancestor of this element in the rendered
// tree but not of Expenses() itself, the same split Clients.tsx's own
// ContractForm uses for the same reason. Closes the mobile Sheet once
// onSubmit reports the Expense actually saved; unconditional on desktop too,
// harmlessly, since setOpenMobile there has nothing to close.
function AutoCloseForm({
  id,
  onSubmit,
  className,
  children,
}: {
  id: string
  onSubmit: (event: React.FormEvent) => Promise<boolean>
  className?: string
  children: React.ReactNode
}) {
  const { setOpenMobile } = useSidebar()
  async function handleSubmit(event: React.FormEvent) {
    if (await onSubmit(event)) setOpenMobile(false)
  }
  return (
    <form id={id} onSubmit={(e) => void handleSubmit(e)} className={className}>
      {children}
    </form>
  )
}

// itemsCents sums a draft breakdown, skipping what is not yet an amount. The
// Expense's own total is never computed from this — ADR-0002 — it is only what
// the form checks the total against and shows the remainder from.
const itemsCents = (items: ItemDraft[]) =>
  items.reduce((sum, it) => sum + (toCentsExpr(it.amount) ?? 0), 0)

// How many Expenses the default recent view holds — see the backend's own
// `limit` ceiling on GET /api/expenses. A selected year fetches unbounded
// (v1.3.0: DataTable paginates client-side instead of the old limit/offset
// page-at-a-time server paging).
const RECENT_LIMIT = 50

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
  const [subcategories, setSubcategories] = useState<Subcategory[]>([])
  // Only ever read for the view dialog's Holding name — a buy Expense on
  // this page is rare (Investments' own form is the usual way one gets a
  // holding_id), but "all info about the Expense" means naming it, not just
  // the id.
  const [holdings, setHoldings] = useState<Holding[]>([])
  const [lists, setLists] = useState<Lists>({
    payers: [],
    payment_methods: [],
    target_cents: 0,
    goal_cents: 0,
    savings_starting_balance_cents: 0,
    net_worth_target_cents: 0,
  })
  const [draft, setDraft] = useState<Draft>(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [detailsOpen, setDetailsOpen] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  // The Expense the "Visualizza" action opened, or null when the dialog is
  // closed — Negozio, Pagato da, Metodo di pagamento, Note and the Item
  // breakdown all live only here now, not in the table itself.
  const [viewing, setViewing] = useState<Expense | null>(null)
  // null is the default view: the most recent RECENT_LIMIT Expenses,
  // unfiltered. Picking a year switches to that year's own Expenses,
  // fetched in full — DataTable's own pagination (and its per-column
  // filters) take it from there client-side.
  const [year, setYear] = useState<string | null>(null)
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

  // No year selected: the recent view, `limit` alone. A year selected: that
  // whole year's Expenses, unbounded — the backend's own handler already
  // answers a bare year-scoped GET in full; DataTable pages through it.
  const expensesPath =
    year === null
      ? `/api/expenses?limit=${RECENT_LIMIT}`
      : `/api/expenses?year=${year}`

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Expense[]>(expensesPath),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Lists>("/api/settings"),
        apiJSON<Holding[]>("/api/holdings"),
        apiJSON<Subcategory[]>("/api/subcategories"),
      ])
        .then(([e, c, l, h, s]) => {
          setExpenses(e)
          setCategories(c)
          setLists(l)
          setHoldings(h)
          setSubcategories(s)
        })
        .catch(() => toast(t.serverUnreachable)),
    [expensesPath]
  )

  useEffect(() => {
    void load()
  }, [load])

  // Returns whether the Expense was actually saved, so the form (which alone
  // knows about the mobile Sheet, via AutoCloseForm below) can close it only
  // once there is something to close for — the same split Clients.tsx's own
  // addContract/ContractForm uses.
  async function submit(event: React.FormEvent): Promise<boolean> {
    event.preventDefault()
    // Cents are computed here and sent as an integer: the API refuses a
    // fractional amount_cents outright, so a bad parse cannot become a
    // silently rounded Expense.
    const cents = toCents(draft.amount)
    if (cents === null || cents <= 0) {
      toast(t.invalidAmount)
      return false
    }

    // Caught here rather than left to the server's opaque failure: a Select
    // stuck showing a stale category after the form reset (the NO_CATEGORY
    // sentinel fix above addresses the visual half of it) would otherwise
    // submit "" for category_id.
    if (draft.category_id === "") {
      toast(t.invalidCategory)
      return false
    }

    // The server refuses an Expense with no Payer, but Payer lives inside the
    // details disclosure — closed, its Select unmounts, so the browser's own
    // required check on it never runs. Checked here instead, opening the
    // disclosure so the field the error is about is what the household sees.
    if (draft.payer.trim() === "") {
      toast(t.invalidPayer)
      setDetailsOpen(true)
      return false
    }

    // A row added and then left alone is not an Item, so it is dropped rather
    // than refused — anything half-filled in is a mistake worth stopping on.
    const filled = draft.items.filter(
      (it) => it.name.trim() !== "" || it.amount !== "" || it.category_id !== ""
    )
    const items = filled.map((it) => {
      const quantity = toQuantity(it.quantity)
      return {
        name: it.name.trim(),
        // Ticket 08: the amount may be a typed "x*y"/"x-y"/"x+y" rather than
        // a plain number — evaluated the same way itemsCents already reads
        // it back, so what gets sent matches what the remainder was checked
        // against.
        amount_cents: toCentsExpr(it.amount) ?? 0,
        category_id: it.category_id,
        // A unit typed against a quantity that failed to parse is not a real
        // pair either, so it is dropped along with it rather than sent alone.
        quantity,
        unit: quantity === null ? "" : it.unit,
        discounted: it.discounted,
      }
    })
    if (
      items.some(
        (it) => !it.name || it.amount_cents <= 0 || it.category_id === ""
      )
    ) {
      toast(t.invalidItem)
      return false
    }
    // Ticket 01: quantity and unit are a pair, the same rule the server
    // enforces — checked here too so the household gets it in Italian rather
    // than as a bare failed save.
    if (items.some((it) => (it.quantity === null) !== (it.unit === ""))) {
      toast(t.invalidItemQuantityUnit)
      return false
    }
    // The server refuses this too, and in English: checking here is what gets
    // the household an Italian sentence rather than a bare failed save.
    if (itemsCents(filled) > cents) {
      toast(t.itemsOverTotal)
      return false
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
      toast(t.expenseNotSaved)
      return false
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
    return true
  }

  async function removeExpense(e: Expense) {
    if (!confirm(t.confirmDeleteExpense(formatCents(e.amount_cents)))) return
    try {
      await api(`/api/expenses/${e.id}`, { method: "DELETE" })
    } catch {
      toast(t.expenseNotDeleted)
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

  // The Expense's own Subcategory picker — a second, independent tag, never
  // narrowed by which Category is chosen: the whole point is that the same
  // one pairs with any of them.
  const pickableSub = pickableSubcategories(
    subcategories,
    "expense",
    subcategories.find((s) => s.id === draft.subcategory_id)
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

  // -1: further back is always available, arbitrarily. +1: a step past the
  // current year is what returns to the recent (unfiltered) view — there is
  // no "current year + 1" to step into instead, since a year that has not
  // happened has no Expenses (Year.tsx's own stepper stops there the same
  // way).
  const stepYear = (delta: number) => {
    setYear((y) => {
      if (delta < 0) return y === null ? thisYear() : String(Number(y) - 1)
      return y !== null && y < thisYear() ? String(Number(y) + 1) : null
    })
  }

  // v1.3.0: DataTable's own sortable headers, per-column filters (Date,
  // Amount, Category — the table's only columns today, matching what it
  // showed before this migration), and client-side pagination. Depends on
  // `categories` since Category filters/sorts/renders by name, not the raw
  // category_id. `editing`'s dimmed row is the one thing this migration
  // deliberately drops: DataTable has no per-row className hook, and a
  // household editing a row already sees it live in the open form sidebar.
  const columns = useMemo<DataTableColumnDef<Expense>[]>(() => {
    const helper = createDataTableColumnHelper<Expense>()
    return [
      helper.accessor("occurred_on", {
        header: t.date,
        meta: { filterVariant: "date-range" },
        cell: (info) => (
          <span className="text-xs text-muted-foreground">
            {formatDate(info.getValue())}
          </span>
        ),
      }),
      helper.accessor("amount_cents", {
        header: t.amount,
        meta: { filterVariant: "range" },
        cell: ({ row }) => (
          <div className="font-medium tabular-nums">
            <SpoilerAmount
              cents={row.original.amount_cents}
              gift={isGiftCategory(categories, row.original.category_id)}
            />
          </div>
        ),
      }),
      helper.accessor((e) => nameOf(categories, e.category_id), {
        id: "category",
        header: t.category,
        meta: { filterVariant: "select" },
        cell: ({ row }) => {
          const category = categories.find(
            (c) => c.id === row.original.category_id
          )
          return category ? (
            <CategoryBadge name={category.name} color={category.color} />
          ) : (
            nameOf(categories, row.original.category_id)
          )
        },
      }),
      helper.accessor("store", {
        header: t.store,
        meta: { filterVariant: "text", hiddenOnMobile: true },
        cell: (info) => info.getValue() || "-",
      }),
      helper.display({
        id: "actions",
        header: t.actions,
        enableSorting: false,
        cell: ({ row }) => {
          const e = row.original
          return (
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
                <DropdownMenuItem onClick={() => setViewing(e)}>
                  <RiEyeLine /> {t.viewExpense}
                </DropdownMenuItem>
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
          )
        },
      }),
    ]
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [categories])

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
              <PeriodLabel>{year ?? t.recentExpenses}</PeriodLabel>
            </PeriodStepper>
          </div>
          <DataTable
            columns={columns}
            data={expenses ?? []}
            getRowId={(e) => String(e.id)}
            initialSorting={[{ id: "occurred_on", desc: true }]}
            paginationPosition="both"
          />
        </div>

        <FormSidebar
          title={editing === null ? t.addExpense : t.editExpense}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          openMobile={openMobileOnArrival}
          footer={
            <div className="flex flex-col gap-2">
              <Button
                type="submit"
                form="expense-form"
                size="lg"
                className="h-12 text-base"
              >
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
          }
        >
          <AutoCloseForm
            id="expense-form"
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

            <Field>
              <FieldLabel htmlFor="occurred_on">{t.date}</FieldLabel>
              <DatePicker
                id="occurred_on"
                value={draft.occurred_on}
                onValueChange={(v) => set("occurred_on", v)}
                className="w-full"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="category">{t.category}</FieldLabel>
              <Select
                value={draft.category_id === "" ? NO_CATEGORY : String(draft.category_id)}
                onValueChange={(v) =>
                  set("category_id", v === NO_CATEGORY ? "" : Number(v))
                }
                required
              >
                <SelectTrigger id="category" className="h-10 w-full">
                  <SelectValue placeholder={t.chooseCategory} />
                </SelectTrigger>
                <SelectContent>
                  {pickable(draft.category_id).map((c) => (
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
                    <SuggestField
                      id="store"
                      value={draft.store}
                      onValueChange={(v) => set("store", v)}
                      suggestPath="/api/stores/suggest"
                      className="h-10"
                    />
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
                    {draft.items.map((it, i) => {
                      // Live, client-side only: the same division the server
                      // does (ADR-0014), read back off the draft as it is
                      // typed, purely for feedback — quantity/unit are all
                      // that is ever sent back.
                      const quantity = toQuantity(it.quantity)
                      const cents = toCentsExpr(it.amount)
                      const pricePerUnit =
                        quantity !== null && cents !== null && cents > 0
                          ? cents / quantity
                          : null
                      return (
                        <div
                          key={i}
                          className="flex flex-col gap-2 rounded-lg border border-input/50 p-2"
                        >
                          <div className="flex items-center justify-between">
                            <span className="text-xs text-muted-foreground">
                              {t.itemName} {i + 1}
                            </span>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon-sm"
                              aria-label={t.removeItem}
                              className="text-lg"
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
                          {/* Ticket 01's quantity/unit/discounted widened what
                              an Item holds past what a single dense row can
                              lay out — each field gets its own responsive
                              Field row instead: stacked (label above control)
                              in the panel's normal 20rem, a label-left row
                              again once the panel is wide enough to offer one
                              (its own @md/field-group container query, not
                              the viewport's — FormSidebar's expand toggle). */}
                          <FieldGroup className="gap-2">
                            <Field orientation="responsive">
                              <FieldLabel htmlFor={`item-name-${i}`}>
                                {t.itemName}
                              </FieldLabel>
                              <SuggestField
                                id={`item-name-${i}`}
                                value={it.name}
                                onValueChange={(v) => setItem(i, { name: v })}
                                suggestPath="/api/items/suggest"
                                className="h-9"
                              />
                            </Field>
                            <Field orientation="responsive">
                              <FieldLabel htmlFor={`item-amount-${i}`}>
                                {t.amount}
                              </FieldLabel>
                              <Input
                                id={`item-amount-${i}`}
                                type="text"
                                inputMode="text"
                                value={it.amount}
                                onChange={(e) =>
                                  setItem(i, { amount: e.target.value })
                                }
                                // Ticket 08: an expression like "2*3,50" is
                                // typed to save doing the math by hand, but
                                // read back as itself, not as €2,00 — so it
                                // is swapped for its own computed value on
                                // blur, the same way the household would
                                // cross it out and write the total instead.
                                onBlur={() => {
                                  const computed = toCentsExpr(it.amount)
                                  if (computed !== null)
                                    setItem(i, { amount: toTyped(computed) })
                                }}
                                placeholder="0,00 oppure 2*3,50"
                                className="h-9"
                              />
                            </Field>
                            <Field orientation="responsive">
                              <FieldLabel htmlFor={`item-category-${i}`}>
                                {t.category}
                              </FieldLabel>
                              <Select
                                value={
                                  it.category_id === ""
                                    ? NO_CATEGORY
                                    : String(it.category_id)
                                }
                                onValueChange={(v) =>
                                  setItem(i, {
                                    category_id: v === NO_CATEGORY ? "" : Number(v),
                                  })
                                }
                              >
                                <SelectTrigger
                                  id={`item-category-${i}`}
                                  className="h-9"
                                >
                                  <SelectValue placeholder={t.chooseCategory} />
                                </SelectTrigger>
                                <SelectContent>
                                  {pickable(it.category_id).map((c) => (
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
                            <Field orientation="responsive">
                              <FieldLabel htmlFor={`item-quantity-${i}`}>
                                {t.itemQuantity}
                              </FieldLabel>
                              <Input
                                id={`item-quantity-${i}`}
                                type="text"
                                inputMode="decimal"
                                value={it.quantity}
                                onChange={(e) =>
                                  setItem(i, { quantity: e.target.value })
                                }
                                className="h-9"
                              />
                            </Field>
                            <Field orientation="responsive">
                              <FieldLabel htmlFor={`item-unit-${i}`}>
                                {t.itemUnit}
                              </FieldLabel>
                              <Select
                                value={it.unit || NO_UNIT}
                                onValueChange={(v) =>
                                  setItem(i, { unit: v === NO_UNIT ? "" : v })
                                }
                              >
                                <SelectTrigger
                                  id={`item-unit-${i}`}
                                  className="h-9"
                                >
                                  <SelectValue placeholder={t.itemUnit} />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value={NO_UNIT}>{t.notSet}</SelectItem>
                                  <SelectItem value="kg">{t.itemUnitKg}</SelectItem>
                                  <SelectItem value="lt">{t.itemUnitLt}</SelectItem>
                                  <SelectItem value="piece">
                                    {t.itemUnitPiece}
                                  </SelectItem>
                                </SelectContent>
                              </Select>
                            </Field>
                            <FieldLabel className="flex w-fit items-center gap-1.5 text-xs font-normal">
                              <Checkbox
                                checked={it.discounted}
                                onCheckedChange={(v) =>
                                  setItem(i, { discounted: v === true })
                                }
                              />
                              {t.itemDiscounted}
                            </FieldLabel>
                            {pricePerUnit !== null && (
                              <FieldDescription>
                                {t.pricePerUnit(
                                  formatPricePerUnit(pricePerUnit),
                                  unitLabel(it.unit)
                                )}
                              </FieldDescription>
                            )}
                          </FieldGroup>
                        </div>
                      )
                    })}
                    <Button
                      type="button"
                      variant="outline"
                      className="h-10"
                      onClick={() =>
                        set("items", [
                          ...draft.items,
                          blankItem(draft.category_id),
                        ])
                      }
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
          </AutoCloseForm>
        </FormSidebar>
      </div>

      {/* Everything the table's own columns used to show inline (Dettagli's
          disclosure, Note's column) plus everything neither ever did (Tax
          year, the Holding, quantity/unit/discounted per Item) — one place
          for the whole Expense now that the table itself only carries what
          most rows actually need at a glance. */}
      <Dialog
        open={viewing !== null}
        onOpenChange={(open) => !open && setViewing(null)}
      >
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
          {viewing && (
            <>
              <DialogHeader>
                <DialogTitle>{t.expenseDetails}</DialogTitle>
              </DialogHeader>
              <dl className="flex flex-col divide-y divide-border text-sm">
                <ViewRow label={t.date} value={formatDate(viewing.occurred_on)} />
                <ViewRow
                  label={t.amount}
                  value={`€ ${formatCents(viewing.amount_cents)}`}
                />
                <ViewRow
                  label={t.category}
                  value={nameOf(categories, viewing.category_id)}
                />
                {viewing.store && <ViewRow label={t.store} value={viewing.store} />}
                {viewing.payer && <ViewRow label={t.payer} value={viewing.payer} />}
                {viewing.payment_method && (
                  <ViewRow
                    label={t.paymentMethod}
                    value={viewing.payment_method}
                  />
                )}
                {viewing.tax_year > 0 && (
                  <ViewRow label={t.taxYear} value={String(viewing.tax_year)} />
                )}
                {viewing.holding_id !== null && (
                  <ViewRow
                    label={t.holding}
                    value={nameOf(holdings, viewing.holding_id)}
                  />
                )}
                {viewing.note && <ViewRow label={t.note} value={viewing.note} />}
              </dl>
              {viewing.items.length > 0 && (
                <div className="flex flex-col gap-2">
                  <span className="text-sm font-medium">{t.items}</span>
                  <ul className="flex flex-col gap-2">
                    {viewing.items.map((it, i) => (
                      <li
                        key={i}
                        className="flex flex-col gap-0.5 rounded-lg border border-input/50 p-2 text-sm"
                      >
                        <div className="flex items-baseline justify-between gap-2">
                          <span className="truncate">
                            {it.name} · {nameOf(categories, it.category_id)}
                          </span>
                          <span className="tabular-nums">
                            € {formatCents(it.amount_cents)}
                          </span>
                        </div>
                        {it.quantity !== null && (
                          <span className="text-xs text-muted-foreground">
                            {it.quantity} {unitLabel(it.unit)}
                            {it.price_per_unit !== null &&
                              ` · ${t.pricePerUnit(
                                formatPricePerUnit(it.price_per_unit),
                                unitLabel(it.unit)
                              )}`}
                            {it.discounted && ` · ${t.itemDiscounted}`}
                          </span>
                        )}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
