package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A recurringExpense is the rent, defined once. It carries an Expense's template
// fields — everything that will be copied onto each generated Expense — plus
// the three that make it recur: which day of the month it lands on, and the
// window it covers.
//
// The window IS the record: there is no active flag, because a boolean cannot
// say what was true in February. Deactivating sets EndMonth to the current
// month, and a report over an old month inside the window always regenerates
// the same rows, whenever it is run. See ADR-0005.
//
// Changing an amount is not an edit: it ends this one and starts another, so
// the months before the change keep the amount that was actually true then.
// That is a workflow the screen offers rather than a rule the API enforces —
// an amount typed wrong on the day it was created still has to be fixable.
//
// There is no Items breakdown. An Item is the exception on a receipt; a fixed
// monthly cost is one number under one Category, and nothing in the spec's
// schema carries a recurring one.
type recurringExpense struct {
	ID            int64  `json:"id"`
	AmountCents   int64  `json:"amount_cents"`
	CategoryID    int64  `json:"category_id"`
	Store         string `json:"store"`
	Payer         string `json:"payer"`
	PaymentMethod string `json:"payment_method"`
	Note          string `json:"note"`

	// The day the money leaves, 1–31. Ticket 13 clamps it to the month's
	// length on the way out — a 31 becomes the 30th in April — which is why
	// it is stored as the day that was meant rather than pre-clamped.
	DayOfMonth int `json:"day_of_month"`

	// The window, both months in monthLayout — zero-padded, so
	// `start_month <= M <= end_month` is a plain string comparison, which is
	// what ticket 13's generation will lean on entirely.
	//
	// EndMonth is empty rather than null when the Recurring is still running,
	// which is what lets a PATCH merge onto the stored row: an omitted field
	// keeps what it had, an empty one clears it. Stored as NULL, so "ongoing"
	// has exactly one spelling.
	StartMonth string `json:"start_month"`
	EndMonth   string `json:"end_month"`
}

// migrateRecurring is schema step 7. It also adds the expense column ticket
// 05's migration deliberately left out: recurring_id points at this table, so
// it could not exist until the table did, and adding it here makes it a real
// foreign key rather than a bare integer. Nothing writes it yet — ticket 13
// generates the Expenses that will carry it.
//
// end_month CHECKs against ” for the reason income's dates do: "ongoing" is
// the absence of an end month, and a column that could hold ” as well would
// have two spellings of it — one of which sorts below every real month and
// would quietly end the window in the year zero.
func migrateRecurring(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE recurring_expense (
		id             INTEGER PRIMARY KEY,
		amount_cents   INTEGER NOT NULL CHECK (amount_cents > 0),
		category_id    INTEGER NOT NULL REFERENCES category(id),
		store          TEXT NOT NULL DEFAULT '',
		payer          TEXT NOT NULL DEFAULT '',
		payment_method TEXT NOT NULL DEFAULT '',
		note           TEXT NOT NULL DEFAULT '',
		day_of_month   INTEGER NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
		start_month    TEXT NOT NULL,
		end_month      TEXT CHECK (end_month <> '' AND end_month >= start_month),
		created_at     TEXT NOT NULL
	) STRICT`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE expense
		ADD COLUMN recurring_id INTEGER REFERENCES recurring_expense(id)`)
	return err
}

