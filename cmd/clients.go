package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// A Client is who money comes from, as one row rather than three spellings of
// the same name — which is what makes "has this Client paid me" a question the
// app can answer at all.
//
// It is deliberately not a freelance-only idea and carries no scope column:
// "Mum" for a birthday gift is as valid a Client as "Studio Rossi", and an
// Income names one or names nobody. An Income references a Client rather than
// copying its name, so a rename fixes every past Income at once — the same
// bargain a Category makes, and the reason hiding rather than deleting is how
// a Client is retired.
type client struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Hidden bool   `json:"hidden"`

	// The Income Category the Income form's picker prefills once this Client
	// is chosen — a suggestion only, never enforced: an Income from this
	// Client can still be saved under any other Income Category (ticket 04).
	// Nil when the Client has none set. A plain reference, like ClientID on an
	// Income: hiding or deleting the Category leaves this column exactly as
	// it is, and nothing here requires it to still accept Income.
	DefaultCategoryID *int64 `json:"default_category_id"`

	// The Payer the Income form's picker prefills once this Client is chosen,
	// on the same suggestion-never-enforced terms as DefaultCategoryID. Label
	// text from the settings list rather than a reference, the same bargain
	// income.payer makes: renaming the list leaves this reading as it was.
	// "" when the Client has none set.
	DefaultPayer string `json:"default_payer"`

	// The sum of this Client's received Incomes — payment_date set,
	// ADR-0003 — computed at read time and never stored (ticket 04). Every
	// read goes through clientSelect, so this is never stale; a PATCH must
	// not let a request smuggle a different number in, which is why
	// handlePatchClient restores it after decoding.
	TotalEarnedCents int64 `json:"total_earned_cents"`
}

// migrateClients is schema step 5. Nothing is seeded: who the household is
// paid by is not something the code can guess, unlike the Category list where
// thirteen guesses save an evening of typing.
func migrateClients(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE client (
		id     INTEGER PRIMARY KEY,
		name   TEXT NOT NULL,
		hidden INTEGER NOT NULL DEFAULT 0 CHECK (hidden IN (0, 1))
	) STRICT`)
	return err
}

// handleListClients returns every Client, hidden ones included: pickers filter
// on `hidden` themselves, and the management screen needs to see what it has
// hidden in order to unhide it. Ordered case-insensitively, because SQLite's
// default collation would otherwise sort "zia Carla" above "Banca".
//
// ponytail: unpaginated, like the Expense list. A household bills a couple of
// dozen Clients; add a filter when a screen needs one.
func handleListClients(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(clientSelect + ` ORDER BY name COLLATE NOCASE`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// An empty list has to marshal as [] rather than null: the frontend
		// maps over it, and an empty list is where every household starts.
		out := []client{}
		for rows.Next() {
			c, err := scanClient(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, c)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateClient(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only the one field a create can set: reading the whole struct would
		// let a POST ask for a Client that is already hidden, which is not a
		// state anything wants, and the column default covers the rest.
		var body struct {
			Name string `json:"name"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		c := client{Name: body.Name}
		if err := c.validate(); err != nil {
			writeInvalid(w, err)
			return
		}

		res, err := db.Exec(`INSERT INTO client (name) VALUES (?)`, c.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if c.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, c)
	}
}

// handlePatchClient renames a Client or hides it. Decoding onto the stored row
// is what makes it partial: hiding a Client says nothing about its name, and
// must not blank it.
//
// Nothing here is protected the way a Base category is: no report resolves a
// Client by identity, so every Client is the household's to rename.
func handlePatchClient(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := findClient(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := c.ID
		totalEarned := c.TotalEarnedCents // computed, not a stored column — see below
		if err := decodeJSON(w, r, &c); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		c.ID = id                        // an id in the body is not a way to move the row
		c.TotalEarnedCents = totalEarned // and nor is total_earned_cents a way to change it
		if err := c.validate(); err != nil {
			writeInvalid(w, err)
			return
		}

		if _, err := db.Exec(`UPDATE client SET name = ?, hidden = ?, default_category_id = ?, default_payer = ? WHERE id = ?`,
			c.Name, c.Hidden, c.DefaultCategoryID, c.DefaultPayer, c.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

// handleDeleteClient removes one, for the Client typed twice by mistake. A
// Client with Incomes behind it is retired by hiding instead, and the delete
// refuses with a 409 rather than removing the name off every Income that
// resolves through it: ticket 09's foreign key is what holds the line, and
// writeError is what turns it into a conflict.
func handleDeleteClient(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := findClient(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM client WHERE id = ?`, c.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findClient loads the Client the path names, writing the response itself when
// there is none — an unparseable id and a missing row are both a 404, because
// from outside they are the same thing: that Client is not there.
func findClient(w http.ResponseWriter, db *sql.DB, rawID string) (client, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return client{}, false
	}
	c, err := scanClient(db.QueryRow(clientSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return client{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return client{}, false
	}
	return c, true
}

// clientExists reports whether a Client row exists for id — checked
// explicitly rather than left to the foreign key, for the same reason
// categoryAccepts is: a bad id should read as an invalid request, not the 409
// writeError turns a constraint failure into.
func clientExists(db *sql.DB, id int64) (bool, error) {
	var found int
	err := db.QueryRow(`SELECT 1 FROM client WHERE id = ?`, id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// The subquery is the whole of ticket 04's total-earned figure: a plain SUM
// scoped to this Client and to received Incomes only (ADR-0003), run at read
// time rather than kept as a column that could drift from what income.go
// actually stores.
const clientSelect = `SELECT id, name, hidden, default_category_id, default_payer,
	(SELECT COALESCE(SUM(amount_cents), 0) FROM income
		WHERE income.client_id = client.id AND income.payment_date IS NOT NULL)
	FROM client`

func scanClient(row interface{ Scan(...any) error }) (client, error) {
	var c client
	err := row.Scan(&c.ID, &c.Name, &c.Hidden, &c.DefaultCategoryID, &c.DefaultPayer, &c.TotalEarnedCents)
	return c, err
}

// validate trims the name on its way past, so "Rossi " and "Rossi" are not two
// rows that look identical in a picker — the exact problem a Client exists to
// solve. No uniqueness constraint: two real Clients can share a name, and the
// household is the one who knows.
func (c *client) validate() error {
	if c.Name = strings.TrimSpace(c.Name); c.Name == "" {
		return errors.New("a client needs a name")
	}
	// Trimmed for the same reason the name is: the prefill has to match a
	// label in the settings list exactly, or the picker offers it twice.
	c.DefaultPayer = strings.TrimSpace(c.DefaultPayer)
	return nil
}
