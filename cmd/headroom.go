package main

import (
	"database/sql"
	"net/http"
	"time"
)

const headroomPath = "/api/reports/headroom"

// headroomHorizonMonths is how far ahead Planned purchases are placed. Short
// on purpose: something that doesn't fit in six months is not a short-term
// purchase but a matter for Savings or a loan (spec's Implementation
// Decisions).
const headroomHorizonMonths = 6

// headroomMonth is one future month's projected Headroom (CONTEXT.md) and the
// running total through it — a negative month lowers the running total.
type headroomMonth struct {
	Month         string `json:"month"`
	HeadroomCents int64  `json:"headroom_cents"`
	RunningCents  int64  `json:"running_cents"`
	// The two parts that differ month to month, so the screen can show how
	// the Headroom was arrived at: what active Contracts still owe this
	// month, and the Recurring expenses due.
	ContractCents  int64 `json:"contract_cents"`
	RecurringCents int64 `json:"recurring_cents"`
}

// headroomReport is the next headroomHorizonMonths months, starting next
// month: this month's leftover is still being spent, so it is never offered.
// Unavailable, with no months, until there are budgetMinHistoryMonths of
// history — the same rule Budget uses, because the typical figures here are
// the same kind of trailing median.
type headroomReport struct {
	Available  bool            `json:"available"`
	Months     []headroomMonth `json:"months"`
	Placements []placement     `json:"placements"`
	// The same-every-month parts of the breakdown, and how much the last 12
	// months weigh against this year's in the typical figures (percent).
	TypicalIncomeCents    int64 `json:"typical_income_cents"`
	TypicalSpendingCents  int64 `json:"typical_spending_cents"`
	GoalCents             int64 `json:"goal_cents"`
	TrailingWeightPercent int64 `json:"trailing_weight_percent"`
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
// best it could have had, the balance at the end of the horizon.
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
			best := int64(0)
			if len(balance) > 0 {
				best = max(balance[len(balance)-1], 0)
			}
			pl.MissingCents = p.AmountCents - best
			pl.SavingsCover = savingsCents >= pl.MissingCents
		}
		out = append(out, pl)
	}
	return out
}

// computeHeadroom projects each future month as
//
//	typical Income + Contract shares − typical non-recurring spending
//	− Recurring expenses due that month − Goal
//
// Typical blends two medians: the last 12 completed months (since the first
// Expense, Budget's window) and this year's completed months, the 12 months
// weighing (12 − n)/12 where n is how many months of this year are over — all
// of it in January, a twelfth in December. Early in the year the baseline is
// last year; as this year's months are recorded they take over, with no jump
// on any particular day. Income excludes Contract Incomes (the shares stand
// in for those) and Investments sales; spending includes Investments — money
// that leaves for a PAC is money the household won't have, and a portfolio
// is never money to count on (CONTEXT.md, Headroom).
func computeHeadroom(db *sql.DB, now func() time.Time) (headroomReport, error) {
	report := headroomReport{Months: []headroomMonth{}}

	firstMonth, err := firstExpenseMonth(db)
	if err != nil || firstMonth == "" {
		return report, err
	}
	thisYear := now().Format("2006")
	var incomes, spending, yearIncomes, yearSpending []int64
	for _, month := range trailingCompletedMonths(now, budgetTrailingMonths) {
		if month < firstMonth {
			continue
		}
		var in, out int64
		if err := db.QueryRow(`SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
			JOIN category c ON c.id = i.category_id
			WHERE substr(i.payment_date, 1, 7) = ? AND i.contract_id IS NULL AND IFNULL(c.code, '') != ?`,
			month, codeInvestments).Scan(&in); err != nil {
			return report, err
		}
		if err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM expense
			WHERE substr(occurred_on, 1, 7) = ? AND recurring_id IS NULL`, month).Scan(&out); err != nil {
			return report, err
		}
		incomes = append(incomes, in)
		spending = append(spending, out)
		if month[:4] == thisYear {
			yearIncomes = append(yearIncomes, in)
			yearSpending = append(yearSpending, out)
		}
	}
	if len(incomes) < budgetMinHistoryMonths {
		return report, nil
	}
	// n counts calendar months, not months with data: the weight follows the
	// year, and a household that started mid-year still has its 12-month
	// median (whatever months exist) to lean on.
	n := int64(now().Month()) - 1
	report.TrailingWeightPercent = (12 - n) * 100 / 12
	report.TypicalIncomeCents = blendCents(incomes, yearIncomes, n)
	report.TypicalSpendingCents = blendCents(spending, yearSpending, n)
	typical := report.TypicalIncomeCents - report.TypicalSpendingCents

	goalCents, err := getSettingCents(db, goalCentsKey)
	if err != nil {
		return report, err
	}
	report.GoalCents = goalCents
	shares, err := contractShares(db, now)
	if err != nil {
		return report, err
	}

	var running int64
	for _, month := range nextMonths(now, headroomHorizonMonths) {
		var recurring int64
		if err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM recurring_expense
			WHERE start_month <= ? AND (end_month IS NULL OR end_month >= ?)`, month, month).Scan(&recurring); err != nil {
			return report, err
		}
		headroom := typical + shares[month] - recurring - goalCents
		running += headroom
		report.Months = append(report.Months, headroomMonth{
			Month: month, HeadroomCents: headroom, RunningCents: running,
			ContractCents: shares[month], RecurringCents: recurring,
		})
	}
	report.Available = true
	return report, nil
}

// blendCents is the weighted typical figure: the trailing median weighing
// (12 − n)/12, this year's median the rest. With no month of this year over
// yet (n = 0, or no data this year) it is the trailing median alone.
func blendCents(trailing, year []int64, n int64) int64 {
	if n == 0 || len(year) == 0 {
		return medianCents(trailing)
	}
	return ((12-n)*medianCents(trailing) + n*medianCents(year)) / 12
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
		for _, month := range nextMonths(now, headroomHorizonMonths) {
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
