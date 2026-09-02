import { useEffect, useState } from "react"

import { t } from "@/strings"

// Placeholder shell. It calls /api/health so that a broken embed or a broken
// dev proxy shows up immediately rather than at the end of the next ticket.
export function App() {
  const [status, setStatus] = useState<string>(t.connecting)

  useEffect(() => {
    fetch("/api/health")
      .then((r) => setStatus(r.ok ? t.connected : t.serverUnreachable))
      .catch(() => setStatus(t.serverUnreachable))
  }, [])

  return (
    <div className="flex min-h-svh p-6">
      <div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
        <h1 className="font-medium">{t.appName}</h1>
        <p className="font-mono text-xs text-muted-foreground">{status}</p>
      </div>
    </div>
  )
}

export default App
