package main

import (
	"database/sql"
	"errors"
	"math"
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
// vehicle the household buys and sells. Buying and selling one is recorded as
// an ordinary Expense or Income under the Investments base category (ticket
// 03); this ticket is the fixed, household-maintained list itself.
//
// Quantity and pricing are both opt-in, per Holding, and both still entirely
// hand-typed (schema steps 17-18) — CONTEXT.md's "never priced or revalued by
// the app" stays true in the sense that matters: nothing here fetches a
// price or runs on a schedule, it only has somewhere to put one the
// household typed in. PaidCents, ValueNowCents, GainLossCents and
// GainLossPercent are all computed at read time, never stored, the same
// bargain a Client's TotalEarnedCents and a Contract's own figures make.
type holding struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	CurrentPriceCents *int64 `json:"current_price_cents"`

	// A hand-typed correction added to the quantity summed from linked
	// Expenses/Incomes — a PAC's own monthly Expenses are generated without
	// a quantity (materialise has no reliable per-month price to derive one
	// from), so this is the household's way to true up QuantityOwned without
	// having to open every one of them. Zero on a Holding nobody has
	// corrected.
	QuantityAdjustment float64 `json:"quantity_adjustment"`

	// QuantityAdjustment plus every linked Expense (a buy) less every linked,
	// paid Income (a sell) — symmetric and simple on purpose: this answers
	// "how much of my own money is tied up here right now," not a
	// capital-gains-accurate weighted cost basis, which is the household's
	// tax software's job, not this app's.
	QuantityOwned float64 `json:"quantity_owned"`
	PaidCents     int64   `json:"paid_cents"`

	// nil whenever CurrentPriceCents is nil — there is nothing to value at.
	// GainLossPercent is also nil when PaidCents is zero, the same
	// divide-by-nothing guard InvoiceTargetCents-style figures use elsewhere.
	ValueNowCents   *int64   `json:"value_now_cents"`
	GainLossCents   *int64   `json:"gain_loss_cents"`
	GainLossPercent *float64 `json:"gain_loss_percent"`
}

// computeHoldingFigures fills in the read-time-only fields from
// CurrentPriceCents, QuantityOwned and PaidCents, which scanHolding has
// already set. Its own function, not inlined into scanHolding, for the same
// reason computeContractFigures is separate from scanContract: every write
// handler needs the stored/summed fields refreshed after its own change, and
// this is the one place that arithmetic lives.
func computeHoldingFigures(h *holding) {
	if h.CurrentPriceCents == nil {
		return
	}
	value := int64(math.Round(h.QuantityOwned * float64(*h.CurrentPriceCents)))
	h.ValueNowCents = &value
	gainLoss := value - h.PaidCents
	h.GainLossCents = &gainLoss
	if h.PaidCents != 0 {
		percent := float64(gainLoss) / float64(h.PaidCents) * 100
		h.GainLossPercent = &percent
	}
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

// migrateHoldingQuantityAndPrice is schema step 17. A Holding was "never
// priced or revalued by the app" (CONTEXT.md) — this is the household opting
// a specific Holding into both, deliberately still by hand: quantity is
// typed on the Expense/Income that bought or sold it (the same optional
// quantity/CHECK shape ticket 01 gave Item), and current_price_cents is
// typed directly onto the Holding whenever the household checks a broker.
// Neither is ever written by this codebase itself — there is still no
// scheduled job, no outbound HTTP call, nothing fetching a price — so the
// CONTEXT.md line stays true in spirit: the app does not go looking for a
// price, it only has somewhere to put one that was handed to it.
//
// current_price_cents is a *price per unit*, not a total position value,
// specifically so that a future price-API integration is "PATCH this field
// periodically" rather than a second schema change: every such API answers
// in price-per-share, and value_now_cents (computed, never stored) is
// quantity_owned × current_price_cents either way.
func migrateHoldingQuantityAndPrice(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE expense ADD COLUMN quantity REAL CHECK (quantity IS NULL OR quantity > 0)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE income ADD COLUMN quantity REAL CHECK (quantity IS NULL OR quantity > 0)`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE holding ADD COLUMN current_price_cents INTEGER CHECK (current_price_cents IS NULL OR current_price_cents >= 0)`)
	return err
}

