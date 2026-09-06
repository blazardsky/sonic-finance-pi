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
  footer,
  className,
  open,
  onOpenChange,
  openMobile,
}: {
  title?: React.ReactNode
  children: React.ReactNode
  // The submit/cancel action bar, kept reachable without scrolling: since
  // the panel above is now sized by its own content and scrolls with the
  // page (no longer viewport-clamped), this is rendered outside that flow,
  // in its own always-on-screen strip — `fixed` to the viewport bottom on
  // desktop, an ordinary flex-column tail on mobile's already viewport-sized
  // Sheet. Not inside a caller's own `<form>` element: give the form an
  // `id` and this its submit button's `form={id}` to keep them associated.
  footer?: React.ReactNode
  className?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
  // Starts the mobile Sheet open, one-shot at mount — for a caller that
  // navigates here wanting the form already showing (e.g. a mobile quick-add
  // shortcut). `open`/`onOpenChange` can't do this themselves: they only
  // reach the desktop panel.
  openMobile?: boolean
}) {
  // The measured content-edge and bottom (viewport px), reported by the
  // `contained` Sidebar below — read by FormSidebarTrigger (which stays
  // genuinely `fixed` even while the panel is closed, so it tracks this edge
  // itself rather than the true viewport edge, ticket 14) and by
  // FormSidebarFooter (which eases its own `fixed` bottom inset as this
  // bottom nears the viewport's, ticket 07).
  const [rect, setRect] = React.useState({ edge: 0, bottom: 0, viewportHeight: 0 })

  return (
    <SidebarProvider
      cookieName={FORM_SIDEBAR_COOKIE_NAME}
      className="contents"
      open={open}
      onOpenChange={onOpenChange}
      defaultOpenMobile={openMobile}
      // 20rem, wider than the main nav's 16rem (SIDEBAR_WIDTH) — this panel
      // carries a whole form, the nav just carries labels (ticket 07).
      style={{ "--sidebar-width": "20rem" } as React.CSSProperties}
    >
      <Sidebar
        side="right"
        collapsible="offcanvas"
        variant="sidebar"
        mobileWidth="100vw"
        themed={false}
        contained
        onContainedRectChange={setRect}
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
        {footer && (
          <FormSidebarFooter
            edge={rect.edge}
            panelBottom={rect.bottom}
            viewportHeight={rect.viewportHeight}
          >
            {footer}
          </FormSidebarFooter>
        )}
      </Sidebar>
      <FormSidebarTrigger edge={rect.edge} />
    </SidebarProvider>
  )
}

// How close to the panel's true end (viewport px) the footer starts easing
// off `bottom-0`, and how much of a gap (viewport px) it settles into once
// there — ticket 07's "sits at bottom-0 ... eases to bottom-3 ... within
// 3rem", at the default 16px root: 3rem and 0.75rem.
const FOOTER_EASE_ZONE_PX = 48
const FOOTER_REST_GAP_PX = 12

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

// The submit/cancel bar, kept reachable without scrolling. Mobile's Sheet is
// already exactly viewport height, so an ordinary flex-column tail item
// already sits at the visible bottom — no positioning trick needed there.
// Desktop's panel is `contained` (in flow, scrolling with the rest of the
// page), so this is `fixed` to the viewport bottom instead, docked to the
// same measured edge as FormSidebarTrigger — `position: sticky` would be the
// simpler way to track "the viewport bottom, until the panel's own end",
// but it only holds against an ancestor that actually scrolls, and this
// app's shell doesn't: it's sized with `min-h-svh` (a floor), so a page
// taller than the viewport grows the whole document instead of clipping
// `main` into its own scrollport — the window is what scrolls, so this
// tracks that directly instead.
function FormSidebarFooter({
  children,
  edge,
  panelBottom,
  viewportHeight,
}: {
  children: React.ReactNode
  edge: number
  panelBottom: number
  viewportHeight: number
}) {
  const { isMobile, open } = useSidebar()

  if (isMobile) {
    return <div className="shrink-0 border-t bg-background p-4">{children}</div>
  }
  // Unlike mobile's Sheet, the desktop panel's own content stays mounted
  // (just width-collapsed) while closed — this bar is `fixed`, outside that
  // collapsing box, so it has to hide itself instead of being clipped along
  // with it.
  if (!open) return null

  // How far below the *current* viewport bottom the panel's true end sits —
  // large while there's plenty of panel left to scroll through, shrinking
  // to 0 right as the viewport bottom reaches it. Bottom-0 (flush) until
  // that's within the ease zone, then a linear ease down to the rest gap, so
  // the bar is never pushed past the panel's own end into whatever (if
  // anything) follows it.
  const distanceToEnd = panelBottom - viewportHeight
  const eased = FOOTER_REST_GAP_PX * (1 - distanceToEnd / FOOTER_EASE_ZONE_PX)
  const bottomInset =
    distanceToEnd >= FOOTER_EASE_ZONE_PX
      ? 0
      : Math.min(FOOTER_REST_GAP_PX, Math.max(0, eased))

  return (
    <div
      style={{ right: `${edge}px`, bottom: `${bottomInset}px` }}
      className="fixed z-20 w-(--sidebar-width) border-t bg-background p-4 shadow-sm"
    >
      {children}
    </div>
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
        // top-20 = same inset as the contained panel (shell + header).
        "fixed top-20 z-20 rounded-none border border-r-0 bg-black text-white shadow-sm transition-[right] duration-200 ease-linear hover:bg-black hover:text-white aria-expanded:bg-black aria-expanded:text-white dark:bg-white dark:text-black dark:hover:bg-white dark:hover:text-black dark:aria-expanded:bg-white dark:aria-expanded:text-black",
        className
      )}
    >
      <Icon />
    </Button>
  )
}

export { FormSidebar, FormSidebarTrigger }
