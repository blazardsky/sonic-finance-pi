// The shapes the API hands out, shared by the screens that read them.

import type { ColorSlot } from "@/lib/palette"

export type Applies = "expense" | "income" | "both"

// The four values Spending intent can hold (spec's Implementation Decisions →
// Schema) — a single enum rather than two flags, so "necessity, but also
// wise" is simply not a representable value. null (on Category, Expense and
// Recurring expense alike) means "nothing recorded", the ordinary case.
export type SpendingIntent =
  | "necessity"
  | "desire"
  | "desire_wise"
  | "desire_bullshit"

export type Category = {
  id: number
  name: string
  applies_to: Applies
  hidden: boolean
  // One of the 9 fixed palette slots (web/src/lib/palette.ts), never empty —
  // a shared grouping signal, not a unique identity (ADR-0016).
  color: ColorSlot
  // A Base category is one the tax summary resolves by identity: it can be
  // hidden, never renamed or deleted, and the server enforces that with a 409.
  base: boolean
  // Base narrowed to specifically the Gift category — the one the spoiler
  // blur resolves by identity, so an Expense list can tell it apart from any
  // other Base category without matching on its (renameable) name.
  gift: boolean
  // Base narrowed the same way, to the Freelance category — the one the
  // Income form reads to decide whether Fattura inviata belongs on screen,
  // without matching on its (renameable) name.
  freelance: boolean
  // Base narrowed the same way, to the Investments category — the one the
  // PAC badge reads, without matching on its (renameable) name.
  investments: boolean
  // The Category's default Spending intent (ticket 02) — a pure client-side
  // seed for a new Expense's own value, resolved only at Expense-creation
  // time. null means no default is set; changing this later never touches
  // any Expense already saved under this Category.
  spending_intent: SpendingIntent | null
}

// A second, independent tag an Expense (or a Recurring expense) can carry
// alongside its Category — "Caffè" rather than a child of "Alimentari"
// specifically, since the same Subcategory freely pairs with whichever
// Category an entry actually used it under. Unlike Category, nothing here is
// ever Base-protected: no report resolves a Subcategory by identity.
export type Subcategory = {
  id: number
  name: string
  applies_to: Applies
  hidden: boolean
  // Same palette and rules as Category.color, added by the same migration.
  color: ColorSlot
}

// Who money comes from, as one row rather than three spellings. Not a
// freelance-only idea: "Mum" for a birthday gift is a valid Client. Hidden
// Clients stay out of the Income picker but still resolve for old Incomes.
export type Client = {
  id: number
  name: string
  hidden: boolean
  // The Income Category the Income form's picker prefills once this Client
  // is chosen — a suggestion only, never enforced: the picker stays freely
  // editable. Null when this Client has none set.
  default_category_id: number | null
  // The Payer the Income form's picker prefills alongside the Category, on
  // the same terms — label text from the settings list, "" when unset.
  default_payer: string
  // The sum of this Client's received Incomes (payment_date set, ADR-0003),
  // computed at read time and never stored.
  total_earned_cents: number
}

export type HoldingType = "etf" | "crypto" | "stock" | "bond" | "other"

// A specific stock, ETF, crypto asset, bond or other investment vehicle the
// household buys and sells. Named and typed from this fixed list rather than
// free text, because the portfolio percentage breakdown (ticket 03) groups by
// it exactly (CONTEXT.md).
export type Holding = {
  id: number
  name: string
  type: HoldingType
  // Per-unit price, in cents, the household typed in by hand — never fetched
  // by the app (CONTEXT.md). Null until one is set, in which case value/
  // gain-loss below stay null too.
  current_price_cents: number | null
  // A hand-typed correction added to the quantity summed from linked
  // Expenses/Incomes — a PAC's own generated Expenses carry no quantity, so
  // this is how quantity_owned gets trued up without opening every one of
  // them. 0 on a Holding nobody has corrected.
  quantity_adjustment: number
  // The cost-basis counterpart to quantity_adjustment (average-purchase-
  // price feature): a hand-typed contribution to paid_cents from manually
  // recorded purchase lots, moved together with quantity_adjustment via the
  // manual_lot PATCH/POST field rather than typed directly. 0 on a Holding
  // with no manual lots.
  cost_adjustment_cents: number
  // quantity_adjustment plus quantity bought minus quantity sold from paid
  // Incomes only (ADR-0003), computed at read time and never stored.
  quantity_owned: number
  // cost_adjustment_cents plus what was spent buying this Holding minus
  // what was received selling it (paid Incomes only), computed at read
  // time.
  paid_cents: number
  // paid_cents ÷ quantity_owned, computed at read time — null when
  // quantity_owned is zero or negative (nothing meaningful to average).
  average_price_per_unit_cents: number | null
  // quantity_owned × current_price_cents, rounded to the nearest cent. Null
  // whenever current_price_cents is.
  value_now_cents: number | null
  // value_now_cents − paid_cents. Null on the same terms as value_now_cents.
  gain_loss_cents: number | null
  // gain_loss_cents as a percentage of paid_cents. Null whenever paid_cents
  // is zero — a percentage of nothing spent is undefined, not zero.
  gain_loss_percent: number | null
}

