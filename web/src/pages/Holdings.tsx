import { useCallback, useEffect, useState } from "react"
import { RiEditLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
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

// The optional current price, in cents, travels through the form as a typed
// euro string like the amount fields elsewhere ("" is unset, never "0").
type Draft = { name: string; type: HoldingType; currentPrice: string }

const blankDraft = (): Draft => ({
  name: "",
  type: "etf",
  currentPrice: "",
})

// The Holding management screen: add and rename, the fixed list ticket 03's
// Buy/Sell form and portfolio breakdown will read from. No hide or delete in
// this version — a fully-sold Holding simply nets to zero (spec's Out of
// Scope) — so unlike Categories and Clients this list is add-and-edit only.
export function Holdings() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null)
  const [error, setError] = useState("")
  const [draft, setDraft] = useState(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)

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
    setDraft({
      name: h.name,
      type: h.type,
      currentPrice:
        h.current_price_cents === null ? "" : toTyped(h.current_price_cents),
    })
    setSidebarOpen(true)
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    // Name, type and price travel together in one PATCH — the API has no
    // endpoint that changes just one of them, so there is nothing to leave
    // out here.
    const ok = await write(
      editing === null ? "/api/holdings" : `/api/holdings/${editing}`,
      {
        method: editing === null ? "POST" : "PATCH",
        body: JSON.stringify({
          name: draft.name,
          type: draft.type,
          current_price_cents: toCents(draft.currentPrice),
        }),
      }
    )
    if (!ok) return
    const wasEditing = editing !== null
    if (!wasEditing) toast(t.added)
    setEditing(null)
    // Adding several similar Holdings in a row is common (a handful of ETFs
    // set up at once) — the type stays, only the name clears. Finishing an
    // edit is a one-off, so it resets fully.
    setDraft((d) => (wasEditing ? blankDraft() : { ...d, name: "" }))
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
                <TableHead className="text-right">{t.valueNow}</TableHead>
                <TableHead className="text-right">{t.gainLoss}</TableHead>
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
                  <TableCell className="text-right tabular-nums">
                    {h.value_now_cents === null
                      ? "—"
                      : `€ ${formatCents(h.value_now_cents)}`}
                  </TableCell>
                  <TableCell
                    className={`text-right tabular-nums ${
                      (h.gain_loss_cents ?? 0) < 0 ? "text-destructive" : ""
                    }`}
                  >
                    {h.gain_loss_cents === null
                      ? "—"
                      : `€ ${formatCents(h.gain_loss_cents)}${
                          h.gain_loss_percent === null
                            ? ""
                            : ` (${h.gain_loss_percent.toFixed(1)}%)`
                        }`}
                  </TableCell>
                  <TableCell>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label={t.editHolding}
                      onClick={() => selectHolding(h)}
                    >
                      <RiEditLine />
                    </Button>
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

            <Field>
              <FieldLabel htmlFor="current-price">{t.currentPrice}</FieldLabel>
              <InputGroup className="h-10">
                <InputGroupAddon className="text-sm">€</InputGroupAddon>
                <InputGroupInput
                  id="current-price"
                  type="text"
                  inputMode="decimal"
                  value={draft.currentPrice}
                  onChange={(e) => set("currentPrice", e.target.value)}
                  placeholder="0,00"
                />
              </InputGroup>
              <FieldDescription>{t.currentPriceHint}</FieldDescription>
            </Field>

          </form>
        </FormSidebar>
      </div>
    </div>
  )
}
