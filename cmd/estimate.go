package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"
)

// estimateReport is the Dashboard's one projected figure (ADR-0010: the
// app's first and only non-cash-basis number). Income, expense and tax are
// each projected independently by scaling last year's full-year total by how
// this year's pace so far compares to last year's at the same point.
//
// Each *_cents is nil rather than zero when its own ratio is undefined —
// taxSummary.NetPercent's own "null, not a wrong zero" rule — so the
// Dashboard can simply skip that one bar. Available is false only when every
// metric is: a household with nothing recorded before this year has no prior
// year to project any of them from (ADR-0010's named consequence), which
// this makes true of all three at once without a separate check for it.
type estimateReport struct {
	Year                 string `json:"year"`
	Available            bool   `json:"available"`
	IncomeEstimateCents  *int64 `json:"income_estimate_cents"`
	ExpenseEstimateCents *int64 `json:"expense_estimate_cents"`
	TaxEstimateCents     *int64 `json:"tax_estimate_cents"`
}

// handleEstimateReport answers the projection for one calendar year, using
// the clock to decide how much of it — and of the year before it — counts as
// "so far". A year that is not a year is refused, exactly as /year and /tax
// refuse one.
func handleEstimateReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year := r.PathValue("year")
		if err := validYear("year", year); err != nil {
			writeInvalid(w, err)
			return
		}
		yearNumber, err := strconv.Atoi(year)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		lastYear := strconv.Itoa(yearNumber - 1)

		// Materialise every month either year's queries can touch, up front:
		// a Recurring tax payment or rent has to exist before any of the
		// sums below can see it, and readExpenseCentsExcludingInvestments's
		// own per-month materialise does not cover the full-year tax query.
		for _, month := range append(monthsOf(year), monthsOf(lastYear)...) {
			if err := materialise(db, now, month); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}

		// The YTD cutoff: however many whole months of `year` are already
		// behind the clock, and the same number of months — Jan onward — of
		// `lastYear`, which is what "at the same point" means.
		completed := estimateCompletedMonths(now, year)
		thisYearMonths := monthsOf(year)[:len(completed)]
		lastYearMonths := monthsOf(lastYear)[:len(completed)]

		thisIncome, err := sumMonths(db, now, thisYearMonths, readIncomeCentsExcludingInvestments)
		lastIncomeYTD, err2 := sumMonths(db, now, lastYearMonths, readIncomeCentsExcludingInvestments)
		lastIncomeFull, err3 := sumMonths(db, now, monthsOf(lastYear), readIncomeCentsExcludingInvestments)
		if err = firstErr(err, err2, err3); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		thisExpense, err := sumMonths(db, now, thisYearMonths, readExpenseCentsExcludingInvestments)
		lastExpenseYTD, err2 := sumMonths(db, now, lastYearMonths, readExpenseCentsExcludingInvestments)
		lastExpenseFull, err3 := sumMonths(db, now, monthsOf(lastYear), readExpenseCentsExcludingInvestments)
		if err = firstErr(err, err2, err3); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		thisTax, err := readTaxPaidCentsThroughMonths(db, year, thisYearMonths)
		lastTaxYTD, err2 := readTaxPaidCentsThroughMonths(db, lastYear, lastYearMonths)
		lastTaxFull, err3 := readTaxPaidCentsThroughMonths(db, lastYear, nil)
		if err = firstErr(err, err2, err3); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		report := estimateReport{
			Year:                 year,
			IncomeEstimateCents:  estimateRatio(lastIncomeFull, thisIncome, lastIncomeYTD),
			ExpenseEstimateCents: estimateRatio(lastExpenseFull, thisExpense, lastExpenseYTD),
			TaxEstimateCents:     estimateRatio(lastTaxFull, thisTax, lastTaxYTD),
		}
		report.Available = report.IncomeEstimateCents != nil ||
			report.ExpenseEstimateCents != nil || report.TaxEstimateCents != nil
		writeJSON(w, http.StatusOK, report)
	}
}

