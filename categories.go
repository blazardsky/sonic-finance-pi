package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// A Category applies to Expenses, to Incomes, or to both, so that the expense
// picker never offers "Freelance" and the income picker never offers
// "Alimentari".
const (
	appliesExpense = "expense"
	appliesIncome  = "income"
	appliesBoth    = "both"
)

// The stable codes of the two Base categories. A non-null code is what marks a
// Category as one the code resolves by identity — the yearly tax summary reads
// these two — and therefore one that rename and delete refuse. Nothing else is
// protected: a seeded Category nothing resolves is the household's to edit.
// See ADR-0008.
const (
	codeFreelance = "freelance"
	codeTaxes     = "taxes"
)

// The names the two Base categories are seeded with. They are ordinary names
// after that — the household can hide either one, and the report still finds
// it by code — but the tests need something to address them by.
const (
	seedFreelanceName = "Freelance"
	seedTaxesName     = "Tasse"
)

// seedCategories is what a fresh database starts with, so that first use is
// not an evening of typing a list. Only the two with a code are protected.
var seedCategories = []category{
	{Name: seedTaxesName, AppliesTo: appliesExpense, code: codeTaxes},
	{Name: "Alimentari", AppliesTo: appliesExpense},
	{Name: "Casa", AppliesTo: appliesExpense},
	{Name: "Bollette", AppliesTo: appliesExpense},
	{Name: "Trasporti", AppliesTo: appliesExpense},
	{Name: "Salute", AppliesTo: appliesExpense},
	{Name: "Svago", AppliesTo: appliesExpense},
	{Name: "Ristoranti", AppliesTo: appliesExpense},
	{Name: "Abbigliamento", AppliesTo: appliesExpense},

	{Name: seedFreelanceName, AppliesTo: appliesIncome, code: codeFreelance},
	{Name: "Stipendio", AppliesTo: appliesIncome},
	{Name: "Regali", AppliesTo: appliesIncome},
	{Name: "Rimborsi", AppliesTo: appliesIncome},
	{Name: "Investimenti", AppliesTo: appliesIncome},

	{Name: "Altro", AppliesTo: appliesBoth},
}

type category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AppliesTo string `json:"applies_to"`
	Hidden    bool   `json:"hidden"`

	// Base is what the UI greys the rename and delete buttons on. The code
	// behind it is not published: it is an implementation detail of the
	// reports, and the household has no use for it.
	Base bool `json:"base"`

	code string
}

// migrateCategories is schema step 1: the table and its seed rows. Seeding
// inside the migration is what makes the list the household's own from then
// on — it runs once, so nothing a later edit removes ever comes back.
func migrateCategories(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE category (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applies_to TEXT NOT NULL CHECK (applies_to IN ('expense', 'income', 'both')),
		hidden     INTEGER NOT NULL DEFAULT 0 CHECK (hidden IN (0, 1)),
		code       TEXT UNIQUE
	) STRICT`); err != nil {
		return err
	}
	for _, c := range seedCategories {
		// A code is either a real code or absent: stored as "" it would
		// compare equal across every unprotected row and collide with the
		// UNIQUE constraint.
		code := sql.NullString{String: c.code, Valid: c.code != ""}
		if _, err := tx.Exec(`INSERT INTO category (name, applies_to, code) VALUES (?, ?, ?)`,
			c.Name, c.AppliesTo, code); err != nil {
			return err
		}
	}
	return nil
}

// handleListCategories returns every Category, hidden ones included: pickers
// filter on `hidden` themselves, and the management screen needs to see what
// it has hidden in order to unhide it. Ordered case-insensitively, because
// SQLite's default collation would otherwise sort "zia" above "Alimentari".
func handleListCategories(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(categorySelect + ` ORDER BY name COLLATE NOCASE`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// An empty list has to marshal as [] rather than null: the frontend
		// maps over it.
		out := []category{}
		for rows.Next() {
			c, err := scanCategory(rows)
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

func handleCreateCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name      string `json:"name"`
			AppliesTo string `json:"applies_to"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		c := category{Name: strings.TrimSpace(body.Name), AppliesTo: body.AppliesTo}
		if err := c.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := db.Exec(`INSERT INTO category (name, applies_to) VALUES (?, ?)`, c.Name, c.AppliesTo)
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

// handlePatchCategory applies whichever of name, applies_to and hidden the
// request carries. A Base category refuses the first two with 409 and accepts
// the third: the tax summary resolves it by identity, so tidying the list must
// not be able to break the report, but quitting freelancing should not leave a
// dead option in the picker forever.
//
// ADR-0008 names rename and delete. applies_to is refused for the same reason
// one step further out: the report reads Tasse on the Expense side, so an edit
// making it income-only leaves no Expense able to land in it — the report
// keeps resolving the Category and keeps finding nothing.
func handlePatchCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := findCategory(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		var body struct {
			Name      *string `json:"name"`
			AppliesTo *string `json:"applies_to"`
			Hidden    *bool   `json:"hidden"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if c.Base && (body.Name != nil || body.AppliesTo != nil) {
			writeError(w, http.StatusConflict, errors.New("a base category cannot be renamed or re-scoped"))
			return
		}

		if body.Name != nil {
			c.Name = strings.TrimSpace(*body.Name)
		}
		if body.AppliesTo != nil {
			c.AppliesTo = *body.AppliesTo
		}
		if body.Hidden != nil {
			c.Hidden = *body.Hidden
		}
		if err := c.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if _, err := db.Exec(`UPDATE category SET name = ?, applies_to = ?, hidden = ? WHERE id = ?`,
			c.Name, c.AppliesTo, c.Hidden, c.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

func handleDeleteCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := findCategory(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if c.Base {
			writeError(w, http.StatusConflict, errors.New("a base category cannot be deleted"))
			return
		}
		if _, err := db.Exec(`DELETE FROM category WHERE id = ?`, c.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findCategory loads the Category the path names, writing the response itself
// when there is none — an unparseable id and a missing row are both a 404,
// because from outside they are the same thing: that Category is not there.
func findCategory(w http.ResponseWriter, db *sql.DB, rawID string) (category, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return category{}, false
	}
	c, err := scanCategory(db.QueryRow(categorySelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return category{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return category{}, false
	}
	return c, true
}

// The one projection every read of a Category uses, and the scan that matches
// it. `code` is reduced to the boolean the API publishes: the code itself is
// an implementation detail of the reports that resolve by it.
const categorySelect = `SELECT id, name, applies_to, hidden, code IS NOT NULL FROM category`

func scanCategory(row interface{ Scan(...any) error }) (category, error) {
	var c category
	err := row.Scan(&c.ID, &c.Name, &c.AppliesTo, &c.Hidden, &c.Base)
	return c, err
}

func (c category) validate() error {
	if c.Name == "" {
		return errors.New("a category needs a name")
	}
	switch c.AppliesTo {
	case appliesExpense, appliesIncome, appliesBoth:
		return nil
	}
	return errors.New("applies_to must be expense, income or both")
}
