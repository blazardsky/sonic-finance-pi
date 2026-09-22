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

	// The cost-basis counterpart to QuantityAdjustment (schema step 20,
	// Titoli's average-purchase-price feature): a hand-typed contribution to
	// PaidCents from manually-recorded purchase lots, on top of whatever
	// linked Expenses/Incomes already sum to. A real column, same as
	// QuantityAdjustment, and freely settable directly — but ordinarily set
	// through the manual_lot request field instead (applyManualLot), which
	// keeps it and QuantityAdjustment moving together so the derived average
	// stays meaningful.
	CostAdjustmentCents int64 `json:"cost_adjustment_cents"`

	// CostAdjustmentCents plus every linked Expense (a buy) less every
	// linked, paid Income (a sell) — the exact same shape QuantityOwned
	// already has, one level of cents up.
	PaidCents int64 `json:"paid_cents"`

	// PaidCents ÷ QuantityOwned, computed at read time — nil when
	// QuantityOwned is zero (nothing to divide by) or negative (sold more
	// than was ever bought, a state with no meaningful average). Never
	// stored: freezing a number here is exactly what schema step 20 is
	// avoiding, since it would drift the moment a linked buy/sell changes.
	AveragePricePerUnitCents *float64 `json:"average_price_per_unit_cents"`

	// nil whenever CurrentPriceCents is nil — there is nothing to value at.
	// GainLossPercent is also nil when PaidCents is zero, the same
	// divide-by-nothing guard InvoiceTargetCents-style figures use elsewhere.
	ValueNowCents   *int64   `json:"value_now_cents"`
	GainLossCents   *int64   `json:"gain_loss_cents"`
	GainLossPercent *float64 `json:"gain_loss_percent"`
}

// manualLot is the PATCH/POST body's optional carrier for the Titoli
// average-purchase-price feature: one purchase lot's quantity and price per
// unit, applied by applyManualLot as either a one-shot addition to
// QuantityAdjustment/CostAdjustmentCents or a full replace of them. Quantity
// is required whenever a lot is sent at all — a price with no quantity to
// weight it against (add or replace) is not a meaningful average, only a
// number with nothing underneath it.
type manualLot struct {
	Quantity          float64 `json:"quantity"`
	PricePerUnitCents float64 `json:"price_per_unit_cents"`
	Replace           bool    `json:"replace"`
}

// validate requires Quantity only when adding — a lot with a price and no
// quantity is not a purchase. Replacing is the deliberate exception: a
// replace's quantity may be omitted (or 0, indistinguishable here, and
// treated the same) to mean "keep the current total quantity, just correct
// the average" — there is still something to weight the new average
// against in that case, namely whatever the Holding's total already is,
// which is exactly what "unless the user is replacing" carves out.
func (m *manualLot) validate() error {
	if !m.Replace && m.Quantity <= 0 {
		return errors.New("a purchase lot needs a quantity greater than zero")
	}
	if m.Quantity < 0 {
		return errors.New("quantity must not be negative")
	}
	if m.PricePerUnitCents < 0 {
		return errors.New("price per unit must not be negative")
	}
	return nil
}

// applyManualLot computes a Holding's new QuantityAdjustment/
// CostAdjustmentCents after recording one purchase lot of `quantity` units
// at `pricePerUnitCents` each. transactionsQuantity/transactionsCostCents
// are what linked Expenses/Incomes already contribute — read fresh from the
// Holding this is called against, never a stale snapshot, so a lot recorded
// moments after a buy or sell posts still lands on the correct total.
//
// Unchecked (add): the lot adds on top of whatever manual correction already
// existed — currentAdjustment/currentCostAdjustmentCents plus this lot.
//
// Checked (replace) with quantity <= 0: keeps the Holding's current total
// quantity (currentAdjustment plus what transactions already contribute)
// and only replaces the average — the "replace-average without a quantity"
// case validate() lets through.
//
// Checked (replace) otherwise: overwrites the running manual correction so the
// Holding's OVERALL quantity/average come out to exactly quantity/
// pricePerUnitCents, regardless of what the manual correction was before —
// solved by subtracting what linked transactions already contribute, since
// QuantityOwned/PaidCents are always transactionsQuantity+adjustment and
// transactionsCostCents+costAdjustment.
func applyManualLot(
	currentAdjustment float64, currentCostAdjustmentCents int64,
	transactionsQuantity float64, transactionsCostCents int64,
	quantity float64, pricePerUnitCents float64, replace bool,
) (newAdjustment float64, newCostAdjustmentCents int64) {
	if replace {
		effectiveQuantity := quantity
		if effectiveQuantity <= 0 {
			// No quantity typed — keep the current total (validate has
			// already refused this unless replace is set).
			effectiveQuantity = currentAdjustment + transactionsQuantity
		}
		lotCostCents := int64(math.Round(effectiveQuantity * pricePerUnitCents))
		return effectiveQuantity - transactionsQuantity, lotCostCents - transactionsCostCents
	}
	lotCostCents := int64(math.Round(quantity * pricePerUnitCents))
	return currentAdjustment + quantity, currentCostAdjustmentCents + lotCostCents
}

