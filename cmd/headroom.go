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
}

// headroomReport is the next headroomHorizonMonths months, starting next
// month: this month's leftover is still being spent, so it is never offered.
// Unavailable, with no months, until there are budgetMinHistoryMonths of
// history — the same rule Budget uses, because the typical figures here are
// the same kind of trailing median.
type headroomReport struct {
	Available bool            `json:"available"`
	Months    []headroomMonth `json:"months"`
}

func handleHeadroomReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := computeHeadroom(db, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

// computeHeadroom projects each future month as
//
//	typical Income + Contract shares − typical non-recurring spending
//	− Recurring expenses due that month − Goal
//
// Typical means the median over Budget's window (trailing completed months
// since the first Expense). Income excludes Contract Incomes (the shares
// stand in for those) and Investments sales; spending includes Investments —
// money that leaves for a PAC is money the household won't have, and a
// portfolio is never money to count on (CONTEXT.md, Headroom).
func computeHeadroom(db *sql.DB, now func() time.Time) (headroomReport, error) {
	report := headroomReport{Months: []headroomMonth{}}

	firstMonth, err := firstExpenseMonth(db)
	if err != nil || firstMonth == "" {
		return report, err
	}
	var incomes, spending []int64
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
	}
	if len(incomes) < budgetMinHistoryMonths {
		return report, nil
	}
	typical := medianCents(incomes) - medianCents(spending)

	goalCents, err := getSettingCents(db, goalCentsKey)
	if err != nil {
		return report, err
	}
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
		report.Months = append(report.Months, headroomMonth{Month: month, HeadroomCents: headroom, RunningCents: running})
	}
	report.Available = true
	return report, nil
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
