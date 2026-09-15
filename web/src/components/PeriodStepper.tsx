import type { ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

// The ‹ [content] › shell every page-level period selector shares. Before
// this, Month.tsx, Year.tsx, YearlyReport.tsx, Expenses.tsx and Incomes.tsx
// each built their own version — same idea, different button sizing and
// spacing every time. The content in between stays the caller's own:
// Month's native <input type="month"> is a genuinely different control (a
// jump-to-any-month picker the others don't have), so this only unifies the
// shell around it, not the input itself.
export function PeriodStepper({
  onPrevious,
  onNext,
  previousLabel,
  nextLabel,
  nextDisabled,
  children,
  className,
}: {
  onPrevious: () => void
  onNext: () => void
  previousLabel: string
  nextLabel: string
  nextDisabled?: boolean
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        aria-label={previousLabel}
        onClick={onPrevious}
      >
        ‹
      </Button>
      {children}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        aria-label={nextLabel}
        disabled={nextDisabled}
        onClick={onNext}
      >
        ›
      </Button>
    </div>
  )
}

// The plain-text period label every PeriodStepper's content is, except
// Month's own <input>: one size/weight so "2026" (Year), "Report annuale ·
// 2026" (YearlyReport) and "2026" / "Recenti" (Expenses, Incomes) all read as
// the same kind of label instead of three different ones. `as` picks the
// tag — Year and YearlyReport use this as their page's own <h1>;
// Expenses/Incomes/Month never had one, and stay plain text rather than
// gaining one as a side effect of sharing this component.
export function PeriodLabel({
  as: Tag = "span",
  children,
}: {
  as?: "span" | "h1"
  children: ReactNode
}) {
  return (
    <Tag className="min-w-16 text-center text-lg font-semibold tabular-nums">
      {children}
    </Tag>
  )
}
