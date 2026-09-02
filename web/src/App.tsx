import { useEffect, useState } from 'react'
import { t } from './strings'

// Placeholder shell. It calls /api/health so that a broken embed or a broken
// dev proxy shows up immediately rather than at the end of the next ticket.
export default function App() {
  const [status, setStatus] = useState<string>(t.connecting)

  useEffect(() => {
    fetch('/api/health')
      .then((r) => (r.ok ? setStatus(t.connected) : setStatus(t.serverUnreachable)))
      .catch(() => setStatus(t.serverUnreachable))
  }, [])

  return (
    <main className="p-8 font-sans">
      <h1 className="text-2xl font-semibold">{t.appName}</h1>
      <p className="text-sm text-gray-500">{status}</p>
    </main>
  )
}
