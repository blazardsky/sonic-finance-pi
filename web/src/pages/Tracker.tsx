import { Fragment, useEffect, useMemo, useRef, useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"

import { ColorDot } from "@/components/ColorDot"
import { Button } from "@/components/ui/button"
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { apiJSON } from "@/lib/api"
import { formatDate, formatPricePerUnit } from "@/lib/money"
import { colorSlots, paletteVar } from "@/lib/palette"
import { colorOf, nameOf } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type {
  Category,
  Subcategory,
  TrackerItem,
  TrackerPoint,
} from "@/types"

type Mode = "product" | "tag" | "category"

// A key for one (name, category) group — the same pair GET /api/tracker
// groups by. Not the name alone: a rare same-named Item under two Categories
// would otherwise collide in the selector below.
function itemKey(item: TrackerItem): string {
  return JSON.stringify([item.name, item.category_id])
}

// The unit a group is priced in: its latest purchase's, "" when that one had
// no quantity to divide by.
const unitOf = (item: TrackerItem) => item.points.at(-1)?.unit ?? ""

const perUnit = (unit: string) => `€/${t.trackerUnitShort(unit)}`

// A chart label: the (lowercased) name capitalized by hand, since CSS
// capitalize would turn "€/kg" into "€/Kg" too.
const withUnit = (name: string, unit: string) => {
  const label = name.charAt(0).toUpperCase() + name.slice(1)
  return unit ? `${label} (${perUnit(unit)})` : label
}

const formatPrice = (p: TrackerPoint) =>
  p.unit
    ? t.pricePerUnit(formatPricePerUnit(p.price), t.trackerUnitShort(p.unit))
    : `€ ${formatPricePerUnit(p.price)}`

// Axis ticks: month and two-digit year, short enough for a phone.
const shortDate = (ms: number) =>
  new Date(ms).toLocaleDateString("it-IT", {
    month: "short",
    year: "2-digit",
    timeZone: "UTC",
  })

// One chart line: a product, with the purchases this view shows of it.
type Series = { label: string; points: TrackerPoint[] }

// One chart row per date, holding whichever series were bought that day —
// so the tooltip lists every product from the same shop together.
// shortcut: two purchases of the same product on one day keep only the
// later; split rows by purchase if that ever matters.
type Row = { t: number; date: string; points: Record<string, TrackerPoint> }

// How the same Item's price has moved over time (CONTEXT.md's Tracker entry),
// entirely read from GET /api/tracker. One chart, switched between a single
// product, every product under a Tag, or every product under a Category —
// one line per product, so they can be compared — and the products listed
// underneath, grouped by Category.
export function Tracker() {
  const [items, setItems] = useState<TrackerItem[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [tags, setTags] = useState<Subcategory[]>([])
  const [error, setError] = useState("")
  const [mode, setMode] = useState<Mode>("product")
  const [selected, setSelected] = useState<Partial<Record<Mode, string>>>({})
  const chartRef = useRef<HTMLElement>(null)

  useEffect(() => {
    let current = true
    Promise.all([
      apiJSON<TrackerItem[]>("/api/tracker"),
      apiJSON<Category[]>("/api/categories"),
      apiJSON<Subcategory[]>("/api/subcategories"),
    ])
      .then(([tracker, cats, subs]) => {
        if (!current) return
        setItems(tracker)
        setCategories(cats)
        setTags(subs)
      })
      .catch(() => {
        if (!current) return
        setError(t.serverUnreachable)
      })
    return () => {
      current = false
    }
  }, [])

  // Products grouped under their Category, both by name — the household reads
  // this page by "what did Alimentari cost", not by hunting for one name.
  const byCategory = useMemo(() => {
    const groups = new Map<number, TrackerItem[]>()
    for (const item of items ?? []) {
      groups.set(item.category_id, [
        ...(groups.get(item.category_id) ?? []),
        item,
      ])
    }
    return [...groups.entries()]
      .map(([id, list]) => ({
        id,
        name: nameOf(categories, id) || t.notSet,
        items: list.sort((a, b) => a.name.localeCompare(b.name)),
      }))
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [items, categories])

  // Only the Tags some purchase actually carries, hidden ones included:
  // hiding takes a Tag out of the pickers, not out of the history.
  const usedTags = useMemo(() => {
    const used = new Set(
      (items ?? []).flatMap((i) => i.points.map((p) => p.subcategory_id))
    )
    return tags
      .filter((s) => used.has(s.id))
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [items, tags])

  const options: Record<Mode, string[]> = {
    product: byCategory.flatMap((g) => g.items.map(itemKey)),
    tag: usedTags.map((s) => String(s.id)),
    category: byCategory.map((g) => String(g.id)),
  }
  const current = options[mode].includes(selected[mode] ?? "")
    ? selected[mode]!
    : options[mode][0]

  function show(m: Mode, value: string) {
    setMode(m)
    setSelected((s) => ({ ...s, [m]: value }))
    chartRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" })
  }

  const series: Series[] = useMemo(() => {
    const all = items ?? []
    if (current === undefined) return []
    if (mode === "product") {
      const item = all.find((i) => itemKey(i) === current)
      return item ? [{ label: withUnit(item.name, unitOf(item)), points: item.points }] : []
    }
    if (mode === "category") {
      return all
        .filter((i) => String(i.category_id) === current)
        .map((i) => ({ label: withUnit(i.name, unitOf(i)), points: i.points }))
    }
    // A Tag is per purchase, so only the purchases carrying it are drawn.
    return all
      .map((i) => ({
        label: withUnit(i.name, unitOf(i)),
        points: i.points.filter((p) => String(p.subcategory_id) === current),
      }))
      .filter((s) => s.points.length > 0)
  }, [items, mode, current])

  const chartConfig = Object.fromEntries(
    series.map((s, i) => [
      `s${i}`,
      {
        label: s.label,
        color: paletteVar(colorSlots[i % (colorSlots.length - 1)], "primary"),
      },
    ])
  ) satisfies ChartConfig

  const rows = useMemo(() => {
    const byDate = new Map<string, Row>()
    series.forEach((s, i) => {
      for (const p of s.points) {
        const row = byDate.get(p.occurred_on) ?? {
          t: Date.parse(p.occurred_on),
          date: p.occurred_on,
          points: {},
        }
        row.points[`s${i}`] = p
        byDate.set(p.occurred_on, row)
      }
    })
    return [...byDate.values()].sort((a, b) => a.t - b.t)
  }, [series])

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.tracker}</h1>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {items?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noTrackerYet}</p>
      )}

      {items && items.length > 0 && (
        <>
          <section ref={chartRef} className="flex scroll-mt-4 flex-col gap-3">
            <h2 className="text-sm font-medium text-muted-foreground">
              {t.trackerPriceTrend}
            </h2>
            <div className="flex flex-wrap items-center gap-2">
              <Tabs value={mode} onValueChange={(v) => setMode(v as Mode)}>
                <TabsList>
                  <TabsTrigger value="product">{t.trackerProduct}</TabsTrigger>
                  <TabsTrigger value="tag">{t.subcategory}</TabsTrigger>
                  <TabsTrigger value="category">{t.category}</TabsTrigger>
                </TabsList>
              </Tabs>
              {current !== undefined && (
                <Select
                  value={current}
                  onValueChange={(v) => setSelected((s) => ({ ...s, [mode]: v }))}
                >
                  <SelectTrigger
                    aria-label={t.trackerChooseItem}
                    className="h-8 min-w-0 max-w-full flex-1 sm:flex-none"
                  >
                    <SelectValue placeholder={t.trackerChooseItem} />
                  </SelectTrigger>
                  <SelectContent>
                    {mode === "product" &&
                      byCategory.map((g) => (
                        <SelectGroup key={g.id}>
                          <SelectLabel>{g.name}</SelectLabel>
                          {g.items.map((item) => (
                            <SelectItem key={itemKey(item)} value={itemKey(item)}>
                              <span className="capitalize">{item.name}</span>
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      ))}
                    {mode === "tag" &&
                      usedTags.map((s) => (
                        <SelectItem key={s.id} value={String(s.id)}>
                          <span className="flex items-center gap-2">
                            <ColorDot color={s.color} />
                            {s.name}
                          </span>
                        </SelectItem>
                      ))}
                    {mode === "category" &&
                      byCategory.map((g) => (
                        <SelectItem key={g.id} value={String(g.id)}>
                          <span className="flex items-center gap-2">
                            <ColorDot color={colorOf(categories, g.id)} />
                            {g.name}
                          </span>
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
              )}
            </div>

            {current === undefined ? (
              <p className="text-sm text-muted-foreground">{t.trackerNoTags}</p>
            ) : (
              <ChartContainer config={chartConfig} className="aspect-auto h-72 w-full">
                <LineChart data={rows} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                  <CartesianGrid vertical={false} />
                  <XAxis
                    dataKey="t"
                    type="number"
                    scale="time"
                    domain={["dataMin", "dataMax"]}
                    padding={{ left: 12, right: 12 }}
                    tickFormatter={shortDate}
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    minTickGap={24}
                  />
                  <YAxis
                    tickFormatter={(v: number) => formatPricePerUnit(v)}
                    tickLine={false}
                    axisLine={false}
                    width={44}
                  />
                  <ChartTooltip
                    cursor={false}
                    content={
                      <ChartTooltipContent
                        labelFormatter={(_, payload) =>
                          formatDate((payload?.[0]?.payload as Row | undefined)?.date ?? "")
                        }
                        formatter={(_, name, item) => {
                          const p = (item.payload as Row).points[String(name)]
                          if (!p) return null
                          return (
                            <div className="flex w-full flex-col gap-0.5">
                              <span className="flex items-center gap-1.5">
                                <span
                                  className="size-2.5 shrink-0 rounded-xs"
                                  style={{ backgroundColor: item.color }}
                                />
                                <span>{chartConfig[String(name)]?.label}</span>
                              </span>
                              <span className="pl-4 text-muted-foreground">
                                <span className="font-mono font-medium text-foreground tabular-nums">
                                  {formatPrice(p)}
                                </span>
                                {p.store && ` · ${p.store}`}
                                {p.discounted && ` · ${t.itemDiscounted}`}
                              </span>
                            </div>
                          )
                        }}
                      />
                    }
                  />
                  {series.length > 1 && (
                    <ChartLegend
                      content={<ChartLegendContent nameKey="value" />}
                    />
                  )}
                  {series.map((_, i) => (
                    <Line
                      key={`s${i}`}
                      name={`s${i}`}
                      dataKey={(row: Row) => row.points[`s${i}`]?.price}
                      type="linear"
                      stroke={`var(--color-s${i})`}
                      strokeWidth={2}
                      dot={{ r: 3, fill: `var(--color-s${i})` }}
                      connectNulls
                      isAnimationActive={false}
                    />
                  ))}
                </LineChart>
              </ChartContainer>
            )}
          </section>

          <section className="flex flex-col gap-2">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t.trackerProduct}</TableHead>
                  <TableHead className="text-right">{t.trackerLastPrice}</TableHead>
                  <TableHead className="text-right">{t.trackerMinPrice}</TableHead>
                  <TableHead className="text-right">{t.trackerMaxPrice}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {byCategory.map((g) => (
                  <Fragment key={g.id}>
                    <TableRow
                      className="hover:bg-transparent"
                      data-state={
                        mode === "category" && current === String(g.id)
                          ? "selected"
                          : undefined
                      }
                    >
                      <TableCell colSpan={4} className="pt-4">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="-ml-2 font-medium"
                          onClick={() => show("category", String(g.id))}
                        >
                          <ColorDot color={colorOf(categories, g.id)} />
                          {g.name}
                        </Button>
                      </TableCell>
                    </TableRow>
                    {g.items.map((item) => {
                      const last = item.points.at(-1)!
                      const byPrice = [...item.points].sort((a, b) => a.price - b.price)
                      const min = byPrice[0]
                      const max = byPrice.at(-1)!
                      const unit = unitOf(item)
                      return (
                        <TableRow
                          key={itemKey(item)}
                          data-state={
                            mode === "product" && current === itemKey(item)
                              ? "selected"
                              : undefined
                          }
                        >
                          <TableCell>
                            <Button
                              variant="link"
                              className="h-auto p-0 font-medium text-foreground capitalize"
                              onClick={() => show("product", itemKey(item))}
                            >
                              {item.name}
                            </Button>
                            {unit && (
                              <span className="ml-1 text-xs text-muted-foreground">
                                {perUnit(unit)}
                              </span>
                            )}
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            € {formatPricePerUnit(last.price)}
                            <div className="text-xs text-muted-foreground">
                              {formatDate(last.occurred_on)}
                            </div>
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            € {formatPricePerUnit(min.price)}
                            <div className="text-xs text-muted-foreground">
                              {min.store || t.notSet}
                            </div>
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            € {formatPricePerUnit(max.price)}
                            <div className="text-xs text-muted-foreground">
                              {max.store || t.notSet}
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </Fragment>
                ))}
              </TableBody>
            </Table>
          </section>
        </>
      )}
    </div>
  )
}
