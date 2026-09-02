import { useCallback, useEffect, useState } from "react"

import { Categories } from "@/Categories"
import { Clients } from "@/Clients"
import { Expenses } from "@/Expenses"
import { Incomes } from "@/Incomes"
import { Login } from "@/Login"
import { Month } from "@/Month"
import { Button } from "@/components/ui/button"
import { t } from "@/strings"

// /api/health doubles as the session check: it sits behind auth like every
// other /api/ route, so a 401 is how the app learns it needs to show the login
// screen. Until it answers, neither the shell nor the login screen is the
// right thing to render — hence "checking", rather than a flash of the wrong
// one on every cold load.
type State = "checking" | "loggedOut" | "connected" | "unreachable"

// The screens, in the order the nav lists them. The month view opens, because
// where the month stands is the first question the app exists to answer;
// logging an Expense is next, and the two management screens sit after it. The
// keys are the strings-file keys too, so the nav labels itself.
const screens = [
  "month",
  "expenses",
  "incomes",
  "categories",
  "clients",
] as const
type Screen = (typeof screens)[number]

export function App() {
  const [state, setState] = useState<State>("checking")
  const [screen, setScreen] = useState<Screen>("month")

  const check = useCallback(() => {
    fetch("/api/health")
      .then((r) => {
        if (r.status === 401) return setState("loggedOut")
        setState(r.ok ? "connected" : "unreachable")
      })
      .catch(() => setState("unreachable"))
  }, [])

  useEffect(() => check(), [check])

  if (state === "loggedOut") return <Login onLoggedIn={check} />

  if (state === "checking") {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <p className="font-mono text-xs text-muted-foreground">
          {t.connecting}
        </p>
      </div>
    )
  }

  if (state === "unreachable") {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <p className="font-mono text-xs text-muted-foreground">
          {t.serverUnreachable}
        </p>
      </div>
    )
  }

  // A row of text links, one per screen, the current one not offered. A later
  // ticket will earn a real nav — this is not it yet.
  return (
    <>
      {screen === "month" && <Month />}
      {screen === "expenses" && <Expenses />}
      {screen === "incomes" && <Incomes />}
      {screen === "categories" && <Categories />}
      {screen === "clients" && <Clients />}
      <div className="mx-auto flex w-full max-w-md justify-end gap-1 px-6 pb-8">
        {screens.map(
          (key) =>
            key !== screen && (
              <Button
                key={key}
                size="sm"
                variant="ghost"
                onClick={() => setScreen(key)}
              >
                {t[key]}
              </Button>
            )
        )}
      </div>
    </>
  )
}

export default App
