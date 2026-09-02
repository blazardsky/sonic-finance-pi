package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// dateLayout is how a date crosses the API and how it is stored: the one
// format `<input type="date">` speaks, and the one that sorts correctly as
// text, which is what lets SQLite order a month without parsing anything.
const dateLayout = "2006-01-02"

// An expense is money leaving the household, on the date it left. Amounts are
// integer cents from the JSON in to the column out — no float ever holds one,
// because a euro like 12.34 has no exact binary form and a single rounding
// makes a month's total stop matching the entries under it.
//
// Store, Payer, Payment method and the note are what make a €42 line
// recognisable in November. Payer and PaymentMethod are the label text itself,
// not a reference into the list it was chosen from — ADR-0001 makes a Payer a
// label rather than an identity, so renaming one later must leave old entries
// reading exactly as they did. Category is the deliberate opposite: it is
// referenced, so a rename fixes every past entry at once.
type expense struct {
	ID            int64  `json:"id"`
	OccurredOn    string `json:"occurred_on"`
	AmountCents   int64  `json:"amount_cents"`
	CategoryID    int64  `json:"category_id"`
	Store         string `json:"store"`
	Payer         string `json:"payer"`
	PaymentMethod string `json:"payment_method"`
	Note          string `json:"note"`
}

// migrateExpenses is schema step 2. It created more columns than ticket 05
// wrote, because a migration case can never be edited once a database has run
// it: store, payer, payment_method and note were empty defaults waiting for
// ticket 06, which now fills them. The spec's remaining column, recurring_id,
// is the one that does need something new — it points at a table ticket 12
// creates, and adding it there keeps it a real foreign key rather than a bare
// integer.
//
// category_id is a foreign key because renaming a Category should fix every
// past entry at once; payer and payment_method are deliberately text, because
// renaming one of those labels must not rewrite what old entries say.
func migrateExpenses(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE expense (
		id             INTEGER PRIMARY KEY,
		occurred_on    TEXT NOT NULL,
		amount_cents   INTEGER NOT NULL CHECK (amount_cents > 0),
		category_id    INTEGER NOT NULL REFERENCES category(id),
		store          TEXT NOT NULL DEFAULT '',
		payer          TEXT NOT NULL DEFAULT '',
		payment_method TEXT NOT NULL DEFAULT '',
		note           TEXT NOT NULL DEFAULT '',
		tax_year       INTEGER,
		created_at     TEXT NOT NULL
	) STRICT`)
	return err
}

// handleListExpenses returns every Expense, newest first: the list sits under
// the add form, so what was just logged has to be the first thing on it. Ties
// on the date fall back to the id, which is insertion order.
//
// ponytail: unpaginated. A household logs a few thousand a year and the
// frontend has one screen; add a month filter when a report needs one.
func handleListExpenses(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(expenseSelect + ` ORDER BY occurred_on DESC, id DESC`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// An empty list has to marshal as [] rather than null: the frontend
		// maps over it.
		out := []expense{}
		for rows.Next() {
			e, err := scanExpense(rows)
			if err != nil {
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

func handleCreateExpense(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var e expense
		// A fractional amount_cents fails to decode into the int64 field, and
		// that is the intended answer: cents are whole, and a client sending
		// euros as a float is a bug to reject rather than round.
		if err := decodeJSON(w, r, &e); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if !checkExpense(w, db, &e) {
			return
		}

		res, err := db.Exec(`INSERT INTO expense
			(occurred_on, amount_cents, category_id, store, payer, payment_method, note, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			e.OccurredOn, e.AmountCents, e.CategoryID,
			e.Store, e.Payer, e.PaymentMethod, e.Note, now().Format(time.RFC3339))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if e.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, e)
	}
}

const expenseSelect = `SELECT id, occurred_on, amount_cents, category_id,
	store, payer, payment_method, note FROM expense`

func scanExpense(row interface{ Scan(...any) error }) (expense, error) {
	var e expense
	err := row.Scan(&e.ID, &e.OccurredOn, &e.AmountCents, &e.CategoryID,
		&e.Store, &e.Payer, &e.PaymentMethod, &e.Note)
	return e, err
}

// validate rejects at the door what the rest of the app would otherwise have
// to keep asking about, and trims the free text on its way past: the one place
// that decides what is valid is also the one place that decides what is
// stored. It touches no database, so every error it returns is a bad request
// and nothing else.
func (e *expense) validate() error {
	e.Store = strings.TrimSpace(e.Store)
	e.Payer = strings.TrimSpace(e.Payer)
	e.PaymentMethod = strings.TrimSpace(e.PaymentMethod)
	e.Note = strings.TrimSpace(e.Note)

	if e.AmountCents <= 0 {
		return errors.New("an expense needs an amount above zero")
	}
	// A date is compared as text everywhere after this, so it has to be
	// exactly the layout. The zero-padded layout makes time.Parse strict about
	// both: it rejects "2026-3-5" for its shape and "2026-02-30" for its day.
	if _, err := time.Parse(dateLayout, e.OccurredOn); err != nil {
		return errors.New("occurred_on must be a real date as YYYY-MM-DD")
	}
	return nil
}

// handlePatchExpense edits one. Decoding onto the stored Expense is what makes
// it partial: a field the body omits keeps the value it was read with, and one
// it sends — including an empty note, which is how a note is cleared —
// replaces it.
//
// Payer and Payment method are not checked against the configured lists, and
// must not be: an Expense saved under a label since renamed away still has to
// be editable, or correcting its amount would mean losing what it says.
func handlePatchExpense(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, ok := findExpense(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := e.ID
		if err := decodeJSON(w, r, &e); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		e.ID = id // an id in the body is not a way to move the row
		if !checkExpense(w, db, &e) {
			return
		}

		if _, err := db.Exec(`UPDATE expense SET occurred_on = ?, amount_cents = ?, category_id = ?,
			store = ?, payer = ?, payment_method = ?, note = ? WHERE id = ?`,
			e.OccurredOn, e.AmountCents, e.CategoryID,
			e.Store, e.Payer, e.PaymentMethod, e.Note, e.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, e)
	}
}

// handleDeleteExpense removes one, so a duplicate entry does not distort the
// month.
func handleDeleteExpense(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, ok := findExpense(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM expense WHERE id = ?`, e.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findExpense loads the Expense the path names, writing the response itself
// when there is none — an unparseable id and a missing row are both a 404,
// because from outside they are the same thing: that Expense is not there.
func findExpense(w http.ResponseWriter, db *sql.DB, rawID string) (expense, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return expense{}, false
	}
	e, err := scanExpense(db.QueryRow(expenseSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return expense{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return expense{}, false
	}
	return e, true
}

// checkExpense validates e and confirms its Category, writing the response
// itself on a refusal. Create and edit share it, so an edit cannot smuggle
// past a rule a create enforces.
//
// The Category is checked rather than left to the foreign key: the FK would
// answer a bad id with a constraint failure, which is a 500 shape, and it
// would not catch an income-only Category at all.
func checkExpense(w http.ResponseWriter, db *sql.DB, e *expense) bool {
	if err := e.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	switch ok, err := categoryAccepts(db, e.CategoryID, appliesExpense); {
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
		return false
	case !ok:
		writeError(w, http.StatusBadRequest, errors.New("that category is not one an expense can go in"))
		return false
	}
	return true
}