export type Expense = {
  id: number
  occurred_on: string
  amount_cents: number
  category_id: number
  // The four details, always strings and never null: they go straight into
  // inputs. Payer and payment method are the label text the Expense was saved
  // with, not a reference — renaming the list leaves them as they read.
  store: string
  payer: string
  payment_method: string
  note: string
  // The optional partial breakdown: the parts of this Expense that belong
  // under a different Category. Always an array, empty for the €7 coffee.
  // Never the source of amount_cents — see ADR-0002.
  items: Item[]
  // The year this payment's tax relates to, and 0 on every Expense that is not
  // a tax one: tax on 2026's income is paid during 2027, so the yearly summary
  // attributes it by this rather than by occurred_on (ADR-0008). The server
  // derives it — 0 sent for a tax Expense comes back as the payment's year, and
  // anything sent for any other Expense comes back as 0.
  tax_year: number
  // The Holding this Expense bought, or null on every Expense that is not a
  // buy — meaningful only under the Investments base category (ticket 03).
  holding_id: number | null
  // A second, independent tag alongside category_id, or null on most
  // Expenses — see Subcategory.
  subcategory_id: number | null
  // How many units of holding_id this Expense bought, or null. Meaningless
  // without a holding_id alongside it — nothing enforces that pairing.
  quantity: number | null
  // This Expense's own Spending intent (ticket 03), independent of its
  // Category's default — that default only ever pre-fills this field at
  // creation time (see Category.spending_intent); from then on this is the
  // one value anything reads. null means "not classified".
  spending_intent: SpendingIntent | null
}

// A part of an Expense under its own Category. No id: an Item is saved as one
// of a set with its Expense and addressed by nothing, so what the form sends
// back is the whole breakdown.
export type Item = {
  name: string
  amount_cents: number
  category_id: number
  // Ticket 01: how much was bought and the unit it was bought in, both null/""
  // together or neither (server-enforced) — "" is Unit's absence, since it is
  // picked from a fixed list (kg/lt/piece) rather than free text.
  quantity: number | null
  unit: string
  // Purely informational (ADR-0014): never affects amount_cents or
  // price_per_unit.
  discounted: boolean
  // Derived server-side from amount_cents and quantity (ADR-0014), in cents
  // per unit — never sent back on write, and null whenever quantity is.
  price_per_unit: number | null
}

// The two short configured lists the Expense form picks from, plus Target and
// Goal (ticket 04) riding the same settings payload — household-set figures,
// not lists, but read/written the same way.
export type Lists = {
  payers: string[]
  payment_methods: string[]
  target_cents: number
  goal_cents: number
  // Savings' one-time starting balance (ticket 07), for pre-app savings —
  // rides the same payload as Target and Goal.
  savings_starting_balance_cents: number
  // The total the household wants Savings plus its portfolio to reach, on
  // the same terms as the starting balance above.
  net_worth_target_cents: number
  // Spending intent's single on/off switch (ticket 01) — off by default;
  // turning it off hides the feature's UI everywhere without touching any
  // Spending intent already recorded.
  spending_intent_enabled: boolean
}

// Budget (computed), Target and Goal (household-set) — cmd/budget.go's
// GET /api/reports/budget. target_cents/goal_cents are always real numbers,
// independent of whether Budget itself is available (they are just
// settings); budget_cents is 0 and meaningless when available is false.
export type BudgetReport = {
  available: boolean
  budget_cents: number
  target_cents: number
  goal_cents: number
}

