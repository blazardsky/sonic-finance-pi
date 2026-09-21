package main

import (
	"net/http"
	"strconv"
	"testing"
)

// A Holding as the API hands it out.
type holdingJSON struct {
	ID                       int64    `json:"id"`
	Name                     string   `json:"name"`
	Type                     string   `json:"type"`
	CurrentPriceCents        *int64   `json:"current_price_cents"`
	QuantityAdjustment       float64  `json:"quantity_adjustment"`
	CostAdjustmentCents      int64    `json:"cost_adjustment_cents"`
	QuantityOwned            float64  `json:"quantity_owned"`
	PaidCents                int64    `json:"paid_cents"`
	AveragePricePerUnitCents *float64 `json:"average_price_per_unit_cents"`
	ValueNowCents            *int64   `json:"value_now_cents"`
	GainLossCents            *int64   `json:"gain_loss_cents"`
	GainLossPercent          *float64 `json:"gain_loss_percent"`
}

// holdingPath addresses one Holding the way the API does.
func holdingPath(id int64) string {
	return "/api/holdings/" + strconv.FormatInt(id, 10)
}

func (a *testApp) holdings(t *testing.T) []holdingJSON {
	t.Helper()
	var got []holdingJSON
	if res := a.get(t, "/api/holdings", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/holdings = %d, want 200", res.StatusCode)
	}
	return got
}

// createHolding adds one and returns it as the API answered, so a test that
// only needs a Holding to exist is one line.
func (a *testApp) createHolding(t *testing.T, name, typ string) holdingJSON {
	t.Helper()
	var got holdingJSON
	res := a.post(t, "/api/holdings", map[string]string{"name": name, "type": typ}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/holdings %q/%q = %d, want 201", name, typ, res.StatusCode)
	}
	return got
}

// Nothing is seeded: which stocks, ETFs, crypto or bonds a household holds is
// not something the code can guess.
func TestAFreshDatabaseHasNoHoldings(t *testing.T) {
	a := newTestApp(t)

	if got := a.holdings(t); len(got) != 0 {
		t.Errorf("a fresh database has %d Holdings, want none: %v", len(got), got)
	}
}

func TestCreatingAHoldingPutsItInTheList(t *testing.T) {
	a := newTestApp(t)

	created := a.createHolding(t, "VWCE", holdingETF)
	if created.ID == 0 {
		t.Error("the created Holding has no id")
	}

	got := a.holdings(t)
	if len(got) != 1 || got[0] != created {
		t.Errorf("the list is %v, want just %v", got, created)
	}
}

// The portfolio breakdown (ticket 03) groups by id, so a rename must not move
// the row.
func TestRenamingAHoldingKeepsItsIdentity(t *testing.T) {
	a := newTestApp(t)
	before := a.createHolding(t, "BTC", holdingCrypto)

	var got holdingJSON
	res := a.patch(t, holdingPath(before.ID), map[string]string{"name": "Bitcoin"}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.ID != before.ID {
		t.Errorf("the renamed Holding has id %d, want %d", got.ID, before.ID)
	}

	list := a.holdings(t)
	if len(list) != 1 || list[0].Name != "Bitcoin" || list[0].ID != before.ID {
		t.Errorf("after the rename the list is %v, want one row %d named Bitcoin", list, before.ID)
	}
}

// A PATCH can change the type alone, the same partial bargain a Client's
// rename makes.
func TestChangingAHoldingsType(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "Rendite Stato", holdingStock)

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]string{"type": holdingBond}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.Type != holdingBond {
		t.Errorf("type = %q, want %q", got.Type, holdingBond)
	}
	if got.Name != "Rendite Stato" {
		t.Errorf("changing the type changed the name: %q", got.Name)
	}
}

func TestHoldingWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"no name", http.MethodPost, "/api/holdings", map[string]string{"type": holdingETF}, http.StatusBadRequest},
		{"blank name", http.MethodPost, "/api/holdings", map[string]string{"name": "  ", "type": holdingETF}, http.StatusBadRequest},
		{"unknown type", http.MethodPost, "/api/holdings", map[string]string{"name": "X", "type": "shares"}, http.StatusBadRequest},
		{"missing type", http.MethodPost, "/api/holdings", map[string]string{"name": "X"}, http.StatusBadRequest},
		{"rename to nothing", http.MethodPatch, holdingPath(h.ID), map[string]string{"name": " ", "type": holdingETF}, http.StatusBadRequest},
		{"unknown id", http.MethodPatch, holdingPath(h.ID + 999), map[string]string{"name": "X", "type": holdingETF}, http.StatusNotFound},
		{"non-numeric id", http.MethodPatch, "/api/holdings/abc", map[string]string{"name": "X", "type": holdingETF}, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.do(t, tc.method, tc.path, tc.body, nil)
			if res.StatusCode != tc.want {
				t.Errorf("%s %s = %d, want %d", tc.method, tc.path, res.StatusCode, tc.want)
			}
		})
	}

	if got := a.holdings(t); len(got) != 1 || got[0] != h {
		t.Errorf("the list is %v, want just the one valid Holding %v", got, h)
	}
}

// "VWCE " and "VWCE" would look identical yet be two rows, which is exactly
// what would silently split one holding's percentage across two — CONTEXT.md.
func TestHoldingNamesAreTrimmed(t *testing.T) {
	a := newTestApp(t)

	if got := a.createHolding(t, "  VWCE  ", holdingETF); got.Name != "VWCE" {
		t.Errorf("created name = %q, want %q", got.Name, "VWCE")
	}
}

// No price set: quantity_owned and paid_cents come straight from the one
// linked buy, and everything price-derived stays nil — there is nothing to
// value at yet.
func TestHoldingFiguresWithNoPriceAreQuantityAndPaidOnly(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)
	investments := a.investments(t)

	a.addExpense(t, map[string]any{
		"amount_cents": 10000, "category_id": investments.ID, "occurred_on": "2026-06-01",
		"holding_id": h.ID, "quantity": 2.5,
	})

	got := a.holdings(t)[0]
	if got.QuantityOwned != 2.5 {
		t.Errorf("quantity_owned = %v, want 2.5", got.QuantityOwned)
	}
	if got.PaidCents != 10000 {
		t.Errorf("paid_cents = %d, want 10000", got.PaidCents)
	}
	if got.ValueNowCents != nil || got.GainLossCents != nil || got.GainLossPercent != nil {
		t.Errorf("value/gain-loss = %v/%v/%v, want all nil with no price set",
			got.ValueNowCents, got.GainLossCents, got.GainLossPercent)
	}
}

// The heart of the ticket: value_now_cents is quantity × price, and
// gain/loss follows from that and what was actually paid.
func TestHoldingFiguresComputeValueAndGainLoss(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)
	investments := a.investments(t)

	a.addExpense(t, map[string]any{
		"amount_cents": 10000, "category_id": investments.ID, "occurred_on": "2026-06-01",
		"holding_id": h.ID, "quantity": 2.0,
	})

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]any{"current_price_cents": 6000}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH current_price_cents = %d, want 200", res.StatusCode)
	}
	// 2 units at 6000 cents each = 12000 value_now; paid 10000; gain 2000,
	// which is 20% of what was paid.
	if got.ValueNowCents == nil || *got.ValueNowCents != 12000 {
		t.Fatalf("value_now_cents = %v, want 12000", got.ValueNowCents)
	}
	if got.GainLossCents == nil || *got.GainLossCents != 2000 {
		t.Fatalf("gain_loss_cents = %v, want 2000", got.GainLossCents)
	}
	if got.GainLossPercent == nil || *got.GainLossPercent != 20 {
		t.Fatalf("gain_loss_percent = %v, want 20", got.GainLossPercent)
	}

	// The list endpoint computes the same figures, not just the single-PATCH
	// response.
	list := a.holdings(t)[0]
	if list.ValueNowCents == nil || *list.ValueNowCents != 12000 {
		t.Errorf("list's value_now_cents = %v, want 12000", list.ValueNowCents)
	}
}

