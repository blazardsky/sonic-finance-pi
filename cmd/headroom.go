package main

import (
	"database/sql"
	"net/http"
	"time"
)

const headroomPath = "/api/reports/headroom"

// headroomHorizonMonths is the most months ahead Planned purchases are placed.
// Short on purpose: something that doesn't fit within the horizon is not a
// short-term purchase but a matter for Savings or a loan (spec's
// Implementation Decisions).
const headroomHorizonMonths = 6

// horizonMonths is the rest of the current year, soonest first: the forecast
// doesn't reach into a year it knows nothing about. At most
// headroomHorizonMonths, and at least one, so December still forecasts
// January.
func horizonMonths(now func() time.Time) []string {
	n := min(max(12-int(now().Month()), 1), headroomHorizonMonths)
	return nextMonths(now, n)
}

// goalBuffer turns a month's leftover into its Headroom (CONTEXT.md). Goal is
// a buffer, not a debt: a leftover above Goal keeps the difference; one below
// Goal, or short by up to Goal, is zero — the money meant for Goal covers
// the gap; only a shortfall beyond Goal is negative.
func goalBuffer(leftover, goal int64) int64 {
	switch {
	case leftover >= goal:
		return leftover - goal
	case leftover >= -goal:
		return 0
	default:
		return leftover + goal
	}
}

// headroomMonth is one future month's projected Headroom (CONTEXT.md) and the
// running total through it — a negative month lowers the running total.
type headroomMonth struct {
	Month         string `json:"month"`
	HeadroomCents int64  `json:"headroom_cents"`
	RunningCents  int64  `json:"running_cents"`
	// What the Headroom was built from, so every figure can be checked by
	// hand: the forecast Income and spending, what active Contracts still owe
	// this month, and the Recurring expenses due.
	ForecastIncomeCents   int64 `json:"forecast_income_cents"`
	ForecastSpendingCents int64 `json:"forecast_spending_cents"`
	ContractCents         int64 `json:"contract_cents"`
	RecurringCents        int64 `json:"recurring_cents"`
	// The Tax reserve (CONTEXT.md) and what it was taken on: the month's
	// Freelance Incomes without a Contract, forecast like Income. Both 0 with
	// nobody self-employed.
	TaxReserveCents        int64 `json:"tax_reserve_cents"`
	FreelanceForecastCents int64 `json:"freelance_forecast_cents"`
}

// headroomReport is the rest of the year (horizonMonths), starting next
// month: this month's leftover is still being spent, so it is never offered.
// Unavailable, with no months, until there are budgetMinHistoryMonths of
// history — the same rule Budget uses.
type headroomReport struct {
	Available  bool            `json:"available"`
	Months     []headroomMonth `json:"months"`
	Placements []placement     `json:"placements"`
	// StartCents is the last completed month's actual leftover, through the
	// Goal buffer — what the running total starts from: real money, not a
	// statistic.
	StartCents int64 `json:"start_cents"`
	// StartMonth is which month StartCents is, so the screen can name it.
	StartMonth string `json:"start_month"`
	GoalCents  int64  `json:"goal_cents"`
	// The Tax reserve taken on last month's freelance Income, the rate used
	// (as a percentage) and where it came from: "history" (completed Tax
	// years) or "fallback" (the household's own percentage). 0, 0 and "" with
	// nobody self-employed.
	StartTaxReserveCents int64   `json:"start_tax_reserve_cents"`
	TaxReservePercent    float64 `json:"tax_reserve_percent"`
	TaxReserveSource     string  `json:"tax_reserve_source"`
}

// placement is where one Planned purchase lands: Month is the first month it
// fits in, or "" when it fits in none — then MissingCents says how far short
// the horizon falls and SavingsCover whether current Savings cover that gap
// (the household's Savings-or-a-loan fork).
type placement struct {
	PlannedID    int64  `json:"planned_id"`
	Month        string `json:"month"`
	MissingCents int64  `json:"missing_cents"`
	SavingsCover bool   `json:"savings_cover"`
}

func handleHeadroomReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := computeHeadroom(db, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		report.Placements = []placement{}
		if report.Available {
			list, err := readPlannedPurchases(db)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			savingsCents, _, err := computeSavingsCents(db, "")
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			report.Placements = place(report.Months, list, savingsCents)
		}
		writeJSON(w, http.StatusOK, report)
	}
}

