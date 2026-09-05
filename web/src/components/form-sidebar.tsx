import * as React from "react"
import { RiAddLine, RiCloseLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarProvider,
  useSidebar,
} from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"
import { t } from "@/lib/strings"

// Distinct from the main nav sidebar's "sidebar_state" cookie (ui/sidebar.tsx)
// so opening/closing a page's form panel never touches the nav's remembered
// state.
const FORM_SIDEBAR_COOKIE_NAME = "form_sidebar_state"

// The right-side "add new" panel shared by every form+list page (Spese,
// Entrate, Ricorrenti, Risparmi, Categorie, Clienti, Investimenti). Wraps the
// existing Sidebar primitive rather than a bespoke panel: same offcanvas
// collapse, same auto-Sheet-on-mobile behaviour, just mirrored to the right
// and on its own persistence key. Takes the page's form as children and has
// no opinion on it — a page that wants to keep the panel open and clear its
// form after submit just does that in its own state, since submitting never
// touches this component. A page that wants a custom close/cancel control
// inside its form can import `useSidebar` from `@/components/ui/sidebar`
// itself — the context it reads is this component's. `open`/`onOpenChange`
// are optional and only control the desktop panel (mobile's Sheet stays
// self-managed, reachable via its own always-visible FAB either way) — a page
// that lets a row action (e.g. a "Modifica" row action) load something into
// the form can pass these to make sure a closed desktop panel reopens to show
// it, rather than updating silently behind a closed panel.
function FormSidebar({
  title,
  children,
  className,
  open,
  onOpenChange,
}: {
  title?: React.ReactNode
  children: React.ReactNode
  className?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
}) {
  return (
    <SidebarProvider
      cookieName={FORM_SIDEBAR_COOKIE_NAME}
      className="contents"
      open={open}
      onOpenChange={onOpenChange}
    >
      <Sidebar
        side="right"
        collapsible="offcanvas"
        variant="sidebar"
        mobileWidth="100vw"
        themed={false}
      >
        {title && (
          <SidebarHeader className="border-b p-4">
            <h2 className="font-heading text-base font-medium">{title}</h2>
          </SidebarHeader>
        )}
        <SidebarContent className={cn("gap-4 p-4", className)}>
          {children}
        </SidebarContent>
      </Sidebar>
      <FormSidebarTrigger />
    </SidebarProvider>
  )
}

// Both states are `fixed` to the real viewport edge — not placed in normal
// flow — because this component can be mounted anywhere in a page's layout,
// and the panel itself is always `fixed` to the viewport regardless of where
// its flex-row placeholder happens to sit. An in-flow trigger button ends up
// visually underneath the (higher-stacking, `fixed`) open panel whenever the
// page's content column is narrower than the viewport; anchoring this to the
// viewport too, above the panel's z-index, is what keeps it reachable in both
// the open and closed state. Desktop: a small edge tab, sliding over to track
// the panel's left edge when open. Mobile: a FAB pinned to the bottom, since
// the panel becomes a full-width Sheet there.
function FormSidebarTrigger({ className }: { className?: string }) {
  const { isMobile, open, openMobile, toggleSidebar } = useSidebar()
  const isOpen = isMobile ? openMobile : open
  const label = isOpen ? t.hideForm : t.showForm
  const Icon = isOpen ? RiCloseLine : RiAddLine

  if (isMobile) {
    return (
      <Button
        type="button"
        variant="default"
        size="icon-lg"
        aria-label={label}
        title={label}
        onClick={toggleSidebar}
        className={cn(
          "fixed right-6 bottom-6 z-40 rounded-full shadow-lg",
          className
        )}
      >
        <Icon />
      </Button>
    )
  }

  return (
    <Button
      type="button"
      variant="secondary"
      size="icon"
      aria-label={label}
      title={label}
      onClick={toggleSidebar}
      className={cn(
        "fixed top-1/2 z-20 -translate-y-1/2 rounded-l-lg rounded-r-none border border-r-0 shadow-sm transition-[right] duration-200 ease-linear",
        open ? "right-(--sidebar-width)" : "right-0",
        className
      )}
    >
      <Icon />
    </Button>
  )
}

export { FormSidebar, FormSidebarTrigger }
