// The imperative half of the toast: call `toast("message")` from anywhere.
// Split from components/toast.tsx (the `<Toast />` renderer) because a file
// that exports both a component and a plain function breaks Vite's fast
// refresh.
export type ToastState = { id: number; message: string } | null

let listener: ((state: ToastState) => void) | null = null
let nextId = 0

export function toast(message: string) {
  listener?.({ id: ++nextId, message })
}

export function subscribeToast(fn: (state: ToastState) => void) {
  listener = fn
  return () => {
    listener = null
  }
}
