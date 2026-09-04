package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// monthCategoryTotal is categoryTotal (cmd/reports.go) with a month column —
// the same trick readDailyBreakdown adds a day column for, one grain up: the
// bucket a grouped query keys on, rather than the substring a per-month query
// filters by.
type monthCategoryTotal struct {
	Month       string `json:"month"`
	CategoryID  int64  `json:"category_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

// fullYearReport is ADR-0011's report: the one screen allowed to run what the
// comment on readBreakdown calls the heaviest query in the app, because it is
// opened on purpose rather than folded into Year.tsx's default load, and
// because ByMonth is computed as a single (month, category)-grouped query —
// the same cost as one month's breakdown, not twelve of them.
//
// Everything Year.tsx/handleYearReport (cmd/reports.go) already answers — the
// year's three numbers and its twelve months, for the cumulative chart — is
// not repeated here; the frontend reads both endpoints together.
type fullYearReport struct {
	Year string `json:"year"`

	ByMonth []monthCategoryTotal `json:"by_month"`

	// The year's Expense with Taxes left out: what was actually spent
	// living, separate from what was paid the state.
	ExpenseExcludingTaxCents int64 `json:"expense_excluding_tax_cents"`

	// Savings (computeSavingsCents, cmd/savings.go) as it stood the moment
	// this year began — a baseline for how the year changed the household's
	// position, not the Savings page's own all-time figure.
	SavingsAtStartCents int64 `json:"savings_at_start_cents"`

	// Median Expense, Income and Net over this year's own completed months.
	// CONTEXT.md's Budget entry names the distinction explicitly: "typical in
	// 2026" must not shift depending on what year it is read from, so this is
	// not Budget's trailing-12-months median (cmd/budget.go).
	MedianExpenseCents int64 `json:"median_expense_cents"`
	MedianIncomeCents  int64 `json:"median_income_cents"`
	MedianNetCents     int64 `json:"median_net_cents"`
}

// handleFullYearReport answers the whole of ticket 07 in one request: the
// grouped breakdown, the tax-excluded total, the historical Savings cutoff,
// and the year-scoped medians. It materialises every month of the year
// first, for the same ADR-0005 reason handleYearReport does — a month
// nobody has opened yet must not read as empty just because this is the
// first screen to ask about it.
//
// readMonthRow's own comment calls itself "the single path every month read
// goes through" — this is the one other entry point, because two of this
// report's three figures (the grouped breakdown, the tax-excluded total) are
// plain SQL with no per-month loop to hang materialise off of. Only
// yearScopedMedians still reads through readMonthRow, and re-materialises
// months this loop already covered — redundant, not wrong, and left alone
// rather than threading a "skip materialise" flag through it for one caller.
func handleFullYearReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year := r.PathValue("year")
		if err := validYear("year", year); err != nil {
			writeInvalid(w, err)
			return
		}
		for _, month := range monthsOf(year) {
			if err := materialise(db, now, month); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}

		report := fullYearReport{Year: year}
		var err error

		if report.ByMonth, err = readMonthCategoryBreakdown(db, year); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if report.ExpenseExcludingTaxCents, err = readYearExpenseExcludingTax(db, year); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		yearNumber, err := strconv.Atoi(year)
		if err != nil {
			// validYear has already parsed this as YYYY; it cannot fail here.
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		priorDec31 := fmt.Sprintf("%d-12-31", yearNumber-1)
		if report.SavingsAtStartCents, _, err = computeSavingsCents(db, priorDec31); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if report.MedianExpenseCents, report.MedianIncomeCents, report.MedianNetCents, err =
			yearScopedMedians(db, now, year); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, report)
	}
}

// readMonthCategoryBreakdown is readBreakdown (cmd/reports.go) with the month
// kept as a column instead of summed away and a year filter instead of a
// month one — the ADR-0011 query, extending readDailyBreakdown's own
// day-column trick to month grain instead of reinventing the attribution
// SQL. Same ADR-0002 attribution (an Item's amount under the Item's own
// Category, the remainder under the Expense's), same zero-drop, same order.
func readMonthCategoryBreakdown(db *sql.DB, year string) ([]monthCategoryTotal, error) {
	rows, err := db.Query(`SELECT share.month, c.id, c.name, SUM(share.amount_cents)
		FROM (
			SELECT substr(e.occurred_on, 1, 7) AS month, e.category_id, e.amount_cents - COALESCE(
				(SELECT SUM(i.amount_cents) FROM item i WHERE i.expense_id = e.id), 0
			) AS amount_cents
			FROM expense e WHERE substr(e.occurred_on, 1, 4) = ?
			UNION ALL
			SELECT substr(e.occurred_on, 1, 7), i.category_id, i.amount_cents FROM item i
			JOIN expense e ON e.id = i.expense_id
			WHERE substr(e.occurred_on, 1, 4) = ?
		) share
		JOIN category c ON c.id = share.category_id
		GROUP BY share.month, c.id, c.name
		HAVING SUM(share.amount_cents) > 0
		ORDER BY share.month, SUM(share.amount_cents) DESC, c.name COLLATE NOCASE`, year, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// [] rather than null: the screen maps over it, same as every other list
	// report here.
	out := []monthCategoryTotal{}
	for rows.Next() {
		var m monthCategoryTotal
		if err := rows.Scan(&m.Month, &m.CategoryID, &m.Category, &m.AmountCents); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// readYearExpenseExcludingTax is the year's Expense total with the Taxes
// Category left out — what was actually spent living, separate from what was
// paid the state (spec's "total for the year excluding the Taxes category").
func readYearExpenseExcludingTax(db *sql.DB, year string) (int64, error) {
	var cents int64
	err := db.QueryRow(`SELECT COALESCE(SUM(e.amount_cents), 0) FROM expense e
		JOIN category c ON c.id = e.category_id
		WHERE substr(e.occurred_on, 1, 4) = ? AND IFNULL(c.code, '') != ?`,
		year, codeTaxes).Scan(&cents)
	return cents, err
}

// yearScopedMedians is this report's own median — Expense, Income and Net
// over the reported year's completed months, deliberately not Budget's
// trailing-12-months window (cmd/budget.go, CONTEXT.md's Budget entry): a
// fully completed past year medians over all twelve of its own months, not
// whatever twelve are trailing from today. Reuses medianCents rather than a
// second copy of the same handful of lines — it is already a plain "median
// of these ints" primitive with nothing Budget-specific in it.
//
// A year with no completed months yet (the current year before any month of
// it has finished, or a future one) medians over nothing and answers zero,
// the same as any other empty-history report here.
func yearScopedMedians(db *sql.DB, now func() time.Time, year string) (medianExpense, medianIncome, medianNet int64, err error) {
	var expenses, incomes, nets []int64
	for _, month := range completedMonthsOf(year, now) {
		m, err := readMonthRow(db, now, month)
		if err != nil {
			return 0, 0, 0, err
		}
		expenses = append(expenses, m.ExpenseCents)
		incomes = append(incomes, m.IncomeCents)
		nets = append(nets, m.NetCents)
	}
	if len(expenses) == 0 {
		return 0, 0, 0, nil
	}
	return medianCents(expenses), medianCents(incomes), medianCents(nets), nil
}

// completedMonthsOf is year's own months that lie strictly before the
// current one, in order — Budget's "in progress, so not completed" rule
// (cmd/budget.go's trailingCompletedMonths), applied to one calendar year
// instead of a rolling 12 months. A year entirely in the past has all
// twelve; the current year has however many have actually finished; a
// future year has none.
func completedMonthsOf(year string, now func() time.Time) []string {
	cur := now().Format(monthLayout)
	months := make([]string, 0, 12)
	for _, month := range monthsOf(year) {
		if month < cur {
			months = append(months, month)
		}
	}
	return months
}
