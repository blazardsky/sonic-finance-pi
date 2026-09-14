package main

import (
	"database/sql"
	"errors"
	"fmt"
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
	// One of categoryColors, never empty — same palette and rules as
	// Category.Color (ADR-0016), added by the same migration step.
	Color string `json:"color"`
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
			Color     string `json:"color"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s := subcategory{Name: strings.TrimSpace(body.Name), AppliesTo: body.AppliesTo, Color: body.Color}
		if s.Color == "" {
			s.Color = defaultColor
		}
		if err := s.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := db.Exec(`INSERT INTO subcategory (name, applies_to, color) VALUES (?, ?, ?)`, s.Name, s.AppliesTo, s.Color)
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
			Color     *string `json:"color"`
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
		if body.Color != nil {
			s.Color = *body.Color
		}
		if err := s.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if _, err := db.Exec(`UPDATE subcategory SET name = ?, applies_to = ?, hidden = ?, color = ? WHERE id = ?`,
			s.Name, s.AppliesTo, s.Hidden, s.Color, s.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

// handleDeleteSubcategory behaves exactly as before with neither query
// parameter present: a bare DELETE, which the FK constraint turns into a 409
// via writeError when something still points at the row. Unlike Category, a
// Subcategory offers two ways forward instead of one, since it's optional
// everywhere it's used: replace_with (validated the same way replace_with is
// for Category, via checkSubcategoryAccepts) moves every reference onto
// another Subcategory before deleting; clear=true nulls those same references
// out instead, touching nothing else on the rows. If both are somehow sent
// together, replace_with wins — deliberately, not left undefined — and
// clear is simply ignored.
func handleDeleteSubcategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := findSubcategory(w, db, r.PathValue("id"))
		if !ok {
			return
		}

		if raw := r.URL.Query().Get("replace_with"); raw != "" {
			replaceID, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("replace_with must be a subcategory id"))
				return
			}
			if !checkSubcategoryAccepts(w, db, replaceID, s.AppliesTo,
				"replace_with must be an existing subcategory accepting the same side as the one being deleted") {
				return
			}
			if err := reassignSubcategoryTo(db, s.ID, replaceID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.URL.Query().Get("clear") == "true" {
			if err := clearSubcategoryFrom(db, s.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if _, err := db.Exec(`DELETE FROM subcategory WHERE id = ?`, s.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// reassignSubcategoryTo moves Expense's and Recurring expense's
// subcategory_id from oldID to newID, then removes the now-unreferenced old
// row, all in one transaction — mirrors reassignCategoryTo. category_id on
// those rows is never touched: pairing a Subcategory with a Category is a
// fact about one Expense, not about the Subcategory itself.
func reassignSubcategoryTo(db *sql.DB, oldID, newID int64) error {
	return runThenDelete(db, []execStmt{
		{`UPDATE expense SET subcategory_id = ? WHERE subcategory_id = ?`, []any{newID, oldID}},
		{`UPDATE recurring_expense SET subcategory_id = ? WHERE subcategory_id = ?`, []any{newID, oldID}},
	}, "subcategory", oldID)
}

// clearSubcategoryFrom nulls out Expense's and Recurring expense's
// subcategory_id wherever it points at id, then removes the row — the
// "remove the tag instead" choice ticket 01 gave Category no equivalent to,
// since a Category is never optional on those rows the way a Subcategory is.
// category_id is never touched, for the same reason reassignSubcategoryTo
// leaves it alone.
func clearSubcategoryFrom(db *sql.DB, id int64) error {
	return runThenDelete(db, []execStmt{
		{`UPDATE expense SET subcategory_id = NULL WHERE subcategory_id = ?`, []any{id}},
		{`UPDATE recurring_expense SET subcategory_id = NULL WHERE subcategory_id = ?`, []any{id}},
	}, "subcategory", id)
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

const subcategorySelect = `SELECT id, name, applies_to, hidden, color FROM subcategory`

func scanSubcategory(row interface{ Scan(...any) error }) (subcategory, error) {
	var s subcategory
	err := row.Scan(&s.ID, &s.Name, &s.AppliesTo, &s.Hidden, &s.Color)
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
	default:
		return errors.New("applies_to must be expense, income or both")
	}
	if !validColor(s.Color) {
		return fmt.Errorf("color must be one of %v", categoryColors)
	}
	return nil
}