// Money owed to or received by the household. It exists from the moment it is
// expected — an invoice sent — and is received only once it has a payment
// date: an empty payment_date is the unpaid state, and unpaid Income counts
// toward no total anywhere (ADR-0003).
//
// One number, the amount that arrived. No gross/net pair and no tax
// percentage: tax is an Expense on the day it is paid (ADR-0004).
export type Income = {
  id: number
  amount_cents: number
  // The "income reason" the screens word differently — the same Category
  // field an Expense carries, referenced so a rename fixes every past Income.
  category_id: number
  // Who it came from, or null: a reimbursement names nobody.
  client_id: number | null
  // The label text the Income was saved with, not a reference — the same
  // bargain an Expense makes.
  payer: string
  // Both dates are "" when absent rather than null: they go straight into
  // date inputs, and "" is how one is cleared on the way back.
  payment_date: string
  invoice_sent_date: string
  note: string
  // The Holding this Income sold, or null on every Income that is not a sell
  // — meaningful only under the Investments base category (ticket 03).
  holding_id: number | null
  // The Contract this Income counts toward, or null for "Extra" — real
  // income from this Client, just outside any agreed total (ticket 05).
  contract_id: number | null
  // Whether this Income still carries its 2€ "marca da bollo elettronica" —
  // true by default for a Freelance invoice over the bollo threshold. The 2€
  // stays inside amount_cents (it did arrive), but a linked Contract's
  // received/accounted figures exclude it since it isn't real revenue.
  bollo_fattura: boolean
  // Optional invoice number on a Freelance Income (ticket 05) — free text,
  // often per-person (Nicco/Sofi). "" when unset.
  invoice_number: string
  // Portion of amount_cents that is "extra" (reimbursement, late fee, …)
  // when the Income is linked to a Contract (ticket 05). Counts toward what
  // was collected (amount_cents includes it) but is subtracted from the
  // Contract's received/accounted figures so it does not eat into the
  // agreed total. 0 when unset or when unlinked.
  extra_cents: number
  // How many units of holding_id this Income sold, or null — the sell-side
  // twin of Expense.quantity.
  quantity: number | null
}

// A total the household expects from a Client over a date range (ticket 05).
// A Client can have more than one over time; two of a Client's Contracts
// never overlap in range (checked at write time). Everything past
// total_cents is computed at read time from the range and whatever Incomes
// ended up linked to it via Income.contract_id, and none of it is stored
// (ADR-0012) — reading the same Contract twice in the same month always
// answers with the same numbers, but reading it next month will not.
export type Contract = {
  id: number
  client_id: number
  start_month: string
  end_month: string
  total_cents: number
  // A straight-line share of time elapsed, capped at total_cents once
  // end_month has passed.
  expected_so_far_cents: number
  // Paid Incomes linked to this Contract (payment_date set, ADR-0003).
  received_cents: number
  // Every Income linked to this Contract regardless of payment status —
  // an invoice already sent counts here even before it is paid, so it is
  // never suggested for invoicing twice.
  accounted_cents: number
  // (total_cents − accounted_cents) ÷ months remaining, recomputed on every
  // read (ADR-0012) — this is not a stored schedule. Once end_month has
  // passed this is the whole remaining shortfall instead, and overdue is
  // true.
  invoice_target_this_month_cents: number
  overdue: boolean
}

// Where a calendar month stands: what came in, what went out, and the
// difference. Money in counts received Incomes only — an unpaid one is money
// that has not arrived and is in none of these numbers (ADR-0003). net_cents
// is negative in a month that spent more than it received.
//
// A year is twelve of these and nothing more — the shape of the year is drawn
// from the three numbers, and no Category breakdown comes with them.
export type MonthRow = {
  month: string
  income_cents: number
  expense_cents: number
  net_cents: number
}

// A month with what the money went on: what the month screen reads.
export type MonthTotals = MonthRow & {
  // Where the money went, biggest share first, always summing to
  // expense_cents. An Item's amount is under the Item's Category and the
  // remainder under the Expense's own, so nothing is ever "uncategorised"
  // (ADR-0002).
  by_category: CategoryTotal[]
}

