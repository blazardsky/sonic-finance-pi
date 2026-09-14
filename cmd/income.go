package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// An income is money owed to or received by the household. It exists from the
// moment it is expected — an invoice sent — and becomes received when it has a
// payment date. Until then it is unpaid and counts toward no total; see
// ADR-0003, which carries the standing hazard: every query that sums Income
// filters on the payment date being there.
//
// One number, the amount that arrived. There is no gross/net pair and no tax
// percentage, because tax is an Expense on the day it is paid and netting the
// Income as well would deduct it twice — ADR-0004.
//
// Category is the "income reason" the Income screens word differently; it is
// the same field, referenced so a rename fixes every past Income at once. The
// Client is referenced for the same reason and is optional: a birthday gift
// names nobody. Payer is the deliberate opposite — the label text itself, so
// renaming the list leaves what old entries say exactly as it was.
type income struct {
	ID          int64  `json:"id"`
	AmountCents int64  `json:"amount_cents"`
	CategoryID  int64  `json:"category_id"`
	ClientID    *int64 `json:"client_id"`
	Payer       string `json:"payer"`

	// Both dates are empty when absent rather than null, which is what lets a
	// PATCH merge onto the stored row: an omitted field keeps what it had, and
	// an empty one clears it — the same bargain the Expense note makes. They
	// are stored as NULL, because "unpaid" is the absence of a payment date
	// and a column that can hold '' as well would have two spellings of it.
	PaymentDate     string `json:"payment_date"`
	InvoiceSentDate string `json:"invoice_sent_date"`

	Note string `json:"note"`

	// The Holding this Income sold, and nil on every Income that is not a
	// sell — meaningful only under the Investments base category (ticket 03).
	// Set directly on insert like CategoryID, and never read by a report
	// outside Investments. A pointer, like ClientID, because "no Holding" is a
	// real answer for every non-Investments Income.
	HoldingID *int64 `json:"holding_id"`

	// The Contract this Income counts toward, and nil for "Extra" — real
	// income from this Client, just outside any agreed total (ticket 05). A
	// pointer for the same reason ClientID is one: unlinked is a real, common
	// answer, not an omission. checkIncome refuses a Contract that does not
	// belong to ClientID, so linking across Clients is not a state a write can
	// reach even though nothing at the database layer forbids it.
	ContractID *int64 `json:"contract_id"`

	// Whether this Income still owes its 2€ "marca da bollo elettronica" —
	// true by default, since every freelance invoice over bolloThresholdCents
	// carries one. The 2€ arrived with the payment like the rest of the
	// amount — AmountCents is never adjusted for it — but it is not real
	// freelance revenue, so a linked Contract's received/accounted figures
	// (computeContractFigures, via contractColumns) subtract it before
	// counting the Income toward the agreed total.
	BolloFattura bool `json:"bollo_fattura"`
}

// bolloThresholdCents is the amount above which an Italian invoice needs a 2€
// marca da bollo — the household's own rounding of the real threshold
// (77.47€) to a plain 77€.
const bolloThresholdCents = 7700

// bolloFatturaCents is the 2€ stamp duty a qualifying Income already carries
// inside AmountCents, in cents.
const bolloFatturaCents = 200

// migrateIncomes is schema step 6. client_id is a plain reference: not
// ON DELETE SET NULL, which would strip the name off every past Income the
// moment a Client was deleted, and not CASCADE, which would delete the money
// along with the name. A plain reference refuses instead, and writeError turns
// that refusal into a 409 — so a Client with Incomes behind it is retired by
// hiding, which is what hiding is for.
//
// Both dates CHECK against ” so that an absent one has exactly one spelling
// in the database. Unpaid Income counting toward no total rests entirely on
// `payment_date IS NULL` — ADR-0003 calls the missing filter the single most
// likely bug here — and a stray ” would be an Income that reads as paid and
// sums as nothing. Enforcing it in the column rather than trusting nullDate
// means a later query cannot be quietly wrong; it fails loudly instead.
func migrateIncomes(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE income (
		id                INTEGER PRIMARY KEY,
		amount_cents      INTEGER NOT NULL CHECK (amount_cents > 0),
		category_id       INTEGER NOT NULL REFERENCES category(id),
		client_id         INTEGER REFERENCES client(id),
		payer             TEXT NOT NULL DEFAULT '',
		payment_date      TEXT CHECK (payment_date <> ''),
		invoice_sent_date TEXT CHECK (invoice_sent_date <> ''),
		note              TEXT NOT NULL DEFAULT '',
		created_at        TEXT NOT NULL
	) STRICT`)
	return err
}

// migrateBolloFattura is schema step 15: bollo_fattura defaults to 1 for
// every existing Income too — every one of them is a real invoice, so the
// same 2€-over-77€ rule already applies retroactively to a Contract's
// received/accounted figures once this ships.
func migrateBolloFattura(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE income ADD COLUMN bollo_fattura INTEGER NOT NULL DEFAULT 1 CHECK (bollo_fattura IN (0, 1))`)
	return err
}

