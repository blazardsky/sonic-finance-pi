import { useCallback, useEffect, useState } from "react"

import { Categories } from "@/pages/Categories"
import { Clients } from "@/pages/Clients"
import { Dashboard } from "@/pages/Dashboard"
import { Expenses } from "@/pages/Expenses"
import { Holdings } from "@/pages/Holdings"
import { Incomes } from "@/pages/Incomes"
import { Login } from "@/pages/Login"
import { Month } from "@/pages/Month"
import { RecurringExpenses } from "@/pages/RecurringExpenses"
import { Savings } from "@/pages/Savings"
import { Settings } from "@/pages/Settings"
import { Year } from "@/pages/Year"
import { YearlyReport } from "@/pages/YearlyReport"
import { AppSidebar } from "@/components/app-sidebar"
import { SiteHeader } from "@/components/site-header"
import { SidebarProvider } from "@/components/ui/sidebar"
import { Toast } from "@/components/toast"
import { t } from "@/lib/strings"

// /api/health doubles as the session check: it sits behind auth like every
// other /api/ route, so a 401 is how the app learns it needs to show the login
// screen. Until it answers, neither the shell nor the login screen is the
// right thing to render — hence "checking", rather than a flash of the wrong
// one on every cold load.
type State = "checking" | "loggedOut" | "connected" | "unreachable"

// The screens. Their sidebar grouping and order lives in app-sidebar.tsx,
// which reads this same type — the keys are the strings-file keys too, so
// the nav labels itself.
export type Screen =
  | "dashboard"
  | "month"
  | "year"
  | "yearReport"
  | "expenses"
  | "incomes"
  | "recurring"
  | "savings"
  | "categories"
  | "clients"
  | "holdings"
  | "settings"

export function App() {
  const [state, setState] = useState<State>("checking")
  const [screen, setScreen] = useState<Screen>("dashboard")
  // Set by the mobile-only "Aggiungi spesa" header shortcut alongside the
  // screen change, so Expenses knows to open its panel on arrival. Cleared as
  // soon as the user leaves Expenses, so a later plain nav back in via the
  // menu doesn't reopen it.
  const [expensesQuickAdd, setExpensesQuickAdd] = useState(false)
  // Adjusted during render (React's own pattern for this) rather than an
  // effect, since an effect setting state right back would cost an extra
  // render for no visible frame in between.
  const [quickAddScreen, setQuickAddScreen] = useState(screen)
  if (screen !== quickAddScreen) {
    setQuickAddScreen(screen)
    if (screen !== "expenses" && expensesQuickAdd) setExpensesQuickAdd(false)
  }
  // The Pi has no RTC. When its clock is unset it generates no Recurring
  // expenses, and a month short a rent with nothing said about it is how a
  // household concludes the app has lost the rent. Assumed fine until health
  // says otherwise, so an older server that does not send the field warns
  // about nothing.
  const [clockOK, setClockOK] = useState(true)

  const check = useCallback(() => {
    fetch("/api/health")
      .then(async (r) => {
        if (r.status === 401) return setState("loggedOut")
        if (r.ok) {
          const health = await r.json()
          setClockOK(health.clock_ok !== false)
        }
        setState(r.ok ? "connected" : "unreachable")
      })
      .catch(() => setState("unreachable"))
  }, [])

  useEffect(() => check(), [check])

  if (state === "loggedOut") return <Login onLoggedIn={check} />

  if (state === "checking") {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <p className="font-mono text-xs text-muted-foreground">
          {t.connecting}
        </p>
      </div>
    )
  }

  if (state === "unreachable") {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <p className="font-mono text-xs text-muted-foreground">
          {t.serverUnreachable}
        </p>
      </div>
    )
  }

  // Header on top, full width; the sidebar and the main content sit in a row
  // below it. Both the frame around everything and the header are the app's
  // blue, distinct from the sidebar and main content's light background.
  return (
    <SidebarProvider>
      <Toast />
      <div className="flex min-h-svh w-full flex-col gap-2 bg-shell p-2 md:gap-3 md:p-3">
        <SiteHeader
          screen={screen}
          onQuickAddExpense={() => {
            setScreen("expenses")
            setExpensesQuickAdd(true)
          }}
        />
        <div className="flex min-h-0 flex-1 gap-2 md:gap-3">
          <AppSidebar screen={screen} onNavigate={setScreen} />
          <main className="min-h-0 flex-1 overflow-auto rounded-sm bg-background shadow-sm">
            {!clockOK && (
              <div className="mx-auto w-full max-w-md px-6 pt-6">
                <p role="alert" className="text-sm text-destructive">
                  {t.clockUnset}
                </p>
              </div>
            )}
            {screen === "dashboard" && <Dashboard />}
            {screen === "month" && <Month />}
            {screen === "year" && <Year />}
            {screen === "yearReport" && <YearlyReport />}
            {screen === "expenses" && <Expenses quickAdd={expensesQuickAdd} />}
            {screen === "incomes" && <Incomes />}
            {screen === "recurring" && <RecurringExpenses />}
            {screen === "savings" && <Savings />}
            {screen === "categories" && <Categories />}
            {screen === "clients" && <Clients />}
            {screen === "holdings" && <Holdings />}
            {screen === "settings" && <Settings />}
          </main>
        </div>
      </div>
    </SidebarProvider>
  )
}

export default App
