import { useEffect, useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldTitle,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Toggle } from "@/components/ui/toggle"
import { api, apiJSON } from "@/lib/api"
import { toCents, toTyped } from "@/lib/money"
import { t } from "@/lib/strings"
import type { Lists } from "@/types"

// Adding a Payer used to mean an SSH session. This screen is the whole of what
// that took: the two configured lists, and the password.
//
// The lists are edited as text, one entry per line, because that is exactly
// the shape the API takes — PUT replaces a whole list — and a line is already
// everything an entry can be: add one, delete one, drag one up to reorder the
// picker. A row-per-entry editor with its own add and remove buttons would be
// three times the code to do less. The Badges under each box are read-only —
// a live "here's what's in the box right now" preview, not a second way to
// edit; the Textarea is still the only thing that does that.
const toText = (values: string[]) => values.join("\n")
const toList = (text: string) =>
  text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)

export function Settings() {
  const [payers, setPayers] = useState("")
  const [paymentMethods, setPaymentMethods] = useState("")
  // Target and Goal (ticket 04) ride the same settings payload as the two
  // lists above, so one save covers all four fields in one request — same
  // amount-typing convention as the Expense/Income forms (toCents/toTyped).
  const [target, setTarget] = useState("")
  const [goal, setGoal] = useState("")
  // Savings' starting balance: set once and left alone, so it lives here with
  // the other household-set numbers rather than on Risparmi.
  const [startingBalance, setStartingBalance] = useState("")
  // Ticket 01: the feature's single switch. Off by default, and rides the
  // same payload/save as the rest of this form rather than its own request —
  // one settings screen, one save.
  const [spendingIntentEnabled, setSpendingIntentEnabled] = useState(false)
  const [listsMessage, setListsMessage] = useState("")
  const [listsError, setListsError] = useState("")

  useEffect(() => {
    apiJSON<Lists>("/api/settings")
      .then((l) => {
        setPayers(toText(l.payers))
        setPaymentMethods(toText(l.payment_methods))
        setTarget(toTyped(l.target_cents))
        setGoal(toTyped(l.goal_cents))
        setStartingBalance(toTyped(l.savings_starting_balance_cents))
        setSpendingIntentEnabled(l.spending_intent_enabled)
      })
      .catch(() => setListsError(t.serverUnreachable))
  }, [])

  async function saveLists(event: React.FormEvent) {
    event.preventDefault()
    setListsMessage("")
    setListsError("")
    const targetCents = toCents(target)
    const goalCents = toCents(goal)
    const startingBalanceCents = toCents(startingBalance)
    if (
      targetCents === null ||
      goalCents === null ||
      startingBalanceCents === null
    ) {
      setListsError(t.invalidAmount)
      return
    }
    const body = {
      payers: toList(payers),
      payment_methods: toList(paymentMethods),
      target_cents: targetCents,
      goal_cents: goalCents,
      savings_starting_balance_cents: startingBalanceCents,
      spending_intent_enabled: spendingIntentEnabled,
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
      setTarget(toTyped(saved.target_cents))
      setGoal(toTyped(saved.goal_cents))
      setStartingBalance(toTyped(saved.savings_starting_balance_cents))
      setSpendingIntentEnabled(saved.spending_intent_enabled)
      setListsMessage(t.listsSaved)
    } catch (res) {
      setListsError(
        res instanceof Response ? t.listsNotSaved : t.serverUnreachable
      )
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-8 p-6">
      <h1 className="font-medium">{t.settings}</h1>

      <div className="flex flex-wrap items-start gap-6">
        <Card className="w-full max-w-md">
          <CardContent>
            <form onSubmit={saveLists} className="flex flex-col gap-4">
              <Field>
                <FieldLabel htmlFor="payers">{t.payersList}</FieldLabel>
                <Textarea
                  id="payers"
                  value={payers}
                  onChange={(e) => setPayers(e.target.value)}
                  rows={4}
                  spellCheck={false}
                />
                <div className="flex flex-wrap gap-1">
                  {toList(payers).map((p) => (
                    <Badge key={p} variant="secondary">
                      {p}
                    </Badge>
                  ))}
                </div>
              </Field>
              <Field>
                <FieldLabel htmlFor="payment_methods">
                  {t.paymentMethodsList}
                </FieldLabel>
                <Textarea
                  id="payment_methods"
                  value={paymentMethods}
                  onChange={(e) => setPaymentMethods(e.target.value)}
                  rows={3}
                  spellCheck={false}
                />
                <div className="flex flex-wrap gap-1">
                  {toList(paymentMethods).map((m) => (
                    <Badge key={m} variant="secondary">
                      {m}
                    </Badge>
                  ))}
                </div>
              </Field>
              <FieldDescription>{t.onePerLine}</FieldDescription>
              <FieldDescription>{t.listsHistorySafe}</FieldDescription>

              <Field>
                <FieldLabel htmlFor="target">{t.target}</FieldLabel>
                <Input
                  id="target"
                  inputMode="decimal"
                  value={target}
                  onChange={(e) => setTarget(e.target.value)}
                  className="h-9"
                />
                <FieldDescription>{t.targetHint}</FieldDescription>
              </Field>
              <Field>
                <FieldLabel htmlFor="goal">{t.savingsGoal}</FieldLabel>
                <Input
                  id="goal"
                  inputMode="decimal"
                  value={goal}
                  onChange={(e) => setGoal(e.target.value)}
                  className="h-9"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="starting-balance">
                  {t.startingBalance}
                </FieldLabel>
                <Input
                  id="starting-balance"
                  inputMode="decimal"
                  value={startingBalance}
                  onChange={(e) => setStartingBalance(e.target.value)}
                  className="h-9"
                />
                <FieldDescription>{t.startingBalanceHint}</FieldDescription>
              </Field>

              <FieldLabel htmlFor="spending-intent-enabled">
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldTitle>{t.spendingIntent}</FieldTitle>
                    <FieldDescription>{t.spendingIntentHint}</FieldDescription>
                  </FieldContent>
                  <Switch
                    id="spending-intent-enabled"
                    checked={spendingIntentEnabled}
                    onCheckedChange={setSpendingIntentEnabled}
                  />
                </Field>
              </FieldLabel>

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
          </CardContent>
        </Card>

        <WorkersCard />
        <PasswordForm />
        <BackupLinks />
      </div>
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
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>{t.changePassword}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field>
            <FieldLabel htmlFor="current-password">
              {t.currentPassword}
            </FieldLabel>
            <Input
              id="current-password"
              type="password"
              autoComplete="current-password"
              required
              value={current}
              onChange={(e) => setCurrent(e.target.value)}
              className="h-9"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="new-password">{t.newPassword}</FieldLabel>
            <Input
              id="new-password"
              type="password"
              autoComplete="new-password"
              required
              minLength={8}
              value={next}
              onChange={(e) => setNext(e.target.value)}
              className="h-9"
            />
            <FieldDescription>{t.changePasswordHint}</FieldDescription>
          </Field>
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
      </CardContent>
    </Card>
  )
}

// Native links, not fetches: the session cookie goes with the click, and the
// browser handles the file. Wrapping them in api() would JSON-header a
// database and then have nowhere to put the bytes.
function BackupLinks() {
  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>{t.backupAndExport}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
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
      </CardContent>
    </Card>
  )
}