// place walks the list in priority order carrying the running Headroom. A
// purchase lands in the first month from which the balance stays at or above
// its amount for the rest of the horizon — a later negative month must not
// push the account below zero after buying — and its amount leaves the
// balance from that month on. One that fits nowhere consumes nothing, so it
// never blocks the smaller ones after it; its gap is measured against the
// best it could have had, the balance at the end of the horizon — a
// negative balance widening the gap rather than counting as zero.
func place(months []headroomMonth, list []plannedPurchase, savingsCents int64) []placement {
	balance := make([]int64, len(months))
	for i, m := range months {
		balance[i] = m.RunningCents
	}
	out := make([]placement, 0, len(list))
	for _, p := range list {
		pl := placement{PlannedID: p.ID}
		// Walking back from the end, lowest is the lowest balance from month i
		// on; the earliest i where it still covers the amount is where it lands.
		at := -1
		lowest := int64(0)
		for i := len(balance) - 1; i >= 0; i-- {
			if i == len(balance)-1 || balance[i] < lowest {
				lowest = balance[i]
			}
			if lowest >= p.AmountCents {
				at = i
			}
		}
		if at >= 0 {
			pl.Month = months[at].Month
			for i := at; i < len(balance); i++ {
				balance[i] -= p.AmountCents
			}
		} else {
			// Not floored at zero: a negative accumulated Headroom is a
			// deficit the purchase has to make up too, so it adds to the gap.
			var best int64
			if len(balance) > 0 {
				best = balance[len(balance)-1]
			}
			pl.MissingCents = p.AmountCents - best
			pl.SavingsCover = savingsCents >= pl.MissingCents
		}
		out = append(out, pl)
	}
	return out
}

