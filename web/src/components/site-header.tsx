import {
  RiAddLine,
  RiComputerLine,
  RiMoonLine,
  RiShoppingBasketLine,
  RiSunLine,
} from "@remixicon/react"

import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"
import { t } from "@/lib/strings"
import type { Screen } from "@/App"

// The header sits on its own dark bg, not the page's — every button in it
// needs its own hover/text colours rather than the defaults meant for a
// light background.
const headerButton =
  "text-header-foreground hover:bg-header-foreground/10 hover:text-header-foreground"

const themeOrder = ["dark", "light", "system"] as const
const themeIcons = {
  dark: RiMoonLine,
  light: RiSunLine,
  system: RiComputerLine,
}

// One button cycling dark → light → system, giving the ThemeProvider's
// existing "d" shortcut a visible control — it had none before this.
function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  const Icon = themeIcons[theme]
  return (
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label={theme}
      className={headerButton}
      onClick={() =>
        setTheme(themeOrder[(themeOrder.indexOf(theme) + 1) % 3])
      }
    >
      <Icon />
    </Button>
  )
}

export function SiteHeader({
  screen,
  onQuickAddExpense,
}: {
  screen: Screen
  onQuickAddExpense: () => void
}) {
  return (
    <header className="flex h-14 shrink-0 items-center gap-2 rounded-sm bg-header px-4 text-header-foreground">
      <SidebarTrigger className={headerButton} />
      <span className="font-heading text-base font-medium">{t[screen]}</span>
      <div className="ml-auto flex items-center gap-1">
        {/* Mobile-only: on desktop, Expenses' own panel is a click away and
            always open by default (Expenses.tsx), so this shortcut would be
            redundant there. Homepage-only: Expenses.tsx only reads its
            quickAdd prop once, at mount — already being on Expenses would
            leave a second tap with nothing to remount and re-read it. */}
        {screen === "dashboard" && (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t.addExpense}
            title={t.addExpense}
            className={cn(headerButton, "md:hidden")}
            onClick={onQuickAddExpense}
          >
            <span className="relative inline-flex">
              <RiShoppingBasketLine />
              <RiAddLine className="absolute -top-1 -right-1 size-2.5 rounded-full bg-header-foreground text-header" />
            </span>
          </Button>
        )}
        <ThemeToggle />
      </div>
    </header>
  )
}
