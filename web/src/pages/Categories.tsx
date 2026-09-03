import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api"
import { t } from "@/lib/strings"
import type { Applies, Category } from "@/types"

const appliesLabels: Record<Applies, string> = {
  expense: t.appliesExpense,
  income: t.appliesIncome,
  both: t.appliesBoth,
}

// The Category management screen: add, rename, hide, delete. Hidden Categories
// stay on this list — it is the only place they can be brought back from — and
// are what every picker filters out.
export function Categories() {
  const [categories, setCategories] = useState<Category[] | null>(null)
  const [error, setError] = useState("")
  const [name, setName] = useState("")
  const [appliesTo, setAppliesTo] = useState<Applies>("expense")
  const [editing, setEditing] = useState<number | null>(null)

  // Returns its promise so a write can wait for the reload it triggers, and
  // sets state from a callback rather than an awaited line, which is what
  // keeps the effect below off React's cascading-render path.
  const load = useCallback(
    () =>
      api("/api/categories")
        .then((res) => res.json())
        .then(setCategories)
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // Every write goes through here so that the reload and the error wording
  // are in one place: a 409 is the only failure worth its own sentence, and it
  // now has two readings — the tax summary resolves this Category, or entries
  // are still pointing at it. Which one it is depends on what was attempted,
  // so the caller supplies the sentence.
  async function write(
    path: string,
    init: RequestInit,
    onConflict: string = t.categoryProtected
  ) {
    setError("")
    try {
      await api(path, init)
      await load()
      return true
    } catch (res) {
      setError(
        res instanceof Response && res.status === 409
          ? onConflict
          : t.serverUnreachable
      )
      return false
    }
  }

  async function add(event: React.FormEvent) {
    event.preventDefault()
    const created = await write("/api/categories", {
      method: "POST",
      body: JSON.stringify({ name, applies_to: appliesTo }),
    })
    if (created) setName("")
  }

  async function rename(category: Category, to: string) {
    setEditing(null)
    if (to.trim() === "" || to === category.name) return
    await write(`/api/categories/${category.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: to }),
    })
  }

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <h1 className="font-medium">{t.categories}</h1>

      <form onSubmit={add} className="flex flex-col gap-2">
        <div className="flex gap-2">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t.categoryName}
            required
            className="h-9"
          />
          <NativeSelect
            value={appliesTo}
            onChange={(e) => setAppliesTo(e.target.value as Applies)}
            aria-label={t.appliesTo}
            className="h-9 w-auto"
          >
            {(["expense", "income", "both"] as const).map((value) => (
              <option key={value} value={value}>
                {appliesLabels[value]}
              </option>
            ))}
          </NativeSelect>
        </div>
        <Button type="submit" size="lg">
          {t.addCategory}
        </Button>
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      <ul className="flex flex-col divide-y divide-border">
        {categories?.map((c) => (
          <li key={c.id} className="flex flex-col gap-1.5 py-2.5">
            {editing === c.id ? (
              <Input
                autoFocus
                defaultValue={c.name}
                className="h-9"
                onBlur={(e) => void rename(c, e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") e.currentTarget.blur()
                  if (e.key === "Escape") setEditing(null)
                }}
              />
            ) : (
              <div className="flex items-baseline justify-between gap-2">
                <span className={c.hidden ? "text-muted-foreground" : ""}>
                  {c.name}
                  {c.hidden && ` · ${t.hidden}`}
                </span>
                <span className="text-xs text-muted-foreground">
                  {appliesLabels[c.applies_to]}
                </span>
              </div>
            )}
            <div className="flex gap-1">
              {/* The server refuses these two on a Base category; disabling
                  them here is so the household is not offered the refusal. */}
              <Button
                size="xs"
                variant="ghost"
                disabled={c.base}
                onClick={() => setEditing(c.id)}
              >
                {t.rename}
              </Button>
              <Button
                size="xs"
                variant="ghost"
                onClick={() =>
                  void write(`/api/categories/${c.id}`, {
                    method: "PATCH",
                    body: JSON.stringify({ hidden: !c.hidden }),
                  })
                }
              >
                {c.hidden ? t.unhide : t.hide}
              </Button>
              <Button
                size="xs"
                variant="destructive"
                disabled={c.base}
                onClick={() => {
                  if (confirm(t.confirmDeleteCategory(c.name)))
                    void write(
                      `/api/categories/${c.id}`,
                      { method: "DELETE" },
                      t.categoryInUse
                    )
                }}
              >
                {t.delete}
              </Button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
