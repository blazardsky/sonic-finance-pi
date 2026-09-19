import * as React from "react"
import {
  RiAddLine,
  RiCloseLine,
  RiContractLeftRightLine,
  RiExpandLeftRightLine,
} from "@remixicon/react"

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

// The panel's normal width.
const DEFAULT_SIDEBAR_WIDTH = "20rem"

// Expanded, the panel isn't wider *in place* — it becomes the same kind of
// overlay this component already uses on mobile: its own backdrop, its own
// z-index above the list and everything else on the page, its own scroll
// region, laid overtop rather than beside or instead of the list. A
// household asking for "more room" wants the form to cover more of the
// screen, not shove the list out of the way or fight it for space in a
// flex row. Not full-width like mobile's takeover — there's a table worth
// leaving a sliver of visible underneath — but comfortably past Field's own
// "responsive" orientation breakpoint (its @md/field-group container query,
// 28rem) so a stacked Item's fields can return to a labelled row.
const EXPANDED_OVERLAY_WIDTH = "32rem"

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
  openMobileWhen,
}: {
  title?: React.ReactNode
  children: React.ReactNode
  // The submit/cancel action bar, kept reachable without scrolling: the
  // panel itself is viewport-height-bounded now (`contained`'s own fixed
  // sizing), so this is just its last flex child, same on every form of the
  // panel (desktop, mobile's Sheet, the expanded overlay) — SidebarContent's
  // own overflow-auto is what scrolls, not the page underneath it. Not
  // inside a caller's own `<form>` element: give the form an `id` and this
  // its submit button's `form={id}` to keep them associated.
  footer?: React.ReactNode
  className?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
  // Starts the mobile Sheet open, one-shot at mount — for a caller that
  // navigates here wanting the form already showing (e.g. a mobile quick-add
  // shortcut). `open`/`onOpenChange` can't do this themselves: they only
  // reach the desktop panel.
  openMobile?: boolean
  // Opens the mobile Sheet whenever this value is non-null (ticket 01) —
  // typically the id of the row being edited. `open` alone cannot do this:
  // pages default it to true for the desktop panel, so setSidebarOpen(true)
  // on "Modifica" is a no-op while the Sheet stays closed.
  openMobileWhen?: number | null
}) {
  // The measured content-edge (viewport px), reported by the `contained`
  // Sidebar below — read by FormSidebarTrigger, which stays genuinely
  // `fixed` even while the panel is closed, so it has to track this edge
  // itself rather than the true viewport edge (ticket 14).
  const [edge, setEdge] = React.useState(0)
  // Desktop-only: mobile's Sheet is already full-width, so there is nothing
  // for this to expand there. Local and unpersisted — it's a working-room
  // toggle for the session, not a layout choice worth remembering across
  // visits the way open/closed already is (its own cookie, above).
  const [expanded, setExpanded] = React.useState(false)

  return (
    <SidebarProvider
      cookieName={FORM_SIDEBAR_COOKIE_NAME}
      className="contents"
      open={open}
      onOpenChange={onOpenChange}
      defaultOpenMobile={openMobile}
      // 20rem, wider than the main nav's 16rem (SIDEBAR_WIDTH) since this
      // panel carries a whole form and the nav just carries labels (ticket
      // 07). Unaffected by `expanded`: that's a separate overlay Sidebar
      // renders on its own terms (`overlay`/`overlayWidth` below), not a
      // wider version of this in-flow width.
      style={{ "--sidebar-width": DEFAULT_SIDEBAR_WIDTH } as React.CSSProperties}
    >
      <OpenMobileWhen value={openMobileWhen} />
      <Sidebar
        side="right"
        collapsible="offcanvas"
        variant="sidebar"
        mobileWidth="100vw"
        themed={false}
        contained
        onContainedRectChange={(r) => setEdge(r.edge)}
        overlay={expanded}
        overlayWidth={EXPANDED_OVERLAY_WIDTH}
      >
        {title && (
          <SidebarHeader className="flex-row items-center justify-between border-b p-4">
            <h2 className="font-heading text-base font-medium">{title}</h2>
            <div className="flex items-center gap-1">
              <FormSidebarExpandToggle
                expanded={expanded}
                onToggle={() => setExpanded((e) => !e)}
              />
              <FormSidebarClose expanded={expanded} />
            </div>
          </SidebarHeader>
        )}
        <SidebarContent className={cn("gap-4 p-4", className)}>
          {children}
        </SidebarContent>
        {footer && <FormSidebarFooter>{footer}</FormSidebarFooter>}
      </Sidebar>
      <FormSidebarTrigger edge={edge} expanded={expanded} />
    </SidebarProvider>
  )
}

// Ticket 01: pages pass the editing id as `openMobileWhen`. Whenever it is a
// real id the Sheet opens, including when swapping from one row to another
// (the dependency is the id itself, not a boolean that would stay true).
function OpenMobileWhen({ value }: { value?: number | null }) {
  const { setOpenMobile } = useSidebar()
  React.useEffect(() => {
    if (value != null) setOpenMobile(true)
  }, [value, setOpenMobile])
  return null
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
// Desktop's width toggle, sat in the header next to the title — mobile's
// Sheet is already full-width, so this has nothing to do there and isn't
// rendered, the same way FormSidebarClose has nothing to do on desktop.
function FormSidebarExpandToggle({
  expanded,
  onToggle,
}: {
  expanded: boolean
  onToggle: () => void
}) {
  const { isMobile } = useSidebar()
  if (isMobile) return null
  const label = expanded ? t.collapseForm : t.expandForm
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-lg"
      aria-label={label}
      title={label}
      onClick={onToggle}
    >
      {expanded ? <RiContractLeftRightLine /> : <RiExpandLeftRightLine />}
    </Button>
  )
}

function FormSidebarClose({ expanded }: { expanded: boolean }) {
  const { isMobile, setOpen, setOpenMobile } = useSidebar()
  if (!isMobile && !expanded) return null
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-lg"
      aria-label={t.hideForm}
      title={t.hideForm}
      onClick={() => (isMobile ? setOpenMobile(false) : setOpen(false))}
    >
      <RiCloseLine />
    </Button>
  )
}

// The submit/cancel bar, kept reachable without scrolling: `contained`'s
// panel is now genuinely `fixed` and height-bounded (viewport, minus the
// header), so this is just its last flex child on every form of the panel —
// desktop, mobile's Sheet, the expanded overlay alike. SidebarContent's own
// `overflow-auto` is what scrolls a form taller than the panel; this bar
// never moves.
function FormSidebarFooter({ children }: { children: React.ReactNode }) {
  return <div className="shrink-0 border-t bg-background p-4">{children}</div>
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
  expanded,
}: {
  className?: string
  edge: number
  expanded: boolean
}) {
  const { isMobile, open, openMobile, toggleSidebar } = useSidebar()
  const isOpen = isMobile ? openMobile : open
  const label = isOpen ? t.hideForm : t.showForm
  const Icon = isOpen ? RiCloseLine : RiAddLine

  // Expanded *and open*, the panel is the same kind of Sheet overlay
  // mobile's `isOpen` check just below already accounts for — FormSidebarClose
  // (rendered inside it) is what closes it, so this has nothing left to do
  // either. `expanded` alone isn't enough: it's a separate, unpersisted
  // "more room" toggle that outlives the panel being closed, so closing an
  // expanded panel (via that same FormSidebarClose) must still leave this as
  // the way back in — otherwise nothing on screen can reopen it at all.
  if (!isMobile && expanded && isOpen) return null

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
