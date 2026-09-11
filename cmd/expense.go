package main

import (
	"database/sql"
	"errors"
	"fmt"
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
	Items         []item `json:"items"`

	// The year this payment's tax relates to, and 0 on every Expense that is
	// not a tax one. Tax on 2026's income is paid during 2027, so the year the
	// money left is not the year the summary attributes it to — hence a field
	// rather than four characters of occurred_on (ADR-0008).
	//
	// It is derived rather than trusted: checkExpense defaults it to the year
	// of the payment and clears it outside a tax Category. It is an override
	// of that default rather than the only place the default lives — a
	// generated Expense goes nowhere near this handler, so the summary applies
	// the same rule to a NULL column (see handleTaxSummary).
	TaxYear int `json:"tax_year"`

	// The Holding this Expense bought, and nil on every Expense that is not a
	// buy — meaningful only under the Investments base category (ticket 03),
	// the same relationship TaxYear has to Taxes. Unlike TaxYear, nothing
	// defaults or clears it: it is set directly on insert like CategoryID, and
	// simply never read by a report outside Investments (spec's Implementation
	// Decisions). A pointer, like income.ClientID, because "no Holding" is a
	// real answer for every non-Investments Expense.
	HoldingID *int64 `json:"holding_id"`
}

// The fixed list a quantity's unit is picked from, the same one-of-a-fixed-
// list bargain a Holding's type makes (ticket 01).
const (
	itemUnitKg    = "kg"
	itemUnitLt    = "lt"
	itemUnitPiece = "piece"
)

// An item is a part of an Expense that belongs under a different Category — a
// book bought during the grocery shop. Items are optional and partial: they
// never have to account for the whole Expense, and whatever they do not cover
// stays under the Expense's own Category. The Expense's amount is never
// derived from them; see ADR-0002.
//
// An Item carries no id across the API, because nothing addresses one: they
// are saved as a set with their Expense, an edit replaces the whole
// breakdown, and so what a request sends is exactly what a later one reads.
type item struct {
	Name        string `json:"name"`
	AmountCents int64  `json:"amount_cents"`
	CategoryID  int64  `json:"category_id"`

	// Ticket 01: how much of the Item was bought, and the unit it was bought
	// in — both present or both absent, refused otherwise (validate). Quantity
	// is a pointer because 0 is not "no quantity", the same reason HoldingID
	// is one; Unit is a plain string with "" as its absence, since it is
	// picked from a fixed list rather than a foreign key. Nil/"" is what every
	// Item saved before this ticket now reads as.
	Quantity *float64 `json:"quantity"`
	Unit     string   `json:"unit"`

	// Purely informational — ADR-0014. Defaults to false and never affects
	// AmountCents or PricePerUnit.
	Discounted bool `json:"discounted"`

	// The derived price per unit (ADR-0014): computed from AmountCents and
	// Quantity wherever an Item is read, nil whenever Quantity is nil, and
	// never stored. A request's own price_per_unit is never trusted —
	// computePricePerUnit overwrites it unconditionally, the same bargain
	// TotalEarnedCents makes on a Client.
	PricePerUnit *float64 `json:"price_per_unit"`
}

// computePricePerUnit fills PricePerUnit from AmountCents and Quantity. It is
// called after validate has already refused a Quantity/Unit mismatch, so it
// never has to guess what an inconsistent pair means — and it is called on
// every read, so what a request sent for price_per_unit never matters.
func (it *item) computePricePerUnit() {
	if it.Quantity == nil {
		it.PricePerUnit = nil
		return
	}
	ppu := float64(it.AmountCents) / *it.Quantity
	it.PricePerUnit = &ppu
}

// migrateItemPricing is schema step 12 (ticket 01): item gains an optional
// quantity/unit pair and a purely informational discounted flag. No column
// gets a value that invents information nobody typed — quantity and unit stay
// NULL and discounted defaults to false, so every Item saved before this ran
// reads exactly as an ordinary one still does.
//
// No CHECK ties quantity to unit, the same reason migrateItems has no CHECK
// against the Expense's total: a row constraint cannot see its own pair
// meaningfully either way round, so that rule is enforced on the way in, in
// validate, where a refusal can say what was wrong.
func migrateItemPricing(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE item ADD COLUMN quantity REAL CHECK (quantity IS NULL OR quantity > 0)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE item ADD COLUMN unit TEXT CHECK (unit IS NULL OR unit IN ('kg', 'lt', 'piece'))`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE item ADD COLUMN discounted INTEGER NOT NULL DEFAULT 0 CHECK (discounted IN (0, 1))`)
	return err
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

