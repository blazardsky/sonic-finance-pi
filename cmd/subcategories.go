package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// A subcategory is a second, independent tag an Expense can carry alongside
// its Category — "Caffè" rather than a child of "Alimentari" specifically,
// because the same subcategory has to freely pair with whichever Category an
// entry actually used it under. There is no link to any Category anywhere in
// this table: pairing them is a fact about one Expense, not about the
// subcategory itself.
//
// Unlike category, nothing here is ever Base-protected: no report resolves a
// subcategory by identity, so every one of them is the household's to rename,
// hide or delete freely.
type subcategory struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AppliesTo string `json:"applies_to"`
	Hidden    bool   `json:"hidden"`
}

// migrateSubcategories is schema step 14: the table itself, plus the nullable
// subcategory_id an Expense and a Recurring expense can each optionally carry.
// No ON DELETE clause, the same as category's own foreign keys: deleting a
// subcategory still referenced by an Expense fails on the constraint itself,
// which writeError already turns into a 409 — the same free in-use guard
// handleDeleteCategory relies on.
func migrateSubcategories(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE subcategory (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applies_to TEXT NOT NULL CHECK (applies_to IN ('expense', 'income', 'both')),
		hidden     INTEGER NOT NULL DEFAULT 0 CHECK (hidden IN (0, 1))
	) STRICT`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE expense ADD COLUMN subcategory_id INTEGER REFERENCES subcategory(id)`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE recurring_expense ADD COLUMN subcategory_id INTEGER REFERENCES subcategory(id)`)
	return err
}

func handleListSubcategories(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(subcategorySelect + ` ORDER BY name COLLATE NOCASE`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		out := []subcategory{}
		for rows.Next() {
			s, err := scanSubcategory(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, s)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateSubcategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name      string `json:"name"`
			AppliesTo string `json:"applies_to"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s := subcategory{Name: strings.TrimSpace(body.Name), AppliesTo: body.AppliesTo}
		if err := s.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := db.Exec(`INSERT INTO subcategory (name, applies_to) VALUES (?, ?)`, s.Name, s.AppliesTo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if s.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, s)
	}
}

func handlePatchSubcategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := findSubcategory(w, db, r.PathValue("id"))
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
		if body.Name != nil {
			s.Name = strings.TrimSpace(*body.Name)
		}
		if body.AppliesTo != nil {
			s.AppliesTo = *body.AppliesTo
		}
		if body.Hidden != nil {
			s.Hidden = *body.Hidden
		}
		if err := s.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if _, err := db.Exec(`UPDATE subcategory SET name = ?, applies_to = ?, hidden = ? WHERE id = ?`,
			s.Name, s.AppliesTo, s.Hidden, s.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

func handleDeleteSubcategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := findSubcategory(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM subcategory WHERE id = ?`, s.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findSubcategory loads the Subcategory the path names, writing the response
// itself when there is none — an unparseable id and a missing row are both a
// 404, because from outside they are the same thing.
func findSubcategory(w http.ResponseWriter, db *sql.DB, rawID string) (subcategory, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return subcategory{}, false
	}
	s, err := scanSubcategory(db.QueryRow(subcategorySelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return subcategory{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return subcategory{}, false
	}
	return s, true
}

const subcategorySelect = `SELECT id, name, applies_to, hidden FROM subcategory`

func scanSubcategory(row interface{ Scan(...any) error }) (subcategory, error) {
	var s subcategory
	err := row.Scan(&s.ID, &s.Name, &s.AppliesTo, &s.Hidden)
	return s, err
}

// subcategoryAccepts reports whether the Subcategory exists and is one an
// entry of this kind can go in, the same rule categoryAccepts answers for
// Category. Hidden is not part of it, for the same reason: hiding takes a
// Subcategory out of the picker, not out of the app.
func subcategoryAccepts(db *sql.DB, id int64, applies string) (bool, error) {
	var found int
	err := db.QueryRow(`SELECT 1 FROM subcategory WHERE id = ? AND applies_to IN (?, ?)`,
		id, applies, appliesBoth).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// checkSubcategoryAccepts confirms one Subcategory exists and can hold money
// of this kind, writing the response itself on a refusal — mirrors
// checkCategoryAccepts.
func checkSubcategoryAccepts(w http.ResponseWriter, db *sql.DB, id int64, applies, refusal string) bool {
	switch ok, err := subcategoryAccepts(db, id, applies); {
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	case !ok:
		writeInvalid(w, errors.New(refusal))
	default:
		return true
	}
	return false
}

func (s subcategory) validate() error {
	if s.Name == "" {
		return errors.New("a subcategory needs a name")
	}
	switch s.AppliesTo {
	case appliesExpense, appliesIncome, appliesBoth:
		return nil
	}
	return errors.New("applies_to must be expense, income or both")
}
