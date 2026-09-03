import { useEffect, useState } from "react"

import { apiJSON } from "@/api"
import { formatCents } from "@/money"
import { t } from "@/strings"
import type { Pending } from "@/types"

// The notice the freelancer used to get from reading a spreadsheet carefully:
// what is owed, and what looks unbilled. It sits on the month screen rather
// than anywhere it has to be looked for — the whole point of ticket 14 is that
// a forgotten invoice surfaces on its own, and nothing here depends on the Pi
// being awake at a particular moment (story 42: no email, no push).
//
// It renders nothing at all when nothing is pending. A household owed nothing
// does not need a heading telling it so, and this sits above the numbers it
// would otherwise be pushing down.
export function PendingPayments() {
  const [pending, setPending] = useState<Pending | null>(null)

  // Not month-scoped: what is owed is owed whichever month the screen is
  // showing, so this is fetched once. A failure leaves it silent — losing the
  // notice is not a reason to blank the totals under it.
  useEffect(() => {
    apiJSON<Pending>("/api/pending-payments")
      .then(setPending)
      .catch(() => setPending(null))
  }, [])

  if (!pending) return null
  const { outstanding, not_yet_invoiced: notYetInvoiced } = pending
  if (outstanding.length === 0 && notYetInvoiced.length === 0) return null

  return (
    <div className="flex flex-col gap-6">
      {/* Money genuinely owed, oldest first. The days are the point: chasing
          is based on a number rather than a feeling. */}
      {outstanding.length > 0 && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.pendingPayments}
          </h2>
          <ul className="flex flex-col divide-y divide-border">
            {outstanding.map((income) => (
              <li
                key={income.id}
                className="flex items-baseline justify-between gap-3 py-2"
              >
                <span className="min-w-0">
                  {/* The Client is who to chase; a Client-less Income has only
                      its reason to say what it is. */}
                  <span className="truncate">
                    {income.client || income.category}
                  </span>{" "}
                  {/* The days, and not the date they are counted from: the
                      ticket asks for a number to chase on, and the Income
                      screen is where the invoice date is read. */}
                  <span className="text-xs text-muted-foreground">
                    {t.waitingDays(income.days_waiting)}
                  </span>
                </span>
                {/* No "+" and no full-strength text: this money has not
                    arrived and is in none of the totals above (ADR-0003). */}
                <span className="whitespace-nowrap text-muted-foreground tabular-nums">
                  € {formatCents(income.amount_cents)}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* The nudge. Not a claim that an invoice is missing — only that one
          usually goes out by now and has not, which is why the hint says what
          the list is made of rather than what to do about it. */}
      {notYetInvoiced.length > 0 && (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t.notYetInvoiced}
          </h2>
          <p className="text-xs text-muted-foreground">
            {t.notYetInvoicedHint}
          </p>
          <ul className="flex flex-col divide-y divide-border">
            {notYetInvoiced.map((client) => (
              <li key={client.client_id} className="truncate py-2">
                {client.client}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}
