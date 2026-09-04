package main

import (
	"database/sql"
	"net/http"
	"time"
)

// billingWindowMonths is how far back the nudge looks for a Client who is
// usually billed by now: the three calendar months before this one. A code
// constant and not a setting — the ticket says so, and a household that has to
// tune the number would have to understand it first. Widening it is a one-line
// change on the day someone asks.
const billingWindowMonths = 3

// Pending payments are the two questions a freelancer asks a spreadsheet on a
// Sunday evening: what am I owed, and what did I forget to invoice. Neither
// list is a record of its own — both are views over Incomes (CONTEXT.md), and
// the word here is "pending payment", never "receivable".
//
// Two lists from one endpoint, because they are read together and one round
// trip is one round trip.
type pendingPayments struct {
	// Money genuinely owed: every Income with no payment date, oldest first.
	Outstanding []outstandingIncome `json:"outstanding"`

	// Invoices that look forgotten: Clients billed inside the window with
	// nothing recorded this month.
	NotYetInvoiced []notYetInvoicedClient `json:"not_yet_invoiced"`
}

// One Income still waiting for its money. Client and Category travel as names
// rather than ids for the reason a category breakdown's do: the screen showing
// this holds neither list, and a hidden Client still has to say what it is
// called — money owed by a name since hidden is still owed.
//
// Client is empty when the Income names nobody: a birthday gift that never
// arrived is money waiting, from nobody in particular.
type outstandingIncome struct {
	ID           int64  `json:"id"`
	AmountCents  int64  `json:"amount_cents"`
	Client       string `json:"client"`
	Category     string `json:"category"`
	WaitingSince string `json:"waiting_since"`
	DaysWaiting  int    `json:"days_waiting"`
	// Empty when no invoice was sent: waiting_since then falls back to the
	// day the Income was typed. The Dashboard's "in attesa" list wants the
	// real invoice date and skips the empty ones.
	InvoiceSentDate string `json:"invoice_sent_date"`
}

// One Client the household has been billing and has not billed this month.
// The id comes along because two Clients can share a name — clients.go leaves
// that to the household — so the name is not something a list can key on.
type notYetInvoicedClient struct {
	ClientID int64  `json:"client_id"`
	Client   string `json:"client"`
}

// billedOn is the date an Income counts as having been billed on: the invoice
// date when there is one, and otherwise the day it was typed. An unpaid Income
// has no payment date by definition — that is what makes it unpaid (ADR-0003)
// — so payment_date is the one date that cannot answer either of this file's
// questions. The fallback is sliced to ten characters rather than parsed,
// because every date in this codebase is compared as text.
//
// It takes the table alias rather than reading a bare column name: the nudge
// asks this of two Incomes in one statement, an outer one and the one inside
// its NOT EXISTS, and which of them a bare name binds to is not something to
// have to reason about at 3am.
func billedOn(income string) string {
	return `COALESCE(` + income + `.invoice_sent_date, substr(` + income + `.created_at, 1, 10))`
}

// handlePendingPayments answers both lists at once. The clock is injected for
// the same reason everything else here injects it — how long something has
// been waiting, and which month is "this month", are the whole content of the
// two lists, and the Pi has no RTC.
//
// A clock the Pi cannot believe is not refused here, for the reason a month
// report is not: what is unpaid is unpaid whatever the clock says, and the
// household asking who owes them still gets the true answer. What an unset
// clock costs is the arithmetic around it — the 1970 window contains no
// Income, so the nudge is empty, and every "days waiting" floors at zero, so
// a three-month-old invoice reads as sent today. That is the one thing here
// that goes quietly wrong, which is why /api/health publishes clock_ok and
// App.tsx warns on it above every screen: unlike generation, there is nothing
// to lose by reading and nothing but the clock to fix.
func handlePendingPayments(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		outstanding, err := readOutstanding(db, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		notYetInvoiced, err := readNotYetInvoiced(db, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, pendingPayments{
			Outstanding: outstanding, NotYetInvoiced: notYetInvoiced,
		})
	}
}