// One Category's share of a month's spend. The name comes with the id, so the
// month screen has no reason to hold the Category list — including for a
// Category since hidden, which still has to say what it was called.
export type CategoryTotal = {
  category_id: number
  category: string
  amount_cents: number
}

// One Holding's share of the portfolio (ticket 07): what was put in net of
// what was taken out, as a percentage of the total across every other
// Holding still standing. A Holding fully sold off nets to zero and is
// simply not one of these rows, never shown at 0% (CONTEXT.md).
export type HoldingBreakdown = {
  holding_id: number
  name: string
  type: HoldingType
  net_cents: number
  percent: number
  // Same read-time figures as Holding's own (computeHoldingFigures): null on
  // the same terms — no current_price_cents typed in means nothing to value
  // at, and gain_loss_percent is also null when net_cents is zero.
  value_now_cents: number | null
  gain_loss_cents: number | null
  gain_loss_percent: number | null
}

// The Savings page's one request (ticket 07) — cmd/savings.go's
// GET /api/reports/savings: the computed, ledger-free Savings figure
// (cumulative received Income minus Expense, excluding Investments, plus
// the one-time starting balance this already folds in), that starting
// balance on its own so the page can show and edit it separately, and the
// portfolio breakdown.
export type SavingsReport = {
  savings_cents: number
  starting_balance_cents: number
  holdings: HoldingBreakdown[]
  combined_cents: number
  // The household-set net worth target, 0 when it has never set one, and the
  // pace combined_cents is closing on it at: twelve times the median month's
  // saving over the trailing completed months, or twelve times Goal when
  // there is not yet enough history to median (cmd/savings.go). Can be zero
  // or negative — a household spending everything it earns is not on its way
  // anywhere, and the page has to say so rather than divide by it.
  net_worth_target_cents: number
  yearly_savings_cents: number
}

// A CategoryTotal with the day it belongs to — the shape both trend charts
// read: GET /api/reports/daily (a rolling 30-day window) and GET
// /api/reports/month/{month}/daily (one calendar month). Same Item-inclusive
// attribution as the month breakdown (ADR-0002), and Investments are not
// excluded (ADR-0009) — a stock buy is a day like any other Category's.
export type DailyCategoryTotal = {
  day: string
  category_id: number
  category: string
  amount_cents: number
}

// The shape of a whole year: the same three numbers a month answers with, over
// twelve months, plus the twelve months themselves, plus extra_income_cents —
// received Income that is not work. months is always twelve long, January
// first, whether or not anything happened in any of them — a year view is a
// shape, and a missing month would be a gap in it.
export type YearTotals = {
  year: string
  income_cents: number
  expense_cents: number
  net_cents: number
  // Received Income that is not work (neither Freelance nor Stipendio).
  extra_income_cents: number
  // The Investments slice of expense_cents — still ordinary spend, just
  // named separately (same relationship tax_paid_cents has to it).
  investments_cents: number
  months: MonthRow[]
}

// One (month, category) line of the full yearly report's breakdown
// (ticket 07) — categoryTotal (above) with the month it belongs to, from
// GET /api/reports/year/{year}/full's single grouped query (ADR-0011).
export type MonthCategoryTotal = {
  month: string
  category_id: number
  category: string
  amount_cents: number
}

// One month's Spending intent split (ticket 05) — already grouped
// server-side into the four buckets the year-review line chart (and ticket
// 06's diagram, summed across the year) derive their shares from. A month
// with nothing classified at all has no entry here, not a row of zeros —
// spec story 21's "flat/empty" rather than a misleading zero.
export type SpendingIntentMonthTotal = {
  month: string
  necessity_cents: number
  desire_cents: number
  wise_cents: number
  bullshit_cents: number
}

// The full yearly report (ticket 07): everything Year.tsx's own
// /api/reports/year/{year} does not already answer. The page reads both
// endpoints together — this one for the Category grid and the figures below,
// the other for the twelve MonthRows the cumulative chart is built from.
export type FullYearReport = {
  year: string
  by_month: MonthCategoryTotal[]
  // Ticket 05/06's own field, behind spending_intent_enabled on the page.
  spending_intent_by_month: SpendingIntentMonthTotal[]
  // The year's Expense with Taxes left out — what was actually spent living.
  expense_excluding_tax_cents: number
  // Savings (SavingsReport.savings_cents' own formula) as it stood on
  // December 31st of the prior year, not the Savings page's all-time figure.
  savings_at_start_cents: number
  // Scoped to this year's own completed months — distinct from Budget's
  // trailing 12-month median (CONTEXT.md's Budget entry).
  median_expense_cents: number
  median_income_cents: number
  median_net_cents: number
}