// estimateCompletedMonths is the Estimate's YTD cutoff: every month of the
// given year that is wholly behind the clock's current month. A year already
// over answers with all twelve; a year not yet begun answers with none —
// months compare correctly as the "YYYY-MM" text they already are.
func estimateCompletedMonths(now func() time.Time, year string) []string {
	cur := now().Format(monthLayout)
	var completed []string
	for _, m := range monthsOf(year) {
		if m < cur {
			completed = append(completed, m)
		}
	}
	return completed
}

// sumMonths folds one of the Investments-excluding readers over a list of
// months — the shape both readExpenseCentsExcludingInvestments and its
// income-side twin below share, so the Estimate's six sums (income/expense x
// this-year-YTD/last-year-YTD/last-year-full) are six calls to this rather
// than six copies of the same loop.
func sumMonths(db *sql.DB, now func() time.Time, months []string,
	read func(*sql.DB, func() time.Time, string) (int64, error)) (int64, error) {
	var total int64
	for _, month := range months {
		cents, err := read(db, now, month)
		if err != nil {
			return 0, err
		}
		total += cents
	}
	return total, nil
}

// readIncomeCentsExcludingInvestments is readMonthRow's Income query with
// the Investments category left out — the Estimate's income-side twin of
// readExpenseCentsExcludingInvestments (ADR-0009): a stock sale is not
// "normal" income and would distort the pace the Estimate projects from.
//
// now is unused — Income has no Recurring machinery (materialise only ever
// generates Expenses), so there is nothing to generate first — but it is
// kept in the signature so this and its Expense-side twin share one type,
// which is what lets sumMonths fold over either.
func readIncomeCentsExcludingInvestments(db *sql.DB, now func() time.Time, month string) (int64, error) {
	var cents int64
	err := db.QueryRow(`SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
		JOIN category c ON c.id = i.category_id
		WHERE i.payment_date IS NOT NULL AND substr(i.payment_date, 1, 7) = ?
		AND IFNULL(c.code, '') != ?`,
		month, codeInvestments).Scan(&cents)
	return cents, err
}

// readTaxPaidCentsThroughMonths is taxSummary's TaxPaidCents query —
// attributed by Tax year rather than by occurred_on (ADR-0008), so
// Investments exclusion does not apply here (nothing tax-categorised is ever
// Investments) — restricted to payments that have actually happened by the
// last of the given months. months == nil means the whole year: exactly what
// taxSummary itself reports, which is what the Estimate's last-year
// full-year total needs. An empty (non-nil) slice is the "no completed
// months yet" case and short-circuits to zero rather than building a cutoff
// no row can be before.
func readTaxPaidCentsThroughMonths(db *sql.DB, year string, months []string) (int64, error) {
	if months != nil && len(months) == 0 {
		return 0, nil
	}
	yearNumber, err := strconv.Atoi(year)
	if err != nil {
		return 0, err
	}
	query := `SELECT COALESCE(SUM(e.amount_cents), 0) FROM expense e
		JOIN category c ON c.id = e.category_id AND c.code = ?
		WHERE COALESCE(e.tax_year, CAST(substr(e.occurred_on, 1, 4) AS INTEGER)) = ?`
	args := []any{codeTaxes, yearNumber}
	if months != nil {
		query += ` AND substr(e.occurred_on, 1, 7) <= ?`
		args = append(args, months[len(months)-1])
	}
	var cents int64
	err = db.QueryRow(query, args...).Scan(&cents)
	return cents, err
}

// estimateRatio is the shape behind all three of the Estimate's figures:
// last year's full-year total, scaled by how this year's pace so far
// compares to last year's at the same point. Nil when last year's
// YTD-at-the-same-point is zero — an undefined ratio, not a very large or
// very small one.
//
// ponytail: plain int64 multiply-then-divide, no float. A household's
// figures would have to run past roughly 90 million euros for
// lastYearFull*thisYearYTD to overflow int64 — widen to a float64
// intermediate if that ever stops being a safe assumption.
func estimateRatio(lastYearFull, thisYearYTD, lastYearYTD int64) *int64 {
	if lastYearYTD == 0 {
		return nil
	}
	estimate := lastYearFull * thisYearYTD / lastYearYTD
	return &estimate
}

// firstErr answers the first non-nil of a handful of errors collected from
// sibling reads that all have to succeed — the same "run them all, report
// whichever broke" shape as any other handler here, without repeating the
// three-deep if-err chain three times over for the Estimate's six reads.
func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
