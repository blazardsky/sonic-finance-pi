package main

import (
	"database/sql"
	"net/http"
	"time"
)

// monthLayout is how a month crosses the API and how it is stored: dates are
// TEXT as YYYY-MM-DD, so a month is the first seven characters of one and
// nothing has to be parsed to group by it — and a Recurring expense's window
// is two columns of exactly this shape, compared as text for the same reason.
// validMonth, in recurring.go, is the one gate that holds anything to it.
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

	// Where the money went, biggest share first. Always present, and always
	// summing to ExpenseCents exactly — the screen maps over it, and a
	// breakdown that did not add up would be worse than no breakdown.
	ByCategory []categoryTotal `json:"by_category"`
}

// One Category's share of a month's spend. The name travels with the id
// because the screen showing this has no other reason to hold the Category
// list, and a hidden Category still has to say what it is called: money spent
// under a name since hidden did not stop having been spent.
type categoryTotal struct {
	CategoryID  int64  `json:"category_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

// handleMonthReport reports one calendar month. The month is the whole input:
// which month to show is the browser's question — the phone in the hand has a
// correct clock and the Pi has no RTC. The clock is injected all the same,
// because generating this month's Recurring expenses is the one thing here
// that has to know what "past" means.
//
// A month that is not a month is refused rather than summed: substr against
// "2026-3" matches nothing, and answering zeros would be indistinguishable
// from a month in which nothing happened. It is refused before generation,
// too — materialise builds a date out of the month it is given.
func handleMonthReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		month := r.PathValue("month")
		if err := validMonth("month", month); err != nil {
			writeInvalid(w, err)
			return
		}
		totals, err := readMonth(db, now, month)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, totals)
	}
}

// readMonth is the single path every month read goes through, and stays that
// way: the Category breakdown reads the same month, and ticket 15's year view
// will read twelve of them through here. One function to hook, rather than one
// per report — which is what makes the line below cover every month read.
//
// Generation comes first, and reading second, which is the whole of ADR-0005:
// there is no scheduler, so the rent exists because someone looked at the
// month. A refusal to generate — a clock the Pi cannot believe — is not a
// refusal to read: the totals of what was typed are still the truth, and
// /api/health is where the screens learn why nothing was generated.
//
// month is trusted to be YYYY-MM by the time it arrives — the handler is where
// that is decided, so the SQL below can compare it as plain text.
func readMonth(db *sql.DB, now func() time.Time, month string) (monthTotals, error) {
	m := monthTotals{Month: month}

	if err := materialise(db, now, month); err != nil {
		return m, err
	}

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

	byCategory, err := readBreakdown(db, month)
	if err != nil {
		return m, err
	}
	m.ByCategory = byCategory

	m.NetCents = m.IncomeCents - m.ExpenseCents
	return m, nil
}

// readBreakdown attributes the month's spend to Categories, and is the whole
// of ADR-0002 expressed as one query: an Item's amount goes to the Item's
// Category, and what the Items do not cover goes to the Expense's own. The
// remainder is never "uncategorised", because there is nowhere else for it to
// be — the Expense has a Category whether or not anything was itemised.
//
// The two halves of the union are the two ways money is attributed, and the
// arithmetic is a subtraction rather than a redistribution, which is what
// makes the total unconditionally equal to SUM(amount_cents): every Expense
// contributes its own amount, split at most once.
//
// A remainder of zero is dropped rather than listed. An Expense its Items
// account for entirely spent nothing under its own Category, and a €0,00 line
// under Casa would read as money that went there.
func readBreakdown(db *sql.DB, month string) ([]categoryTotal, error) {
	rows, err := db.Query(`SELECT c.id, c.name, SUM(share.amount_cents)
		FROM (
			SELECT e.category_id, e.amount_cents - COALESCE(
				(SELECT SUM(i.amount_cents) FROM item i WHERE i.expense_id = e.id), 0
			) AS amount_cents
			FROM expense e WHERE substr(e.occurred_on, 1, 7) = ?
			UNION ALL
			SELECT i.category_id, i.amount_cents FROM item i
			JOIN expense e ON e.id = i.expense_id
			WHERE substr(e.occurred_on, 1, 7) = ?
		) share
		JOIN category c ON c.id = share.category_id
		GROUP BY c.id, c.name
		HAVING SUM(share.amount_cents) > 0
		ORDER BY SUM(share.amount_cents) DESC, c.name COLLATE NOCASE`, month, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// An empty month has to marshal as [] rather than null: the screen maps
	// over it, and most households have months with nothing in them.
	out := []categoryTotal{}
	for rows.Next() {
		var ct categoryTotal
		if err := rows.Scan(&ct.CategoryID, &ct.Category, &ct.AmountCents); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}

// recentLimit is how many entries the home screen lists. It sits under three
// totals on a phone, so the number is what fits above the fold rather than a
// page size — there is no second page, and the two full lists are one tap
// away.
const recentLimit = 10

// A recentEntry is one thing the household typed, in either direction.
// Direction is which of the two it is, which is what the screen signs it by.
// This is a read-only projection and not a third entity — nothing is ever
// saved in this shape.
//
// Date is the day the money moved: an Expense's occurred_on, an Income's
// payment date — and empty for an Income that has not been paid, because money
// that has not arrived has no day it arrived on. An empty date is therefore
// exactly the unpaid state (ADR-0003), and the screen has to show it as such:
// this list sits directly under a money-in total that excludes it, and an
// unpaid invoice reading as €800 received would be that ADR's named bug on
// screen. invoice_sent_date is deliberately not a fallback — "how long has
// this been sitting" is a different question, asked by ticket 14.
type recentEntry struct {
	Direction   string `json:"direction"`
	ID          int64  `json:"id"`
	Date        string `json:"date"`
	AmountCents int64  `json:"amount_cents"`
	Category    string `json:"category"`
}

// The two directions money moves, which are the two entities this list is a
// projection of. Not "kind": CONTEXT.md keeps that word out of the codebase.
const (
	towardsExpense = "expense"
	towardsIncome  = "income"
)

// handleRecentEntries lists the last few entries so that what was just typed
// is visible. Ordered by when it was typed rather than by the date on it: the
// list exists to confirm an entry landed, and a receipt found in a coat pocket
// is exactly the one most worth showing.
//
// It is not scoped to a month, and deliberately: the home screen steps through
// months, and an entry backdated to last month still has to appear the moment
// it is saved.
func handleRecentEntries(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// created_at is what this is ordered by and never what it shows: the
		// household did not type that date, and for an Income it would be a
		// second spelling of a payment that may not have happened.
		//
		// The tie-break is there only so two identical requests answer in the
		// same order. Which way a tie falls does not matter: created_at is
		// stored to the second, entries typed in the same second are equally
		// "just typed", and ordering this list is all it is for.
		rows, err := db.Query(`SELECT ? AS direction, e.id AS id, e.occurred_on AS date,
				e.amount_cents, c.name, e.created_at AS typed_at
			FROM expense e JOIN category c ON c.id = e.category_id
			UNION ALL
			SELECT ?, i.id, COALESCE(i.payment_date, ''),
				i.amount_cents, c.name, i.created_at
			FROM income i JOIN category c ON c.id = i.category_id
			ORDER BY typed_at DESC, id DESC, direction
			LIMIT ?`, towardsExpense, towardsIncome, recentLimit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// [] rather than null: the screen maps over it, and an empty list is
		// where every household starts.
		out := []recentEntry{}
		for rows.Next() {
			var e recentEntry
			var typedAt string
			if err := rows.Scan(&e.Direction, &e.ID, &e.Date, &e.AmountCents,
				&e.Category, &typedAt); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, e)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
