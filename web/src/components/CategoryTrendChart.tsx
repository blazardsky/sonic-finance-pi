import { Bar, BarChart, XAxis } from "recharts"

import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { formatCents } from "@/lib/money"
import type { TrendBucket } from "@/lib/trend"

// The one stacked bar chart both trend charts are — a day's or a week's spend
// as a stack of Category segments — so the Dashboard's rolling chart and the
// Month page's weekly chart draw from the same query shape and are read the
// same way. dataKey is the Category id as a string, never the name: the
// ChartConfig it drives keys the same way, so a rename never desyncs a bar
// from its colour.
export function CategoryTrendChart({
  buckets,
  categories,
}: {
  buckets: TrendBucket[]
  categories: { id: number; name: string; color: string }[]
}) {
  const chartConfig = Object.fromEntries(
    categories.map((c) => [String(c.id), { label: c.name, color: c.color }])
  ) satisfies ChartConfig

  const data = buckets.map((b) => ({
    label: b.label,
    ...Object.fromEntries(
      categories.map((c) => [String(c.id), b.amounts[c.id] ?? 0])
    ),
  }))

  return (
    <ChartContainer config={chartConfig} className="aspect-auto h-64 w-full">
      <BarChart data={data} margin={{ top: 4, right: 8, left: 8, bottom: 0 }}>
        <XAxis
          dataKey="label"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          interval="preserveStartEnd"
        />
        <ChartTooltip
          cursor={false}
          content={
            <ChartTooltipContent
              formatter={(value, name, item) => (
                <>
                  <div
                    className="h-2.5 w-2.5 shrink-0 rounded-[2px]"
                    style={{ backgroundColor: item.color }}
                  />
                  <div className="flex flex-1 justify-between gap-2 leading-none">
                    <span className="text-muted-foreground">
                      {categories.find((c) => String(c.id) === name)?.name ??
                        name}
                    </span>
                    <span className="font-mono font-medium tabular-nums">
                      € {formatCents(Number(value))}
                    </span>
                  </div>
                </>
              )}
            />
          }
        />
        <ChartLegend content={<ChartLegendContent />} />
        {categories.map((c) => (
          <Bar
            key={c.id}
            dataKey={String(c.id)}
            stackId="spend"
            fill={`var(--color-${c.id})`}
            radius={0}
          />
        ))}
      </BarChart>
    </ChartContainer>
  )
}
