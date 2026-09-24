import { useCallback, useEffect, useState } from "react"
import {
  RiArrowDownLine,
  RiArrowUpLine,
  RiDeleteBinLine,
  RiEditLine,
  RiMoreLine,
  RiShoppingBagLine,
} from "@remixicon/react"

import { ColorDot } from "@/components/ColorDot"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { FormSidebar } from "@/components/form-sidebar"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api, apiJSON } from "@/lib/api"
import { formatCents, toCents, toTyped } from "@/lib/money"
import { pickableCategories } from "@/lib/pickers"
import { t } from "@/lib/strings"
import { toast } from "@/lib/toast"
import type {
  Category,
  HeadroomReport,
  Placement,
  PlannedPurchase,
} from "@/types"

type Draft = {
  label: string
  amount: string
  category_id: number | null
}

const blankDraft = (): Draft => ({ label: "", amount: "", category_id: null })

// "2026-04" → "apr 2026", the same short names the year's rows use.
function monthLabel(month: string) {
  const [y, m] = month.split("-").map(Number)
  return `${t.monthsShort[m - 1]} ${y}`
}

const signedCents = (cents: number) =>
  `${cents < 0 ? "−" : ""}€ ${formatCents(Math.abs(cents))}`

// The Acquisti programmati page: the household's priority-ordered list of
// Planned purchases (CONTEXT.md). Not a DataTable: its sorting and filters
// would fight the one order that matters here, the household's own, which
// only the up/down controls change.
export function PlannedPurchases({
  onBought,
}: {
  // "Comprato": hands the Planned purchase to the Expense form, which
  // deletes it once the Expense is saved.
  onBought: (p: PlannedPurchase) => void
}) {
  const [planned, setPlanned] = useState<PlannedPurchase[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [headroom, setHeadroom] = useState<HeadroomReport | null>(null)
  const [draft, setDraft] = useState(blankDraft)
  const [editing, setEditing] = useState<number | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [error, setError] = useState("")

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }))

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<PlannedPurchase[]>("/api/planned-purchases"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<HeadroomReport>("/api/reports/headroom"),
      ])
        .then(([p, c, h]) => {
          setPlanned(p)
          setCategories(c)
          setHeadroom(h)
        })
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
      toast(t.plannedNotSaved)
      return false
    }
  }

  function select(p: PlannedPurchase) {
    setEditing(p.id)
    setDraft({
      label: p.label,
      amount: toTyped(p.amount_cents),
      category_id: p.category_id,
    })
    setError("")
    setSidebarOpen(true)
  }

  function reset() {
    setEditing(null)
    setDraft(blankDraft())
    setError("")
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const cents = toCents(draft.amount)
    if (cents === null || cents <= 0) {
      setError(t.invalidAmount)
      return
    }
    const ok = await write(
      editing === null
        ? "/api/planned-purchases"
        : `/api/planned-purchases/${editing}`,
      {
        method: editing === null ? "POST" : "PUT",
        body: JSON.stringify({
          label: draft.label,
          amount_cents: cents,
          category_id: draft.category_id,
        }),
      }
    )
    if (!ok) return
    if (editing === null) toast(t.added)
    reset()
  }

  async function remove(p: PlannedPurchase) {
    if (!confirm(t.confirmDeletePlanned(p.label))) return
    if (await write(`/api/planned-purchases/${p.id}`, { method: "DELETE" })) {
      if (editing === p.id) reset()
    }
  }

  const move = (p: PlannedPurchase, direction: "up" | "down") =>
    write(`/api/planned-purchases/${p.id}/move`, {
      method: "POST",
      body: JSON.stringify({ direction }),
    })

  const pickable = pickableCategories(
    categories,
    "expense",
    categories.find((c) => c.id === draft.category_id)
  )
  const list = planned ?? []
  const placements = new Map(
    (headroom?.placements ?? []).map((pl) => [pl.planned_id, pl])
  )
  const months = headroom?.months ?? []
  const lastRunning = months.at(-1)?.running_cents ?? 0
  const nextMonth = months[0]

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
      <h1 className="font-medium">{t.planned}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          {/* Two figures, the way the Dashboard's totals read: next month's
              forecast Headroom, and the Headroom accumulated to the end of
              the horizon. Which month each Planned purchase lands in is on
              the list itself. */}
          {headroom && !headroom.available && (
            <p className="text-sm text-muted-foreground">
              {t.headroomUnavailable}
            </p>
          )}
          {headroom?.available && nextMonth && (
            <div className="flex flex-col gap-2">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">
                      {t.headroomForecastMonthly}
                    </CardTitle>
                  </CardHeader>
                  <CardContent elevated className="flex flex-col gap-1">
                    <p
                      className={`text-2xl font-medium tabular-nums ${nextMonth.headroom_cents < 0 ? "text-destructive" : ""}`}
                    >
                      {signedCents(nextMonth.headroom_cents)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {monthLabel(nextMonth.month)}
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">
                      {t.headroomAccumulated(months.length)}
                    </CardTitle>
                  </CardHeader>
                  <CardContent elevated className="flex flex-col gap-1">
                    <p
                      className={`text-2xl font-medium tabular-nums ${lastRunning < 0 ? "text-destructive" : ""}`}
                    >
                      {signedCents(lastRunning)}
                    </p>
                    {/* The accumulated figure starts from last month's real
                        leftover, which is why it can be well above the sum
                        of the monthly forecasts: this names it. */}
                    <p className="text-xs text-muted-foreground">
                      {t.headroomIncludesStart(
                        monthLabel(headroom.start_month)
                      )}{" "}
                      <span className="font-medium text-foreground tabular-nums">
                        {signedCents(headroom.start_cents)}
                      </span>
                    </p>
                  </CardContent>
                </Card>
              </div>
              <p className="text-xs text-muted-foreground">
                {t.headroomProjection}
              </p>
            </div>
          )}

          {planned?.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t.noPlannedYet}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="hidden w-10 sm:table-cell">
                    {t.priority}
                  </TableHead>
                  <TableHead>{t.plannedLabel}</TableHead>
                  <TableHead className="text-right">{t.amount}</TableHead>
                  {headroom?.available && (
                    <TableHead className="hidden sm:table-cell">
                      {t.plannedWhen}
                    </TableHead>
                  )}
                  <TableHead className="w-28 text-right">{t.actions}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {list.map((p, i) => {
                  const category = categories.find(
                    (c) => c.id === p.category_id
                  )
                  return (
                    <TableRow key={p.id}>
                      <TableCell className="hidden text-muted-foreground tabular-nums sm:table-cell">
                        {i + 1}
                      </TableCell>
                      <TableCell>
                        <span className="flex items-center gap-2">
                          {category && <ColorDot color={category.color} />}
                          {p.label}
                        </span>
                        {/* On phones the "Quando" column is folded in here. */}
                        {headroom?.available && (
                          <span className="text-xs whitespace-normal text-muted-foreground sm:hidden">
                            <PlacementText
                              placement={placements.get(p.id)}
                              horizon={months.length}
                            />
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right font-medium tabular-nums">
                        € {formatCents(p.amount_cents)}
                      </TableCell>
                      {headroom?.available && (
                        <TableCell className="hidden sm:table-cell">
                          <PlacementText
                            placement={placements.get(p.id)}
                            horizon={months.length}
                          />
                        </TableCell>
                      )}
                      <TableCell>
                        <div className="flex items-center justify-end">
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            aria-label={t.moveUp}
                            title={t.moveUp}
                            disabled={i === 0}
                            onClick={() => void move(p, "up")}
                          >
                            <RiArrowUpLine />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            aria-label={t.moveDown}
                            title={t.moveDown}
                            disabled={i === list.length - 1}
                            onClick={() => void move(p, "down")}
                          >
                            <RiArrowDownLine />
                          </Button>
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
                              <DropdownMenuItem onClick={() => onBought(p)}>
                                <RiShoppingBagLine /> {t.bought}
                              </DropdownMenuItem>
                              <DropdownMenuItem onClick={() => select(p)}>
                                <RiEditLine /> {t.editPlanned}
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                variant="destructive"
                                onClick={() => void remove(p)}
                              >
                                <RiDeleteBinLine /> {t.delete}
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </div>

        <FormSidebar
          title={editing === null ? t.addPlanned : t.editPlanned}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          openMobileWhen={editing}
          footer={
            <div className="flex flex-col gap-2">
              <Button
                type="submit"
                form="planned-form"
                size="lg"
                className="h-12 text-base"
              >
                {editing === null ? t.addPlanned : t.save}
              </Button>
              {editing !== null && (
                <Button type="button" variant="ghost" onClick={reset}>
                  {t.cancel}
                </Button>
              )}
            </div>
          }
        >
          <form
            id="planned-form"
            onSubmit={submit}
            className="flex flex-col gap-3"
          >
            <Field>
              <FieldLabel htmlFor="planned-label">{t.plannedLabel}</FieldLabel>
              <Input
                id="planned-label"
                value={draft.label}
                onChange={(e) => set("label", e.target.value)}
                required
                className="h-10"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="planned-amount">{t.amount}</FieldLabel>
              <InputGroup className="h-12">
                <InputGroupAddon className="text-xl">€</InputGroupAddon>
                <InputGroupInput
                  id="planned-amount"
                  type="text"
                  inputMode="decimal"
                  value={draft.amount}
                  onChange={(e) => set("amount", e.target.value)}
                  placeholder="0,00"
                  required
                  className="text-2xl"
                />
              </InputGroup>
            </Field>

            <Field>
              <FieldLabel htmlFor="planned-category">{t.category}</FieldLabel>
              <Select
                value={
                  draft.category_id === null
                    ? "none"
                    : String(draft.category_id)
                }
                onValueChange={(v) =>
                  set("category_id", v === "none" ? null : Number(v))
                }
              >
                <SelectTrigger id="planned-category" className="h-10 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t.chooseSubcategory}</SelectItem>
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

            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
          </form>
        </FormSidebar>
      </div>
    </div>
  )
}

// Where one Planned purchase lands, or, when it fits in none of the horizon's
// months, how far short it falls and whether Savings cover the gap.
function PlacementText({
  placement,
  horizon,
}: {
  placement: Placement | undefined
  horizon: number
}) {
  if (!placement) return null
  if (placement.month) return <>{monthLabel(placement.month)}</>
  return (
    <span className="flex flex-col text-xs">
      <span className="text-destructive">{t.doesNotFit(horizon)}</span>
      <span className="text-muted-foreground">
        {t.missingAmount(formatCents(placement.missing_cents))} ·{" "}
        {placement.savings_cover ? t.savingsCoverGap : t.loanNeeded}
      </span>
    </span>
  )
}