// A sell reduces both quantity_owned and paid_cents — the simple, symmetric
// "how much of my own money is tied up here" the doc comment describes —
// but only once it is actually paid (ADR-0003); an invoiced-not-yet-paid
// sell must not move either figure yet.
func TestHoldingFiguresNetBuyAndSell(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "BTC", holdingCrypto)
	investments := a.investments(t)

	a.addExpense(t, map[string]any{
		"amount_cents": 100000, "category_id": investments.ID, "occurred_on": "2026-06-01",
		"holding_id": h.ID, "quantity": 1.0,
	})
	sell := a.addIncome(t, map[string]any{
		"amount_cents": 40000, "category_id": investments.ID,
		"holding_id": h.ID, "quantity": 0.4,
	})

	unpaid := a.holdings(t)[0]
	if unpaid.QuantityOwned != 1.0 || unpaid.PaidCents != 100000 {
		t.Fatalf("before payment: quantity/paid = %v/%d, want 1/100000 — an unpaid sell must not count",
			unpaid.QuantityOwned, unpaid.PaidCents)
	}

	a.patch(t, incomePath(sell.ID), map[string]any{"payment_date": "2026-06-01"}, nil)

	after := a.holdings(t)[0]
	if after.QuantityOwned != 0.6 {
		t.Errorf("quantity_owned after the paid sell = %v, want 0.6", after.QuantityOwned)
	}
	if after.PaidCents != 60000 {
		t.Errorf("paid_cents after the paid sell = %d, want 60000", after.PaidCents)
	}
}

// gain_loss_percent has nothing to divide by when nothing has been paid yet
// (a Holding with only a price and no buys) — nil rather than a divide by
// zero, the same guard InvoiceTargetCents-style figures use elsewhere.
func TestHoldingGainLossPercentIsNilWhenNothingWasPaid(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	var got holdingJSON
	a.patch(t, holdingPath(h.ID), map[string]any{"current_price_cents": 6000}, &got)
	if got.PaidCents != 0 {
		t.Fatalf("paid_cents = %d, want 0 — nothing was ever bought", got.PaidCents)
	}
	if got.GainLossPercent != nil {
		t.Errorf("gain_loss_percent = %v, want nil — dividing by zero paid cents", got.GainLossPercent)
	}
	// value_now_cents and gain_loss_cents are still well-defined (0 owned ×
	// any price is 0 value, 0 minus 0 paid is 0 gain) — only the percentage
	// has nothing to be a percentage of.
	if got.ValueNowCents == nil || *got.ValueNowCents != 0 {
		t.Errorf("value_now_cents = %v, want 0", got.ValueNowCents)
	}
}

// total_earned-style forgery, applied to a Holding: quantity_owned and
// paid_cents are computed from linked Expenses/Incomes, never something a
// PATCH body gets to set directly.
func TestPatchingAHoldingCannotForgeQuantityOrPaid(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)
	investments := a.investments(t)
	a.addExpense(t, map[string]any{
		"amount_cents": 10000, "category_id": investments.ID, "occurred_on": "2026-06-01",
		"holding_id": h.ID, "quantity": 2.0,
	})

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]any{
		"quantity_owned": 999999.0, "paid_cents": 1,
	}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.QuantityOwned != 2.0 || got.PaidCents != 10000 {
		t.Errorf("quantity/paid = %v/%d, want the real 2/10000, not the forged claim",
			got.QuantityOwned, got.PaidCents)
	}
}

