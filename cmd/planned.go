package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// A Planned purchase is something the household intends to buy soon
// (CONTEXT.md): not an Expense, no money has left. Position is its place in
// the household-ordered priority list, lower first — the only thing that
// decides which one gets Headroom first.
type plannedPurchase struct {
	ID          int64  `json:"id"`
	Label       string `json:"label"`
	AmountCents int64  `json:"amount_cents"`
	CategoryID  *int64 `json:"category_id"`
	Position    int64  `json:"position"`
}

func migratePlannedPurchase(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE planned_purchase (
		id           INTEGER PRIMARY KEY,
		label        TEXT    NOT NULL,
		amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
		category_id  INTEGER REFERENCES category(id),
		position     INTEGER NOT NULL
	)`)
	return err
}

const plannedSelect = `SELECT id, label, amount_cents, category_id, position FROM planned_purchase`

func scanPlanned(row interface{ Scan(...any) error }) (plannedPurchase, error) {
	var p plannedPurchase
	err := row.Scan(&p.ID, &p.Label, &p.AmountCents, &p.CategoryID, &p.Position)
	return p, err
}

// readPlannedPurchases answers the list in priority order — the order
// placement walks it in, so it is also the order the screen shows.
func readPlannedPurchases(db *sql.DB) ([]plannedPurchase, error) {
	rows, err := db.Query(plannedSelect + ` ORDER BY position, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []plannedPurchase{}
	for rows.Next() {
		p, err := scanPlanned(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func handleListPlanned(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := readPlannedPurchases(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

// plannedBody is what create and update both take: the whole editable shape,
// so an update is a replace, like the other record forms. Position is not
// part of it — only a reorder changes the order.
type plannedBody struct {
	Label       string `json:"label"`
	AmountCents int64  `json:"amount_cents"`
	CategoryID  *int64 `json:"category_id"`
}

// decodePlanned reads and validates the body, writing the response itself on
// a refusal.
func decodePlanned(w http.ResponseWriter, r *http.Request, db *sql.DB) (plannedBody, bool) {
	var body plannedBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return body, false
	}
	body.Label = strings.TrimSpace(body.Label)
	if body.Label == "" {
		writeInvalid(w, errors.New("a planned purchase needs a label"))
		return body, false
	}
	if body.AmountCents <= 0 {
		writeInvalid(w, errors.New("amount_cents must be positive"))
		return body, false
	}
	if body.CategoryID != nil && !checkCategoryAccepts(w, db, *body.CategoryID, appliesExpense,
		"category_id must be an existing expense category") {
		return body, false
	}
	return body, true
}

// handleCreatePlanned adds one at the bottom of the list: adding never
// reshuffles the household's priorities.
func handleCreatePlanned(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodePlanned(w, r, db)
		if !ok {
			return
		}
		res, err := db.Exec(`INSERT INTO planned_purchase (label, amount_cents, category_id, position)
			VALUES (?, ?, ?, (SELECT COALESCE(MAX(position), 0) + 1 FROM planned_purchase))`,
			body.Label, body.AmountCents, body.CategoryID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		id, err := res.LastInsertId()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		p, err := scanPlanned(db.QueryRow(plannedSelect+` WHERE id = ?`, id))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, p)
	}
}

func handleUpdatePlanned(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := findPlanned(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		body, ok := decodePlanned(w, r, db)
		if !ok {
			return
		}
		p.Label, p.AmountCents, p.CategoryID = body.Label, body.AmountCents, body.CategoryID
		if _, err := db.Exec(`UPDATE planned_purchase SET label = ?, amount_cents = ?, category_id = ? WHERE id = ?`,
			p.Label, p.AmountCents, p.CategoryID, p.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

// handleDeletePlanned removes one outright — whether it was bought (the
// screen deletes it once the Expense is saved) or just dropped. Nothing
// references a Planned purchase, and gaps in position are harmless: only the
// order matters.
func handleDeletePlanned(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := findPlanned(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM planned_purchase WHERE id = ?`, p.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleReorderPlanned takes the whole list in its new priority order — what
// a drag and drop leaves behind — and renumbers positions to match. The ids
// must be exactly the existing ones, each once: a list that has changed
// meanwhile (another screen added or deleted one) is refused, not merged.
func handleReorderPlanned(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			IDs []int64 `json:"ids"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		tx, err := db.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		existing := map[int64]bool{}
		rows, err := tx.Query(`SELECT id FROM planned_purchase`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			existing[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		mismatch := errors.New("ids must list every planned purchase exactly once")
		if len(body.IDs) != len(existing) {
			writeInvalid(w, mismatch)
			return
		}
		for _, id := range body.IDs {
			if !existing[id] {
				writeInvalid(w, mismatch)
				return
			}
			delete(existing, id) // a repeated id finds nothing the second time
		}

		for i, id := range body.IDs {
			if _, err := tx.Exec(`UPDATE planned_purchase SET position = ? WHERE id = ?`, i+1, id); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		list, err := readPlannedPurchases(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

// findPlanned loads the Planned purchase the path names, writing a 404 itself
// when there is none — an unparseable id and a missing row are the same thing
// from outside.
func findPlanned(w http.ResponseWriter, db *sql.DB, rawID string) (plannedPurchase, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return plannedPurchase{}, false
	}
	p, err := scanPlanned(db.QueryRow(plannedSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return plannedPurchase{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return plannedPurchase{}, false
	}
	return p, true
}
