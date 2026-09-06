import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { apiJSON } from "@/lib/api"
import { formatCents, toCents, toTyped } from "@/lib/money"
import { t } from "@/lib/strings"
import type { HoldingType, Lists, SavingsReport } from "@/types"

// Same labels Holdings.tsx uses for the fixed type list — each screen keeps
// its own copy rather than sharing one, the existing convention here.
const typeLabels: Record<HoldingType, string> = {
  etf: t.holdingTypeETF,
  crypto: t.holdingTypeCrypto,
  stock: t.holdingTypeStock,
  bond: t.holdingTypeBond,
  other: t.holdingTypeOther,
}

// The Risparmi page: a read-only recap, no entries added here except the
// starting balance — the total Savings figure and the portfolio breakdown
// (ticket 07), both off the one /api/reports/savings request. Buying and
// selling a Holding happens on the Investimenti page instead.
export function Savings() {
  const [error, setError] = useState("")

  const [savings, setSavings] = useState<SavingsReport | null>(null)
  const [startingBalance, setStartingBalance] = useState("")
  const [balanceError, setBalanceError] = useState("")
  const [balanceMessage, setBalanceMessage] = useState("")

  const load = useCallback(
    () =>
      apiJSON<SavingsReport>("/api/reports/savings")
        .then((s) => {
          setSavings(s)
          setStartingBalance(toTyped(s.starting_balance_cents))
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // The starting balance rides the same /api/settings payload as Target and
  // Goal (cmd/setting.go) — sending just this one field leaves payers,
  // payment methods, Target and Goal exactly as they were read.
  async function saveStartingBalance(event: React.FormEvent) {
    event.preventDefault()
    setBalanceError("")
    setBalanceMessage("")
    const cents = toCents(startingBalance)
    if (cents === null) {
      setBalanceError(t.invalidAmount)
      return
    }
    try {
      const saved = await apiJSON<Lists>("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ savings_starting_balance_cents: cents }),
      })
      setStartingBalance(toTyped(saved.savings_starting_balance_cents))
      setBalanceMessage(t.listsSaved)
      await load()
    } catch {
      setBalanceError(t.startingBalanceNotSaved)
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.savings}</h1>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      <div className="flex flex-wrap gap-4">
        <Card className="min-w-56">
          <CardHeader>
            <CardTitle>{t.savings}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-medium tabular-nums">
              € {formatCents(savings?.savings_cents ?? 0)}
            </p>
          </CardContent>
        </Card>

        <Card className="min-w-72 flex-1">
          <CardHeader>
            <CardTitle>{t.startingBalance}</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              onSubmit={saveStartingBalance}
              className="flex flex-col gap-1.5"
            >
              <div className="flex gap-2">
                <InputGroup className="h-9">
                  <InputGroupAddon>€</InputGroupAddon>
                  <InputGroupInput
                    inputMode="decimal"
                    value={startingBalance}
                    onChange={(e) => setStartingBalance(e.target.value)}
                  />
                </InputGroup>
                <Button type="submit" variant="outline">
                  {t.save}
                </Button>
              </div>
              <span className="text-xs text-muted-foreground">
                {t.startingBalanceHint}
              </span>
              {balanceError && (
                <p role="alert" className="text-sm text-destructive">
                  {balanceError}
                </p>
              )}
              {balanceMessage && (
                <p role="status" className="text-sm text-muted-foreground">
                  {balanceMessage}
                </p>
              )}
            </form>
          </CardContent>
        </Card>
      </div>

      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-medium">{t.portfolio}</h2>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t.holding}</TableHead>
              <TableHead>{t.holdingType}</TableHead>
              <TableHead className="text-right">{t.amount}</TableHead>
              <TableHead className="text-right">%</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(savings?.holdings ?? []).map((h) => (
              <TableRow key={h.holding_id}>
                <TableCell>{h.name}</TableCell>
                <TableCell>{typeLabels[h.type]}</TableCell>
                <TableCell className="text-right tabular-nums">
                  € {formatCents(h.net_cents)}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {h.percent.toFixed(1)}%
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {(savings?.holdings ?? []).length === 0 && (
          <p className="text-sm text-muted-foreground">
            {t.noHoldingsInPortfolio}
          </p>
        )}
      </div>
    </div>
  )
}
