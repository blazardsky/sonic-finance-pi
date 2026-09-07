import { Fragment, useCallback, useEffect, useState } from "react"
import {
  RiAddLine,
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
import { useSidebar } from "@/components/ui/sidebar"
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
import { pickableCategories, withSaved } from "@/lib/pickers"
import { t } from "@/lib/strings"
import type { Category, Client, Contract, Lists } from "@/types"

// The Default-category picker's unset state — Radix Select refuses an
// empty-string item value, so "not set" gets a sentinel, mapped back to null
// on the way out.
const NONE = "__none__"

// The Client management screen: add, edit, hide, delete. Hidden Clients stay
// on this list — it is the only place one can be brought back from — and are
// what the Income picker filters out.
//
// The table is view-only: name, Categoria predefinita and Contratti are read
// here, never edited here. Everything that changes a Client lands in the
// sidebar via its row's own Azioni menu — Modifica cliente (name + Categoria
// predefinita, the same add/edit shape Incomes.tsx uses) and Aggiungi
// contratto are two separate entries, not one combined form: a Contract
// can't exist before its Client does (the API scopes creation under
// /api/clients/:id/contracts), and conflating "add a Client" with "add this
// Client's Contract" was the exact coupling this screen used to have.
export function Clients() {
  const [clients, setClients] = useState<Client[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  // Only for the Payer picker's options — everything else /api/settings
  // carries is another screen's business.
  const [payers, setPayers] = useState<string[]>([])
  const [error, setError] = useState("")
  const [sidebarOpen, setSidebarOpen] = useState(true)

  // The Client form: id null means "add a new Client", otherwise the id being
  // edited — same shape Incomes.tsx uses for its own add/edit form. Mutually
  // exclusive with contractFor below: selecting one clears the other.
  const [editing, setEditing] = useState<number | null>(null)
  const [name, setName] = useState("")
  const [defaultCategoryId, setDefaultCategoryId] = useState<number | null>(null)
  const [defaultPayer, setDefaultPayer] = useState("")

  // The Client currently getting a new Contract in the sidebar — its own
  // mode, entirely separate from the Client form above.
  const [contractFor, setContractFor] = useState<number | null>(null)
  const [newContract, setNewContract] = useState({
    start_month: "",
    end_month: "",
    amount: "",
  })
  const [contractError, setContractError] = useState("")

  // Contracts, per Client — loaded lazily the first time a row is expanded
  // rather than for every Client up front, since most reads of this screen
  // care about nothing past the name and the total.
  const [expanded, setExpanded] = useState<number | null>(null)
  const [contracts, setContracts] = useState<Contract[]>([])

  function resetForm() {
    setEditing(null)
    setName("")
    setDefaultCategoryId(null)
    setDefaultPayer("")
  }

  // Loads a Client into the form and makes sure the panel holding it is
  // actually visible, same reasoning as Incomes.tsx's selectIncome.
  function selectClient(c: Client) {
    setContractFor(null)
    setEditing(c.id)
    setName(c.name)
    setDefaultCategoryId(c.default_category_id)
    setDefaultPayer(c.default_payer)
    setSidebarOpen(true)
  }

  function startContract(c: Client) {
    resetForm()
    setContractFor(c.id)
    setNewContract({ start_month: "", end_month: "", amount: "" })
    setContractError("")
    setSidebarOpen(true)
  }

  const load = useCallback(
    () =>
      Promise.all([
        apiJSON<Client[]>("/api/clients"),
        apiJSON<Category[]>("/api/categories"),
        apiJSON<Lists>("/api/settings"),
      ])
        .then(([cl, c, l]) => {
          setClients(cl)
          setCategories(c)
          setPayers(l.payers)
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

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    if (editing === null) {
      // The server only accepts a name on create (cmd/clients.go) — a fresh
      // Client is never born hidden, and either default is a second request
      // right after, same as picking one up in an edit.
      setError("")
      let created: Client
      try {
        created = await apiJSON<Client>("/api/clients", {
          method: "POST",
          body: JSON.stringify({ name }),
        })
      } catch {
        setError(t.clientNotSaved)
        return
      }
      if (defaultCategoryId !== null || defaultPayer !== "") {
        await write(`/api/clients/${created.id}`, {
          method: "PATCH",
          body: JSON.stringify({
            default_category_id: defaultCategoryId,
            default_payer: defaultPayer,
          }),
        })
      } else {
        await load()
      }
      resetForm()
      toast(t.added)
      return
    }

    if (
      await write(`/api/clients/${editing}`, {
        method: "PATCH",
        body: JSON.stringify({
          name,
          default_category_id: defaultCategoryId,
          default_payer: defaultPayer,
        }),
      })
    ) {
      resetForm()
    }
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
    setExpanded(clientId)
    await refreshContracts(clientId)
  }

  // Returns whether the Contract was saved, so the sidebar (which alone knows
  // about the mobile Sheet) can close itself only once there is something to
  // close for.
  async function addContract(clientId: number, event: React.FormEvent) {
    event.preventDefault()
    const cents = toCents(newContract.amount)
    if (cents === null || cents <= 0) {
      setContractError(t.invalidAmount)
      return false
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
      return false
    }
    toast(t.added)
    setNewContract({ start_month: "", end_month: "", amount: "" })
    setContractFor(null)
    setSidebarOpen(false)
    // `contracts` backs whichever row is expanded in the table, which may be
    // a different Client than the one this form is for — only refresh it
    // when the two agree.
    if (expanded === clientId) await refreshContracts(clientId)
    return true
  }

  const contractForClient = clients?.find((c) => c.id === contractFor) ?? null

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
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
                <TableHead>{t.contracts}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {clients?.map((c) => (
                <Fragment key={c.id}>
                  <TableRow
                    className={
                      editing === c.id || contractFor === c.id
                        ? "opacity-50"
                        : ""
                    }
                  >
                    <TableCell className="whitespace-normal">
                      <span className={c.hidden ? "text-muted-foreground" : ""}>
                        {c.name}
                        {c.hidden && ` · ${t.hiddenClient}`}
                      </span>
                    </TableCell>
                    <TableCell>
                      {categories.find((cat) => cat.id === c.default_category_id)
                        ?.name ?? t.notSet}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      € {formatCents(c.total_earned_cents)}
                    </TableCell>
                    <TableCell>
                      {/* View-only: the Client's own Contracts, read here and
                          added from the sidebar instead. */}
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
                    </TableCell>
                    <TableCell>
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
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => selectClient(c)}>
                            <RiEditLine /> {t.editClient}
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => startContract(c)}>
                            <RiAddLine /> {t.addContract}
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
                                ).then((ok) => {
                                  if (!ok) return
                                  if (editing === c.id) resetForm()
                                  if (contractFor === c.id) setContractFor(null)
                                })
                            }}
                          >
                            <RiDeleteBinLine /> {t.delete}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                  {/* The expanded Client's Contracts: a nested row right under its
                    own Client rather than a second Table, so the list itself is
                    unchanged when nothing is expanded. */}
                  {expanded === c.id && (
                    <TableRow>
                      <TableCell colSpan={5} className="bg-muted/30">
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
          title={
            contractForClient
              ? t.addContractFor(contractForClient.name)
              : editing === null
                ? t.addClient
                : t.editClient
          }
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          footer={
            contractForClient ? (
              <div className="flex flex-col gap-2">
                <Button
                  type="submit"
                  form="contract-form"
                  size="lg"
                  className="h-12 text-base"
                >
                  {t.addContract}
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setContractFor(null)}
                >
                  {t.cancel}
                </Button>
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                <Button
                  type="submit"
                  form="client-form"
                  size="lg"
                  className="h-12 text-base"
                >
                  {editing === null ? t.addClient : t.save}
                </Button>
                {editing !== null && (
                  <Button type="button" variant="ghost" onClick={resetForm}>
                    {t.cancel}
                  </Button>
                )}
              </div>
            )
          }
        >
          {contractForClient ? (
            <ContractForm
              clientId={contractForClient.id}
              newContract={newContract}
              setNewContract={setNewContract}
              contractError={contractError}
              onSubmit={addContract}
            />
          ) : (
            <form
              id="client-form"
              onSubmit={submit}
              className="flex flex-col gap-3 md:pb-24"
            >
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

              <Field>
                <FieldLabel htmlFor="default-category">
                  {t.defaultCategory}
                </FieldLabel>
                <Select
                  value={defaultCategoryId == null ? NONE : String(defaultCategoryId)}
                  onValueChange={(v) =>
                    setDefaultCategoryId(v === NONE ? null : Number(v))
                  }
                >
                  <SelectTrigger id="default-category" className="h-10 w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NONE}>{t.notSet}</SelectItem>
                    {pickableCategories(
                      categories,
                      "income",
                      categories.find((cat) => cat.id === defaultCategoryId)
                    ).map((cat) => (
                      <SelectItem key={cat.id} value={String(cat.id)}>
                        {cat.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel htmlFor="default-payer">{t.defaultPayer}</FieldLabel>
                <Select
                  value={defaultPayer === "" ? NONE : defaultPayer}
                  onValueChange={(v) => setDefaultPayer(v === NONE ? "" : v)}
                >
                  <SelectTrigger id="default-payer" className="h-10 w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NONE}>{t.notSet}</SelectItem>
                    {withSaved(payers, defaultPayer || undefined).map((p) => (
                      <SelectItem key={p} value={p}>
                        {p}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </form>
          )}
        </FormSidebar>
      </div>
    </div>
  )
}

// Its own component only so it can reach useSidebar(): closing the desktop
// panel (the page's own sidebarOpen state) says nothing about the mobile
// Sheet, which is self-managed by the Sidebar primitive itself and only
// reachable from inside its Provider — form-sidebar.tsx's own documented
// escape hatch for exactly this case.
function ContractForm({
  clientId,
  newContract,
  setNewContract,
  contractError,
  onSubmit,
}: {
  clientId: number
  newContract: { start_month: string; end_month: string; amount: string }
  setNewContract: React.Dispatch<
    React.SetStateAction<{ start_month: string; end_month: string; amount: string }>
  >
  contractError: string
  onSubmit: (clientId: number, event: React.FormEvent) => Promise<boolean>
}) {
  const { setOpenMobile } = useSidebar()

  async function handleSubmit(event: React.FormEvent) {
    if (await onSubmit(clientId, event)) setOpenMobile(false)
  }

  return (
    <form
      id="contract-form"
      onSubmit={(e) => void handleSubmit(e)}
      className="flex flex-col gap-3 md:pb-24"
    >
      <Field>
        <FieldLabel htmlFor="start-month">{t.startMonth}</FieldLabel>
        <Input
          id="start-month"
          type="month"
          required
          value={newContract.start_month}
          onChange={(e) =>
            setNewContract((d) => ({ ...d, start_month: e.target.value }))
          }
          className="h-10"
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="end-month">{t.endMonth}</FieldLabel>
        <Input
          id="end-month"
          type="month"
          required
          value={newContract.end_month}
          onChange={(e) =>
            setNewContract((d) => ({ ...d, end_month: e.target.value }))
          }
          className="h-10"
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="contract-total">{t.contractTotal}</FieldLabel>
        <Input
          id="contract-total"
          type="text"
          inputMode="decimal"
          placeholder="0,00"
          required
          value={newContract.amount}
          onChange={(e) =>
            setNewContract((d) => ({ ...d, amount: e.target.value }))
          }
          className="h-10"
        />
      </Field>
      {contractError && (
        <p role="alert" className="text-sm text-destructive">
          {contractError}
        </p>
      )}
    </form>
  )
}