// computeHoldingFigures fills in the read-time-only fields from
// CurrentPriceCents, QuantityOwned and PaidCents, which scanHolding has
// already set. Its own function, not inlined into scanHolding, for the same
// reason computeContractFigures is separate from scanContract: every write
// handler needs the stored/summed fields refreshed after its own change, and
// this is the one place that arithmetic lives.
func computeHoldingFigures(h *holding) {
	if h.QuantityOwned > 0 {
		avg := float64(h.PaidCents) / h.QuantityOwned
		h.AveragePricePerUnitCents = &avg
	}
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

// migrateHoldingCostAdjustment is schema step 20: CostAdjustmentCents, the
// cost-basis counterpart to quantity_adjustment — Titoli's average-purchase-
// price feature needs a manually-typed cost contribution to weight against
// the manually-typed quantity one, since PaidCents before this step comes
// only from linked Expenses/Incomes with no hand-typed correction at all.
// Defaults to 0 so every existing Holding reads as "no manual cost
// correction yet," the same bargain quantity_adjustment's own default made.
func migrateHoldingCostAdjustment(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE holding ADD COLUMN cost_adjustment_cents INTEGER NOT NULL DEFAULT 0`)
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
		out, err := readHoldings(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// readHoldings is every Holding, figures and all — handleListHoldings' own
// query, pulled out so readHoldingBreakdown (cmd/savings.go) can read the
// same QuantityOwned/PaidCents/ValueNowCents/GainLossCents arithmetic rather
// than a second copy of it in SQL.
func readHoldings(db *sql.DB) ([]holding, error) {
	rows, err := db.Query(holdingSelect + ` ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// An empty list has to marshal as [] rather than null: the frontend
	// maps over it, and an empty list is where every household starts.
	out := []holding{}
	for rows.Next() {
		h, err := scanHolding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// A newly created Holding can set current_price_cents right away (harmless,
// even if nothing is linked to it yet), but QuantityOwned and PaidCents are
// computed from linked Expenses/Incomes that cannot possibly exist yet — set
// explicitly rather than trusted from whatever a forged request body claims.
//
// An optional manual_lot seeds QuantityAdjustment/CostAdjustmentCents the
// same way it corrects them later (applyManualLot) — with nothing linked
// yet, transactionsQuantity/transactionsCostCents are both zero, so add and
// replace land on the same result: the lot's own quantity and average.
func handleCreateHolding(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			holding
			ManualLot *manualLot `json:"manual_lot"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h := body.holding
		if err := h.validate(); err != nil {
			writeInvalid(w, err)
			return
		}
		if body.ManualLot != nil {
			if err := body.ManualLot.validate(); err != nil {
				writeInvalid(w, err)
				return
			}
			h.QuantityAdjustment, h.CostAdjustmentCents = applyManualLot(
				h.QuantityAdjustment, h.CostAdjustmentCents, 0, 0,
				body.ManualLot.Quantity, body.ManualLot.PricePerUnitCents, body.ManualLot.Replace,
			)
		}

		res, err := db.Exec(`INSERT INTO holding (name, type, current_price_cents, quantity_adjustment, cost_adjustment_cents) VALUES (?, ?, ?, ?, ?)`,
			h.Name, h.Type, h.CurrentPriceCents, h.QuantityAdjustment, h.CostAdjustmentCents)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if h.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		h.QuantityOwned, h.PaidCents = h.QuantityAdjustment, h.CostAdjustmentCents
		computeHoldingFigures(&h)
		writeJSON(w, http.StatusCreated, h)
	}
}

// handlePatchHolding renames a Holding, changes its type, sets/clears its
// current_price_cents, or corrects its quantity_adjustment/
// cost_adjustment_cents — directly, or through manual_lot (applyManualLot),
// which computes both together from the household's Q/price/replace inputs
// instead of asking them to work out the raw correction numbers themselves.
// Decoding onto the stored row is what makes it partial, the same bargain a
// Client's PATCH makes. The two transaction sums are captured before
// decoding and QuantityOwned/PaidCents are rebuilt from them afterward —
// computed, never something a PATCH body gets to claim directly, the same
// protection handlePatchContract gives ReceivedCents/AccountedCents — but
// QuantityAdjustment/CostAdjustmentCents are real columns and freely
// writable (directly or via manual_lot), which is exactly what those two
// rebuilds fold back in. Nothing else here is protected the way a Base
// category is: no report resolves one Holding by identity, so every Holding
// is the household's to rename.
func handlePatchHolding(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, ok := findHolding(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		id := h.ID
		quantityFromTransactions := h.QuantityOwned - h.QuantityAdjustment
		costFromTransactions := h.PaidCents - h.CostAdjustmentCents
		currentAdjustment, currentCostAdjustment := h.QuantityAdjustment, h.CostAdjustmentCents

		var body struct {
			holding
			ManualLot *manualLot `json:"manual_lot"`
		}
		body.holding = h
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h = body.holding
		h.ID = id // an id in the body is not a way to move the row

		if body.ManualLot != nil {
			if err := body.ManualLot.validate(); err != nil {
				writeInvalid(w, err)
				return
			}
			h.QuantityAdjustment, h.CostAdjustmentCents = applyManualLot(
				currentAdjustment, currentCostAdjustment,
				quantityFromTransactions, costFromTransactions,
				body.ManualLot.Quantity, body.ManualLot.PricePerUnitCents, body.ManualLot.Replace,
			)
		}
		h.QuantityOwned = h.QuantityAdjustment + quantityFromTransactions
		h.PaidCents = h.CostAdjustmentCents + costFromTransactions
		if err := h.validate(); err != nil {
			writeInvalid(w, err)
			return
		}

		if _, err := db.Exec(`UPDATE holding SET name = ?, type = ?, current_price_cents = ?, quantity_adjustment = ?, cost_adjustment_cents = ? WHERE id = ?`,
			h.Name, h.Type, h.CurrentPriceCents, h.QuantityAdjustment, h.CostAdjustmentCents, h.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		h.ValueNowCents, h.GainLossCents, h.GainLossPercent, h.AveragePricePerUnitCents = nil, nil, nil, nil
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
// relies on. quantity_adjustment/cost_adjustment_cents ride along raw;
// scanHolding adds them to the summed quantity/amount to get
// QuantityOwned/PaidCents.
const holdingSelect = `SELECT id, name, type, current_price_cents, quantity_adjustment, cost_adjustment_cents,
	(SELECT COALESCE(SUM(quantity), 0) FROM expense WHERE expense.holding_id = holding.id) -
		(SELECT COALESCE(SUM(quantity), 0) FROM income WHERE income.holding_id = holding.id AND income.payment_date IS NOT NULL),
	(SELECT COALESCE(SUM(amount_cents), 0) FROM expense WHERE expense.holding_id = holding.id) -
		(SELECT COALESCE(SUM(amount_cents), 0) FROM income WHERE income.holding_id = holding.id AND income.payment_date IS NOT NULL)
	FROM holding`

func scanHolding(row interface{ Scan(...any) error }) (holding, error) {
	var h holding
	var quantityFromTransactions float64
	var costFromTransactions int64
	if err := row.Scan(&h.ID, &h.Name, &h.Type, &h.CurrentPriceCents, &h.QuantityAdjustment, &h.CostAdjustmentCents, &quantityFromTransactions, &costFromTransactions); err != nil {
		return holding{}, err
	}
	h.QuantityOwned = h.QuantityAdjustment + quantityFromTransactions
	h.PaidCents = h.CostAdjustmentCents + costFromTransactions
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
