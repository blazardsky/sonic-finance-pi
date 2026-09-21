import { useEffect, useState } from "react"
import { RiCloseLine, RiDeleteBinLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { api, apiJSON } from "@/lib/api"
import { toast } from "@/lib/toast"
import { t } from "@/lib/strings"

// One card per FTS5 vocabulary (typeahead.go's store_fts/item_name_fts) —
// both append-only by design (ADR-0015), so this is the only place a stale
// or duplicate spelling ("CoopVoce" next to "Coopvoce") ever leaves the
// suggestion list. Deleting here never touches a past Expense/Item — Store
// stays free text either way (ADR-0013) — it only stops the name being
// offered again.
function VocabularyCard({ title, path }: { title: string; path: string }) {
  const [values, setValues] = useState<string[] | null>(null)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState("")

  const load = () =>
    apiJSON<string[]>(path)
      .then((v) => {
        setValues(v)
        setSelected(new Set())
      })
      .catch(() => toast(t.serverUnreachable))

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function remove(names: string[]) {
    try {
      await api(path, { method: "DELETE", body: JSON.stringify({ values: names }) })
    } catch {
      toast(t.suggestionsNotDeleted)
      return
    }
    toast(t.suggestionsDeleted)
    await load()
  }

  const filtered = (values ?? []).filter((v) =>
    v.toLowerCase().includes(filter.trim().toLowerCase())
  )
  const allSelected = filtered.length > 0 && filtered.every((v) => selected.has(v))

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <Input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder={t.suggestionsSearchPlaceholder}
          className="h-9"
        />

        {values?.length === 0 && (
          <p className="text-sm text-muted-foreground">{t.suggestionsNoneYet}</p>
        )}

        {filtered.length > 0 && (
          <>
            <FieldLabel className="flex w-fit items-center gap-2 text-xs font-normal text-muted-foreground">
              <Checkbox
                checked={allSelected}
                onCheckedChange={(v) =>
                  setSelected(v === true ? new Set(filtered) : new Set())
                }
              />
              {t.suggestionsSelectAll}
            </FieldLabel>

            <ul className="flex max-h-80 flex-col gap-1 overflow-y-auto">
              {filtered.map((value) => (
                <li
                  key={value}
                  className="flex items-center justify-between gap-2 rounded-md px-1 py-1 hover:bg-muted/50"
                >
                  <FieldLabel className="flex min-w-0 flex-1 items-center gap-2 font-normal">
                    <Checkbox
                      checked={selected.has(value)}
                      onCheckedChange={(v) =>
                        setSelected((s) => {
                          const next = new Set(s)
                          if (v === true) next.add(value)
                          else next.delete(value)
                          return next
                        })
                      }
                    />
                    <span className="truncate">{value}</span>
                  </FieldLabel>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t.delete}
                    onClick={() => void remove([value])}
                  >
                    <RiCloseLine />
                  </Button>
                </li>
              ))}
            </ul>

            <Button
              type="button"
              variant="destructive"
              size="sm"
              className="w-fit gap-1.5"
              disabled={selected.size === 0}
              onClick={() => void remove([...selected])}
            >
              <RiDeleteBinLine className="size-4" />
              {t.suggestionsDeleteSelected(selected.size)}
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  )
}

export function Suggestions() {
  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.suggestions}</h1>
      <p className="text-sm text-muted-foreground">{t.suggestionsHint}</p>
      <div className="flex flex-wrap items-start gap-6">
        <VocabularyCard title={t.suggestionsStores} path="/api/stores" />
        <VocabularyCard title={t.suggestionsItems} path="/api/items/names" />
      </div>
    </div>
  )
}