// migrateItems is schema step 4. ON DELETE CASCADE is the schema saying what
// ADR-0002 says: an Item has no life outside its Expense, so deleting the
// Expense takes the breakdown with it rather than failing on the reference.
//
// There is no CHECK against the Expense's total, because a row constraint
// cannot see its siblings' amounts. That sum is enforced on the way in, where
// the refusal can say what was wrong.
func migrateItems(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE item (
		id           INTEGER PRIMARY KEY,
		expense_id   INTEGER NOT NULL REFERENCES expense(id) ON DELETE CASCADE,
		name         TEXT NOT NULL,
		amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
		category_id  INTEGER NOT NULL REFERENCES category(id)
	) STRICT`)
	return err
}

// handleListExpenses returns every Expense, newest first: the list sits under
// the add form, so what was just logged has to be the first thing on it. Ties
// on the date fall back to the id, which is insertion order.
//
// Optional query params, all off by default (a bare GET still returns every
// Expense unfiltered — Investments' own history read depends on that):
// `year` (YYYY) scopes to one year, and `limit`/`offset` page through
// whatever `year` (or the absence of it) already selected. The frontend uses
// `limit` alone for its default "last 50" view and adds `year`+`offset` only
// once a household actually pages into an older year — the ponytail this
// replaces ("unpaginated... add a filter when a report needs one") named this
// exact need.
func handleListExpenses(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := itemsByExpense(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		query := expenseSelect
		var args []any
		if year := r.URL.Query().Get("year"); year != "" {
			if len(year) != 4 {
				writeError(w, http.StatusBadRequest, fmt.Errorf("year must be 4 digits"))
				return
			}
			if _, err := strconv.Atoi(year); err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("year must be numeric"))
				return
			}
			query += ` WHERE occurred_on LIKE ?`
			args = append(args, year+"-%")
		}
		query += ` ORDER BY occurred_on DESC, id DESC`
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				writeError(w, http.StatusBadRequest, fmt.Errorf("limit must be a positive integer"))
				return
			}
			query += ` LIMIT ?`
			args = append(args, limit)
			if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
				offset, err := strconv.Atoi(offsetStr)
				if err != nil || offset < 0 {
					writeError(w, http.StatusBadRequest, fmt.Errorf("offset must be a non-negative integer"))
					return
				}
				query += ` OFFSET ?`
				args = append(args, offset)
			}
		}

		rows, err := db.Query(query, args...)
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
			e.Items = append(e.Items, items[e.ID]...)
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

		// One transaction, because an Expense and its breakdown are one save:
		// a failure partway must not leave a €62 shop with half a book in it.
		tx, err := db.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(`INSERT INTO expense
			(occurred_on, amount_cents, category_id, store, payer, payment_method, note,
			 tax_year, holding_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.OccurredOn, e.AmountCents, e.CategoryID, e.Store, e.Payer, e.PaymentMethod,
			e.Note, nullYear(e.TaxYear), e.HoldingID, now().Format(time.RFC3339))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if e.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := syncStoreFTS(tx, e.Store); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := insertItems(tx, e.ID, e.Items); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, e)
	}
}

const expenseSelect = `SELECT id, occurred_on, amount_cents, category_id,
	store, payer, payment_method, note, COALESCE(tax_year, 0), holding_id FROM expense`

