package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

// monthLayout is how a month crosses the API and how it is stored inside a
// date: dates are TEXT as YYYY-MM-DD, so a month is the first seven characters
// of one and nothing has to be parsed to group by it.
const monthLayout = "2006-01"

// monthTotals answers the question the app exists to answer: where does this
// month stand. Money in is received Income only — an Income with no payment
// date is money that has not arrived, and counts toward nothing (ADR-0003).
//
// The difference is computed here rather than left to the client, because it
// is the number the household actually reads and there is no second opinion to
// be had about it.
type monthTotals struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
	NetCents     int64  `json:"net_cents"`
}

// handleMonthReport reports one calendar month. The month is the whole input:
// no handler here reads the clock, so which month to show is the browser's
// question — the phone in the hand has a correct clock and the Pi has no RTC.
//
// A month that is not a month is refused rather than summed: substr against
// "2026-3" matches nothing, and answering zeros would be indistinguishable
// from a month in which nothing happened.
func handleMonthReport(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		month := r.PathValue("month")
		if _, err := time.Parse(monthLayout, month); err != nil {
			writeInvalid(w, errors.New("a month must be a real month as YYYY-MM"))
			return
		}
		totals, err := readMonth(db, month)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, totals)
	}
}

// readMonth is the single path every month read goes through, and is meant to
// stay that way: ticket 11's Category breakdown reads the same month, and
// ticket 13 materialises the month's Recurring expenses before anything is
// summed (ADR-0005). One function to hook, rather than one per report.
//
// month is trusted to be YYYY-MM by the time it arrives — the handler is where
// that is decided, so the SQL below can compare it as plain text.
func readMonth(db *sql.DB, month string) (monthTotals, error) {
	m := monthTotals{Month: month}

	if err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM expense
		WHERE substr(occurred_on, 1, 7) = ?`, month).Scan(&m.ExpenseCents); err != nil {
		return m, err
	}

	// The filter ADR-0003 exists for. It is redundant against the substr on
	// the line below it — a NULL payment date matches no month — and it stays
	// anyway: this is the invariant the codebase least wants to have to
	// re-derive, and every query that sums Income is meant to say it out loud.
	//
	// An Income belongs to the month its money arrived, so payment_date is the
	// only date it can be grouped by. invoice_sent_date answers "how long has
	// this been sitting", which is ticket 14's question.
	if err := db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM income
		WHERE payment_date IS NOT NULL AND substr(payment_date, 1, 7) = ?`,
		month).Scan(&m.IncomeCents); err != nil {
		return m, err
	}

	m.NetCents = m.IncomeCents - m.ExpenseCents
	return m, nil
}
