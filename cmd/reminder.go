package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A Reminder is a short household-labeled toggle for a manual action done
// outside the app — a bank transfer, a payment made by hand — not tied to any
// Expense or Recurring expense (CONTEXT.md). Turned on once the labeled thing
// is done; reads as off again from the first of the next calendar month.
//
// SetForMonth is deliberately not part of the API: it is what collapse
// compares against now(), not something a screen has any use for — the same
// reason recurring_id is never published on an Expense.
type reminder struct {
	ID          int64   `json:"id"`
	Label       string  `json:"label"`
	Enabled     bool    `json:"enabled"`
	SetForMonth *string `json:"-"`
}

// handleListReminders answers every Reminder, each collapsed against the
// current month — ticket 06's whole mechanism: nothing ever flips the stored
// bit back to false on its own, this is what makes November's "did the
// transfer" read as off again in December with no scheduler involved
// (ADR-0005, the same shape materialise uses for Recurring expenses).
func handleListReminders(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(reminderSelect + ` ORDER BY id`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		current := now().Format(monthLayout)
		out := []reminder{}
		for rows.Next() {
			rem, err := scanReminder(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			rem.collapse(current)
			out = append(out, rem)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// handleCreateReminder defines one, always starting disabled: a Reminder is
// only ever turned on by the household doing the thing it names.
func handleCreateReminder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Label string `json:"label"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		rem := reminder{Label: strings.TrimSpace(body.Label)}
		if rem.Label == "" {
			writeInvalid(w, errors.New("a reminder needs a label"))
			return
		}

		res, err := db.Exec(`INSERT INTO reminder (label) VALUES (?)`, rem.Label)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if rem.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, rem)
	}
}

// handlePatchReminder is the only way a Reminder's state moves: enabling
// stamps the current month, disabling is a plain flip that leaves
// set_for_month alone (there's nothing to stamp when turning something off).
// Label is not editable here — ticket 06 gives a Reminder create and delete,
// not rename.
func handlePatchReminder(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rem, ok := findReminder(w, db, r.PathValue("id"))
		if !ok {
			return
		}

		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		rem.Enabled = body.Enabled
		if rem.Enabled {
			month := now().Format(monthLayout)
			rem.SetForMonth = &month
		}

		if _, err := db.Exec(`UPDATE reminder SET enabled = ?, set_for_month = ? WHERE id = ?`,
			rem.Enabled, rem.SetForMonth, rem.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, rem)
	}
}

// handleDeleteReminder removes one outright. Nothing else references a
// Reminder (spec, Reminders), so unlike Category/Client this needs no
// hide/archive path.
func handleDeleteReminder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rem, ok := findReminder(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM reminder WHERE id = ?`, rem.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findReminder loads the Reminder the path names, writing the response
// itself when there is none — an unparseable id and a missing row are both a
// 404, because from outside they are the same thing: that Reminder is not
// there. Uncollapsed: a PATCH needs the raw set_for_month to leave it alone
// when disabling.
func findReminder(w http.ResponseWriter, db *sql.DB, rawID string) (reminder, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return reminder{}, false
	}
	rem, err := scanReminder(db.QueryRow(reminderSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return reminder{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return reminder{}, false
	}
	return rem, true
}

const reminderSelect = `SELECT id, label, enabled, set_for_month FROM reminder`

func scanReminder(row interface{ Scan(...any) error }) (reminder, error) {
	var rem reminder
	err := row.Scan(&rem.ID, &rem.Label, &rem.Enabled, &rem.SetForMonth)
	return rem, err
}

// collapse is ticket 06's whole read-time rule: a stored true only reads as
// on when set_for_month is still the real current month. Whether the row
// itself gets written back to false is deliberately left alone (ponytail: no
// extra write on the read path) — what the API returns is the contract, and
// the next enable overwrites set_for_month anyway.
func (rem *reminder) collapse(current string) {
	if rem.SetForMonth == nil || *rem.SetForMonth != current {
		rem.Enabled = false
	}
}
