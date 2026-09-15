import { Fragment, useEffect, useMemo, useState } from "react"
import { Line, LineChart, XAxis } from "recharts"

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
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
import { apiJSON } from "@/lib/api"
import { formatCents, formatDate } from "@/lib/money"
import { t } from "@/lib/strings"
import type { Category, TrackerItem } from "@/types"

// A key for one (name, category) group — the same pair GET /api/tracker
// groups by. Not the name alone: a rare same-named Item under two Categories
// would otherwise collide in the selector below.
function itemKey(item: TrackerItem): string {
  return JSON.stringify([item.name, item.category_id])
}

const chartConfig = {
  average: { label: t.trackerYearlyAverage, color: "var(--primary)" },
} satisfies ChartConfig

// How much the same Item has cost over time and where it was cheaper
// (CONTEXT.md's Tracker entry, ticket 06) — entirely read from ticket 05's
// GET /api/tracker. A single chart with an item selector rather than one line
// chart per Item: the household can end up tracking dozens of grocery items,
// and a page that grows a whole chart per Item would turn into a wall of
// near-empty sparklines rather than something anyone actually reads.
export function Tracker() {
  const [items, setItems] = useState<TrackerItem[] | null>(null)
  const [categories, setCategories] = useState<Category[] | null>(null)
  const [error, setError] = useState("")
  const [selected, setSelected] = useState<string | null>(null)

  useEffect(() => {
    let current = true
    Promise.all([
      apiJSON<TrackerItem[]>("/api/tracker"),
      apiJSON<Category[]>("/api/categories"),
    ])
      .then(([tracker, cats]) => {
        if (!current) return
        setItems(tracker)
        setCategories(cats)
      })
      .catch(() => {
        if (!current) return
        setError(t.serverUnreachable)
      })
    return () => {
      current = false
    }
  }, [])

  const categoryName = useMemo(() => {
    const byId = new Map((categories ?? []).map((c) => [c.id, c.name]))
    return (id: number) => byId.get(id) ?? t.notSet
  }, [categories])

  // Grouped by Category first, product name second, rather than the flat
  // per-product order /api/tracker returns them in — a household tracking
  // dozens of grocery items reads this page by "what did Alimentari cost",
  // not by hunting for one name interleaved with every other Category's.
  const sortedItems = useMemo(() => {
    if (!items) return items
    return [...items].sort((a, b) => {
      const byCategory = categoryName(a.category_id).localeCompare(
        categoryName(b.category_id)
      )
      return byCategory !== 0 ? byCategory : a.name.localeCompare(b.name)
    })
  }, [items, categoryName])

  // Defaults to the first group once the data is in, and falls back the same
  // way if the previously selected one ever stopped existing.
  const selectedItem =
    items?.find((i) => itemKey(i) === selected) ?? items?.[0] ?? null
  if (items && items.length > 0 && selected === null) {
    setSelected(itemKey(items[0]))
  }

  const chartData = (selectedItem?.yearly_average ?? []).map((y) => ({
    year: String(y.year),
    average: Math.round(y.average_price),
  }))

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
          <section className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-sm font-medium text-muted-foreground">
                {t.trackerYearlyAverage}
              </h2>
              <Select value={selected ?? undefined} onValueChange={setSelected}>
                <SelectTrigger aria-label={t.trackerChooseItem} className="h-8">
                  <SelectValue placeholder={t.trackerChooseItem} />
                </SelectTrigger>
                <SelectContent>
                  {sortedItems?.map((item) => (
                    <SelectItem key={itemKey(item)} value={itemKey(item)}>
                      <span className="capitalize">{item.name}</span>
                      {" · "}
                      {categoryName(item.category_id)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <ChartContainer config={chartConfig} className="aspect-auto h-64 w-full">
              <LineChart data={chartData} margin={{ top: 4, right: 8, left: 8, bottom: 0 }}>
                <XAxis dataKey="year" tickLine={false} axisLine={false} tickMargin={8} />
                <ChartTooltip
                  cursor={false}
                  content={
                    <ChartTooltipContent
                      formatter={(value) => (
                        <span className="font-mono font-medium tabular-nums">
                          € {formatCents(Number(value))}
                        </span>
                      )}
                    />
                  }
                />
                <Line
                  dataKey="average"
                  type="monotone"
                  stroke="var(--color-average)"
                  strokeWidth={2}
                  dot
                />
              </LineChart>
            </ChartContainer>
          </section>

          <section className="flex flex-col gap-2">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t.itemName}</TableHead>
                  <TableHead>{t.category}</TableHead>
                  <TableHead>{t.store}</TableHead>
                  <TableHead className="text-right">{t.trackerLastPrice}</TableHead>
                  <TableHead>{t.date}</TableHead>
                  <TableHead className="text-right">{t.trackerMinPrice}</TableHead>
                  <TableHead className="text-right">{t.trackerMaxPrice}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedItems?.map((item) => (
                  <Fragment key={itemKey(item)}>
                    {item.stores.map((s, i) => (
                      <TableRow key={s.store}>
                        {i === 0 && (
                          <>
                            <TableCell
                              rowSpan={item.stores.length}
                              className="align-top font-medium capitalize"
                            >
                              {item.name}
                            </TableCell>
                            <TableCell
                              rowSpan={item.stores.length}
                              className="align-top text-xs text-muted-foreground"
                            >
                              {categoryName(item.category_id)}
                            </TableCell>
                          </>
                        )}
                        <TableCell className="capitalize">{s.store}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          € {formatCents(Math.round(s.last_price))}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatDate(s.last_occurred_on)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          € {formatCents(Math.round(s.min_price))}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          € {formatCents(Math.round(s.max_price))}
                        </TableCell>
                      </TableRow>
                    ))}
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