func scanExpense(row interface{ Scan(...any) error }) (expense, error) {
	// The frontend maps over the breakdown, so an Expense without one has to
	// marshal as [] rather than null. The caller fills in any Items there are.
	e := expense{Items: []item{}}
	err := row.Scan(&e.ID, &e.OccurredOn, &e.AmountCents, &e.CategoryID,
		&e.Store, &e.Payer, &e.PaymentMethod, &e.Note, &e.TaxYear, &e.HoldingID)
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
	// Ticket 08: "whose money was it" is never left unanswered going forward.
	// Checked after trimming, so " " is caught as the empty string it is —
	// an existing row saved before this rule still reads with an empty Payer,
	// because nothing here rewrites what is already stored; it only refuses a
	// write that would (re)create the gap.
	if e.Payer == "" {
		return errors.New("an expense needs a payer")
	}
	// A date is compared as text everywhere after this, so it has to be
	// exactly the layout. The zero-padded layout makes time.Parse strict about
	// both: it rejects "2026-3-5" for its shape and "2026-02-30" for its day.
	if _, err := time.Parse(dateLayout, e.OccurredOn); err != nil {
		return errors.New("occurred_on must be a real date as YYYY-MM-DD")
	}
	// 0 is the absence of a Tax year, and anything else has to be a year the
	// summary can be asked for: the column takes any integer, so a 26 typed
	// for 2026 would otherwise be stored and attributed to nothing at all.
	// Through validYear rather than a range of its own, so "a year" has one
	// definition in this codebase and not two that could drift.
	if e.TaxYear != 0 {
		if err := validYear("tax_year", strconv.Itoa(e.TaxYear)); err != nil {
			return err
		}
	}

	// The breakdown marshals as [] rather than null, and is checked against
	// the Expense's own amount — never the other way around. ADR-0002: a
	// partial itemisation must not silently shrink a €62 shop into a €14 one,
	// and a remainder below zero has no meaning, so it is refused instead.
	if e.Items == nil {
		e.Items = []item{}
	}
	var covered int64
	for i := range e.Items {
		it := &e.Items[i]
		it.Name = strings.TrimSpace(it.Name)
		if it.Name == "" {
			return errors.New("every item needs a name")
		}
		if it.AmountCents <= 0 {
			return fmt.Errorf("the item %q needs an amount above zero", it.Name)
		}
		// Ticket 01: quantity and unit are a pair — a quantity with no unit
		// does not say what it counts, and a unit with no quantity has
		// nothing to divide by — so both present or both absent is the only
		// shape accepted.
		if (it.Quantity == nil) != (it.Unit == "") {
			return fmt.Errorf("the item %q needs both a quantity and a unit, or neither", it.Name)
		}
		if it.Quantity != nil {
			if *it.Quantity <= 0 {
				return fmt.Errorf("the item %q needs a quantity above zero", it.Name)
			}
			switch it.Unit {
			case itemUnitKg, itemUnitLt, itemUnitPiece:
			default:
				return fmt.Errorf("the item %q has a unit that must be kg, lt or piece", it.Name)
			}
		}
		// Computed here, after the pair above is known consistent, and
		// unconditionally: whatever a request sent for price_per_unit is
		// replaced rather than trusted.
		it.computePricePerUnit()
		// Compared inside the loop, so covered never runs past one item's
		// amount above the total and an int64 has no chance to overflow.
		if covered += it.AmountCents; covered > e.AmountCents {
			return errors.New("the items add up to more than the expense total")
		}
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
//
// Moving a generated Expense to another month skips the month it left, for the
// reason deleting one does: generation asks whether that (Recurring expense,
// month) has an Expense, and after the move it does not — so the next look at
// the old month would put a second rent there for a payment that happened
// once. Correcting a date inside the month skips nothing.
func handlePatchExpense(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, ok := findExpense(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := e.ID
		// Items are the one field decoding cannot merge into: an Item has no
		// id, so a shorter list sent for a longer stored one would leave the
		// old names and amounts showing through the gaps. The breakdown is
		// emptied first and replaced wholesale, and only put back if the body
		// turned out to say nothing about it.
		stored := e.Items
		e.Items = nil
		if err := decodeJSON(w, r, &e); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if e.Items == nil {
			e.Items = stored
		}
		e.ID = id // an id in the body is not a way to move the row
		if !checkExpense(w, db, &e) {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		// Before the UPDATE, while the stored row still says which month it is
		// leaving. OccurredOn is the validated layout by now, so its first
		// seven characters are the month it is moving to.
		if err := skipMonth(tx, e.ID, e.OccurredOn[:len(monthLayout)]); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if _, err := tx.Exec(`UPDATE expense SET occurred_on = ?, amount_cents = ?, category_id = ?,
			store = ?, payer = ?, payment_method = ?, note = ?, tax_year = ?, holding_id = ? WHERE id = ?`,
			e.OccurredOn, e.AmountCents, e.CategoryID, e.Store, e.Payer, e.PaymentMethod,
			e.Note, nullYear(e.TaxYear), e.HoldingID, e.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := syncStoreFTS(tx, e.Store); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if _, err := tx.Exec(`DELETE FROM item WHERE expense_id = ?`, e.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := insertItems(tx, e.ID, e.Items); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, e)
	}
}

// handleDeleteExpense removes one, so a duplicate entry does not distort the
// month — including a generated one, which is deletable under exactly the same
// rules as a typed one and answers the same 204.
//
// What is different is what deleting a generated one leaves behind: a skip for
// its (Recurring expense, month), because generation is otherwise a pure
// function of the window and the next look at the month would put the rent
// straight back. ADR-0005 — the month the rent was not paid has to stay the
// month the rent was not paid.
//
// One transaction, because a skip written without the delete would silence a
// rent that is still there. skipMonth is shared with the edit below, which has
// the same problem the moment it moves a generated Expense out of its month.
func handleDeleteExpense(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, ok := findExpense(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		// Before the delete, while the row is still there to be read: a
		// deleted Expense is keeping no month at all.
		if err := skipMonth(tx, e.ID, ""); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if _, err := tx.Exec(`DELETE FROM expense WHERE id = ?`, e.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := tx.Commit(); err != nil {
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

	// The breakdown comes with it: an edit that says nothing about Items has
	// to keep them, and one that replaces them has to answer with what stuck.
	items, err := itemsByExpense(db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return expense{}, false
	}
	e.Items = append(e.Items, items[e.ID]...)
	return e, true
}

// itemsByExpense loads every Expense's breakdown at once, grouped by the
// Expense it belongs to and each group in the order it was saved. One query
// for the whole table rather than one per Expense: a household's entire
// history of Items is a few hundred rows, which is cheaper to read whole than
// to filter twice.
func itemsByExpense(db *sql.DB) (map[int64][]item, error) {
	rows, err := db.Query(`SELECT expense_id, name, amount_cents, category_id, quantity, unit, discounted
		FROM item ORDER BY expense_id, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64][]item{}
	for rows.Next() {
		var id int64
		var it item
		var unit sql.NullString
		if err := rows.Scan(&id, &it.Name, &it.AmountCents, &it.CategoryID,
			&it.Quantity, &unit, &it.Discounted); err != nil {
			return nil, err
		}
		it.Unit = unit.String
		it.computePricePerUnit()
		out[id] = append(out[id], it)
	}
	return out, rows.Err()
}

// insertItems writes an Expense's breakdown. Items are always written as a
// whole set — that is what lets one carry no id of its own.
func insertItems(tx *sql.Tx, expenseID int64, items []item) error {
	for _, it := range items {
		if _, err := tx.Exec(`INSERT INTO item (expense_id, name, amount_cents, category_id, quantity, unit, discounted)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, expenseID, it.Name, it.AmountCents, it.CategoryID,
			it.Quantity, nullString(it.Unit), it.Discounted); err != nil {
			return err
		}
		// Ticket 03: item_name_fts is an append-only vocabulary of every name
		// ever typed, kept in step with a plain INSERT beside the item's own
		// — see migrateItemStoreFTS.
		if _, err := tx.Exec(`INSERT INTO item_name_fts(name) VALUES (?)`, it.Name); err != nil {
			return err
		}
	}
	return nil
}

// nullString is nullYear for Unit: "" (no quantity/unit pair) has to store
// NULL, not the empty string, since the CHECK constraint only allows the
// fixed list or NULL.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullYear is nullDate for a Tax year: an Expense that has none stores NULL
// rather than 0, so a summary asking for a year cannot match one that is not a
// tax payment at all.
func nullYear(year int) any {
	if year == 0 {
		return nil
	}
	return year
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
		writeInvalid(w, err)
		return false
	}
	if !checkCategoryAccepts(w, db, e.CategoryID, appliesExpense,
		"that category is not one an expense can go in") {
		return false
	}
	// The Tax year is derived here rather than taken on trust, which is what
	// makes "an Expense with a tax_year" and "an Expense the tax summary
	// counts" the same set of rows. Outside a tax Category it is cleared —
	// including on an Expense moved out of one, which would otherwise keep a
	// year nothing reads until the day it is moved back — and inside one it
	// defaults to the year of the payment, editable because tax on 2026's
	// income is paid during 2027 (ADR-0008).
	//
	// The first four characters of a date are its year: OccurredOn is the
	// validated layout by now, and validate() has already refused anything
	// else. The error is unreachable for the same reason, so it is dropped.
	// Sliced by the layout's own length, like every other date slice here.
	tax, err := isTaxCategory(db, e.CategoryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return false
	}
	switch {
	case !tax:
		e.TaxYear = 0
	case e.TaxYear == 0:
		e.TaxYear, _ = strconv.Atoi(e.OccurredOn[:len(yearLayout)])
	}

	// An Item is money in a Category exactly as its Expense is, so it answers
	// to the same rule. Sharing the Expense's own Category is deliberately not
	// checked: pointless, but nobody's business to forbid.
	for _, it := range e.Items {
		if !checkCategoryAccepts(w, db, it.CategoryID, appliesExpense,
			fmt.Sprintf("the item %q is not in a category an expense can go in", it.Name)) {
			return false
		}
	}
	return true
}
