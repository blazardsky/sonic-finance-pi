import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { api, apiJSON } from "@/lib/api"
import { t } from "@/lib/strings"
import type { Lists } from "@/types"

// Adding a Payer used to mean an SSH session. This screen is the whole of what
// that took: the two configured lists, and the password.
//
// The lists are edited as text, one entry per line, because that is exactly
// the shape the API takes — PUT replaces a whole list — and a line is already
// everything an entry can be: add one, delete one, drag one up to reorder the
// picker. A row-per-entry editor with its own add and remove buttons would be
// three times the code to do less.
const toText = (values: string[]) => values.join("\n")
const toList = (text: string) =>
  text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)

export function Settings() {
  const [payers, setPayers] = useState("")
  const [paymentMethods, setPaymentMethods] = useState("")
  const [listsMessage, setListsMessage] = useState("")
  const [listsError, setListsError] = useState("")

  useEffect(() => {
    apiJSON<Lists>("/api/settings")
      .then((l) => {
        setPayers(toText(l.payers))
        setPaymentMethods(toText(l.payment_methods))
      })
      .catch(() => setListsError(t.serverUnreachable))
  }, [])

  async function saveLists(event: React.FormEvent) {
    event.preventDefault()
    setListsMessage("")
    setListsError("")
    const body = {
      payers: toList(payers),
      payment_methods: toList(paymentMethods),
    }
    // Refused here as well as by the server: a picker with no options is a
    // dead end, and the household should hear about it before the round trip.
    if (body.payers.length === 0 || body.payment_methods.length === 0) {
      setListsError(t.emptyList)
      return
    }
    try {
      const saved = await apiJSON<Lists>("/api/settings", {
        method: "PUT",
        body: JSON.stringify(body),
      })
      // Redrawn from what came back, not from what was typed: the trimming and
      // the blank-line dropping are the server's, and the box should show what
      // the pickers will actually offer.
      setPayers(toText(saved.payers))
      setPaymentMethods(toText(saved.payment_methods))
      setListsMessage(t.listsSaved)
    } catch (res) {
      setListsError(
        res instanceof Response ? t.listsNotSaved : t.serverUnreachable
      )
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-md flex-col gap-8 p-6">
      <h1 className="font-medium">{t.settings}</h1>

      <form onSubmit={saveLists} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1.5 text-sm">
          {t.payersList}
          <Textarea
            value={payers}
            onChange={(e) => setPayers(e.target.value)}
            rows={4}
            spellCheck={false}
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          {t.paymentMethodsList}
          <Textarea
            value={paymentMethods}
            onChange={(e) => setPaymentMethods(e.target.value)}
            rows={3}
            spellCheck={false}
          />
        </label>
        <p className="text-sm text-muted-foreground">{t.onePerLine}</p>
        <p className="text-sm text-muted-foreground">{t.listsHistorySafe}</p>
        {listsError && (
          <p role="alert" className="text-sm text-destructive">
            {listsError}
          </p>
        )}
        {listsMessage && (
          <p role="status" className="text-sm text-muted-foreground">
            {listsMessage}
          </p>
        )}
        <Button type="submit" size="lg">
          {t.save}
        </Button>
      </form>

      <PasswordForm />
      <BackupLinks />
    </div>
  )
}

// Its own form because it is its own request, with its own refusals: a wrong
// current password and an unsaved list have nothing to say to each other.
function PasswordForm() {
  const [current, setCurrent] = useState("")
  const [next, setNext] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [busy, setBusy] = useState(false)

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setBusy(true)
    setMessage("")
    setError("")
    try {
      await api("/api/settings/password", {
        method: "POST",
        body: JSON.stringify({ current_password: current, new_password: next }),
      })
      setCurrent("")
      setNext("")
      setMessage(t.passwordChanged)
    } catch (res) {
      if (!(res instanceof Response)) setError(t.serverUnreachable)
      else if (res.status === 401) setError(t.wrongCurrentPassword)
      else if (res.status === 400) setError(t.passwordTooShort)
      else setError(t.passwordNotChanged)
    }
    setBusy(false)
  }

  return (
    <form
      onSubmit={submit}
      className="flex flex-col gap-4 border-t border-border pt-8"
    >
      <h2 className="text-sm font-medium">{t.changePassword}</h2>
      <label className="flex flex-col gap-1.5 text-sm">
        {t.currentPassword}
        <Input
          type="password"
          autoComplete="current-password"
          required
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
          className="h-9"
        />
      </label>
      <label className="flex flex-col gap-1.5 text-sm">
        {t.newPassword}
        <Input
          type="password"
          autoComplete="new-password"
          required
          minLength={8}
          value={next}
          onChange={(e) => setNext(e.target.value)}
          className="h-9"
        />
      </label>
      <p className="text-sm text-muted-foreground">{t.changePasswordHint}</p>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      {message && (
        <p role="status" className="text-sm text-muted-foreground">
          {message}
        </p>
      )}
      <Button type="submit" size="lg" disabled={busy}>
        {t.changePassword}
      </Button>
    </form>
  )
}

// Native links, not fetches: the session cookie goes with the click, and the
// browser handles the file. Wrapping them in api() would JSON-header a
// database and then have nowhere to put the bytes.
function BackupLinks() {
  return (
    <div className="flex flex-col gap-4 border-t border-border pt-8">
      <h2 className="text-sm font-medium">{t.backupAndExport}</h2>
      <p className="text-sm text-muted-foreground">{t.backupHint}</p>
      <Button asChild size="lg">
        <a href="/api/backup">{t.downloadBackup}</a>
      </Button>
      <Button asChild size="lg" variant="outline">
        <a href="/api/export/expenses.csv">{t.exportExpenses}</a>
      </Button>
      <Button asChild size="lg" variant="outline">
        <a href="/api/export/incomes.csv">{t.exportIncomes}</a>
      </Button>
    </div>
  )
}
