import { useCallback, useEffect, useMemo, useState } from "react"
import {
  RiAddLine,
  RiDeleteBinLine,
  RiEditLine,
  RiEyeLine,
  RiEyeOffLine,
  RiMoreLine,
} from "@remixicon/react"

import { Button } from "@/components/ui/button"
import {
  createDataTableColumnHelper,
  DataTable,
  type DataTableColumnDef,
} from "@/components/data-table"
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
import { api, apiJSON } from "@/lib/api"
import { FormSidebar } from "@/components/form-sidebar"
import { toast } from "@/lib/toast"
import { formatCents, formatMonth, toCents, toTyped } from "@/lib/money"
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

  // The Client currently getting a new (or edited) Contract in the sidebar —
  // its own mode, entirely separate from the Client form above.
  const [contractFor, setContractFor] = useState<number | null>(null)
  // null while adding; the Contract's own id while editing an existing one —
  // same shape Incomes.tsx's editing/blankDraft split uses, one form serving
  // both.
  const [editingContractId, setEditingContractId] = useState<number | null>(null)
  const [newContract, setNewContract] = useState({
    start_month: "",
    end_month: "",
    amount: "",
  })
  const [contractError, setContractError] = useState("")

  // Bumped after any Contract save so the currently-expanded row's own
  // ClientContracts (below) remounts and refetches — it otherwise only
  // fetches once, on first expand.
  const [contractsVersion, setContractsVersion] = useState(0)

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
    setEditingContractId(null)
    setNewContract({ start_month: "", end_month: "", amount: "" })
    setContractError("")
    setSidebarOpen(true)
  }

  // Loads an existing Contract into the same form startContract's "add"
  // mode uses — addContract below tells the two apart by editingContractId,
  // the same way Incomes.tsx's own form serves add and edit from one draft.
  function startEditContract(clientId: number, ct: Contract) {
    resetForm()
    setContractFor(clientId)
    setEditingContractId(ct.id)
    setNewContract({
      start_month: ct.start_month,
      end_month: ct.end_month,
      amount: toTyped(ct.total_cents),
    })
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

  // Returns whether the Contract was saved, so the sidebar (which alone knows
  // about the mobile Sheet) can close itself only once there is something to
  // close for. editingContractId picks POST vs PATCH — the same one draft
  // serves both, as startEditContract's own comment explains.
  async function addContract(clientId: number, event: React.FormEvent) {
    event.preventDefault()
    const cents = toCents(newContract.amount)
    if (cents === null || cents <= 0) {
      setContractError(t.invalidAmount)
      return false
    }
    setContractError("")
    const wasEditing = editingContractId !== null
    try {
      await api(
        wasEditing
          ? `/api/clients/${clientId}/contracts/${editingContractId}`
          : `/api/clients/${clientId}/contracts`,
        {
          method: wasEditing ? "PATCH" : "POST",
          body: JSON.stringify({
            start_month: newContract.start_month,
            end_month: newContract.end_month,
            total_cents: cents,
          }),
        }
      )
    } catch {
      setContractError(t.contractNotSaved)
      return false
    }
    // A correction is finished silently, the same as Incomes.tsx's own edit
    // path — only a brand new Contract gets the "added" toast.
    if (!wasEditing) toast(t.added)
    setNewContract({ start_month: "", end_month: "", amount: "" })
    setContractFor(null)
    setEditingContractId(null)
    setSidebarOpen(false)
    // Bumps the currently-expanded row's ClientContracts remount key —
    // harmless if the saved Contract's Client isn't the one expanded.
    setContractsVersion((v) => v + 1)
    return true
  }

  const contractForClient = clients?.find((c) => c.id === contractFor) ?? null

  // v1.3.0: DataTable's own sortable headers, per-column filters (Name,
  // Default Category, Total Earned — the table's only columns today; the
  // former "Contratti" toggle column is replaced by DataTable's own
  // auto-injected expand toggle) and row expansion for Contracts. Default
  // Category filters/sorts by name via an accessorFn, not the raw id.
  // `editing`/`contractFor`'s dimmed row is dropped, same reasoning
  // Expenses.tsx's own migration used.
  const columns = useMemo<DataTableColumnDef<Client>[]>(() => {
    const helper = createDataTableColumnHelper<Client>()
    return [
      helper.accessor("name", {
        header: t.clientName,
        meta: { filterVariant: "text" },
        cell: ({ row }) => (
          <span className={row.original.hidden ? "text-muted-foreground" : ""}>
            {row.original.name}
            {row.original.hidden && ` · ${t.hiddenClient}`}
          </span>
        ),
      }),
      helper.accessor(
        (c) =>
          categories.find((cat) => cat.id === c.default_category_id)?.name ??
          t.notSet,
        {
          id: "default_category",
          header: t.defaultCategory,
          meta: { filterVariant: "select" },
        }
      ),
      helper.accessor("total_earned_cents", {
        header: t.totalEarned,
        meta: { filterVariant: "range" },
        cell: ({ row }) => (
          <div className="text-right tabular-nums">
            € {formatCents(row.original.total_earned_cents)}
          </div>
        ),
      }),
      helper.display({
        id: "actions",
        header: t.actions,
        enableSorting: false,
        cell: ({ row }) => {
          const c = row.original
          return (
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
          )
        },
      }),
    ]
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [categories, editing, contractFor])

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

          <DataTable
            columns={columns}
            data={clients ?? []}
            getRowId={(c) => String(c.id)}
            renderSubRow={(row) => (
              <ClientContracts
                key={contractsVersion}
                clientId={row.original.id}
                onEditContract={(ct) => startEditContract(row.original.id, ct)}
              />
            )}
          />
        </div>

        <FormSidebar
          title={
            contractForClient
              ? editingContractId === null
                ? t.addContractFor(contractForClient.name)
                : t.editContractFor(contractForClient.name)
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
                  {editingContractId === null ? t.addContract : t.save}
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setContractFor(null)
                    setEditingContractId(null)
                  }}
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

// A Client's Contracts, fetched on mount rather than passed down — DataTable
// only mounts a `renderSubRow` when its row is actually expanded (and
// unmounts it on collapse), which is what gives this the same "fetch lazily,
// only for the row actually opened" behavior the previous hand-rolled
// expand toggle had. The parent bumps ClientContracts' `key` after any
// Contract save so this remounts (and refetches) instead of going stale.
function ClientContracts({
  clientId,
  onEditContract,
}: {
  clientId: number
  onEditContract: (contract: Contract) => void
}) {
  const [contracts, setContracts] = useState<Contract[] | null>(null)

  useEffect(() => {
    let cancelled = false
    apiJSON<Contract[]>(`/api/clients/${clientId}/contracts`).then((cs) => {
      if (!cancelled) setContracts(cs)
    })
    return () => {
      cancelled = true
    }
  }, [clientId])

  if (contracts === null) return null

  return (
    <div className="flex flex-col gap-3 py-2">
      {contracts.length === 0 && (
        <p className="text-sm text-muted-foreground">{t.noContractsYet}</p>
      )}
      {contracts.map((ct) => (
        <div
          key={ct.id}
          className="flex flex-col gap-1 rounded-lg border p-2 text-sm"
        >
          <div className="flex items-center justify-between gap-2 font-medium">
            <span>
              {formatMonth(ct.start_month)} – {formatMonth(ct.end_month)}
            </span>
            <span className="flex items-center gap-1">
              <span className="tabular-nums">
                € {formatCents(ct.total_cents)}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t.editContract}
                onClick={() => onEditContract(ct)}
              >
                <RiEditLine />
              </Button>
            </span>
          </div>
          <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-muted-foreground">
            <span>
              {t.expectedSoFar}: € {formatCents(ct.expected_so_far_cents)}
            </span>
            <span>
              {t.contractReceived}: € {formatCents(ct.received_cents)}
            </span>
            <span className={ct.overdue ? "text-destructive" : ""}>
              {t.invoiceTarget}: €{" "}
              {formatCents(ct.invoice_target_this_month_cents)}
              {ct.overdue && ` (${t.contractOverdue})`}
            </span>
          </div>
        </div>
      ))}
    </div>
  )
}
