import { useEffect, useState } from "react"

// Delays a fast-changing value by ms — a typeahead fetch reads this instead
// of the raw keystroke, so it fires once typing pauses rather than once per
// character.
export function useDebouncedValue<T>(value: T, ms: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), ms)
    return () => clearTimeout(timer)
  }, [value, ms])
  return debounced
}