// quantity_adjustment is the one figure here that is a real column, not a
// computed one — a household correcting for a PAC's own quantity-less
// Expenses moves QuantityOwned by exactly the adjustment, on top of whatever
// linked Expenses/Incomes already summed to.
func TestQuantityAdjustmentCorrectsQuantityOwned(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)
	investments := a.investments(t)
	a.addExpense(t, map[string]any{
		"amount_cents": 10000, "category_id": investments.ID, "occurred_on": "2026-06-01",
		"holding_id": h.ID, "quantity": 2.0,
	})

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]any{"quantity_adjustment": 0.5}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH quantity_adjustment = %d, want 200", res.StatusCode)
	}
	if got.QuantityAdjustment != 0.5 {
		t.Errorf("quantity_adjustment = %v, want 0.5", got.QuantityAdjustment)
	}
	if got.QuantityOwned != 2.5 {
		t.Errorf("quantity_owned = %v, want 2.5 (2.0 from the buy plus the 0.5 correction)", got.QuantityOwned)
	}
	if got.PaidCents != 10000 {
		t.Errorf("paid_cents = %d, want 10000 — the correction is quantity-only", got.PaidCents)
	}

	// The list endpoint applies the same correction, not just the single-PATCH
	// response.
	list := a.holdings(t)[0]
	if list.QuantityOwned != 2.5 {
		t.Errorf("list's quantity_owned = %v, want 2.5", list.QuantityOwned)
	}
}

// A Holding created with current_price_cents already set (rare, but not
// forbidden) still starts at zero quantity/paid — nothing can possibly be
// linked to a Holding that did not exist a moment ago.
func TestCreatingAHoldingWithAPriceStartsAtZeroPosition(t *testing.T) {
	a := newTestApp(t)
	var got holdingJSON
	res := a.post(t, "/api/holdings",
		map[string]any{"name": "VWCE", "type": holdingETF, "current_price_cents": 6000}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST = %d, want 201", res.StatusCode)
	}
	if got.QuantityOwned != 0 || got.PaidCents != 0 {
		t.Errorf("quantity/paid = %v/%d, want 0/0 for a brand new Holding", got.QuantityOwned, got.PaidCents)
	}
	if got.ValueNowCents == nil || *got.ValueNowCents != 0 {
		t.Errorf("value_now_cents = %v, want 0 (0 units × any price)", got.ValueNowCents)
	}
}

func TestHoldingsAreListedCaseInsensitivelyByName(t *testing.T) {
	a := newTestApp(t)
	for _, name := range []string{"zcash", "Bitcoin", "amazon", "VWCE"} {
		a.createHolding(t, name, holdingOther)
	}

	var got []string
	for _, h := range a.holdings(t) {
		got = append(got, h.Name)
	}
	want := []string{"amazon", "Bitcoin", "VWCE", "zcash"}
	if len(got) != len(want) {
		t.Fatalf("the list is %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the list is %v, want %v", got, want)
		}
	}
}