// migrateSkips is schema step 8: the record of a generated Expense the
// household deleted. ADR-0005 — deleting one has to make it stay deleted, and
// generation is otherwise a pure function of the window, so the only place
// "the month I did not pay the rent" can be written down is a row of its own.
//
// The month is the whole of the value: (recurring_id, month) is the primary
// key, so re-skipping the same month is a no-op rather than a second row. ON
// DELETE CASCADE for the reason an Item's is — a skip has no life outside the
// Recurring expense it belongs to, and once the definition is gone there is
// nothing left to skip.
func migrateSkips(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE recurring_skip (
		recurring_id INTEGER NOT NULL REFERENCES recurring_expense(id) ON DELETE CASCADE,
		month        TEXT NOT NULL,
		PRIMARY KEY (recurring_id, month)
	) STRICT`)
	return err
}

// handleListRecurring returns every Recurring expense, running and ended
// alike. Which are still running is not a field: it is the window compared
// against the current month, and the screen makes that comparison against the
// phone's clock rather than the Pi's — the Pi has no RTC.
//
// Oldest first, unlike the entry lists: this is a list of fixed monthly costs,
// not of things just typed, and the one set up first is the one that has been
// paid longest. Unpaginated with no ponytail note, unlike the entry lists: a
// household has a handful of these in total, not a few thousand a year.
func handleListRecurring(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(recurringSelect + ` ORDER BY id`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// An empty list has to marshal as [] rather than null: the frontend
		// maps over it, and an empty list is where every household starts.
		out := []recurringExpense{}
		for rows.Next() {
			rec, err := scanRecurring(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, rec)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// handleCreateRecurring defines one. Both recurring-only fields default from
// the clock rather than being demanded of the form: switching the rent on in
// March means it starts in March, and the day it lands on is the day it is
// being set up. Either can be sent explicitly — a start month set back is how
// a cost that has been running since January is entered in March, and it is a
// deliberate answer, not an accident to guard against.
func handleCreateRecurring(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rec recurringExpense
		if err := decodeJSON(w, r, &rec); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if rec.StartMonth == "" {
			rec.StartMonth = now().Format(monthLayout)
		}
		if rec.DayOfMonth == 0 {
			rec.DayOfMonth = now().Day()
		}
		if !checkRecurring(w, db, &rec) {
			return
		}

		res, err := db.Exec(`INSERT INTO recurring_expense
			(amount_cents, category_id, store, payer, payment_method, note,
			 day_of_month, start_month, end_month, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			rec.AmountCents, rec.CategoryID, rec.Store, rec.Payer, rec.PaymentMethod,
			rec.Note, rec.DayOfMonth, rec.StartMonth, nullDate(rec.EndMonth),
			now().Format(time.RFC3339))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if rec.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, rec)
	}
}

// handlePatchRecurring edits one. Decoding onto the stored row is what makes
// it partial, the same bargain an Income makes: an omitted field keeps its
// value, and a sent one — including an empty end_month — replaces it.
//
// Deactivating is this handler and nothing else: the screen sends the current
// month as end_month, because that is all deactivating is. There is no
// separate endpoint for it and no flag to flip, per ADR-0005.
//
// The one merge this refuses is the reverse: clearing an end month that is
// already set. That is reactivation, and ADR-0005 answers it with a new
// definition rather than a reopened one — "the gap months genuinely had no
// payment", and generation would read the reopened window as a mandate to
// fill every one of them. An end month typed wrongly is still moveable; only
// removing it altogether is refused, because only that invents months.
func handlePatchRecurring(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec, ok := findRecurring(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id, ended := rec.ID, rec.EndMonth
		if err := decodeJSON(w, r, &rec); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		rec.ID = id // an id in the body is not a way to move the row
		if ended != "" && strings.TrimSpace(rec.EndMonth) == "" {
			writeInvalid(w, errors.New("a recurring expense that has ended is restarted by creating a new one, not by clearing its end month"))
			return
		}
		if !checkRecurring(w, db, &rec) {
			return
		}

		if _, err := db.Exec(`UPDATE recurring_expense SET amount_cents = ?, category_id = ?,
			store = ?, payer = ?, payment_method = ?, note = ?,
			day_of_month = ?, start_month = ?, end_month = ? WHERE id = ?`,
			rec.AmountCents, rec.CategoryID, rec.Store, rec.Payer, rec.PaymentMethod,
			rec.Note, rec.DayOfMonth, rec.StartMonth, nullDate(rec.EndMonth), rec.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, rec)
	}
}

