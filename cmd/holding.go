package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// The fixed list a Holding's type is picked from. Unlike Store, it must match
// exactly — CONTEXT.md — because the portfolio breakdown (ticket 03) groups by
// it.
const (
	holdingETF    = "etf"
	holdingCrypto = "crypto"
	holdingStock  = "stock"
	holdingBond   = "bond"
	holdingOther  = "other"
)

// A Holding is a specific stock, ETF, crypto asset, bond or other investment
// vehicle the household buys and sells — never priced or revalued by the app,
// per CONTEXT.md. Buying and selling one is recorded as an ordinary Expense or
// Income under the Investments base category (ticket 03); this ticket is only
// the fixed, household-maintained list itself.
type holding struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// migrateInvestments is schema step 9: the Holding table, the nullable
// holding_id an Expense/Income/Recurring expense can carry, and the existing
// "Investimenti" category promoted in place to the new protected Investments
// base category. One step, not three, because a database with any of these
// and not the others is not a state ticket 03 can build on.
func migrateInvestments(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE holding (
		id   INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL CHECK (type IN ('etf', 'crypto', 'stock', 'bond', 'other'))
	) STRICT`); err != nil {
		return err
	}

	// holding_id is meaningful only when the row's Category is the Investments
	// base category — the same relationship tax_year already has to Taxes. Not
	// copied by anything yet: ticket 03 is what starts writing it.
	if _, err := tx.Exec(`ALTER TABLE expense ADD COLUMN holding_id INTEGER REFERENCES holding(id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE income ADD COLUMN holding_id INTEGER REFERENCES holding(id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE recurring_expense ADD COLUMN holding_id INTEGER REFERENCES holding(id)`); err != nil {
		return err
	}

	// The row is updated in place, never inserted again: a second "Investimenti"
	// would leave the category list with two confusingly similar entries. The
	// `code IS NULL` guard is defense in depth rather than load-bearing —
	// migrateStep never re-runs a completed step — for a row already promoted
	// some other way.
	_, err := tx.Exec(`UPDATE category SET applies_to = ?, code = ? WHERE name = ? AND code IS NULL`,
		appliesBoth, codeInvestments, seedInvestmentiName)
	return err
}

// handleListHoldings returns every Holding. Unlike Category and Client there
// is no hidden flag in this version (see the spec's Out of Scope), so there is
// nothing for a picker to filter out.
//
// ponytail: unpaginated, like Client — a household holds a few dozen Holdings
// at most.
func handleListHoldings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(holdingSelect + ` ORDER BY name COLLATE NOCASE`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// An empty list has to marshal as [] rather than null: the frontend
		// maps over it, and an empty list is where every household starts.
		out := []holding{}
		for rows.Next() {
			h, err := scanHolding(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, h)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateHolding(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var h holding
		if err := decodeJSON(w, r, &h); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := h.validate(); err != nil {
			writeInvalid(w, err)
			return
		}

		res, err := db.Exec(`INSERT INTO holding (name, type) VALUES (?, ?)`, h.Name, h.Type)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if h.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, h)
	}
}

// handlePatchHolding renames a Holding or changes its type. Decoding onto the
// stored row is what makes it partial, the same bargain a Client's PATCH
// makes. Nothing here is protected the way a Base category is: no report
// resolves one Holding by identity, so every Holding is the household's to
// rename.
func handlePatchHolding(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, ok := findHolding(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := h.ID
		if err := decodeJSON(w, r, &h); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h.ID = id // an id in the body is not a way to move the row
		if err := h.validate(); err != nil {
			writeInvalid(w, err)
			return
		}

		if _, err := db.Exec(`UPDATE holding SET name = ?, type = ? WHERE id = ?`,
			h.Name, h.Type, h.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, h)
	}
}

// findHolding loads the Holding the path names, writing the response itself
// when there is none — an unparseable id and a missing row are both a 404,
// because from outside they are the same thing: that Holding is not there.
func findHolding(w http.ResponseWriter, db *sql.DB, rawID string) (holding, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return holding{}, false
	}
	h, err := scanHolding(db.QueryRow(holdingSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return holding{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return holding{}, false
	}
	return h, true
}

const holdingSelect = `SELECT id, name, type FROM holding`

func scanHolding(row interface{ Scan(...any) error }) (holding, error) {
	var h holding
	err := row.Scan(&h.ID, &h.Name, &h.Type)
	return h, err
}

// validate trims the name on its way past, so "VWCE " and "VWCE" are not two
// rows the portfolio breakdown would silently split one holding's percentage
// across (CONTEXT.md), and checks type against the fixed list the CHECK
// constraint also enforces — checked here first so a bad type reads as a 400
// rather than the 500 writeError would otherwise turn a CHECK failure into.
func (h *holding) validate() error {
	if h.Name = strings.TrimSpace(h.Name); h.Name == "" {
		return errors.New("a holding needs a name")
	}
	switch h.Type {
	case holdingETF, holdingCrypto, holdingStock, holdingBond, holdingOther:
		return nil
	}
	return errors.New("type must be one of etf, crypto, stock, bond, other")
}
