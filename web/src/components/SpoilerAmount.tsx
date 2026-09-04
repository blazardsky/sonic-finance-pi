import { formatCents } from "@/lib/money"

// The Gift spoiler (spec's Gift section): a Gift Expense's amount reads
// blurred everywhere it appears in a list or grid, and reveals on hover
// (desktop) or tap (mobile) — CSS-only, so it blurs again the instant the
// pointer or focus moves on, with nothing to reset when the screen changes.
// gift=false renders the amount plain, so every call site can pass it
// unconditionally rather than branching itself.
export function SpoilerAmount({
  cents,
  gift,
}: {
  cents: number
  gift: boolean
}) {
  const amount = `€ ${formatCents(cents)}`
  if (!gift) return <>{amount}</>
  return (
    <span
      tabIndex={0}
      className="select-none rounded blur-sm transition-[filter] duration-150 hover:blur-none focus:blur-none"
    >
      {amount}
    </span>
  )
}