// handleDeleteRecurring removes one that should never have existed. Once it
// has generated an Expense the foreign key refuses, and writeError turns that
// into a 409 — which is the right answer: the money did leave the account, so
// the way to stop a real Recurring expense is to end its window, not to erase
// the months it already produced.
func handleDeleteRecurring(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec, ok := findRecurring(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		if _, err := db.Exec(`DELETE FROM recurring_expense WHERE id = ?`, rec.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// findRecurring loads the Recurring expense the path names, writing the
// response itself when there is none — an unparseable id and a missing row are
// both a 404, because from outside they are the same thing.
func findRecurring(w http.ResponseWriter, db *sql.DB, rawID string) (recurringExpense, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return recurringExpense{}, false
	}
	rec, err := scanRecurring(db.QueryRow(recurringSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return recurringExpense{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return recurringExpense{}, false
	}
	return rec, true
}

// The one projection every read uses, and the scan that matches it. end_month
// is flattened to ” on the way out, because that is the shape the API
// publishes and the shape a PATCH merges onto.
const recurringSelect = `SELECT id, amount_cents, category_id, store, payer,
	payment_method, note, day_of_month, start_month, COALESCE(end_month, '')
	FROM recurring_expense`

func scanRecurring(row interface{ Scan(...any) error }) (recurringExpense, error) {
	var rec recurringExpense
	err := row.Scan(&rec.ID, &rec.AmountCents, &rec.CategoryID, &rec.Store, &rec.Payer,
		&rec.PaymentMethod, &rec.Note, &rec.DayOfMonth, &rec.StartMonth, &rec.EndMonth)
	return rec, err
}

// validate rejects at the door what generation would otherwise have to keep
// asking about, and trims the free text on its way past. It touches no
// database, so every error it returns is a bad request and nothing else.
func (rec *recurringExpense) validate() error {
	rec.Store = strings.TrimSpace(rec.Store)
	rec.Payer = strings.TrimSpace(rec.Payer)
	rec.PaymentMethod = strings.TrimSpace(rec.PaymentMethod)
	rec.Note = strings.TrimSpace(rec.Note)
	rec.StartMonth = strings.TrimSpace(rec.StartMonth)
	rec.EndMonth = strings.TrimSpace(rec.EndMonth)

	if rec.AmountCents <= 0 {
		return errors.New("a recurring expense needs an amount above zero")
	}
	// 29, 30 and 31 are all allowed: the day that was meant is what is
	// stored, and the short months are handled where the date is built.
	if rec.DayOfMonth < 1 || rec.DayOfMonth > 31 {
		return errors.New("day_of_month must be between 1 and 31")
	}
	if err := validMonth("start_month", rec.StartMonth); err != nil {
		return err
	}
	if rec.EndMonth == "" {
		return nil
	}
	if err := validMonth("end_month", rec.EndMonth); err != nil {
		return err
	}
	// A window that ends before it starts covers no month at all, which is
	// not a way to say "never ran" — it is a typo. Both are zero-padded, so
	// comparing them as text is comparing them as months.
	if rec.EndMonth < rec.StartMonth {
		return errors.New("end_month cannot be before start_month")
	}
	return nil
}

// validMonth accepts a month and nothing else. The zero-padded layout makes
// time.Parse strict about the shape, so "2026-3" is refused for the same
// reason "2026-3-5" is refused as a date: everything downstream compares
// these as text.
func validMonth(what, value string) error {
	if _, err := time.Parse(monthLayout, value); err != nil {
		return errors.New(what + " must be a real month as YYYY-MM")
	}
	return nil
}

// checkRecurring validates rec and confirms its Category, writing the response
// itself on a refusal. Create and edit share it, so an edit cannot smuggle
// past a rule a create enforces.
//
// The Category is checked rather than left to the foreign key for the reason
// an Expense's is: an FK answers a bad id with a constraint failure, which
// writeError reads as a 409, and it would not catch an income-only Category
// at all. What this defines is Expenses, so it answers to the Expense rule.
func checkRecurring(w http.ResponseWriter, db *sql.DB, rec *recurringExpense) bool {
	if err := rec.validate(); err != nil {
		writeInvalid(w, err)
		return false
	}
	return checkCategoryAccepts(w, db, rec.CategoryID, appliesExpense,
		"that category is not one an expense can go in")
}

// clockFloor is the date the Pi's clock is judged against, as a date rather
// than a time.Time because a time.Time cannot be a constant and this has to be
// one: nothing at runtime, and nothing in the test suite, is allowed to move
// the line generation refuses below. Compared as text, like every other date
// in this codebase.
//
// The Zero W has no real-time clock. It boots at 1970 until NTP answers, and
// 1970 is what this exists to catch — story 72, "so that a boot without
// network time does not write 1970 into my records". It is deliberately not
// the build date: a floor near the build would also refuse a Pi whose clock is
// merely stale by a few months, and for that Pi the months it would generate
// are real past months inside the window, which is the right answer rather
// than a wrong one. There is nothing to gain by refusing those and a household
// with no rent to lose by it.
//
// Refusing is the recoverable half of the choice: nothing is generated until
// the clock is right, and the next read after that generates everything the
// window owes. Generating against a wrong clock is what cannot be undone.
const clockFloor = "2020-01-01"

// clockSane is the one question every caller asks of the clock: is now late
// enough to be believed. /api/health publishes the answer so the screens can
// say so out loud, because a household seeing no rent needs to know why.
func clockSane(now func() time.Time) bool {
	return now().Format(dateLayout) >= clockFloor
}

// logClockUnset says it loudly, and says it the same way wherever it is said:
// once at startup, and again at every read that would have generated. Two
// wordings of this would have drifted, and this is the line whoever is
// wondering where the rent went has to be able to find.
func logClockUnset(now func() time.Time) {
	log.Printf("CLOCK UNSET: the clock reads %s, before %s — the Pi has booted without "+
		"network time. REFUSING to generate recurring expenses until it is corrected",
		now().Format(time.RFC3339), clockFloor)
}

// materialise creates the Expenses the given month owes, and is the whole of
// ADR-0005's "no scheduler": every read that covers a month calls this first,
// so the rent appears because someone looked, not because a timer fired on a
// Pi that may have been switched off. It is called for months long past as
// readily as for this one — a month nobody opened at the time is not
// permanently empty.
//
// Idempotent, and by one statement rather than by a check-then-insert: the
// NOT EXISTS pair is evaluated inside the INSERT that depends on it, so two
// screens loading the same month at once cannot both decide the rent is
// missing. What makes an Expense "already generated" is its recurring_id and
// its month, which is exactly the pair the spec keys generation on.
//
// Nothing is generated past the current month: next month's rent has not been
// paid, and an Expense is money that left.
func materialise(db *sql.DB, now func() time.Time, month string) error {
	if !clockSane(now) {
		logClockUnset(now)
		return nil
	}
	if month > now().Format(monthLayout) {
		return nil
	}
	// The clamp: a rent that leaves on the 31st leaves on the 30th in April,
	// and on the 28th in February. The month's length is arithmetic Go already
	// does — day 0 of the next month is the last of this one — so SQL is left
	// with a plain min() rather than a date expression to decode at 3am.
	start, err := time.Parse(monthLayout, month)
	if err != nil {
		return err
	}
	lastDay := start.AddDate(0, 1, 0).AddDate(0, 0, -1).Day()

	// Every template field is copied, because a generated Expense has to be
	// indistinguishable from a typed one — including recurring_id, which is
	// not published over the API and exists only so this statement can tell
	// what it already did, and so a delete knows which month to skip.
	_, err = db.Exec(`INSERT INTO expense
		(occurred_on, amount_cents, category_id, store, payer, payment_method,
		 note, recurring_id, created_at)
		SELECT printf('%s-%02d', ?, min(r.day_of_month, ?)), r.amount_cents, r.category_id,
			r.store, r.payer, r.payment_method, r.note, r.id, ?
		FROM recurring_expense r
		WHERE r.start_month <= ? AND ? <= COALESCE(r.end_month, ?)
		AND NOT EXISTS (SELECT 1 FROM expense e
			WHERE e.recurring_id = r.id AND substr(e.occurred_on, 1, 7) = ?)
		AND NOT EXISTS (SELECT 1 FROM recurring_skip s
			WHERE s.recurring_id = r.id AND s.month = ?)`,
		month, lastDay, now().Format(time.RFC3339),
		month, month, month, month, month)
	return err
}

// skipMonth writes down that a Recurring expense's month is not to be
// generated again. Which month is read from the stored row rather than from
// anything a client sent, and OR IGNORE because (recurring_id, month) is the
// primary key: a month skipped twice is skipped once.
//
// staying is the month the Expense is keeping — the month a PATCH is moving it
// to, or "" for a delete, which keeps none. That is the whole difference
// between the two callers: a correction inside the month skips nothing,
// because the rent is still sitting in the month it was paid in.
//
// A typed Expense has no recurring_id, so this inserts nothing for one and
// neither caller needs a branch to keep in step with its own write.
func skipMonth(tx *sql.Tx, expenseID int64, staying string) error {
	_, err := tx.Exec(`INSERT OR IGNORE INTO recurring_skip (recurring_id, month)
		SELECT recurring_id, substr(occurred_on, 1, 7) FROM expense
		WHERE id = ? AND recurring_id IS NOT NULL
		AND substr(occurred_on, 1, 7) <> ?`, expenseID, staying)
	return err
}