// migrateHoldingQuantityAdjustment is schema step 18: the hand-typed
// correction QuantityOwned adds to whatever linked Expenses/Incomes sum to —
// a PAC's own generated Expenses carry no quantity (materialise has no
// per-month price to derive one from), so without this the household would
// have to open every one of them by hand to keep QuantityOwned true. Signed,
// unlike current_price_cents: a correction can go either way.
func migrateHoldingQuantityAdjustment(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE holding ADD COLUMN quantity_adjustment REAL NOT NULL DEFAULT 0`)
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

// A newly created Holding can set current_price_cents right away (harmless,
// even if nothing is linked to it yet), but QuantityOwned and PaidCents are
// computed from linked Expenses/Incomes that cannot possibly exist yet — set
// explicitly rather than trusted from whatever a forged request body claims.
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

		res, err := db.Exec(`INSERT INTO holding (name, type, current_price_cents, quantity_adjustment) VALUES (?, ?, ?, ?)`,
			h.Name, h.Type, h.CurrentPriceCents, h.QuantityAdjustment)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if h.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		h.QuantityOwned, h.PaidCents = h.QuantityAdjustment, 0
		computeHoldingFigures(&h)
		writeJSON(w, http.StatusCreated, h)
	}
}

// handlePatchHolding renames a Holding, changes its type, sets/clears its
// current_price_cents, or corrects its quantity_adjustment. Decoding onto the
// stored row is what makes it partial, the same bargain a Client's PATCH
// makes. PaidCents and the quantity summed from linked Expenses/Incomes are
// restored after decoding — computed, never something a PATCH body gets to
// claim directly, the same protection handlePatchContract gives
// ReceivedCents/AccountedCents — but QuantityAdjustment is a real column and
// freely writable, so QuantityOwned is rebuilt from the (possibly just
// changed) adjustment plus the untouched transaction sum. Nothing else here
// is protected the way a Base category is: no report resolves one Holding by
// identity, so every Holding is the household's to rename.
func handlePatchHolding(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, ok := findHolding(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := h.ID
		quantityFromTransactions, paidCents := h.QuantityOwned-h.QuantityAdjustment, h.PaidCents
		if err := decodeJSON(w, r, &h); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h.ID = id // an id in the body is not a way to move the row
		h.PaidCents = paidCents
		h.QuantityOwned = h.QuantityAdjustment + quantityFromTransactions
		if err := h.validate(); err != nil {
			writeInvalid(w, err)
			return
		}

		if _, err := db.Exec(`UPDATE holding SET name = ?, type = ?, current_price_cents = ?, quantity_adjustment = ? WHERE id = ?`,
			h.Name, h.Type, h.CurrentPriceCents, h.QuantityAdjustment, h.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		h.ValueNowCents, h.GainLossCents, h.GainLossPercent = nil, nil, nil
		computeHoldingFigures(&h)
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

// The two subqueries are the quantity and amount summed from linked
// Expenses (a buy) less linked, paid Incomes (a sell) — ADR-0003's
// payment_date filter, the same as clientSelect's own total-earned subquery
// relies on. quantity_adjustment rides along raw; scanHolding adds it to the
// summed quantity to get QuantityOwned.
const holdingSelect = `SELECT id, name, type, current_price_cents, quantity_adjustment,
	(SELECT COALESCE(SUM(quantity), 0) FROM expense WHERE expense.holding_id = holding.id) -
		(SELECT COALESCE(SUM(quantity), 0) FROM income WHERE income.holding_id = holding.id AND income.payment_date IS NOT NULL),
	(SELECT COALESCE(SUM(amount_cents), 0) FROM expense WHERE expense.holding_id = holding.id) -
		(SELECT COALESCE(SUM(amount_cents), 0) FROM income WHERE income.holding_id = holding.id AND income.payment_date IS NOT NULL)
	FROM holding`

func scanHolding(row interface{ Scan(...any) error }) (holding, error) {
	var h holding
	var quantityFromTransactions float64
	if err := row.Scan(&h.ID, &h.Name, &h.Type, &h.CurrentPriceCents, &h.QuantityAdjustment, &quantityFromTransactions, &h.PaidCents); err != nil {
		return holding{}, err
	}
	h.QuantityOwned = h.QuantityAdjustment + quantityFromTransactions
	computeHoldingFigures(&h)
	return h, nil
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
