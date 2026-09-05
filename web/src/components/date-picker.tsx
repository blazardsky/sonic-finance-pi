import * as React from "react"
import { format, parseISO } from "date-fns"
import { it } from "date-fns/locale"
import { RiCalendarLine } from "@remixicon/react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { t } from "@/lib/strings"

// Standard shadcn Popover+Calendar recipe, shared so every form field that
// wants a date picks it up the same way instead of hand-rolling the popover.
// value/onValueChange speak the API's plain YYYY-MM-DD, matching the native
// date inputs this replaces.
function DatePicker({
  id,
  value,
  onValueChange,
  placeholder = t.chooseDate,
  disabled,
  className,
}: {
  id?: string
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
  className?: string
}) {
  const [open, setOpen] = React.useState(false)
  const selected = value ? parseISO(value) : undefined

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          disabled={disabled}
          className={cn(
            "h-10 justify-start font-normal",
            !value && "text-muted-foreground",
            className
          )}
        >
          <RiCalendarLine className="size-4" />
          {selected ? format(selected, "d MMMM yyyy", { locale: it }) : placeholder}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0">
        <Calendar
          mode="single"
          locale={it}
          selected={selected}
          onSelect={(date) => {
            if (!date) return
            onValueChange(format(date, "yyyy-MM-dd"))
            setOpen(false)
          }}
        />
      </PopoverContent>
    </Popover>
  )
}

export { DatePicker }
