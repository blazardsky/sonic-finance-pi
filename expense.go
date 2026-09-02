package main

import (
	"database/sql"
	"errors"
	"net/http"
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
type expense struct {
	ID          int64  `json:"id"`
	OccurredOn  string `json:"occurred_on"`
	AmountCents int64  `json:"amount_cents"`
	CategoryID  int64  `json:"category_id"`
}

// migrateExpenses is schema step 2. It creates more columns than this ticket
// writes: a migration case can never be edited once a database has run it, and
// the fields 06 fills in need nothing that does not already exist, so they are
// cheaper as empty defaults here than as an ALTER TABLE later. The spec's
// remaining column, recurring_id, is the one that does need something new — it
// points at a table ticket 12 creates, and adding it there keeps it a real
// foreign key rather than a bare integer.
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
			var e expense
			if err := rows.Scan(&e.ID, &e.OccurredOn, &e.AmountCents, &e.CategoryID); err != nil {
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
		if err := e.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		// The Category is checked rather than left to the foreign key: the FK
		// would answer a bad id with a constraint failure, which is a 500
		// shape, and it would not catch an income-only Category at all.
		switch ok, err := categoryAccepts(db, e.CategoryID, appliesExpense); {
		case err != nil:
			writeError(w, http.StatusInternalServerError, err)
			return
		case !ok:
			writeError(w, http.StatusBadRequest, errors.New("that category is not one an expense can go in"))
			return
		}

		res, err := db.Exec(`INSERT INTO expense (occurred_on, amount_cents, category_id, created_at)
			VALUES (?, ?, ?, ?)`, e.OccurredOn, e.AmountCents, e.CategoryID, now().Format(time.RFC3339))
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

const expenseSelect = `SELECT id, occurred_on, amount_cents, category_id FROM expense`

// validate rejects at the door what the rest of the app would otherwise have
// to keep asking about. It touches no database, so every error it returns is a
// bad request and nothing else.
func (e *expense) validate() error {
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
