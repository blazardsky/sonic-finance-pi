import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api"
import { t } from "@/lib/strings"
import type { Holding, HoldingType } from "@/types"

const typeLabels: Record<HoldingType, string> = {
  etf: t.holdingTypeETF,
  crypto: t.holdingTypeCrypto,
  stock: t.holdingTypeStock,
  bond: t.holdingTypeBond,
  other: t.holdingTypeOther,
}

// The Holding management screen: add and rename, the fixed list ticket 03's
// Buy/Sell form and portfolio breakdown will read from. No hide or delete in
// this version — a fully-sold Holding simply nets to zero (spec's Out of
// Scope) — so unlike Categories and Clients this list is add-and-edit only.
export function Holdings() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null)
  const [error, setError] = useState("")
  const [name, setName] = useState("")
  const [type, setType] = useState<HoldingType>("etf")
  const [editing, setEditing] = useState<number | null>(null)

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

  async function add(event: React.FormEvent) {
    event.preventDefault()
    const created = await write("/api/holdings", {
      method: "POST",
      body: JSON.stringify({ name, type }),
    })
    if (created) setName("")
  }

  async function rename(holding: Holding, to: string) {
    setEditing(null)
    if (to.trim() === "" || to === holding.name) return
    await write(`/api/holdings/${holding.id}`, {
      method: "PATCH",
      body: JSON.stringify({ type: holding.type, name: to }),
    })
  }

  async function changeType(holding: Holding, to: HoldingType) {
    if (to === holding.type) return
    await write(`/api/holdings/${holding.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: holding.name, type: to }),
    })
  }

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <h1 className="font-medium">{t.holdings}</h1>

      <form onSubmit={add} className="flex flex-col gap-2">
        <div className="flex gap-2">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t.holdingName}
            required
            className="h-9"
          />
          <NativeSelect
            value={type}
            onChange={(e) => setType(e.target.value as HoldingType)}
            aria-label={t.holdingType}
            className="h-9 w-auto"
          >
            {(Object.keys(typeLabels) as HoldingType[]).map((value) => (
              <option key={value} value={value}>
                {typeLabels[value]}
              </option>
            ))}
          </NativeSelect>
        </div>
        <Button type="submit" size="lg">
          {t.addHolding}
        </Button>
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {holdings?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noHoldingsYet}</p>
      )}

      <ul className="flex flex-col divide-y divide-border">
        {holdings?.map((h) => (
          <li key={h.id} className="flex flex-col gap-1.5 py-2.5">
            {editing === h.id ? (
              <Input
                autoFocus
                defaultValue={h.name}
                className="h-9"
                onBlur={(e) => void rename(h, e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") e.currentTarget.blur()
                  if (e.key === "Escape") setEditing(null)
                }}
              />
            ) : (
              <div className="flex items-baseline justify-between gap-2">
                <span>{h.name}</span>
                <NativeSelect
                  value={h.type}
                  onChange={(e) =>
                    void changeType(h, e.target.value as HoldingType)
                  }
                  aria-label={t.holdingType}
                  className="h-7 w-auto text-xs"
                >
                  {(Object.keys(typeLabels) as HoldingType[]).map((value) => (
                    <option key={value} value={value}>
                      {typeLabels[value]}
                    </option>
                  ))}
                </NativeSelect>
              </div>
            )}
            <div className="flex gap-1">
              <Button size="xs" variant="ghost" onClick={() => setEditing(h.id)}>
                {t.rename}
              </Button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
