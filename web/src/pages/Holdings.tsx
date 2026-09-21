import { useCallback, useEffect, useMemo, useState } from "react"
import { RiEditLine, RiEyeLine, RiMoneyEuroCircleLine, RiMoreLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  createDataTableColumnHelper,
  DataTable,
  type DataTableColumnDef,
} from "@/components/data-table"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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
import { api } from "@/lib/api"
import { FormSidebar } from "@/components/form-sidebar"
import { ViewRow } from "@/components/ViewRow"
import { toast } from "@/lib/toast"
import {
  formatCents,
  formatPricePerUnit,
  quantityTyped,
  toCents,
  toQuantity,
  toTyped,
} from "@/lib/money"
import { t } from "@/lib/strings"
import type { Holding, HoldingType } from "@/types"

const typeLabels: Record<HoldingType, string> = {
  etf: t.holdingTypeETF,
  crypto: t.holdingTypeCrypto,
  stock: t.holdingTypeStock,
  bond: t.holdingTypeBond,
  other: t.holdingTypeOther,
}

// initialQuantity/initialAveragePrice only matter while creating (spec:
// "optionally set initial quotas") — both/neither, same bargain a manual
// lot always makes; unused once editing an existing Holding's name/type.
type Draft = {
  name: string
  type: HoldingType
  initialQuantity: string
  initialAveragePrice: string
}

const blankDraft = (): Draft => ({
  name: "",
  type: "etf",
  initialQuantity: "",
  initialAveragePrice: "",
})

// One purchase lot as typed: quantity and its average price both travel as
// plain strings, "" meaning unset, the same bargain every euro/quantity
// input in this app makes. replace picks add-once vs overwrite (see
// manualLotBody below) — meaningless on its own without a quantity/price,
// so it's not surfaced at all in the create form, only the edit dialog.
type PricingDraft = {
  currentPrice: string
  lotQuantity: string
  lotAveragePrice: string
  replace: boolean
}

// A lot needs both a quantity and a price, or neither — a price with
// nothing to weight it against (or a quantity with no price recorded) is
// not a meaningful purchase, per the household's own spec. Returns null on
// "neither" (nothing to send), the parsed pair on "both", and throws
// (caught by the caller) on a mismatched or unparseable one/the-other.
// replace only, since "replace-average without a quantity" is meaningless
// while adding (a lot always needs its own quantity) but well-defined while
// replacing — the server keeps the Holding's current total quantity and
// only replaces the average (applyManualLot, cmd/holding.go). 0 is what
// signals "no quantity typed" to it, same as an omitted field would.
function parseLot(
  quantityInput: string,
  priceTyped: string,
  replace: boolean
): { quantity: number; price_per_unit_cents: number } | null {
  const quantityEmpty = quantityInput.trim() === ""
  const priceEmpty = priceTyped.trim() === ""
  if (quantityEmpty && priceEmpty) return null
  if (priceEmpty) throw new Error(t.invalidManualLot)
  const price = toCents(priceTyped)
  if (price === null) throw new Error(t.invalidManualLot)
  if (quantityEmpty) {
    if (!replace) throw new Error(t.invalidManualLot)
    return { quantity: 0, price_per_unit_cents: price }
  }
  const quantity = toQuantity(quantityInput)
  if (quantity === null) throw new Error(t.invalidManualLot)
  return { quantity, price_per_unit_cents: price }
}

