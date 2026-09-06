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
  openMobile,
}: {
  title?: React.ReactNode
  children: React.ReactNode
  className?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
  // Starts the mobile Sheet open, one-shot at mount — for a caller that
  // navigates here wanting the form already showing (e.g. a mobile quick-add
  // shortcut). `open`/`onOpenChange` can't do this themselves: they only
  // reach the desktop panel.
  openMobile?: boolean
}) {
  // The measured content-edge (viewport px), reported by the `contained`
  // Sidebar below — shared with FormSidebarTrigger so its own `fixed`
  // position tracks the same edge rather than the true viewport edge
  // (ticket 14).
  const [edge, setEdge] = React.useState(0)

  return (
    <SidebarProvider
      cookieName={FORM_SIDEBAR_COOKIE_NAME}
      className="contents"
      open={open}
      onOpenChange={onOpenChange}
      defaultOpenMobile={openMobile}
    >
      <Sidebar
        side="right"
        collapsible="offcanvas"
        variant="sidebar"
        mobileWidth="100vw"
        themed={false}
        contained
        onContainedEdgeChange={setEdge}
      >
        {title && (
          <SidebarHeader className="flex-row items-center justify-between border-b p-4">
            <h2 className="font-heading text-base font-medium">{title}</h2>
            <FormSidebarClose />
          </SidebarHeader>
        )}
        <SidebarContent className={cn("gap-4 p-4", className)}>
          {children}
        </SidebarContent>
      </Sidebar>
      <FormSidebarTrigger edge={edge} />
    </SidebarProvider>
  )
}

// Mobile's dedicated close control, sat in the header next to the title
// rather than reusing the outside FAB as a toggle. Radix's Dialog treats any
// tap outside its Content as a dismiss and closes it before that tap's own
// click handler runs — the FAB, living outside the Sheet, raced its own
// onClick against that auto-dismiss and the two cancelled out, so the
// "close" tap visibly did nothing. Living inside the Content (the header is
// part of it) sidesteps that race entirely. Desktop has no equivalent: its
// panel is never full-width, so FormSidebarTrigger's edge tab is always
// reachable and stays the only control there.
function FormSidebarClose() {
  const { isMobile, setOpenMobile } = useSidebar()
  if (!isMobile) return null
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-lg"
      aria-label={t.hideForm}
      title={t.hideForm}
      onClick={() => setOpenMobile(false)}
    >
      <RiCloseLine />
    </Button>
  )
}

// Both states are `fixed` to the real viewport edge — not placed in normal
// flow — because this component can be mounted anywhere in a page's layout,
// and the panel itself is always `fixed` regardless of where its flex-row
// placeholder happens to sit. An in-flow trigger ends up visually underneath
// the (higher-stacking, `fixed`) open panel; anchoring this to the same
// measured edge, above the panel's z-index, is what keeps it reachable in
// both the open and closed state. Desktop: a small edge tab, sliding over to
// track the panel's left edge when open. Mobile: a FAB pinned to the bottom,
// since the panel becomes a full-width Sheet there.
function FormSidebarTrigger({
  className,
  edge,
}: {
  className?: string
  edge: number
}) {
  const { isMobile, open, openMobile, toggleSidebar } = useSidebar()
  const isOpen = isMobile ? openMobile : open
  const label = isOpen ? t.hideForm : t.showForm
  const Icon = isOpen ? RiCloseLine : RiAddLine

  if (isMobile) {
    // Open-only here — FormSidebarClose (rendered inside the Sheet) is what
    // closes it once open, so this FAB has nothing left to do while open and
    // simply isn't rendered.
    if (isOpen) return null
    return (
      <Button
        type="button"
        variant="default"
        size="icon-lg"
        aria-label={label}
        title={label}
        onClick={toggleSidebar}
        className={cn(
          // size-16 (64px) overrides icon-lg's own size-9 — the biggest tap
          // target on the page, since it's the primary way in to logging an
          // entry.
          "fixed right-6 bottom-6 z-40 size-16 shadow-lg",
          className
        )}
      >
        <RiAddLine className="size-6" />
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
      style={{ right: open ? `calc(${edge}px + var(--sidebar-width))` : `${edge}px` }}
      className={cn(
        // A translate-based top-1/2 centering (Tailwind v4 compiles that to
        // the standalone CSS `translate` property, not `transform`) left
        // Chromium's hit-test region stale on this `fixed`, dynamically
        // re-positioned button — real clicks landed ~16px off from the
        // painted position. calc() avoids `translate` entirely.
        "fixed top-[calc(50%-1rem)] z-20 rounded-l-lg rounded-r-none border border-r-0 shadow-sm transition-[right] duration-200 ease-linear",
        className
      )}
    >
      <Icon />
    </Button>
  )
}

export { FormSidebar, FormSidebarTrigger }