// The year in tax terms: the actual figures, as against the invoicing
// software's forecasts. received_cents is freelance Income only and counts
// received money alone (ADR-0003); tax_paid_cents is the tax Category
// attributed by Tax year rather than by payment date, because tax on one
// year's income is paid during the next (ADR-0008). One total, no per-tax-type
// breakdown.
//
// net_percent is null rather than 0 when nothing was received: a percentage of
// nothing is not zero percent, and the screen has to say nothing instead.
export type TaxSummary = {
  year: string
  received_cents: number
  tax_paid_cents: number
  net_cents: number
  net_percent: number | null
}

// The Dashboard's one projected figure (ADR-0010: the app's first and only
// non-cash-basis number) — cmd/estimate.go's GET /api/reports/estimate/{year}.
// Each *_estimate_cents is last year's full-year total scaled by how this
// year's pace so far (whole completed months) compares to last year's at the
// same point, excluding Investments on the income/expense side (ADR-0009);
// tax is attributed by Tax year (ADR-0008) instead.
//
// Null rather than 0 when its own ratio is undefined — last year had no
// YTD-at-the-same-point to compare against — the same convention
// TaxSummary.net_percent already uses, so the Dashboard skips that one bar
// rather than draw it against a fabricated number. available is false only
// when every metric is: there is no prior year to project any of them from.
export type EstimateReport = {
  year: string
  available: boolean
  income_estimate_cents: number | null
  expense_estimate_cents: number | null
  tax_estimate_cents: number | null
}

// One thing the household typed, in either direction, as the home screen's
// list reads it. Not month-scoped and ordered by when it was typed rather than
// by the date on it: the list exists to confirm an entry landed, including one
// backdated to a month the screen is not showing.
//
// date is the day the money moved — occurred_on for an Expense, the payment
// date for an Income — and "" for an Income that has not been paid, because
// money that has not arrived has no day it arrived on. An empty date is
// therefore the unpaid state, and the screen must show it as one: this list
// sits under a money-in total that excludes it (ADR-0003).
//
// payer is whose money an Expense left ("" on an Income). client is who an
// Income came from ("" on an Expense, and on an Income that names nobody).
export type RecentEntry = {
  direction: "expense" | "income"
  id: number
  date: string
  amount_cents: number
  category: string
  payer: string
  client: string
  // True for an Expense resolving to the Gift category, always false on an
  // Income — the spoiler blur (spec's Gift section) is Expense-only.
  is_gift: boolean
}

// The rent, defined once. It carries an Expense's template fields — every one
// of them is copied onto the Expenses it produces — plus the day of the month
// it lands on and the window it covers.
//
// There is no active flag: end_month is the whole of it (ADR-0005). Empty
// means still running, and a month is covered when start_month <= it <=
// end_month. Both are YYYY-MM and zero-padded, so that is a string comparison.
//
// Changing an amount is not an edit: the old definition is ended and a new one
// started, so the months before the change keep the amount that was true then.
export type Recurring = {
  id: number
  amount_cents: number
  category_id: number
  store: string
  payer: string
  payment_method: string
  note: string
  day_of_month: number
  start_month: string
  end_month: string
  // The Holding a recurring investment (PAC) buys, or null on every other
  // Recurring expense — a genuine template property, unlike tax_year, so
  // materialise copies it onto each generated Expense (ticket 06).
  holding_id: number | null
  // A second, independent tag alongside category_id, copied onto each
  // generated Expense the same way holding_id is — see Subcategory.
  subcategory_id: number | null
  // Ticket 04: this Recurring expense's own Spending intent, copied onto
  // each generated Expense the same way category_id is.
  spending_intent: SpendingIntent | null
}

// A labeled on/off toggle for a manual action the app doesn't automate — a
// bank transfer, a payment made by hand (ticket 06). set_for_month never
// crosses the API: enabled already comes back collapsed to false once the
// server's clock has moved past the month it was last turned on for.
export type Reminder = {
  id: number
  label: string
  enabled: boolean
}

