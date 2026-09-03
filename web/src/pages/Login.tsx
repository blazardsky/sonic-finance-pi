import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { t } from "@/lib/strings"

// The login screen ships in the static bundle, which the server hands out
// without a session — the shell has to be able to render its own way in.
export function Login({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [busy, setBusy] = useState(false)

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError("")
    try {
      const res = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      })
      if (res.ok) {
        onLoggedIn()
        return
      }
      setError(res.status === 401 ? t.wrongPassword : t.serverUnreachable)
    } catch {
      setError(t.serverUnreachable)
    }
    setBusy(false)
  }

  return (
    <div className="flex min-h-svh items-center justify-center p-6">
      <form onSubmit={submit} className="flex w-full max-w-xs flex-col gap-4">
        <h1 className="font-medium">{t.appName}</h1>
        <label className="flex flex-col gap-1.5 text-sm">
          {t.password}
          <Input
            type="password"
            name="password"
            autoFocus
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={error !== ""}
            aria-describedby={error ? "login-error" : undefined}
            className="h-9"
          />
        </label>
        {error && (
          <p id="login-error" role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <Button type="submit" size="lg" disabled={busy}>
          {busy ? t.loggingIn : t.logIn}
        </Button>
      </form>
    </div>
  )
}