// The "Dati lavoratori" card: its own save, sending only its own fields — the
// settings PUT leaves everything it isn't sent as it was, so this never
// touches the lists, Target or Goal. The fields behind a switch are hidden,
// not cleared, while it is off: switching it back on finds them as they were.
function WorkersCard() {
  const [selfEmployed, setSelfEmployed] = useState(false)
  const [percent, setPercent] = useState("33")
  const [taxMonths, setTaxMonths] = useState<number[]>([])
  const [bonusPaychecks, setBonusPaychecks] = useState(false)
  const [bonusMonths, setBonusMonths] = useState<number[]>([])
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")

  const show = (l: Lists) => {
    setSelfEmployed(l.self_employed)
    setPercent(String(l.tax_reserve_fallback_percent))
    setTaxMonths(l.tax_months)
    setBonusPaychecks(l.bonus_paychecks)
    setBonusMonths(l.bonus_months)
  }

  useEffect(() => {
    apiJSON<Lists>("/api/settings")
      .then(show)
      .catch(() => setError(t.serverUnreachable))
  }, [])

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setMessage("")
    setError("")
    const value = Number(percent)
    if (
      percent.trim() === "" ||
      !Number.isInteger(value) ||
      value < 0 ||
      value > 100
    ) {
      setError(t.invalidPercent)
      return
    }
    try {
      show(
        await apiJSON<Lists>("/api/settings", {
          method: "PUT",
          body: JSON.stringify({
            self_employed: selfEmployed,
            tax_reserve_fallback_percent: value,
            tax_months: taxMonths,
            bonus_paychecks: bonusPaychecks,
            bonus_months: bonusMonths,
          }),
        })
      )
      setMessage(t.workersSaved)
    } catch (res) {
      setError(
        res instanceof Response ? t.workersNotSaved : t.serverUnreachable
      )
    }
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>{t.workersData}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="flex flex-col gap-4">
          <FieldLabel htmlFor="self-employed">
            <Field orientation="horizontal">
              <FieldContent>
                <FieldTitle>{t.selfEmployed}</FieldTitle>
              </FieldContent>
              <Switch
                id="self-employed"
                checked={selfEmployed}
                onCheckedChange={setSelfEmployed}
              />
            </Field>
          </FieldLabel>
          {selfEmployed && (
            <>
              <Field>
                <FieldLabel htmlFor="tax-percent">
                  {t.taxReservePercent}
                </FieldLabel>
                <Input
                  id="tax-percent"
                  inputMode="numeric"
                  value={percent}
                  onChange={(e) => setPercent(e.target.value)}
                  className="h-9"
                />
                <FieldDescription>{t.taxReservePercentHint}</FieldDescription>
              </Field>
              <MonthToggles
                label={t.taxMonths}
                months={taxMonths}
                onChange={setTaxMonths}
              />
            </>
          )}

          <FieldLabel htmlFor="bonus-paychecks">
            <Field orientation="horizontal">
              <FieldContent>
                <FieldTitle>{t.bonusPaychecks}</FieldTitle>
              </FieldContent>
              <Switch
                id="bonus-paychecks"
                checked={bonusPaychecks}
                onCheckedChange={setBonusPaychecks}
              />
            </Field>
          </FieldLabel>
          {bonusPaychecks && (
            <MonthToggles
              label={t.bonusMonths}
              months={bonusMonths}
              onChange={setBonusMonths}
            />
          )}

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
          <Button type="submit" size="lg">
            {t.save}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

// Twelve month buttons (1–12), pressed for the months in the set.
function MonthToggles({
  label,
  months,
  onChange,
}: {
  label: string
  months: number[]
  onChange: (months: number[]) => void
}) {
  return (
    <Field>
      <FieldTitle>{label}</FieldTitle>
      <div className="grid grid-cols-6 gap-1">
        {t.monthsShort.map((name, i) => {
          const month = i + 1
          return (
            <Toggle
              key={month}
              variant="outline"
              size="sm"
              className="capitalize"
              pressed={months.includes(month)}
              onPressedChange={(on) =>
                onChange(
                  on
                    ? [...months, month].sort((a, b) => a - b)
                    : months.filter((m) => m !== month)
                )
              }
            >
              {name}
            </Toggle>
          )
        })}
      </div>
    </Field>
  )
}
