// One label/value row in a "view details" dialog — plain text, not an input.
// Shared by Expenses.tsx and Incomes.tsx, whose view dialogs are both a
// fixed dl of whatever the simplified table itself no longer shows inline.
export function ViewRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-2">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="max-w-[70%] text-right break-words whitespace-normal">{value}</dd>
    </div>
  )
}
