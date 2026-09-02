import { useCallback, useEffect, useState } from "react"

import { Login } from "@/Login"
import { t } from "@/strings"

// /api/health doubles as the session check: it sits behind auth like every
// other /api/ route, so a 401 is how the app learns it needs to show the login
// screen. Until it answers, neither the shell nor the login screen is the
// right thing to render — hence "checking", rather than a flash of the wrong
// one on every cold load.
type State = "checking" | "loggedOut" | "connected" | "unreachable"

// Placeholder shell, until a later ticket gives it something to show.
export function App() {
  const [state, setState] = useState<State>("checking")

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

  return (
    <div className="flex min-h-svh p-6">
      <div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
        <h1 className="font-medium">{t.appName}</h1>
        <p className="font-mono text-xs text-muted-foreground">
          {state === "connected" ? t.connected : t.serverUnreachable}
        </p>
      </div>
    </div>
  )
}

export default App
