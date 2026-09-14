package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
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

// The 9-slot color palette (ADR-0016, ticket 04): a Category or Subcategory's
// color is one of these keys, never an arbitrary value. blue-gray is the
// neutral default for a genuinely uncolored one — not a hue snap target, just
// a ninth entry. Defined once here so the CHECK constraint's literal list, the
// Go-side validator, and the migration's snap targets never drift apart.
var categoryColors = []string{
	"blue", "orange", "aqua", "yellow", "magenta", "green", "violet", "red", "blue-gray",
}

const defaultColor = "blue-gray"

func validColor(c string) bool {
	for _, v := range categoryColors {
		if c == v {
			return true
		}
	}
	return false
}

// The stable codes of the four Base categories. A non-null code is what marks
// a Category as one the code resolves by identity — the yearly tax summary
// reads Freelance and Taxes, Budget/Target/Estimate exclude Investments, the
// spoiler blur resolves Gift — and therefore one that rename and delete
// refuse. Nothing else is protected: a seeded Category nothing resolves is
// the household's to edit. See ADR-0008.
const (
	codeFreelance   = "freelance"
	codeTaxes       = "taxes"
	codeInvestments = "investments"
	codeGift        = "gift"
)

// The names the four Base categories are seeded with. They are ordinary
// names after that — the household can hide any one, and the report still
// finds it by code — but the tests need something to address them by.
const (
	seedFreelanceName   = "Freelance"
	seedTaxesName       = "Tasse"
	seedStipendioName   = "Stipendio"
	seedInvestmentiName = "Investimenti" // promoted to codeInvestments by migrateInvestments
	seedRegaliName      = "Regali"       // promoted to codeGift by migrateGiftContractsReminders
)

// seedCategories is what a fresh database starts with, so that first use is
// not an evening of typing a list. Only the ones with a code are protected —
// Investimenti gets its code later, in migrateInvestments, not here.
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
	{Name: seedStipendioName, AppliesTo: appliesIncome},
	{Name: seedRegaliName, AppliesTo: appliesIncome},
	{Name: "Rimborsi", AppliesTo: appliesIncome},
	{Name: seedInvestmentiName, AppliesTo: appliesIncome},

	{Name: "Altro", AppliesTo: appliesBoth},
}