// The weighted-average math itself (schema step 21, Titoli's average-
// purchase-price feature), against applyManualLot directly — no DB, no
// HTTP, just the arithmetic every add/replace PATCH ultimately calls.
func TestApplyManualLot(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		currentAdjustment           float64
		currentCostAdjustmentCents  int64
		transactionsQuantity        float64
		transactionsCostCents       int64
		quantity, pricePerUnitCents float64
		replace                     bool
		wantAdjustment              float64
		wantCostAdjustmentCents     int64
	}{
		{
			// 5 @ 50.00 (from nothing) then add 20 @ 45.00 => 25 quotes,
			// avg 46.00 — the exact example the household gave.
			name:              "add on top of an existing add",
			currentAdjustment: 5, currentCostAdjustmentCents: 25000, // 5 @ 50.00 = 250.00
			transactionsQuantity: 0, transactionsCostCents: 0,
			quantity: 20, pricePerUnitCents: 4500, replace: false,
			wantAdjustment: 25, wantCostAdjustmentCents: 115000, // 250.00 + 900.00 = 1150.00, ÷25 = 46.00
		},
		{
			// The first lot ever recorded, from a Holding with nothing
			// manual and nothing linked yet — add and replace must agree
			// here (handleCreateHolding relies on exactly this).
			name:              "first lot ever, add",
			currentAdjustment: 0, currentCostAdjustmentCents: 0,
			transactionsQuantity: 0, transactionsCostCents: 0,
			quantity: 5, pricePerUnitCents: 5000, replace: false,
			wantAdjustment: 5, wantCostAdjustmentCents: 25000,
		},
		{
			name:              "first lot ever, replace",
			currentAdjustment: 0, currentCostAdjustmentCents: 0,
			transactionsQuantity: 0, transactionsCostCents: 0,
			quantity: 5, pricePerUnitCents: 5000, replace: true,
			wantAdjustment: 5, wantCostAdjustmentCents: 25000,
		},
		{
			// Replace with linked transactions already contributing 3
			// units @ 50.00 (15000 cents) and a stale manual correction
			// from before — replace overwrites the manual side so the
			// OVERALL total comes out to 20 @ 40.00 regardless of either.
			name:              "replace overwrites regardless of existing manual correction, net of live transactions",
			currentAdjustment: 25, currentCostAdjustmentCents: 115000, // stale, must be ignored
			transactionsQuantity: 3, transactionsCostCents: 15000,
			quantity: 20, pricePerUnitCents: 4000, replace: true,
			wantAdjustment:          17,    // 20 - 3, so total quantity = 17 + 3 = 20
			wantCostAdjustmentCents: 65000, // (20*4000) - 15000 = 65000, so total paid = 65000+15000 = 80000 = 20*4000
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotAdjustment, gotCostAdjustmentCents := applyManualLot(
				tc.currentAdjustment, tc.currentCostAdjustmentCents,
				tc.transactionsQuantity, tc.transactionsCostCents,
				tc.quantity, tc.pricePerUnitCents, tc.replace,
			)
			if gotAdjustment != tc.wantAdjustment {
				t.Errorf("adjustment = %v, want %v", gotAdjustment, tc.wantAdjustment)
			}
			if gotCostAdjustmentCents != tc.wantCostAdjustmentCents {
				t.Errorf("cost adjustment cents = %v, want %v", gotCostAdjustmentCents, tc.wantCostAdjustmentCents)
			}
		})
	}
}

// The same math, end to end through the PATCH endpoint: two adds in a row
// land on the household's own example — 5 @ 50.00, then add 20 @ 45.00,
// 25 quotes at an average of 46.00.
func TestManualLotAddWeightsTheAverage(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]any{
		"manual_lot": map[string]any{"quantity": 5, "price_per_unit_cents": 5000, "replace": false},
	}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH manual_lot (first) = %d, want 200", res.StatusCode)
	}
	if got.QuantityOwned != 5 || got.PaidCents != 25000 {
		t.Fatalf("after the first lot: quantity/paid = %v/%d, want 5/25000", got.QuantityOwned, got.PaidCents)
	}

	res = a.patch(t, holdingPath(h.ID), map[string]any{
		"manual_lot": map[string]any{"quantity": 20, "price_per_unit_cents": 4500, "replace": false},
	}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH manual_lot (second) = %d, want 200", res.StatusCode)
	}
	if got.QuantityOwned != 25 {
		t.Errorf("quantity_owned = %v, want 25", got.QuantityOwned)
	}
	if got.PaidCents != 115000 {
		t.Errorf("paid_cents = %d, want 115000", got.PaidCents)
	}
	if got.AveragePricePerUnitCents == nil || *got.AveragePricePerUnitCents != 4600 {
		t.Errorf("average_price_per_unit_cents = %v, want 4600", got.AveragePricePerUnitCents)
	}
}

