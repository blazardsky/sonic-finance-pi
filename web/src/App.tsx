import { useCallback, useEffect, useState } from "react"

import { Categories } from "@/Categories"
import { Expenses } from "@/Expenses"
import { Login } from "@/Login"
import { Button } from "@/components/ui/button"
import { t } from "@/strings"

// /api/health doubles as the session check: it sits behind auth like every
// other /api/ route, so a 401 is how the app learns it needs to show the login
// screen. Until it answers, neither the shell nor the login screen is the
// right thing to render — hence "checking", rather than a flash of the wrong
// one on every cold load.
type State = "checking" | "loggedOut" | "connected" | "unreachable"

export function App() {
  const [state, setState] = useState<State>("checking")
  const [screen, setScreen] = useState<"expenses" | "categories">("expenses")

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

  // Two screens and a text link between them. Logging an Expense is what the
  // app is for, so it is what opens; Categories is somewhere you go once in a
  // while. Later tickets will earn a real nav — this is not it yet.
  return (
    <>
      {screen === "expenses" ? <Expenses /> : <Categories />}
      <div className="mx-auto flex w-full max-w-md justify-end px-6 pb-8">
        <Button
          size="sm"
          variant="ghost"
          onClick={() =>
            setScreen(screen === "expenses" ? "categories" : "expenses")
          }
        >
          {screen === "expenses" ? t.categories : t.expenses}
        </Button>
      </div>
    </>
  )
}

export default App