type category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AppliesTo string `json:"applies_to"`
	Hidden    bool   `json:"hidden"`
	// One of categoryColors, never empty (ADR-0016) — a loose grouping signal,
	// not a unique identity, so two unrelated Categories sharing one is normal.
	Color string `json:"color"`

	// Base is what the UI greys the rename and delete buttons on. The code
	// behind it is not published: it is an implementation detail of the
	// reports, and the household has no use for it.
	Base bool `json:"base"`

	// Gift is Base narrowed to specifically the one Category the spoiler blur
	// resolves by identity. Unlike code, this one is published: the frontend
	// has to tell an Expense in Gift apart from any other Base category to
	// blur its amount, and a household-editable name is not a safe thing to
	// match it by (spoiler ticket).
	Gift bool `json:"gift"`

	// Freelance is Base narrowed the same way Gift is, for the Income form:
	// whether Fattura inviata is worth asking for depends on this Category
	// being Freelance, and a renamed "Freelance" is still exactly as much
	// Freelance as before.
	Freelance bool `json:"freelance"`

	// Investments is Base narrowed the same way Gift and Freelance are, for
	// the PAC badge (ticket 02, `PAC` in CONTEXT.md): a Recurring expense is a
	// PAC only when its Category is specifically Investments, not any Base
	// category with the same applies_to, and a renamed "Investimenti" is
	// still exactly as much Investments as before.
	Investments bool `json:"investments"`

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
			Color     string `json:"color"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		c := category{Name: strings.TrimSpace(body.Name), AppliesTo: body.AppliesTo, Color: body.Color}
		if c.Color == "" {
			c.Color = defaultColor
		}
		if err := c.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := db.Exec(`INSERT INTO category (name, applies_to, color) VALUES (?, ?, ?)`, c.Name, c.AppliesTo, c.Color)
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
		// Decoded onto a presence-detecting struct rather than straight onto c,
		// unlike a Client or Expense PATCH: the Base-category guard below has to
		// know whether name/applies_to were sent at all, even a same-value
		// resend, and decoding onto the loaded row directly cannot tell that
		// apart from the field being merely omitted.
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
		if body.Color != nil {
			c.Color = *body.Color
		}
		if err := c.validate(); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if _, err := db.Exec(`UPDATE category SET name = ?, applies_to = ?, hidden = ?, color = ? WHERE id = ?`,
			c.Name, c.AppliesTo, c.Hidden, c.Color, c.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

// handleDeleteCategory refuses a Base category outright, replace_with or not
// (ADR-0008, checked first). With no replace_with it behaves exactly as
// before: a bare DELETE, which the FK constraint turns into a 409 via
// writeError when something still points at the row. With replace_with, that
// target is validated the same way a Category on an Expense/Income write is —
// checkCategoryAccepts, so "must exist" and "must accept this side" are one
// rule, not two — and then every reference is moved onto it before the old
// row goes, in one transaction, so nothing is ever left pointing at a
// half-deleted Category.
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

		raw := r.URL.Query().Get("replace_with")
		if raw == "" {
			if _, err := db.Exec(`DELETE FROM category WHERE id = ?`, c.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		replaceID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("replace_with must be a category id"))
			return
		}
		if !checkCategoryAccepts(w, db, replaceID, c.AppliesTo,
			"replace_with must be an existing category accepting the same side as the one being deleted") {
			return
		}

		if err := reassignCategoryTo(db, c.ID, replaceID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// execStmt is one parameterized statement for runThenDelete to run.
type execStmt struct {
	query string
	args  []any
}

// runThenDelete runs every stmt, then deletes the row `id` names from table,
// all in one transaction — the shared shape behind reassignCategoryTo,
// reassignSubcategoryTo and clearSubcategoryFrom: a failure partway must
// never leave some entries reassigned (or cleared) and others still pointing
// at a row about to vanish.
func runThenDelete(db *sql.DB, stmts []execStmt, table string, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, s := range stmts {
		if _, err := tx.Exec(s.query, s.args...); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM `+table+` WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// reassignCategoryTo moves every column ticket 01 lists from oldID to newID —
// Expense, Item and Income category, Recurring expense's template category,
// and a Client's default — then removes the now-unreferenced old row.
func reassignCategoryTo(db *sql.DB, oldID, newID int64) error {
	return runThenDelete(db, []execStmt{
		{`UPDATE expense SET category_id = ? WHERE category_id = ?`, []any{newID, oldID}},
		{`UPDATE item SET category_id = ? WHERE category_id = ?`, []any{newID, oldID}},
		{`UPDATE income SET category_id = ? WHERE category_id = ?`, []any{newID, oldID}},
		{`UPDATE recurring_expense SET category_id = ? WHERE category_id = ?`, []any{newID, oldID}},
		{`UPDATE client SET default_category_id = ? WHERE default_category_id = ?`, []any{newID, oldID}},
	}, "category", oldID)
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
// it. `code` is reduced to the four booleans the API publishes — Base
// (protected at all), Gift (protected as specifically the spoiler's Category),
// Freelance (the Income form's Fattura inviata field), and Investments (the
// PAC badge) — the code itself stays an implementation detail of the reports
// that resolve by it.
var categorySelect = fmt.Sprintf(
	`SELECT id, name, applies_to, hidden, color, code IS NOT NULL, IFNULL(code, '') = '%s', IFNULL(code, '') = '%s', IFNULL(code, '') = '%s' FROM category`,
	codeGift, codeFreelance, codeInvestments)

func scanCategory(row interface{ Scan(...any) error }) (category, error) {
	var c category
	err := row.Scan(&c.ID, &c.Name, &c.AppliesTo, &c.Hidden, &c.Color, &c.Base, &c.Gift, &c.Freelance, &c.Investments)
	return c, err
}

// categoryAccepts reports whether the Category exists and is one an entry of
// this kind can go in — "Freelance" is never an Expense. Hidden is not part of
// it: hiding takes a Category out of the picker, not out of the app.
func categoryAccepts(db *sql.DB, id int64, applies string) (bool, error) {
	var found int
	err := db.QueryRow(`SELECT 1 FROM category WHERE id = ? AND applies_to IN (?, ?)`,
		id, applies, appliesBoth).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// isTaxCategory reports whether this Category is the Taxes Base one — the
// question a Tax year is only kept for the answer to. Resolved by code and not
// by name, like every other read of a Base category: the household cannot
// rename this one, but the report must not depend on that being true.
//
// Hidden is not part of it, for the reason it is not part of categoryAccepts:
// a tax payment recorded under a Category since hidden is still a tax payment,
// and the summary still has to count it.
func isTaxCategory(db *sql.DB, id int64) (bool, error) {
	var found int
	err := db.QueryRow(`SELECT 1 FROM category WHERE id = ? AND code = ?`, id, codeTaxes).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// checkCategoryAccepts confirms one Category exists and can hold money of this
// kind, writing the response itself on a refusal — a refusal being a bad
// request the client can be told about, in the words the caller chose.
// Expenses, their Items and Incomes all answer to the same rule, from here.
func checkCategoryAccepts(w http.ResponseWriter, db *sql.DB, id int64, applies, refusal string) bool {
	switch ok, err := categoryAccepts(db, id, applies); {
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
	case !ok:
		writeInvalid(w, errors.New(refusal))
	default:
		return true
	}
	return false
}

func (c category) validate() error {
	if c.Name == "" {
		return errors.New("a category needs a name")
	}
	switch c.AppliesTo {
	case appliesExpense, appliesIncome, appliesBoth:
	default:
		return errors.New("applies_to must be expense, income or both")
	}
	if !validColor(c.Color) {
		return fmt.Errorf("color must be one of %v", categoryColors)
	}
	return nil
}

// migrateCategoryColor is schema step 16: category and subcategory each gain
// a color column, then every existing Category row (Subcategory never had a
// color, so it simply takes the column default) is backfilled from the
// retired frontend generator so the switch to a fixed palette does not
// visually scramble what the household is already used to (ADR-0016).
func migrateCategoryColor(tx *sql.Tx) error {
	colorDDL := `TEXT NOT NULL DEFAULT 'blue-gray' CHECK (color IN ('blue','orange','aqua','yellow','magenta','green','violet','red','blue-gray'))`
	if _, err := tx.Exec(`ALTER TABLE category ADD COLUMN color ` + colorDDL); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE subcategory ADD COLUMN color ` + colorDDL); err != nil {
		return err
	}

	rows, err := tx.Query(`SELECT id FROM category`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, id := range ids {
		if _, err := tx.Exec(`UPDATE category SET color = ? WHERE id = ?`, nearestVividSlot(legacyHue(id)), id); err != nil {
			return err
		}
	}
	return nil
}

// vividSlots is the spec's hue-angle table for the 8 vivid palette slots —
// blue-gray is deliberately absent, since it is never a snap target.
var vividSlots = []struct {
	name string
	hue  float64
}{
	{"red", 0.4},
	{"orange", 17.0},
	{"yellow", 40.8},
	{"green", 120.0},
	{"aqua", 158.5},
	{"blue", 212.8},
	{"violet", 248.8},
	{"magenta", 337.4},
}

// legacyHue reproduces the retired frontend color function exactly, so the
// migration snaps to the same hue that function would have rendered.
func legacyHue(id int64) float64 {
	return math.Mod(float64(id)*137.508, 360)
}

// nearestVividSlot picks whichever vividSlots entry is angularly closest to
// hue, by circular distance on the 0-360 wheel (the wrap-around at 360/0 must
// not make 350 and 10 look 340 apart).
func nearestVividSlot(hue float64) string {
	best := vividSlots[0].name
	bestDist := math.Inf(1)
	for _, s := range vividSlots {
		d := math.Abs(hue - s.hue)
		if d > 180 {
			d = 360 - d
		}
		if d < bestDist {
			bestDist = d
			best = s.name
		}
	}
	return best
}
