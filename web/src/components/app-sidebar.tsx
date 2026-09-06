import {
  RiBarChart2Line,
  RiCalendarLine,
  RiDashboardLine,
  RiExchangeFundsLine,
  RiFileChartLine,
  RiFileList3Line,
  RiPieChartLine,
  RiPriceTag3Line,
  RiRepeatLine,
  RiSettings3Line,
  RiShoppingBag3Line,
  RiTeamLine,
  RiWallet3Line,
} from "@remixicon/react"

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { t } from "@/lib/strings"
import type { Screen } from "@/App"

const icons: Record<Screen, typeof RiDashboardLine> = {
  dashboard: RiDashboardLine,
  month: RiCalendarLine,
  year: RiBarChart2Line,
  yearReport: RiFileChartLine,
  expenses: RiShoppingBag3Line,
  incomes: RiWallet3Line,
  recurring: RiRepeatLine,
  investments: RiExchangeFundsLine,
  savings: RiPieChartLine,
  categories: RiPriceTag3Line,
  clients: RiTeamLine,
  holdings: RiFileList3Line,
  settings: RiSettings3Line,
}

// Three groups, the same altitude the screens are already ordered at: where
// things stand, what gets typed, what gets managed. Risparmi is a read-only
// recap (no entries besides the starting balance), so it sits with Overview
// rather than Entries.
const groups: { label: string; screens: Screen[] }[] = [
  {
    label: t.navOverview,
    screens: ["dashboard", "month", "year", "savings"],
  },
  {
    label: t.navEntries,
    screens: ["expenses", "incomes", "recurring", "investments"],
  },
  {
    label: t.navManagement,
    screens: ["categories", "clients", "holdings", "settings"],
  },
]

function Nav({
  screen,
  onNavigate,
}: {
  screen: Screen
  onNavigate: (screen: Screen) => void
}) {
  return (
    <div className="flex flex-1 flex-col gap-5 overflow-auto p-3">
      {groups.map((group) => (
        <SidebarGroup key={group.label} className="p-0">
          <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu className="gap-1.5">
              {group.screens.map((key) => {
                const Icon = icons[key]
                return (
                  <SidebarMenuItem key={key}>
                    <SidebarMenuButton
                      size="lg"
                      isActive={key === screen}
                      onClick={() => onNavigate(key)}
                    >
                      <Icon />
                      <span>{t[key]}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      ))}
    </div>
  )
}

// A flush footer strip, not a floating chip: full width, no margin, no
// pointer cursor — it names the app, it isn't a control.
function Brand() {
  return (
    <div className="cursor-default rounded-b-sm bg-brand px-3 py-2.5 font-heading text-sm text-brand-foreground">
      <span className="font-bold uppercase">Sonic</span> Finance
    </div>
  )
}

// A plain flex column on desktop, a Sheet drawer on mobile — the shadcn
// Sidebar component's own two branches, without its third: fixed desktop
// positioning is built for a full-bleed page, and fights the padded, framed
// layout the shell (App.tsx) uses here. SidebarProvider still supplies the
// shared mobile/open state both branches read from, including desktop's own
// open/closed toggle (SiteHeader's trigger), which this reads directly since
// there is no <Sidebar> here to read it for us.
export function AppSidebar({
  screen,
  onNavigate,
}: {
  screen: Screen
  onNavigate: (screen: Screen) => void
}) {
  const { isMobile, open, openMobile, setOpenMobile } = useSidebar()

  if (isMobile) {
    return (
      <Sheet open={openMobile} onOpenChange={setOpenMobile}>
        <SheetContent
          side="left"
          className="flex w-72 flex-col bg-sidebar p-0 text-sidebar-foreground"
        >
          <SheetHeader className="sr-only">
            <SheetTitle>{t.appName}</SheetTitle>
            <SheetDescription>{t.appName}</SheetDescription>
          </SheetHeader>
          <Nav
            screen={screen}
            onNavigate={(s) => {
              onNavigate(s)
              setOpenMobile(false)
            }}
          />
          <Brand />
        </SheetContent>
      </Sheet>
    )
  }

  if (!open) return null

  return (
    <div className="flex w-64 shrink-0 flex-col rounded-sm bg-sidebar text-sidebar-foreground shadow-sm">
      <Nav screen={screen} onNavigate={onNavigate} />
      <Brand />
    </div>
  )
}