// readOutstanding lists what is owed, oldest first. Every unpaid Income is
// here, freelance or not: a reimbursement that never arrived is money waiting
// as much as an invoice is. The freelance filter belongs to the other list.
//
// The tie-break on id is there only so two identical requests answer in the
// same order — several invoices sent on one day are equally old.
func readOutstanding(db *sql.DB, now func() time.Time) ([]outstandingIncome, error) {
	rows, err := db.Query(`SELECT i.id, i.amount_cents, COALESCE(cl.name, ''), c.name,
			`+billedOn("i")+` AS waiting_since, COALESCE(i.invoice_sent_date, '')
		FROM income i
		JOIN category c ON c.id = i.category_id
		LEFT JOIN client cl ON cl.id = i.client_id
		WHERE i.payment_date IS NULL
		ORDER BY waiting_since, i.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	today := startOfDay(now)

	// [] rather than null: the screen maps over it, and being owed nothing is
	// the state a household hopes to be in.
	out := []outstandingIncome{}
	for rows.Next() {
		var e outstandingIncome
		if err := rows.Scan(&e.ID, &e.AmountCents, &e.Client, &e.Category, &e.WaitingSince, &e.InvoiceSentDate); err != nil {
			return nil, err
		}
		e.DaysWaiting = daysSince(today, e.WaitingSince)
		out = append(out, e)
	}
	return out, rows.Err()
}

// readNotYetInvoiced lists the Clients who look like a forgotten invoice — the
// nudge: at least one freelance Income inside the window, and nothing
// freelance recorded this month. Freelance is resolved by the Base category's
// code rather than by name, so renaming "Freelance" leaves this working
// (ADR-0008).
//
// Both halves are freelance, which is the ticket's "limited to freelance-
// category income" read as one rule rather than two. A gift is not an invoice
// in either direction: one arriving in February must not start a nudge, and
// one arriving in March must not clear it — a relative sending money is not
// evidence that a Client was billed.
//
// Hidden Clients are left out: hiding is how a Client is retired, and the app
// has no business chasing invoices for someone the household stopped billing.
// That is the opposite of what recording an Income does with hidden, and
// deliberately — one is a correction to history, this is a prompt to act.
func readNotYetInvoiced(db *sql.DB, now func() time.Time) ([]notYetInvoicedClient, error) {
	month := now().Format(monthLayout)
	// Parsing the month back gives the 1st, which is what makes the subtraction
	// safe: AddDate on the 31st of May would land in March.
	start, err := time.Parse(monthLayout, month)
	if err != nil {
		return nil, err
	}
	from := start.AddDate(0, -billingWindowMonths, 0).Format(monthLayout)

	rows, err := db.Query(`SELECT cl.id, cl.name
		FROM client cl
		JOIN income i ON i.client_id = cl.id
		JOIN category c ON c.id = i.category_id AND c.code = ?
		WHERE cl.hidden = 0
		AND substr(`+billedOn("i")+`, 1, 7) >= ? AND substr(`+billedOn("i")+`, 1, 7) < ?
		AND NOT EXISTS (SELECT 1 FROM income n
			JOIN category nc ON nc.id = n.category_id AND nc.code = ?
			WHERE n.client_id = cl.id AND substr(`+billedOn("n")+`, 1, 7) = ?)
		GROUP BY cl.id, cl.name
		ORDER BY cl.name COLLATE NOCASE`,
		codeFreelance, from, month, codeFreelance, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []notYetInvoicedClient{}
	for rows.Next() {
		var c notYetInvoicedClient
		if err := rows.Scan(&c.ClientID, &c.Client); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// startOfDay is the clock as a calendar date, so that "days waiting" counts
// days rather than 24-hour blocks: an invoice sent yesterday evening has been
// waiting a day this morning, not zero.
//
// The date the clock reads is anchored in UTC rather than in its own zone,
// because that is where time.Parse anchors the stored dates it is subtracted
// from — a stored date carries no zone at all. Anchoring the two differently
// is off by the offset, which on a Pi running Italian time is enough to report
// an invoice sent on the 1st as waiting 32 days on the 3rd of the next month;
// it was, on the first run of the real binary. UTC on both sides also has no
// DST, so every subtraction here is a whole number of days.
func startOfDay(now func() time.Time) time.Time {
	t := now()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// daysSince counts whole days from a stored date to today, and never counts
// backwards: an invoice dated in the future — a typo, or a Pi whose clock has
// not caught up — has been waiting no days rather than minus forty. An
// unparseable date answers zero for the same reason, which the column's CHECK
// and validate() between them already make unreachable.
func daysSince(today time.Time, date string) int {
	since, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0
	}
	days := int(today.Sub(since).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
