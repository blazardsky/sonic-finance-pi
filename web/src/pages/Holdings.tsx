import { useCallback, useEffect, useState } from "react"
import { RiEditLine, RiEyeLine, RiMoneyEuroCircleLine, RiMoreLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api } from "@/lib/api"
import { FormSidebar } from "@/components/form-sidebar"
import { ViewRow } from "@/components/ViewRow"
import { toast } from "@/lib/toast"
import { formatCents, toCents, toTyped } from "@/lib/money"
import { t } from "@/lib/strings"
import type { Holding, HoldingType } from "@/types"

const typeLabels: Record<HoldingType, string> = {
  etf: t.holdingTypeETF,
  crypto: t.holdingTypeCrypto,
  stock: t.holdingTypeStock,
  bond: t.holdingTypeBond,
  other: t.holdingTypeOther,
}

type Draft = { name: string; type: HoldingType }

const blankDraft = (): Draft => ({ name: "", type: "etf" })

// The price/quantity-adjustment form travels as typed strings, the same
// bargain every euro/quantity input in this app makes: "" is unset, never
// "0". Unlike the quantity fields on Expense/Income, the adjustment is
// signed, so it does not go through toQuantity (which refuses anything not
// positive) — parseAdjustment below is its own, looser parser.
type PricingDraft = { currentPrice: string; quantityAdjustment: string }

function parseAdjustment(typed: string): number {
  const n = Number(typed.trim().replace(",", "."))
  return Number.isFinite(n) ? n : 0
}

// The Holding management screen: add and rename, the fixed list ticket 03's
// Buy/Sell form and portfolio breakdown read from. No hide or delete in this
// version — a fully-sold Holding simply nets to zero (spec's Out of Scope) —
// so unlike Categories and Clients this list is add-and-edit only.
//
// Value now and gain/loss sit behind a Dettagli view dialog rather than in
// the table itself (the same simplification Expenses/Incomes' own view
// dialogs made), and current_price_cents/quantity_adjustment — the two
// hand-typed corrections — are edited through their own small dialog, the
// same shape Categories.tsx uses for a Subcategory's colour: a focused edit,
// separate from the Name/Type sidebar form.
export function Holdings() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null)
  const [error, setError] = useState("")
  const [draft, setDraft] = useState(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [viewing, setViewing] = useState<Holding | null>(null)
  const [pricing, setPricing] = useState<Holding | null>(null)
  const [pricingDraft, setPricingDraft] = useState<PricingDraft>({
    currentPrice: "",
    quantityAdjustment: "",
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
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  async function write(path: string, init: RequestInit) {
    setError("")
    try {
      await api(path, init)
      await load()
      return true
    } catch {
      setError(t.holdingNotSaved)
      return false
    }
  }

  // Loads a Holding into the form and makes sure the panel holding it is
  // actually visible, same as every other edit action in this effort — the
  // form updates whether it is on screen or not.
  function selectHolding(h: Holding) {
    setEditing(h.id)
    setDraft({ name: h.name, type: h.type })
    setSidebarOpen(true)
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const ok = await write(
      editing === null ? "/api/holdings" : `/api/holdings/${editing}`,
      { method: editing === null ? "POST" : "PATCH", body: JSON.stringify(draft) }
    )
    if (!ok) return
    if (editing === null) toast(t.added)
    setEditing(null)
    // Adding several similar Holdings in a row is common (a handful of ETFs
    // set up at once) — the type stays, only the name clears.
    setDraft((d) => ({ ...d, name: "" }))
  }

  function openPricing(h: Holding) {
    setPricing(h)
    setPricingDraft({
      currentPrice:
        h.current_price_cents === null ? "" : toTyped(h.current_price_cents),
      quantityAdjustment:
        h.quantity_adjustment === 0 ? "" : String(h.quantity_adjustment).replace(".", ","),
    })
  }

  async function submitPricing(event: React.FormEvent) {
    event.preventDefault()
    if (!pricing) return
    const id = pricing.id
    setPricing(null)
    await write(`/api/holdings/${id}`, {
      method: "PATCH",
      body: JSON.stringify({
        current_price_cents: toCents(pricingDraft.currentPrice),
        quantity_adjustment: parseAdjustment(pricingDraft.quantityAdjustment),
      }),
    })
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
      <h1 className="font-medium">{t.holdings}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}

          {holdings?.length === 0 && (
            <p className="text-sm text-muted-foreground">{t.noHoldingsYet}</p>
          )}

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.holdingName}</TableHead>
                <TableHead>{t.holdingType}</TableHead>
                <TableHead className="text-right">{t.quantityOwned}</TableHead>
                <TableHead className="text-right">{t.paid}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {holdings?.map((h) => (
                <TableRow key={h.id}>
                  <TableCell>{h.name}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {typeLabels[h.type]}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {h.quantity_owned}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    € {formatCents(h.paid_cents)}
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
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
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
                <FieldLabel htmlFor="quantity-adjustment">
                  {t.quantityAdjustment}
                </FieldLabel>
                <Input
                  id="quantity-adjustment"
                  type="text"
                  inputMode="decimal"
                  value={pricingDraft.quantityAdjustment}
                  onChange={(e) =>
                    setPricingDraft((d) => ({ ...d, quantityAdjustment: e.target.value }))
                  }
                  placeholder="0"
                  className="h-10"
                />
                <FieldDescription>{t.quantityAdjustmentHint}</FieldDescription>
              </Field>

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
