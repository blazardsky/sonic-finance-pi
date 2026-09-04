import { useCallback, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api, apiJSON } from "@/lib/api"
import { formatCents } from "@/lib/money"
import { pickableCategories } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Client } from "@/types"

// The Client management screen: add, rename, hide, delete. Hidden Clients stay
// on this list — it is the only place one can be brought back from — and are
// what the Income picker filters out.
//
// Simpler than Categories by exactly what a Client is not: no scope to choose,
// and nothing protected, because no report resolves a Client by identity.
export function Clients() {
  const [clients, setClients] = useState<Client[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [error, setError] = useState("")
  const [name, setName] = useState("")
  const [editing, setEditing] = useState<number | null>(null)

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Client[]>("/api/clients"),
        apiJSON<Category[]>("/api/categories"),
      ])
        .then(([cl, c]) => {
          setClients(cl)
          setCategories(c)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // Every write goes through here so the reload and the error wording are in
  // one place. onConflict is the one sentence a caller has to supply itself: a
  // 409 means something is still pointing at this Client, and only the caller
  // knows what was being attempted.
  async function write(path: string, init: RequestInit, onConflict?: string) {
    setError("")
    try {
      await api(path, init)
      await load()
      return true
    } catch (res) {
      // A refusal the Pi answered and a Pi that is not there read differently
      // to whoever is holding the phone: one is worth another try at the
      // form, the other is worth walking to the other room.
      if (!(res instanceof Response)) setError(t.serverUnreachable)
      else
        setError(
          res.status === 409 && onConflict ? onConflict : t.clientNotSaved
        )
      return false
    }
  }

  async function add(event: React.FormEvent) {
    event.preventDefault()
    if (
      await write("/api/clients", {
        method: "POST",
        body: JSON.stringify({ name }),
      })
    )
      setName("")
  }

  async function rename(c: Client, to: string) {
    setEditing(null)
    if (to.trim() === "" || to === c.name) return
    await write(`/api/clients/${c.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: to }),
    })
  }

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-6 p-6">
      <h1 className="font-medium">{t.clients}</h1>

      <form onSubmit={add} className="flex flex-col gap-2">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t.clientName}
          required
          className="h-9"
        />
        <Button type="submit" size="lg">
          {t.addClient}
        </Button>
      </form>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {clients?.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noClientsYet}</p>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t.clientName}</TableHead>
            <TableHead>{t.defaultCategory}</TableHead>
            <TableHead className="text-right">{t.totalEarned}</TableHead>
            <TableHead>{t.actions}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {clients?.map((c) => (
            <TableRow key={c.id}>
              <TableCell className="whitespace-normal">
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
                  <span className={c.hidden ? "text-muted-foreground" : ""}>
                    {c.name}
                    {c.hidden && ` · ${t.hiddenClient}`}
                  </span>
                )}
              </TableCell>
              <TableCell>
                <NativeSelect
                  value={c.default_category_id ?? ""}
                  onChange={(e) =>
                    void write(`/api/clients/${c.id}`, {
                      method: "PATCH",
                      body: JSON.stringify({
                        default_category_id:
                          e.target.value === "" ? null : Number(e.target.value),
                      }),
                    })
                  }
                  className="h-9"
                >
                  <option value="">{t.notSet}</option>
                  {pickableCategories(
                    categories,
                    "income",
                    categories.find((cat) => cat.id === c.default_category_id)
                  ).map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </NativeSelect>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                € {formatCents(c.total_earned_cents)}
              </TableCell>
              <TableCell>
                <div className="flex gap-1">
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => setEditing(c.id)}
                  >
                    {t.rename}
                  </Button>
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() =>
                      void write(`/api/clients/${c.id}`, {
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
                    onClick={() => {
                      if (confirm(t.confirmDeleteClient(c.name)))
                        void write(
                          `/api/clients/${c.id}`,
                          { method: "DELETE" },
                          t.clientInUse
                        )
                    }}
                  >
                    {t.delete}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
