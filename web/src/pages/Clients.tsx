import { Fragment, useCallback, useEffect, useState } from "react"
import {
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiDeleteBinLine,
  RiEditLine,
  RiEyeLine,
  RiEyeOffLine,
  RiMoreLine,
} from "@remixicon/react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api, apiJSON } from "@/lib/api"
import { FormSidebar } from "@/components/form-sidebar"
import { toast } from "@/lib/toast"
import { formatCents, formatMonth, toCents } from "@/lib/money"
import { pickableCategories } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Client, Contract } from "@/types"

// The Default-category picker's unset state — Radix Select refuses an
// empty-string item value, so "not set" gets a sentinel, mapped back to null
// on the way out.
const NONE = "__none__"

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
  const [sidebarOpen, setSidebarOpen] = useState(true)

  // Contracts, per Client — loaded lazily the first time a row is expanded
  // rather than for every Client up front, since most reads of this screen
  // care about nothing past the name and the total.
  const [expanded, setExpanded] = useState<number | null>(null)
  const [contracts, setContracts] = useState<Contract[]>([])
  const [newContract, setNewContract] = useState({
    start_month: "",
    end_month: "",
    amount: "",
  })
  const [contractError, setContractError] = useState("")

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
    ) {
      setName("")
      toast(t.added)
    }
  }

  async function rename(c: Client, to: string) {
    setEditing(null)
    if (to.trim() === "" || to === c.name) return
    await write(`/api/clients/${c.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: to }),
    })
  }

  // Shared by expanding a row and saving a new Contract into it — both end
  // with the same "load what this Client has now" round trip.
  const refreshContracts = (clientId: number) =>
    apiJSON<Contract[]>(`/api/clients/${clientId}/contracts`).then(setContracts)

  // Expanding a row loads that Client's Contracts; collapsing it (or
  // expanding a different one) just hides them again — nothing here is worth
  // caching across two Clients' rows.
  async function toggleExpanded(clientId: number) {
    if (expanded === clientId) {
      setExpanded(null)
      return
    }
    setContractError("")
    setNewContract({ start_month: "", end_month: "", amount: "" })
    setExpanded(clientId)
    await refreshContracts(clientId)
  }

  async function addContract(clientId: number, event: React.FormEvent) {
    event.preventDefault()
    const cents = toCents(newContract.amount)
    if (cents === null || cents <= 0) {
      setContractError(t.invalidAmount)
      return
    }
    setContractError("")
    try {
      await api(`/api/clients/${clientId}/contracts`, {
        method: "POST",
        body: JSON.stringify({
          start_month: newContract.start_month,
          end_month: newContract.end_month,
          total_cents: cents,
        }),
      })
    } catch {
      setContractError(t.contractNotSaved)
      return
    }
    setNewContract({ start_month: "", end_month: "", amount: "" })
    await refreshContracts(clientId)
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6">
      <h1 className="font-medium">{t.clients}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
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
                <Fragment key={c.id}>
                  <TableRow>
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
                      <Select
                        value={
                          c.default_category_id == null
                            ? NONE
                            : String(c.default_category_id)
                        }
                        onValueChange={(v) =>
                          void write(`/api/clients/${c.id}`, {
                            method: "PATCH",
                            body: JSON.stringify({
                              default_category_id:
                                v === NONE ? null : Number(v),
                            }),
                          })
                        }
                      >
                        <SelectTrigger
                          aria-label={t.defaultCategory}
                          className="h-9 w-full"
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value={NONE}>{t.notSet}</SelectItem>
                          {pickableCategories(
                            categories,
                            "income",
                            categories.find(
                              (cat) => cat.id === c.default_category_id
                            )
                          ).map((cat) => (
                            <SelectItem key={cat.id} value={String(cat.id)}>
                              {cat.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      € {formatCents(c.total_earned_cents)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {/* Not an item in the actions menu: this toggles a
                            disclosure right under the row, not a one-shot
                            action, so it stays its own visible control —
                            the expand/collapse behavior itself is unchanged. */}
                        <Button
                          size="xs"
                          variant={expanded === c.id ? "secondary" : "ghost"}
                          onClick={() => void toggleExpanded(c.id)}
                        >
                          {t.contracts}
                          {expanded === c.id ? (
                            <RiArrowUpSLine />
                          ) : (
                            <RiArrowDownSLine />
                          )}
                        </Button>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon-sm"
                              aria-label={t.actions}
                            >
                              <RiMoreLine />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent
                            align="end"
                            // Without this, Radix returns focus to the "..."
                            // trigger once the menu closes, racing (and
                            // beating) the name cell's fresh autoFocus input.
                            onCloseAutoFocus={(e) => e.preventDefault()}
                          >
                            <DropdownMenuItem
                              // Deferred a tick past the click, same as
                              // Categories.tsx's Rinomina: mounting the
                              // autoFocus input immediately loses the race
                              // against Radix's own close teardown.
                              onClick={() =>
                                setTimeout(() => setEditing(c.id), 0)
                              }
                            >
                              <RiEditLine /> {t.rename}
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() =>
                                void write(`/api/clients/${c.id}`, {
                                  method: "PATCH",
                                  body: JSON.stringify({ hidden: !c.hidden }),
                                })
                              }
                            >
                              {c.hidden ? <RiEyeLine /> : <RiEyeOffLine />}{" "}
                              {c.hidden ? t.unhide : t.hide}
                            </DropdownMenuItem>
                            <DropdownMenuItem
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
                              <RiDeleteBinLine /> {t.delete}
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </TableCell>
                  </TableRow>
                  {/* The expanded Client's Contracts: a nested row right under its
                    own Client rather than a second Table, so the list itself is
                    unchanged when nothing is expanded. */}
                  {expanded === c.id && (
                    <TableRow>
                      <TableCell colSpan={4} className="bg-muted/30">
                        <div className="flex flex-col gap-3 py-2">
                          {contracts.length === 0 && (
                            <p className="text-sm text-muted-foreground">
                              {t.noContractsYet}
                            </p>
                          )}
                          {contracts.map((ct) => (
                            <div
                              key={ct.id}
                              className="flex flex-col gap-1 rounded-lg border p-2 text-sm"
                            >
                              <div className="flex justify-between font-medium">
                                <span>
                                  {formatMonth(ct.start_month)} –{" "}
                                  {formatMonth(ct.end_month)}
                                </span>
                                <span className="tabular-nums">
                                  € {formatCents(ct.total_cents)}
                                </span>
                              </div>
                              <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-muted-foreground">
                                <span>
                                  {t.expectedSoFar}: €{" "}
                                  {formatCents(ct.expected_so_far_cents)}
                                </span>
                                <span>
                                  {t.contractReceived}: €{" "}
                                  {formatCents(ct.received_cents)}
                                </span>
                                <span
                                  className={ct.overdue ? "text-destructive" : ""}
                                >
                                  {t.invoiceTarget}: €{" "}
                                  {formatCents(ct.invoice_target_this_month_cents)}
                                  {ct.overdue && ` (${t.contractOverdue})`}
                                </span>
                              </div>
                            </div>
                          ))}

                          <form
                            onSubmit={(e) => void addContract(expanded, e)}
                            className="flex flex-wrap items-end gap-2"
                          >
                            <label className="flex flex-col gap-1 text-xs">
                              {t.startMonth}
                              <Input
                                type="month"
                                required
                                value={newContract.start_month}
                                onChange={(e) =>
                                  setNewContract((d) => ({
                                    ...d,
                                    start_month: e.target.value,
                                  }))
                                }
                                className="h-9"
                              />
                            </label>
                            <label className="flex flex-col gap-1 text-xs">
                              {t.endMonth}
                              <Input
                                type="month"
                                required
                                value={newContract.end_month}
                                onChange={(e) =>
                                  setNewContract((d) => ({
                                    ...d,
                                    end_month: e.target.value,
                                  }))
                                }
                                className="h-9"
                              />
                            </label>
                            <label className="flex flex-col gap-1 text-xs">
                              {t.contractTotal}
                              <Input
                                type="text"
                                inputMode="decimal"
                                placeholder="0,00"
                                required
                                value={newContract.amount}
                                onChange={(e) =>
                                  setNewContract((d) => ({
                                    ...d,
                                    amount: e.target.value,
                                  }))
                                }
                                className="h-9 w-28"
                              />
                            </label>
                            <Button type="submit" size="sm">
                              {t.addContract}
                            </Button>
                          </form>
                          {contractError && (
                            <p role="alert" className="text-sm text-destructive">
                              {contractError}
                            </p>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        </div>

        <FormSidebar
          title={t.addClient}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
        >
          <form onSubmit={add} className="flex flex-col gap-3">
            <Field>
              <FieldLabel htmlFor="name">{t.clientName}</FieldLabel>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t.clientName}
                required
                className="h-10"
              />
            </Field>
            <Button type="submit" size="lg" className="h-12 text-base">
              {t.addClient}
            </Button>
          </form>
        </FormSidebar>
      </div>
    </div>
  )
}
