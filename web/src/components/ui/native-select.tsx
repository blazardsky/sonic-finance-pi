import * as React from "react"

import { cn } from "@/lib/utils"

// A plain <select>, wearing Input's chrome. Native so that a phone gives its
// own picker wheel, which is the whole point on a screen used one-handed at a
// till — and cheaper than a listbox component that would have to reimplement
// it. The chrome lived inline on two screens before it lived here.
function NativeSelect({ className, ...props }: React.ComponentProps<"select">) {
  return (
    <select
      data-slot="native-select"
      className={cn(
        "h-8 w-full min-w-0 rounded-lg border border-input bg-transparent px-2 text-base transition-colors outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30",
        className
      )}
      {...props}
    />
  )
}

export { NativeSelect }
