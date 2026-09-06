import { useEffect, useState } from "react"

import { subscribeToast, type ToastState } from "@/lib/toast"

// One `<Toast />` mounted once at the app root (App.tsx) is the only
// renderer; `toast()` (lib/toast.ts) is how every page reaches it.
export function Toast() {
  const [state, setState] = useState<ToastState>(null)

  useEffect(() => subscribeToast(setState), [])

  useEffect(() => {
    if (!state) return
    const timer = setTimeout(() => setState(null), 2000)
    return () => clearTimeout(timer)
  }, [state])

  if (!state) return null

  return (
    <div
      key={state.id}
      role="status"
      // z-[60]: above the mobile action sidebar's Sheet drawer (SheetOverlay
      // and SheetContent are both z-50, and stacked below this in the DOM),
      // so the toast still reads on top of it once the drawer is open.
      className="fixed bottom-6 left-1/2 z-[60] -translate-x-1/2 animate-in fade-in slide-in-from-bottom-2 rounded-lg bg-foreground px-4 py-2 text-sm text-background shadow-lg"
    >
      {state.message}
    </div>
  )
}