// computeHeadroom forecasts each horizon month M as
//
//	leftover = Income(M) + Contract share(M) − spending(M) − Recurring due(M)
//	           − Tax reserve(M)
//	Headroom = goalBuffer(leftover, Goal)
//
// and accumulates from last month's actual leftover, taken the same way.
// The Tax reserve applies only with someone self-employed (cmd/taxreserve.go).
// Income(M) and spending(M) are seasonal: last year's M−1, M and M+1, blended
// with this year's completed months as they are recorded (seasonalCents). Income leaves out Contract
// Incomes (the shares stand in for those) and investment sales; spending
// leaves out Recurring-generated Expenses (the Recurring due stand in for
// those) and keeps Investments — money that leaves for a PAC is money the
// household won't have, and a portfolio is never money to count on
// (CONTEXT.md, Headroom).
func computeHeadroom(db *sql.DB, now func() time.Time) (headroomReport, error) {
	report := headroomReport{Months: []headroomMonth{}}

	firstMonth, err := firstExpenseMonth(db)
	if err != nil || firstMonth == "" {
		return report, err
	}
	// Each month's totals are read once, whichever windows it falls in. A
	// month before the first Expense is not a real zero and is left out, the
	// same rule Budget uses.
	cache := map[string]moneyTotals{}
	collect := func(months []string) ([]moneyTotals, error) {
		var out []moneyTotals
		for _, m := range months {
			if m < firstMonth {
				continue
			}
			t, ok := cache[m]
			if !ok {
				var err error
				if t, err = monthMoney(db, m, true); err != nil {
					return nil, err
				}
				cache[m] = t
			}
			out = append(out, t)
		}
		return out, nil
	}

	trailing, err := collect(trailingCompletedMonths(now, budgetTrailingMonths))
	if err != nil || len(trailing) < budgetMinHistoryMonths {
		return report, err
	}
	year, err := collect(completedMonthsThisYear(now))
	if err != nil {
		return report, err
	}
	// n counts calendar months, not months with data: the weight follows the
	// year.
	n := int64(now().Month()) - 1

	goalCents, err := getSettingCents(db, goalCentsKey)
	if err != nil {
		return report, err
	}
	report.GoalCents = goalCents

	// The Tax reserve applies only with someone self-employed; switched off,
	// every figure is what it was before the reserve existed.
	w, err := readWorkers(db)
	if err != nil {
		return report, err
	}
	reserve := w.SelfEmployed
	var rate int64
	if reserve {
		if rate, report.TaxReserveSource, err = taxReserveRate(db, now, firstMonth, w.TaxReserveFallbackPercent); err != nil {
			return report, err
		}
		report.TaxReservePercent = float64(rate) / 100
		// A 0% rate reserves nothing, so taking tax payments out of spending
		// would make them vanish from the forecast: treat it as no reserve.
		if rate == 0 {
			reserve = false
			report.TaxReserveSource = ""
		}
	}
	income := func(t moneyTotals) int64 { return t.incomeCents }
	freelance := func(t moneyTotals) int64 { return t.freelanceCents }
	// While the reserve applies, tax payments leave spending: the reserve
	// already sets that money aside month by month, and counting both would
	// count the tax twice.
	spending := func(t moneyTotals) int64 {
		if reserve {
			return t.spendingCents - t.taxCents
		}
		return t.spendingCents
	}
	forecast := func(season []moneyTotals, f func(moneyTotals) int64) int64 {
		return seasonalCents(pick(season, f), pick(year, f), pick(trailing, f), n)
	}

	horizon := horizonMonths(now)
	shares, err := contractShares(db, now, horizon)
	if err != nil {
		return report, err
	}
	// Last month's actual leftover, its Recurring expenses generated first so
	// a rent no screen happened to read yet still counts.
	report.StartMonth = trailingCompletedMonths(now, 1)[0]
	if err := materialise(db, now, report.StartMonth); err != nil {
		return report, err
	}
	last, err := monthMoney(db, report.StartMonth, false)
	if err != nil {
		return report, err
	}
	if reserve {
		report.StartTaxReserveCents = reserveCents(last.freelanceCents, rate)
	}
	report.StartCents = goalBuffer(last.incomeCents-spending(last)-report.StartTaxReserveCents, goalCents)

	running := report.StartCents
	for _, month := range horizon {
		season, err := collect(yearAgoWindow(month))
		if err != nil {
			return report, err
		}
		m := headroomMonth{
			Month:                 month,
			ForecastIncomeCents:   forecast(season, income),
			ForecastSpendingCents: forecast(season, spending),
			ContractCents:         shares[month],
		}
		if m.RecurringCents, err = recurringDue(db, month, reserve); err != nil {
			return report, err
		}
		if reserve {
			m.FreelanceForecastCents = forecast(season, freelance)
			m.TaxReserveCents = reserveCents(m.ContractCents+m.FreelanceForecastCents, rate)
		}
		m.HeadroomCents = goalBuffer(m.ForecastIncomeCents+m.ContractCents-m.ForecastSpendingCents-m.RecurringCents-m.TaxReserveCents, goalCents)
		running += m.HeadroomCents
		m.RunningCents = running
		report.Months = append(report.Months, m)
	}
	report.Available = true
	return report, nil
}

// pick is one figure of each month's totals.
func pick(ts []moneyTotals, f func(moneyTotals) int64) []int64 {
	out := make([]int64, len(ts))
	for i, t := range ts {
		out[i] = f(t)
	}
	return out
}

// recurringDue is what the Recurring expenses running in month add up to.
// While the Tax reserve applies, a Recurring expense in the Taxes category —
// an instalment plan — is left out: the reserve already stands in for it.
func recurringDue(db *sql.DB, month string, withoutTaxes bool) (int64, error) {
	var cents int64
	err := db.QueryRow(`SELECT COALESCE(SUM(r.amount_cents), 0) FROM recurring_expense r
		JOIN category c ON c.id = r.category_id
		WHERE r.start_month <= ? AND (r.end_month IS NULL OR r.end_month >= ?)
		AND NOT (? AND IFNULL(c.code, '') = ?)`,
		month, month, withoutTaxes, codeTaxes).Scan(&cents)
	return cents, err
}