// Replace overwrites the manual correction so the Holding's overall
// quantity/average land exactly on what was typed, net of a linked buy that
// keeps contributing on top.
func TestManualLotReplaceOverwritesNetOfLiveTransactions(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "BTC", holdingCrypto)
	investments := a.investments(t)

	a.addExpense(t, map[string]any{
		"amount_cents": 15000, "category_id": investments.ID, "occurred_on": "2026-06-01",
		"holding_id": h.ID, "quantity": 3.0,
	})

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]any{
		"manual_lot": map[string]any{"quantity": 20, "price_per_unit_cents": 4000, "replace": true},
	}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH manual_lot replace = %d, want 200", res.StatusCode)
	}
	if got.QuantityOwned != 20 {
		t.Errorf("quantity_owned = %v, want 20 (the replace's own total, live buy included)", got.QuantityOwned)
	}
	if got.PaidCents != 80000 {
		t.Errorf("paid_cents = %d, want 80000 (20 × 4000)", got.PaidCents)
	}
	if got.AveragePricePerUnitCents == nil || *got.AveragePricePerUnitCents != 4000 {
		t.Errorf("average_price_per_unit_cents = %v, want 4000", got.AveragePricePerUnitCents)
	}

	// A second buy after the replace still counts — the average is derived
	// fresh every read, never frozen.
	a.addExpense(t, map[string]any{
		"amount_cents": 4000, "category_id": investments.ID, "occurred_on": "2026-06-02",
		"holding_id": h.ID, "quantity": 1.0,
	})
	after := a.holdings(t)[0]
	if after.QuantityOwned != 21 || after.PaidCents != 84000 {
		t.Errorf("after the extra buy: quantity/paid = %v/%d, want 21/84000", after.QuantityOwned, after.PaidCents)
	}
}

// Creating a Holding with an initial lot seeds quantity/average exactly like
// an add on a fresh one — the spec's own "seed the count/average exactly
// like an add" rule.
func TestCreatingAHoldingWithAManualLotSeedsQuantityAndAverage(t *testing.T) {
	a := newTestApp(t)

	var got holdingJSON
	res := a.post(t, "/api/holdings", map[string]any{
		"name": "VWCE", "type": holdingETF,
		"manual_lot": map[string]any{"quantity": 10, "price_per_unit_cents": 5500, "replace": false},
	}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST with manual_lot = %d, want 201", res.StatusCode)
	}
	if got.QuantityOwned != 10 {
		t.Errorf("quantity_owned = %v, want 10", got.QuantityOwned)
	}
	if got.PaidCents != 55000 {
		t.Errorf("paid_cents = %d, want 55000", got.PaidCents)
	}
	if got.AveragePricePerUnitCents == nil || *got.AveragePricePerUnitCents != 5500 {
		t.Errorf("average_price_per_unit_cents = %v, want 5500", got.AveragePricePerUnitCents)
	}
}

// A lot needs a quantity to weight its price against, whether adding or
// replacing — a price with no quantity is not a meaningful average.
func TestManualLotWithoutQuantityIsRejected(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"zero quantity, add", map[string]any{"quantity": 0, "price_per_unit_cents": 5000, "replace": false}},
		{"negative quantity, replace", map[string]any{"quantity": -1, "price_per_unit_cents": 5000, "replace": true}},
		{"negative price", map[string]any{"quantity": 5, "price_per_unit_cents": -1, "replace": false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.patch(t, holdingPath(h.ID), map[string]any{"manual_lot": tc.body}, nil)
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("PATCH manual_lot %v = %d, want 400", tc.body, res.StatusCode)
			}
		})
	}

	// Rejected — the Holding must be unchanged.
	got := a.holdings(t)[0]
	if got.QuantityOwned != 0 || got.PaidCents != 0 {
		t.Errorf("quantity/paid = %v/%d, want 0/0 — the rejected lots must not apply", got.QuantityOwned, got.PaidCents)
	}
}

// quantity_owned and paid_cents (and now average_price_per_unit_cents) stay
// forgery-proof the same way ticket 03 already established — a manual_lot's
// own computed result is what lands, never a raw claim.
func TestPatchingAHoldingCannotForgeTheAverage(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]any{
		"average_price_per_unit_cents": 1.0, "quantity_owned": 999.0, "paid_cents": 1,
	}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.QuantityOwned != 0 || got.PaidCents != 0 || got.AveragePricePerUnitCents != nil {
		t.Errorf("quantity/paid/average = %v/%d/%v, want 0/0/nil — nothing was ever bought",
			got.QuantityOwned, got.PaidCents, got.AveragePricePerUnitCents)
	}
}
