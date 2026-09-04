package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// monthLayout is how a month crosses the API and how it is stored: dates are
// TEXT as YYYY-MM-DD, so a month is the first seven characters of one and
// nothing has to be parsed to group by it — and a Recurring expense's window
// is two columns of exactly this shape, compared as text for the same reason.
// validMonth, in recurring.go, is the one gate that holds anything to it.
const monthLayout = "2006-01"

// monthRow is where a month stands, in three numbers: money in is received
// Income only — an Income with no payment date is money that has not arrived,
// and counts toward nothing (ADR-0003).
//
// The difference is computed here rather than left to the client, because it
// is the number the household actually reads and there is no second opinion to
// be had about it.
//
// It is a type of its own because a year is twelve of these and nothing more:
// the year view draws a shape out of the three numbers, and a year that also
// carried twelve Category breakdowns would be twelve of the heaviest query in
// the app for a screen that reads none of them. Embedded rather than nested,
// so a month report's JSON is one flat object exactly as it was.
type monthRow struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
	NetCents     int64  `json:"net_cents"`
}

// monthTotals answers the question the app exists to answer: where does this
// month stand, and what did the money go on.
type monthTotals struct {
	monthRow

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

// readMonth is a month with its breakdown: what the month screen reads.
func readMonth(db *sql.DB, now func() time.Time, month string) (monthTotals, error) {
	row, err := readMonthRow(db, now, month)
	if err != nil {
		return monthTotals{monthRow: row}, err
	}
	byCategory, err := readBreakdown(db, month)
	return monthTotals{monthRow: row, ByCategory: byCategory}, err
}

// readMonthRow is the single path every month read goes through, and stays
// that way: the month screen reads one through readMonth, and the year view
// reads twelve. One function to hook, rather than one per report — which is
// what makes the line below cover every month read.
//
// Generation comes first, and reading second, which is the whole of ADR-0005:
// there is no scheduler, so the rent exists because someone looked at the
// month. A refusal to generate — a clock the Pi cannot believe — is not a
// refusal to read: the totals of what was typed are still the truth, and
// /api/health is where the screens learn why nothing was generated.
//
// month is trusted to be YYYY-MM by the time it arrives — the handler is where
// that is decided, so the SQL below can compare it as plain text.
func readMonthRow(db *sql.DB, now func() time.Time, month string) (monthRow, error) {
	m := monthRow{Month: month}

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

// readExpenseCentsExcludingInvestments is readMonthRow's Expense query with
// the Investments category left out — the exclusion Budget/Target (ticket 04)
// and the yearly Estimate (ticket 05) share: both exist to answer "is this
// normal spending," and a lumpy stock buy would wreck that (ADR-0009). It
// materialises first, for the same reason readMonthRow does: a past month
// nobody has opened yet must not read as an empty one just because Budget is
// the first thing to ask about it.
func readExpenseCentsExcludingInvestments(db *sql.DB, now func() time.Time, month string) (int64, error) {
	if err := materialise(db, now, month); err != nil {
		return 0, err
	}
	var cents int64
	err := db.QueryRow(`SELECT COALESCE(SUM(e.amount_cents), 0) FROM expense e
		JOIN category c ON c.id = e.category_id
		WHERE substr(e.occurred_on, 1, 7) = ? AND IFNULL(c.code, '') != ?`,
		month, codeInvestments).Scan(&cents)
	return cents, err
}

// dailyCategoryTotal is readBreakdown's categoryTotal with a day column: one
// Category's share of one day's spend, the shape both trend charts read.
type dailyCategoryTotal struct {
	Day         string `json:"day"`
	CategoryID  int64  `json:"category_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

// readDailyBreakdown is readBreakdown over a date range instead of a month,
// with the day it groups by kept rather than summed away. Same ADR-0002
// attribution, same shape of query — one query, GROUP BY on day and Category
// instead of Category alone — used by both trend charts: the rolling window
// and the one-month view differ only in which two dates bound the range, so
// they share this rather than each carrying a near-identical query.
//
// Investments are not excluded (ADR-0009, and the ticket says so explicitly)
// — unlike Budget/Estimate, a trend chart's job is showing what actually
// happened, and a stock buy is a day like any other Category's.
//
// from and to are trusted to be YYYY-MM-DD by the time they arrive — both
// callers build them from a validated month or from the clock, never from
// request input directly.
func readDailyBreakdown(db *sql.DB, from, to string) ([]dailyCategoryTotal, error) {
	rows, err := db.Query(`SELECT share.day, c.id, c.name, SUM(share.amount_cents)
		FROM (
			SELECT e.occurred_on AS day, e.category_id, e.amount_cents - COALESCE(
				(SELECT SUM(i.amount_cents) FROM item i WHERE i.expense_id = e.id), 0
			) AS amount_cents
			FROM expense e WHERE e.occurred_on BETWEEN ? AND ?
			UNION ALL
			SELECT e.occurred_on, i.category_id, i.amount_cents FROM item i
			JOIN expense e ON e.id = i.expense_id
			WHERE e.occurred_on BETWEEN ? AND ?
		) share
		JOIN category c ON c.id = share.category_id
		GROUP BY share.day, c.id, c.name
		HAVING SUM(share.amount_cents) > 0
		ORDER BY share.day, SUM(share.amount_cents) DESC, c.name COLLATE NOCASE`, from, to, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// [] rather than null: the screen maps over it, same as every other list
	// report here.
	out := []dailyCategoryTotal{}
	for rows.Next() {
		var d dailyCategoryTotal
		if err := rows.Scan(&d.Day, &d.CategoryID, &d.Category, &d.AmountCents); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// dailyWindowDays is the Dashboard's rolling window — a fixed constant per
// the ticket, not a household setting, and inclusive of today: 30 days
// ending today means today and the 29 before it.
const dailyWindowDays = 30

// handleDailyReport answers the trailing window ending "today", wherever the
// injected clock says that is — the same clock, and the same Pi-has-no-RTC
// reason, as every other report here. It does not materialise: the window
// slides daily regardless of which months anyone has opened, and whichever
// month report the household reads for the current month already generates
// that month's Recurring expenses.
func handleDailyReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		to := now()
		from := to.AddDate(0, 0, -(dailyWindowDays - 1)).Format(dateLayout)
		daily, err := readDailyBreakdown(db, from, to.Format(dateLayout))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, daily)
	}
}

// handleMonthDailyReport answers one calendar month's daily breakdown — what
// the Month page's weekly chart buckets into Monday-start weeks itself
// (there is deliberately no separate weekly endpoint). It materialises the
// month first, the same as handleMonthReport: this is read independently of
// /api/reports/month/{month}, so it is its own path to ADR-0005's "a month
// nobody opened is not silently empty forever" rather than a free ride on the
// other endpoint happening to be called first.
func handleMonthDailyReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		month := r.PathValue("month")
		if err := validMonth("month", month); err != nil {
			writeInvalid(w, err)
			return
		}
		if err := materialise(db, now, month); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		start, err := time.Parse(monthLayout, month)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		from := start.Format(dateLayout)
		// The month's last day, the same way recurring.go clamps a rent to
		// it: one month on, one day back.
		to := start.AddDate(0, 1, 0).AddDate(0, 0, -1).Format(dateLayout)

		daily, err := readDailyBreakdown(db, from, to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, daily)
	}
}

// recentLimit is how many entries the home screen lists, combined across both
// directions before the client splits them into its separate expense and
// income cards — wide enough that one direction being quiet for a while (an
// Expense-only day, a burst of Recurring generation) does not crowd the other
// out of the shared pool entirely. Still not a page size — there is no second
// page, and the two full lists are one tap away.
const recentLimit = 20

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
//
// Payer is whose money an Expense left (empty on an Income). Client is who
// an Income came from (empty on an Expense, and on an Income that names
// nobody). Names rather than ids, for the same reason Category is: this list
// is read on a screen that holds neither picker.
type recentEntry struct {
	Direction   string `json:"direction"`
	ID          int64  `json:"id"`
	Date        string `json:"date"`
	AmountCents int64  `json:"amount_cents"`
	Category    string `json:"category"`
	Payer       string `json:"payer"`
	Client      string `json:"client"`

	// IsGift is true for an Expense resolving to the Gift Category, and always
	// false on an Income — the spoiler blur is an Expense-only behaviour (spec's
	// Gift section). Resolved here by code, not by matching Category against a
	// hardcoded name: the household can rename Category freely, and this list
	// already reads Category as a name for display, not for identity.
	IsGift bool `json:"is_gift"`
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
				e.amount_cents, c.name, e.payer, '' AS client,
				IFNULL(c.code, '') = ? AS is_gift, e.created_at AS typed_at
			FROM expense e JOIN category c ON c.id = e.category_id
			UNION ALL
			SELECT ?, i.id, COALESCE(i.payment_date, ''),
				i.amount_cents, c.name, '', COALESCE(cl.name, ''), 0, i.created_at
			FROM income i JOIN category c ON c.id = i.category_id
			LEFT JOIN client cl ON cl.id = i.client_id
			ORDER BY typed_at DESC, id DESC, direction
			LIMIT ?`, towardsExpense, codeGift, towardsIncome, recentLimit)
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
				&e.Category, &e.Payer, &e.Client, &e.IsGift, &typedAt); err != nil {
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

// yearLayout is how a year crosses the API: four digits, which is the first
// four characters of a date and of a month, and a Tax year written out. Like
// every other date shape here, it sorts correctly as text.
const yearLayout = "2006"

// yearTotals is the shape of a whole year: the same three numbers a month
// answers with, over twelve months, and the twelve months themselves so the
// screen can draw what the year looked like month by month.
//
// No Category breakdown, for the year or for its months. The ticket asks for
// the shape of the year and the year in tax terms; a twelve-month breakdown is
// twelve runs of the heaviest query in the app, on a Pi Zero W, for a screen
// that would have to sum them back together to say anything. The month screen
// is one tap away and answers it for the month the household is asking about.
type yearTotals struct {
	Year         string `json:"year"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
	NetCents     int64  `json:"net_cents"`

	// Received Income that is not work — neither Freelance nor Stipendio.
	// The Dashboard names it under the year's income total; work is the
	// rest of IncomeCents. Still cash-basis and still excluding unpaid
	// (ADR-0003), same as IncomeCents itself.
	ExtraIncomeCents int64 `json:"extra_income_cents"`

	// Always twelve, January first, whether or not anything happened in any
	// of them: a year view is a shape, and a missing month would be a gap in
	// it rather than an empty one.
	Months []monthRow `json:"months"`
}

// handleYearReport reports one calendar year, month by month. Which year is
// the browser's question for the reason which month is: the phone has a
// correct clock and the Pi has no RTC.
//
// A year that is not a year is refused rather than summed, exactly as a month
// is: "26" matches no date, and answering zeros would be indistinguishable
// from a year in which nothing happened.
func handleYearReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year := r.PathValue("year")
		if err := validYear("year", year); err != nil {
			writeInvalid(w, err)
			return
		}
		totals := yearTotals{Year: year}
		for _, month := range monthsOf(year) {
			// Through readMonthRow, which is what makes a year view generate
			// the Recurring expenses each of its twelve months owes —
			// ADR-0005's "a month nobody opened at the time is not silently
			// empty forever" is most of what a year view is for.
			m, err := readMonthRow(db, now, month)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			totals.IncomeCents += m.IncomeCents
			totals.ExpenseCents += m.ExpenseCents
			totals.Months = append(totals.Months, m)
		}
		totals.NetCents = totals.IncomeCents - totals.ExpenseCents

		// ponytail: Stipendio is matched by seed name, not a code — it is
		// not a Base category, and promoting it for one Dashboard footnote
		// is a migration nobody asked for. If the household renames it,
		// salary starts counting as extra; give it a code then.
		if err := db.QueryRow(`SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
			JOIN category c ON c.id = i.category_id
			WHERE i.payment_date IS NOT NULL AND substr(i.payment_date, 1, 4) = ?
			AND IFNULL(c.code, '') <> ?
			AND c.name <> ?`,
			year, codeFreelance, seedStipendioName).Scan(&totals.ExtraIncomeCents); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, totals)
	}
}

// taxSummary is the figure the invoicing software cannot give: what was really
// received and what was really paid, after the fact. "Received X, paid Y in
// tax, net Z%" — one total, no per-tax-type breakdown (ADR-0008).
//
// The two halves are counted differently on purpose. Received is cash: money
// in the account during this year, and unpaid invoices are not money
// (ADR-0003). Tax paid is attributed by Tax year, because tax on one year's
// income is paid during the next, and a cash-basis figure would look like a
// tax rate while being nothing of the kind.
type taxSummary struct {
	Year          string `json:"year"`
	ReceivedCents int64  `json:"received_cents"`
	TaxPaidCents  int64  `json:"tax_paid_cents"`
	NetCents      int64  `json:"net_cents"`

	// What was kept, as a percentage of what was received. Null rather than
	// zero when nothing was received: a percentage of nothing is not 0%, and
	// a screen reading "net 0%" against an empty year would be a wrong answer
	// where there is no answer. A percentage is not money, so this is the one
	// float in the codebase.
	NetPercent *float64 `json:"net_percent"`
}

// handleTaxSummary answers the year in tax terms. It generates the year's
// months first, for the reason every other report does — a tax payment can be
// a Recurring expense like any other, and the summary must not be short one
// because nobody happened to open the month it lands in.
//
// ponytail: it generates this year's twelve months and not the next year's,
// where a payment attributed back to this one would have been made. The year
// view beside this report generates whichever year the household is looking
// at, so the months not covered here are covered the moment anyone looks at
// them. Widen it if a tax payment ever goes missing for a year nobody opened.
func handleTaxSummary(db *sql.DB, now func() time.Time) http.HandlerFunc {
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

		s := taxSummary{Year: year}

		// The ADR-0003 filter, said out loud for the reason the month report
		// says it: an invoice sent is not income received, and this is the
		// number the household compares against its invoicing software.
		//
		// Freelance is resolved by the Base category's code and never by name
		// — the household can hide that Category but not rename it, and the
		// report must not depend on the second half of that being true. Only
		// freelance: employment income arrives already taxed and a gift is not
		// income, so counting either makes the percentage meaningless.
		if err := db.QueryRow(`SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
			JOIN category c ON c.id = i.category_id AND c.code = ?
			WHERE i.payment_date IS NOT NULL AND substr(i.payment_date, 1, 4) = ?`,
			codeFreelance, year).Scan(&s.ReceivedCents); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		// And the half that is not cash: tax_year, not occurred_on. The
		// Category is the join and the Tax year is the filter, because both
		// have to hold — ADR-0008's Y is "Expenses in the taxes base Category,
		// attributed by Tax year".
		//
		// The COALESCE is what makes a generated tax payment count. A Tax year
		// is an override of a default rather than a fact only a form can
		// supply: checkExpense stores that default on anything typed, and
		// materialise — which copies template fields and knows nothing about
		// tax — leaves NULL. Applying the same default here rather than at
		// generation covers the rows already generated on the Pi as well as
		// the ones still to come, and there is then exactly one rule: a tax
		// Expense that says nothing belongs to the year it was paid in.
		//
		// Both sides are integers so the comparison needs no affinity to be
		// applied to it: the column is INTEGER, the substring is CAST, and the
		// year is bound as a number rather than as the text it arrived as.
		yearNumber, err := strconv.Atoi(year)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := db.QueryRow(`SELECT COALESCE(SUM(e.amount_cents), 0) FROM expense e
			JOIN category c ON c.id = e.category_id AND c.code = ?
			WHERE COALESCE(e.tax_year, CAST(substr(e.occurred_on, 1, 4) AS INTEGER)) = ?`,
			codeTaxes, yearNumber).Scan(&s.TaxPaidCents); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		s.NetCents = s.ReceivedCents - s.TaxPaidCents
		if s.ReceivedCents > 0 {
			percent := float64(s.NetCents) * 100 / float64(s.ReceivedCents)
			s.NetPercent = &percent
		}
		writeJSON(w, http.StatusOK, s)
	}
}

// monthsOf is the twelve months of a year, January first. Built by
// concatenation rather than by date arithmetic: a year is the first four
// characters of a month, so there is nothing to carry.
//
// year is trusted to be four digits by the time it arrives — the handlers are
// where that is decided.
func monthsOf(year string) []string {
	months := make([]string, 0, 12)
	for m := 1; m <= 12; m++ {
		months = append(months, fmt.Sprintf("%s-%02d", year, m))
	}
	return months
}

// validYear accepts a year and nothing else, and is validMonth one level up:
// the layout makes time.Parse strict about the shape, so "26" is refused for
// the same reason "2026-3" is refused as a month. Everything downstream
// compares these as text — including a Tax year, which is an integer in the
// column and a year in every sentence about it.
func validYear(what, value string) error {
	if _, err := time.Parse(yearLayout, value); err != nil {
		return errors.New(what + " must be a real year as YYYY")
	}
	return nil
}