// seasonalCents is one forecast figure: last year's season (season) weighing
// (12 − n)/12, this year's completed months (year) the rest — last year
// leads early in the year, this year's own record takes over as it grows.
// With no season on record, this year alone; with no month of this year
// over (January), the season alone; with neither — a history that started
// last autumn, seen in January — the plain trailing median.
func seasonalCents(season, year, trailing []int64, n int64) int64 {
	switch {
	case len(season) == 0 && len(year) == 0:
		return medianCents(trailing)
	case len(season) == 0:
		return medianCents(year)
	case n == 0 || len(year) == 0:
		return medianCents(season)
	}
	return ((12-n)*medianCents(season) + n*medianCents(year)) / 12
}

// yearAgoWindow is last year's month before, the month itself, and the month
// after: October's window is last September, October and November.
func yearAgoWindow(month string) []string {
	m, _ := time.Parse(monthLayout, month)
	return []string{
		m.AddDate(-1, -1, 0).Format(monthLayout),
		m.AddDate(-1, 0, 0).Format(monthLayout),
		m.AddDate(-1, 1, 0).Format(monthLayout),
	}
}

// completedMonthsThisYear is January through last month of the current year;
// none in January.
func completedMonthsThisYear(now func() time.Time) []string {
	return trailingCompletedMonths(now, int(now().Month())-1)
}

// moneyTotals is one past month's Income and spending, as monthMoney reads
// them, with the two parts the Tax reserve needs singled out: the freelance
// Income within incomeCents, and the tax payments within spendingCents.
type moneyTotals struct {
	incomeCents, spendingCents int64
	freelanceCents, taxCents   int64
}

// monthMoney is one month's received Income (investment sales never count)
// and its Expenses (Investments always do). For a forecast it also leaves out
// Contract Incomes and Recurring-generated Expenses, which the forecast adds
// back month by month as Contract shares and Recurring due; for last month's
// actual leftover they count, because that money really moved.
func monthMoney(db *sql.DB, month string, forecast bool) (moneyTotals, error) {
	incomeQuery := `SELECT COALESCE(SUM(i.amount_cents), 0),
		COALESCE(SUM(CASE WHEN c.code = ? THEN i.amount_cents END), 0)
		FROM income i JOIN category c ON c.id = i.category_id
		WHERE substr(i.payment_date, 1, 7) = ? AND IFNULL(c.code, '') != ?`
	expenseQuery := `SELECT COALESCE(SUM(e.amount_cents), 0),
		COALESCE(SUM(CASE WHEN c.code = ? THEN e.amount_cents END), 0)
		FROM expense e JOIN category c ON c.id = e.category_id
		WHERE substr(e.occurred_on, 1, 7) = ?`
	if forecast {
		incomeQuery += ` AND i.contract_id IS NULL`
		expenseQuery += ` AND e.recurring_id IS NULL`
	}
	var t moneyTotals
	if err := db.QueryRow(incomeQuery, codeFreelance, month, codeInvestments).Scan(&t.incomeCents, &t.freelanceCents); err != nil {
		return t, err
	}
	err := db.QueryRow(expenseQuery, codeTaxes, month).Scan(&t.spendingCents, &t.taxCents)
	return t, err
}

// contractShares is what each active Contract still owes, spread evenly over
// the months it has left from next month on, summed per month. Cash basis:
// what is still owed is the total minus what has actually been received. A
// Contract that ends this month or earlier has no future month to spread
// over and contributes nothing.
func contractShares(db *sql.DB, now func() time.Time, horizon []string) (map[string]int64, error) {
	next := nextMonths(now, 1)[0]
	rows, err := db.Query(contractSelect+` WHERE contract.end_month >= ?`, next)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := map[string]int64{}
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		owed := c.TotalCents - c.ReceivedCents
		if owed <= 0 {
			continue
		}
		from := max(next, c.StartMonth)
		left, err := monthsInclusive(from, c.EndMonth)
		if err != nil {
			return nil, err
		}
		for _, month := range horizon {
			if month >= from && month <= c.EndMonth {
				shares[month] += owed / int64(left)
			}
		}
	}
	return shares, rows.Err()
}

// nextMonths is the n calendar months after the current one, soonest first —
// trailingCompletedMonths's mirror.
func nextMonths(now func() time.Time, n int) []string {
	cur, _ := time.Parse(monthLayout, now().Format(monthLayout))
	months := make([]string, n)
	for i := range months {
		months[i] = cur.AddDate(0, i+1, 0).Format(monthLayout)
	}
	return months
}