// The Holding management screen: add and rename, the fixed list ticket 03's
// Buy/Sell form and portfolio breakdown read from. No hide or delete in this
// version — a fully-sold Holding simply nets to zero (spec's Out of Scope) —
// so unlike Categories and Clients this list is add-and-edit only.
//
// Value now and gain/loss sit behind a Dettagli view dialog rather than in
// the table itself (the same simplification Expenses/Incomes' own view
// dialogs made), and current_price_cents plus a purchase lot (quantity +
// average price, added to or replacing the running quantity_adjustment/
// cost_adjustment_cents correction — average-purchase-price feature) are
// edited through their own small dialog, the same shape Categories.tsx uses
// for a Subcategory's colour: a focused edit,
// separate from the Name/Type sidebar form.
export function Holdings() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null)
  const [draft, setDraft] = useState(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [viewing, setViewing] = useState<Holding | null>(null)
  const [pricing, setPricing] = useState<Holding | null>(null)
  const [pricingDraft, setPricingDraft] = useState<PricingDraft>({
    currentPrice: "",
    lotQuantity: "",
    lotAveragePrice: "",
    replace: false,
  })

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  // Returns its promise so a write can wait for the reload it triggers, and
  // sets state from a callback rather than an awaited line, which is what
  // keeps the effect below off React's cascading-render path.
  const load = useCallback(
    () =>
      api("/api/holdings")
        .then((res) => res.json())
        .then(setHoldings)
        .catch(() => toast(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  async function write(path: string, init: RequestInit) {
    try {
      await api(path, init)
      await load()
      return true
    } catch {
      toast(t.holdingNotSaved)
      return false
    }
  }

  // Loads a Holding into the form and makes sure the panel holding it is
  // actually visible, same as every other edit action in this effort — the
  // form updates whether it is on screen or not.
  function selectHolding(h: Holding) {
    setEditing(h.id)
    // initialQuantity/initialAveragePrice are create-only (see Draft's own
    // comment) — editing an existing Holding's name/type never touches them.
    setDraft({ name: h.name, type: h.type, initialQuantity: "", initialAveragePrice: "" })
    setSidebarOpen(true)
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    let lot
    try {
      // Creating a Holding is always an add — there's no existing total to
      // replace yet.
      lot = parseLot(draft.initialQuantity, draft.initialAveragePrice, false)
    } catch (e) {
      toast(e instanceof Error ? e.message : t.invalidManualLot)
      return
    }
    const ok = await write(
      editing === null ? "/api/holdings" : `/api/holdings/${editing}`,
      {
        method: editing === null ? "POST" : "PATCH",
        body: JSON.stringify({
          name: draft.name,
          type: draft.type,
          // A lot only ever applies while creating — editing this same form
          // later never carries initialQuantity/initialAveragePrice again
          // (selectHolding always blanks them), so this is never sent twice.
          ...(lot && { manual_lot: { ...lot, replace: false } }),
        }),
      }
    )
    if (!ok) return
    if (editing === null) toast(t.added)
    setEditing(null)
    // Adding several similar Holdings in a row is common (a handful of ETFs
    // set up at once) — the type stays, only the name and lot clear.
    setDraft((d) => ({ ...d, name: "", initialQuantity: "", initialAveragePrice: "" }))
  }

  function openPricing(h: Holding) {
    setPricing(h)
    setPricingDraft({
      currentPrice:
        h.current_price_cents === null ? "" : toTyped(h.current_price_cents),
      lotQuantity: "",
      lotAveragePrice: "",
      replace: false,
    })
  }

  async function submitPricing(event: React.FormEvent) {
    event.preventDefault()
    if (!pricing) return
    let lot
    try {
      lot = parseLot(pricingDraft.lotQuantity, pricingDraft.lotAveragePrice, pricingDraft.replace)
    } catch (e) {
      toast(e instanceof Error ? e.message : t.invalidManualLot)
      return
    }
    const id = pricing.id
    setPricing(null)
    await write(`/api/holdings/${id}`, {
      method: "PATCH",
      body: JSON.stringify({
        // Left blank, current_price_cents is left out of the body entirely
        // rather than sent as null — correcting the quotas or the average
        // price should never require touching (or accidentally clearing) an
        // already-recorded market price.
        ...(pricingDraft.currentPrice.trim() !== "" && {
          current_price_cents: toCents(pricingDraft.currentPrice),
        }),
        ...(lot && { manual_lot: { ...lot, replace: pricingDraft.replace } }),
      }),
    })
  }

  // v1.3.0: DataTable's own sortable headers and per-column filters — same
  // columns as today (Name, Type, Quantity Owned, Paid, Actions).
  const columns = useMemo<DataTableColumnDef<Holding>[]>(() => {
    const helper = createDataTableColumnHelper<Holding>()
    return [
      helper.accessor("name", {
        header: t.holdingName,
        meta: { filterVariant: "text" },
      }),
      helper.accessor((h) => typeLabels[h.type], {
        id: "type",
        header: t.holdingType,
        meta: { filterVariant: "select" },
        cell: (info) => (
          <span className="text-xs text-muted-foreground">{info.getValue()}</span>
        ),
      }),
      helper.accessor("quantity_owned", {
        header: t.quantityOwned,
        meta: { filterVariant: "range" },
        cell: (info) => (
          <div className="text-right tabular-nums">{info.getValue()}</div>
        ),
      }),
      helper.accessor("paid_cents", {
        header: t.paid,
        meta: { filterVariant: "range" },
        cell: ({ row }) => (
          <div className="text-right tabular-nums">
            € {formatCents(row.original.paid_cents)}
          </div>
        ),
      }),
      helper.accessor("average_price_per_unit_cents", {
        header: t.averagePurchasePrice,
        meta: { filterVariant: "range" },
        cell: ({ row }) => {
          const avg = row.original.average_price_per_unit_cents
          return (
            <div className="text-right tabular-nums">
              {avg === null ? "" : `€ ${formatPricePerUnit(avg)}`}
            </div>
          )
        },
      }),
      helper.display({
        id: "actions",
        header: t.actions,
        enableSorting: false,
        cell: ({ row }) => {
          const h = row.original
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
                <DropdownMenuItem onClick={() => setViewing(h)}>
                  <RiEyeLine /> {t.viewHolding}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => openPricing(h)}>
                  <RiMoneyEuroCircleLine /> {t.editPriceAndQuantity}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => selectHolding(h)}>
                  <RiEditLine /> {t.editHolding}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      }),
    ]
  }, [])

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
      <h1 className="font-medium">{t.holdings}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          {holdings?.length === 0 && (
            <p className="text-sm text-muted-foreground">{t.noHoldingsYet}</p>
          )}

          <DataTable
            columns={columns}
            data={holdings ?? []}
            getRowId={(h) => String(h.id)}
          />
        </div>

        <FormSidebar
          title={editing === null ? t.addHolding : t.editHolding}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          footer={
            <div className="flex flex-col gap-2">
              <Button
                type="submit"
                form="holding-form"
                size="lg"
                className="h-12 text-base"
              >
                {editing === null ? t.addHolding : t.save}
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
            </div>
          }
        >
          <form
            id="holding-form"
            onSubmit={submit}
            className="flex flex-col gap-3"
          >
            <Field>
              <FieldLabel htmlFor="name">{t.holdingName}</FieldLabel>
              <Input
                id="name"
                value={draft.name}
                onChange={(e) => set("name", e.target.value)}
                placeholder={t.holdingName}
                required
                className="h-10"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="type">{t.holdingType}</FieldLabel>
              <Select
                value={draft.type}
                onValueChange={(v) => set("type", v as HoldingType)}
              >
                <SelectTrigger id="type" className="h-10 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {(Object.keys(typeLabels) as HoldingType[]).map((value) => (
                    <SelectItem key={value} value={value}>
                      {typeLabels[value]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            {/* Create-only (Draft's own comment) — editing an existing
                Holding's name/type never shows these; further lots go
                through the pricing dialog's own trio instead. */}
            {editing === null && (
              <>
                <Field>
                  <FieldLabel htmlFor="initial-quantity">
                    {t.manualLotQuantity}
                  </FieldLabel>
                  <Input
                    id="initial-quantity"
                    type="text"
                    inputMode="decimal"
                    value={draft.initialQuantity}
                    onChange={(e) => set("initialQuantity", e.target.value)}
                    placeholder="0"
                    className="h-10"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="initial-average-price">
                    {t.averagePurchasePrice}
                  </FieldLabel>
                  <InputGroup className="h-10">
                    <InputGroupAddon className="text-sm">€</InputGroupAddon>
                    <InputGroupInput
                      id="initial-average-price"
                      type="text"
                      inputMode="decimal"
                      value={draft.initialAveragePrice}
                      onChange={(e) => set("initialAveragePrice", e.target.value)}
                      placeholder="0,00"
                    />
                  </InputGroup>
                  <FieldDescription>{t.initialManualLotHint}</FieldDescription>
                </Field>
              </>
            )}
          </form>
        </FormSidebar>
      </div>

      <Dialog open={viewing !== null} onOpenChange={(open) => !open && setViewing(null)}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
          {viewing && (
            <>
              <DialogHeader>
                <DialogTitle>{t.holdingDetails}</DialogTitle>
              </DialogHeader>
              <dl className="flex flex-col divide-y divide-border text-sm">
                <ViewRow label={t.holdingType} value={typeLabels[viewing.type]} />
                <ViewRow
                  label={t.quantityOwned}
                  value={String(viewing.quantity_owned)}
                />
                <ViewRow label={t.paid} value={`€ ${formatCents(viewing.paid_cents)}`} />
                {viewing.average_price_per_unit_cents !== null && (
                  <ViewRow
                    label={t.averagePurchasePrice}
                    value={`€ ${formatPricePerUnit(viewing.average_price_per_unit_cents)}`}
                  />
                )}
                {viewing.current_price_cents !== null && (
                  <ViewRow
                    label={t.currentPrice}
                    value={`€ ${formatCents(viewing.current_price_cents)}`}
                  />
                )}
                {viewing.value_now_cents !== null && (
                  <ViewRow
                    label={t.valueNow}
                    value={`€ ${formatCents(viewing.value_now_cents)}`}
                  />
                )}
                {viewing.gain_loss_cents !== null && (
                  <ViewRow
                    label={t.gainLoss}
                    value={`€ ${formatCents(viewing.gain_loss_cents)}${
                      viewing.gain_loss_percent === null
                        ? ""
                        : ` (${viewing.gain_loss_percent.toFixed(1)}%)`
                    }`}
                  />
                )}
              </dl>
            </>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={pricing !== null} onOpenChange={(open) => !open && setPricing(null)}>
        <DialogContent>
          {pricing && (
            <form onSubmit={submitPricing} className="flex flex-col gap-4">
              <DialogHeader>
                <DialogTitle>{t.editPriceAndQuantity}</DialogTitle>
              </DialogHeader>

              <Field>
                <FieldLabel htmlFor="current-price">{t.currentPrice}</FieldLabel>
                <InputGroup className="h-10">
                  <InputGroupAddon className="text-sm">€</InputGroupAddon>
                  <InputGroupInput
                    id="current-price"
                    type="text"
                    inputMode="decimal"
                    value={pricingDraft.currentPrice}
                    onChange={(e) =>
                      setPricingDraft((d) => ({ ...d, currentPrice: e.target.value }))
                    }
                    placeholder="0,00"
                  />
                </InputGroup>
                <FieldDescription>{t.currentPriceHint}</FieldDescription>
              </Field>

              <Field>
                <FieldLabel htmlFor="lot-quantity">{t.manualLotQuantity}</FieldLabel>
                <Input
                  id="lot-quantity"
                  type="text"
                  inputMode="decimal"
                  value={pricingDraft.lotQuantity}
                  onChange={(e) =>
                    setPricingDraft((d) => ({ ...d, lotQuantity: e.target.value }))
                  }
                  placeholder={t.currentlyOwned(quantityTyped(pricing.quantity_owned))}
                  className="h-10"
                />
                <FieldDescription>{t.manualLotHint}</FieldDescription>
              </Field>

              <Field>
                <FieldLabel htmlFor="lot-average-price">
                  {t.averagePurchasePrice}
                </FieldLabel>
                <InputGroup className="h-10">
                  <InputGroupAddon className="text-sm">€</InputGroupAddon>
                  <InputGroupInput
                    id="lot-average-price"
                    type="text"
                    inputMode="decimal"
                    value={pricingDraft.lotAveragePrice}
                    onChange={(e) =>
                      setPricingDraft((d) => ({ ...d, lotAveragePrice: e.target.value }))
                    }
                    placeholder="0,00"
                  />
                </InputGroup>
              </Field>

              <FieldLabel htmlFor="replace-lot" className="font-normal">
                <Checkbox
                  id="replace-lot"
                  checked={pricingDraft.replace}
                  onCheckedChange={(v) =>
                    setPricingDraft((d) => ({ ...d, replace: v === true }))
                  }
                />
                {t.replaceManualLot}
              </FieldLabel>
              <FieldDescription>{t.replaceManualLotHint}</FieldDescription>

              <DialogFooter>
                <Button type="submit">{t.save}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
