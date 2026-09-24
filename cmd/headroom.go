package main

import (
	"database/sql"
	"net/http"
	"time"
)

const headroomPath = "/api/reports/headroom"

// headroomHorizonMonths is the most months ahead Planned purchases are placed.
// Short on purpose: something that doesn't fit in six months is not a
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
	GoalCents  int64 `json:"goal_cents"`
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
		// suffixMin[i] is the lowest balance from month i to the end.
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
//	Headroom = goalBuffer(leftover, Goal)
//
// and accumulates from lastMonthHeadroom. Income(M) and spending(M) are
// seasonal: last year's M−1, M and M+1, blended with this year's completed
// months as they are recorded (seasonalCents). Income leaves out Contract
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
	cache := map[string]forecastTotals{}
	collect := func(months []string) (incomes, spending []int64, err error) {
		for _, m := range months {
			if m < firstMonth {
				continue
			}
			t, ok := cache[m]
			if !ok {
				if t, err = readForecastTotals(db, m); err != nil {
					return nil, nil, err
				}
				cache[m] = t
			}
			incomes = append(incomes, t.incomeCents)
			spending = append(spending, t.spendingCents)
		}
		return incomes, spending, nil
	}

	trailingIn, trailingOut, err := collect(trailingCompletedMonths(now, budgetTrailingMonths))
	if err != nil || len(trailingIn) < budgetMinHistoryMonths {
		return report, err
	}
	yearIn, yearOut, err := collect(completedMonthsThisYear(now))
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
	shares, err := contractShares(db, now)
	if err != nil {
		return report, err
	}
	report.StartCents, err = lastMonthHeadroom(db, now, goalCents)
	if err != nil {
		return report, err
	}

	running := report.StartCents
	for _, month := range horizonMonths(now) {
		seasonIn, seasonOut, err := collect(yearAgoWindow(month))
		if err != nil {
			return report, err
		}
		income := seasonalCents(seasonIn, yearIn, trailingIn, n)
		spending := seasonalCents(seasonOut, yearOut, trailingOut, n)
		var recurring int64
		if err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM recurring_expense
			WHERE start_month <= ? AND (end_month IS NULL OR end_month >= ?)`, month, month).Scan(&recurring); err != nil {
			return report, err
		}
		headroom := goalBuffer(income+shares[month]-spending-recurring, goalCents)
		running += headroom
		report.Months = append(report.Months, headroomMonth{
			Month: month, HeadroomCents: headroom, RunningCents: running,
			ForecastIncomeCents: income, ForecastSpendingCents: spending,
			ContractCents: shares[month], RecurringCents: recurring,
		})
	}
	report.Available = true
	return report, nil
}

// forecastTotals is what one past month contributes to a forecast: its
// non-Contract Income (investment sales left out) and its non-recurring
// spending (Investments kept).
type forecastTotals struct {
	incomeCents, spendingCents int64
}

func readForecastTotals(db *sql.DB, month string) (forecastTotals, error) {
	var t forecastTotals
	if err := db.QueryRow(`SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
		JOIN category c ON c.id = i.category_id
		WHERE substr(i.payment_date, 1, 7) = ? AND i.contract_id IS NULL AND IFNULL(c.code, '') != ?`,
		month, codeInvestments).Scan(&t.incomeCents); err != nil {
		return t, err
	}
	err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM expense
		WHERE substr(occurred_on, 1, 7) = ? AND recurring_id IS NULL`, month).Scan(&t.spendingCents)
	return t, err
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

// lastMonthHeadroom is the last completed month's actual leftover through the
// Goal buffer: every Income received in it — Contract Incomes included, they
// arrived — except investment sales, minus every Expense, Recurring and
// Investments alike. The month's Recurring expenses are generated first, so a
// rent no screen happened to read yet still counts.
func lastMonthHeadroom(db *sql.DB, now func() time.Time, goalCents int64) (int64, error) {
	month := trailingCompletedMonths(now, 1)[0]
	if err := materialise(db, now, month); err != nil {
		return 0, err
	}
	var in, out int64
	if err := db.QueryRow(`SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
		JOIN category c ON c.id = i.category_id
		WHERE substr(i.payment_date, 1, 7) = ? AND IFNULL(c.code, '') != ?`,
		month, codeInvestments).Scan(&in); err != nil {
		return 0, err
	}
	if err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM expense
		WHERE substr(occurred_on, 1, 7) = ?`, month).Scan(&out); err != nil {
		return 0, err
	}
	return goalBuffer(in-out, goalCents), nil
}

// contractShares is what each active Contract still owes, spread evenly over
// the months it has left from next month on, summed per month. Cash basis:
// what is still owed is the total minus what has actually been received. A
// Contract that ends this month or earlier has no future month to spread
// over and contributes nothing.
func contractShares(db *sql.DB, now func() time.Time) (map[string]int64, error) {
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
		for _, month := range horizonMonths(now) {
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