// The two lists a freelancer used to get from reading a spreadsheet: money
// genuinely owed, and invoices that look forgotten. Both are views over
// Incomes and never records of their own — the word is "pending payment",
// never "receivable".
export type Pending = {
  outstanding: OutstandingIncome[]
  not_yet_invoiced: NotYetInvoicedClient[]
  // One entry per Client with a currently active Contract still owing
  // something this month — empty when none is active.
  contracts_due_this_month: ClientContractDue[]
}

export type ClientContractDue = {
  client_id: number
  client: string
  // This month's own straight-line share of what the Contract still owes
  // (ADR-0012), not the whole remaining total.
  due_cents: number
}

// One Income still waiting for its money, oldest first. days_waiting counts
// calendar days from waiting_since — the invoice date, or the day the Income
// was typed when no invoice was sent — and never counts backwards.
//
// client and category are names rather than ids: this list is read on a screen
// that holds neither list, and a Client since hidden still has to say what it
// is called. client is "" for an Income that names nobody.
export type OutstandingIncome = {
  id: number
  amount_cents: number
  client: string
  category: string
  waiting_since: string
  days_waiting: number
  // "" when no invoice was sent — waiting_since then falls back to the day
  // the Income was typed, and every reader of this list shows the entry
  // regardless.
  invoice_sent_date: string
}

// One Client billed in the previous three calendar months with no freelance
// Income recorded this month. Freelance Income on both sides, so a gift from a
// relative neither starts a nudge nor clears one.
//
// The id is what the list keys on: two Clients can share a name.
export type NotYetInvoicedClient = {
  client_id: number
  client: string
}

// One Store's figures for one (name, category) group in the Tracker
// (cmd/tracker.go): last price is the most recently occurred Expense's, plus
// the min/max ever seen there. Cents, but not always whole — an Item bought
// with a quantity is priced per unit (amount_cents ÷ quantity), so this can
// carry a fraction of a cent.
export type TrackerStorePrice = {
  store: string
  last_price: number
  last_occurred_on: string
  min_price: number
  max_price: number
}

// One calendar year's average price for a Tracker group, across every Store
// folded together — unlike TrackerStorePrice, which is scoped to one.
export type TrackerYearlyPrice = {
  year: number
  average_price: number
}

// One (normalized name, Category) group GET /api/tracker answers with: how
// much the same Item has cost over time and where it was cheaper. name is
// already normalized (trimmed/lowercased) server-side for this endpoint only
// — never re-normalize it here.
export type TrackerItem = {
  name: string
  category_id: number
  stores: TrackerStorePrice[]
  yearly_average: TrackerYearlyPrice[]
}

// A Planned purchase (CONTEXT.md): something the household intends to buy
// soon. The list comes back in priority order; position is that order.
export type PlannedPurchase = {
  id: number
  label: string
  amount_cents: number
  category_id: number | null
  position: number
}

// What "Comprato" hands the Expense form: the Planned purchase to delete once
// the Expense is saved, and what to prefill it with.
export type ExpensePrefill = {
  plannedId: number
  label: string
  amount_cents: number
  category_id: number | null
}

// Headroom (CONTEXT.md): one future month's projected leftover once its
// Expenses and Goal are set aside, and the running total through it. The next
// 6 months from next month; unavailable (no months) with under 3 months of
// history.
export type HeadroomMonth = {
  month: string
  headroom_cents: number
  running_cents: number
  // What the Headroom was built from: the forecast Income and spending, what
  // active Contracts still owe this month, and the Recurring expenses due.
  forecast_income_cents: number
  forecast_spending_cents: number
  contract_cents: number
  recurring_cents: number
}

// Where one Planned purchase lands: month is "" when it fits in none of the
// 6 months — then missing_cents is the gap and savings_cover says whether
// current Savings cover it.
export type Placement = {
  planned_id: number
  month: string
  missing_cents: number
  savings_cover: boolean
}

export type HeadroomReport = {
  available: boolean
  months: HeadroomMonth[]
  placements: Placement[]
  // Last month's actual leftover through the Goal buffer: the running
  // totals start from it.
  start_cents: number
  goal_cents: number
}