// handleListIncomes returns every Income, paid and unpaid alike: the payment
// state is a field on the row, and the screens are what decide which they
// want. Newest first by invoice date, falling back to the payment date for an
// Income never invoiced (a gift, a salary) — the closest thing an Income has
// to Expense's own occurred_on — and finally to id for the rare Income with
// neither date yet.
//
// Optional query params, the same shape handleListExpenses offers: `year`
// (YYYY) scopes to one year of that same date, and `limit`/`offset` page
// through whatever `year` (or the absence of it) already selected.
func handleListIncomes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := incomeSelect
		var args []any
		if year := r.URL.Query().Get("year"); year != "" {
			if len(year) != 4 {
				writeError(w, http.StatusBadRequest, fmt.Errorf("year must be 4 digits"))
				return
			}
			if _, err := strconv.Atoi(year); err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("year must be numeric"))
				return
			}
			query += ` WHERE COALESCE(invoice_sent_date, payment_date) LIKE ?`
			args = append(args, year+"-%")
		}
		query += ` ORDER BY COALESCE(invoice_sent_date, payment_date) DESC, id DESC`
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				writeError(w, http.StatusBadRequest, fmt.Errorf("limit must be a positive integer"))
				return
			}
			query += ` LIMIT ?`
			args = append(args, limit)
			if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
				offset, err := strconv.Atoi(offsetStr)
				if err != nil || offset < 0 {
					writeError(w, http.StatusBadRequest, fmt.Errorf("offset must be a non-negative integer"))
					return
				}
				query += ` OFFSET ?`
				args = append(args, offset)
			}
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// An empty list has to marshal as [] rather than null: the frontend
		// maps over it, and an empty list is where every household starts.
		out := []income{}
		for rows.Next() {
			in, err := scanIncome(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, in)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateIncome(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// On by default: a request that says nothing about bollo_fattura
		// gets true, and decodeJSON only overwrites fields the body actually
		// names, so an explicit false still lands.
		in := income{BolloFattura: true}
		// A fractional amount_cents fails to decode into the int64 field, and
		// that is the intended answer: cents are whole, and a client sending
		// euros as a float is a bug to reject rather than round.
		if err := decodeJSON(w, r, &in); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if !checkIncome(w, db, &in) {
			return
		}

		res, err := db.Exec(`INSERT INTO income
			(amount_cents, category_id, client_id, payer, payment_date, invoice_sent_date, note, holding_id, contract_id, bollo_fattura, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.AmountCents, in.CategoryID, in.ClientID, in.Payer,
			nullDate(in.PaymentDate), nullDate(in.InvoiceSentDate), in.Note, in.HoldingID, in.ContractID, in.BolloFattura,
			now().Format(time.RFC3339))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if in.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, in)
	}
}

// handlePatchIncome edits one. Decoding onto the stored Income is what makes
// it partial: a field the body omits keeps the value it was read with, and one
// it sends — including an empty payment date, which is how a payment typed
// onto the wrong Income is taken back, and a null client_id, which is how a
// Client is cleared — replaces it.
//
// Recording a payment weeks after the invoice went out is this handler and
// nothing else: the same row grows a second date.
func handlePatchIncome(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := findIncome(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := in.ID
		if err := decodeJSON(w, r, &in); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		in.ID = id // an id in the body is not a way to move the row
		if !checkIncome(w, db, &in) {
			return
		}

		if _, err := db.Exec(`UPDATE income SET amount_cents = ?, category_id = ?, client_id = ?,
			payer = ?, payment_date = ?, invoice_sent_date = ?, note = ?, holding_id = ?, contract_id = ?, bollo_fattura = ? WHERE id = ?`,
			in.AmountCents, in.CategoryID, in.ClientID, in.Payer,
			nullDate(in.PaymentDate), nullDate(in.InvoiceSentDate), in.Note, in.HoldingID, in.ContractID, in.BolloFattura, in.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, in)
	}
}

// handleDeleteIncome removes one, for the Income recorded twice.
func handleDeleteIncome(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := findIncome(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM income WHERE id = ?`, in.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findIncome loads the Income the path names, writing the response itself when
// there is none — an unparseable id and a missing row are both a 404, because
// from outside they are the same thing: that Income is not there.
func findIncome(w http.ResponseWriter, db *sql.DB, rawID string) (income, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return income{}, false
	}
	in, err := scanIncome(db.QueryRow(incomeSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return income{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return income{}, false
	}
	return in, true
}

// The one projection every read of an Income uses, and the scan that matches
// it. The two nullable dates are flattened to ” on the way out, because that
// is the shape the API publishes and the shape a PATCH merges onto.
const incomeSelect = `SELECT id, amount_cents, category_id, client_id, payer,
	COALESCE(payment_date, ''), COALESCE(invoice_sent_date, ''), note, holding_id, contract_id, bollo_fattura FROM income`

func scanIncome(row interface{ Scan(...any) error }) (income, error) {
	var in income
	err := row.Scan(&in.ID, &in.AmountCents, &in.CategoryID, &in.ClientID, &in.Payer,
		&in.PaymentDate, &in.InvoiceSentDate, &in.Note, &in.HoldingID, &in.ContractID, &in.BolloFattura)
	return in, err
}

// nullDate turns the API's empty date into the NULL the column stores, so an
// unpaid Income has exactly one spelling in the database — which is what the
// totals of ticket 10 onward filter on.
func nullDate(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// validate rejects at the door what the rest of the app would otherwise have
// to keep asking about, and trims the free text on its way past. It touches no
// database, so every error it returns is a bad request and nothing else.
func (in *income) validate() error {
	in.Payer = strings.TrimSpace(in.Payer)
	in.Note = strings.TrimSpace(in.Note)
	in.PaymentDate = strings.TrimSpace(in.PaymentDate)
	in.InvoiceSentDate = strings.TrimSpace(in.InvoiceSentDate)

	if in.AmountCents <= 0 {
		return errors.New("an income needs an amount above zero")
	}
	// Ticket 08: "whose money was it" is never left unanswered going forward.
	// Checked after trimming, so " " is caught as the empty string it is —
	// an existing row saved before this rule still reads with an empty Payer,
	// because nothing here rewrites what is already stored; it only refuses a
	// write that would (re)create the gap.
	if in.Payer == "" {
		return errors.New("an income needs a payer")
	}
	if err := optionalDate("payment_date", in.PaymentDate); err != nil {
		return err
	}
	return optionalDate("invoice_sent_date", in.InvoiceSentDate)
}

// optionalDate accepts a date that may be absent, and nothing else. Absent is
// a real answer for both of an Income's dates — one is the unpaid state — but
// a date that is there is compared as text everywhere after this, so it has to
// be exactly the layout. The zero-padded layout makes time.Parse strict about
// both: it rejects "2026-3-5" for its shape and "2026-02-30" for its day.
func optionalDate(what, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(dateLayout, value); err != nil {
		return errors.New(what + " must be a real date as YYYY-MM-DD")
	}
	return nil
}

// checkIncome validates in and confirms what it references, writing the
// response itself on a refusal. Create and edit share it, so an edit cannot
// smuggle past a rule a create enforces.
//
// Both references are checked rather than left to the foreign keys: an FK
// would answer a bad id with a constraint failure, which writeError reads as a
// 409 — the right answer for a delete blocked by a row still pointing at it,
// and the wrong one for a write naming something that was never there. The
// Category check also catches what an FK cannot see at all: an expense-only
// Category is a real row, and still not somewhere income can land. It is the
// same gate an Expense and its Items pass, asking for the other direction.
func checkIncome(w http.ResponseWriter, db *sql.DB, in *income) bool {
	if err := in.validate(); err != nil {
		writeInvalid(w, err)
		return false
	}
	if !checkCategoryAccepts(w, db, in.CategoryID, appliesIncome,
		"that category is not one an income can go in") {
		return false
	}
	// Hidden is deliberately not part of it: hiding takes a Client out of the
	// pickers, not out of the app, and an Income being corrected months later
	// still belongs to whoever paid it.
	if in.ClientID != nil {
		switch ok, err := clientExists(db, *in.ClientID); {
		case err != nil:
			writeError(w, http.StatusInternalServerError, err)
			return false
		case !ok:
			writeInvalid(w, errors.New("that client does not exist"))
			return false
		}
	}
	// Linking is always an explicit choice, never a guess (spec, Out of
	// Scope), but it is not a free one: a Contract belongs to one Client, and
	// an Income linked to a Contract from some other Client would make that
	// Contract's received/accounted figures count money that was never its
	// own. Unlinked stays the default an omitted contract_id gives for free.
	if in.ContractID != nil {
		switch clientID, ok, err := contractClientID(db, *in.ContractID); {
		case err != nil:
			writeError(w, http.StatusInternalServerError, err)
			return false
		case !ok:
			writeInvalid(w, errors.New("that contract does not exist"))
			return false
		case in.ClientID == nil || *in.ClientID != clientID:
			writeInvalid(w, errors.New("a contract can only be linked to an income from its own client"))
			return false
		}
	}
	return true
}
