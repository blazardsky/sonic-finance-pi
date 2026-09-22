package main

import (
	"net/http"
	"testing"
)

// Savings as the API hands it out (ticket 07).
type savingsJSON struct {
	SavingsCents         int64              `json:"savings_cents"`
	StartingBalanceCents int64              `json:"starting_balance_cents"`
	Holdings             []holdingBreakdown `json:"holdings"`
	CombinedCents        int64              `json:"combined_cents"`
	NetWorthTargetCents  int64              `json:"net_worth_target_cents"`
	YearlySavingsCents   int64              `json:"yearly_savings_cents"`
}

func (a *testApp) savings(t *testing.T) savingsJSON {
	t.Helper()
	var got savingsJSON
	if res := a.get(t, savingsPath, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", savingsPath, res.StatusCode)
	}
	return got
}

// A fresh database has nothing to add or subtract: Savings is 0, no starting
// balance, and no Holdings in the breakdown.
func TestAFreshDatabaseHasZeroSavings(t *testing.T) {
	a := newTestApp(t)

	got := a.savings(t)
	if got.SavingsCents != 0 || got.StartingBalanceCents != 0 || len(got.Holdings) != 0 {
		t.Fatalf("fresh savings = %+v, want all zero/empty", got)
	}
}

// The whole ticket in one number: cumulative received Income minus Expense,
// all time, with no month scoping at all.
func TestSavingsIsCumulativeIncomeMinusExpense(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	stipendio := a.category(t, "Stipendio")

	a.addExpense(t, map[string]any{"occurred_on": "2025-01-10", "amount_cents": 5000, "category_id": alimentari.ID})
	a.addExpense(t, map[string]any{"occurred_on": "2026-03-01", "amount_cents": 3000, "category_id": alimentari.ID})
	a.addIncome(t, map[string]any{"amount_cents": 20000, "category_id": stipendio.ID, "payment_date": "2025-06-15"})

	got := a.savings(t)
	if want := int64(20000 - 5000 - 3000); got.SavingsCents != want {
		t.Errorf("savings_cents = %d, want %d", got.SavingsCents, want)
	}
}

// ADR-0003: an Income only counts once it has a payment date. An invoice sent
// but not yet paid must not inflate Savings.
func TestSavingsExcludesUnpaidIncome(t *testing.T) {
	a := newTestApp(t)
	freelance := a.category(t, "Freelance")

	before := a.savings(t)
	a.addIncome(t, map[string]any{"amount_cents": 99999, "category_id": freelance.ID})

	after := a.savings(t)
	if after.SavingsCents != before.SavingsCents {
		t.Errorf("savings_cents = %d after an unpaid Income, want unchanged %d", after.SavingsCents, before.SavingsCents)
	}
}

// ADR-0009: a buy/sell under the Investments category must not move Savings
// at all — it is neither spent nor saved, it moved into a Holding.
func TestSavingsExcludesTheInvestmentsCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	investments := a.investments(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)

	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 10000, "category_id": alimentari.ID})
	before := a.savings(t)

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-01", "amount_cents": 500000,
		"category_id": investments.ID, "holding_id": vwce.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 200000, "category_id": investments.ID,
		"payment_date": "2026-02-10", "holding_id": vwce.ID,
	})

	after := a.savings(t)
	if after.SavingsCents != before.SavingsCents {
		t.Errorf("savings_cents = %d after a buy/sell, want unchanged %d", after.SavingsCents, before.SavingsCents)
	}
}

// The starting balance is a plain setting, round-tripped through the existing
// PUT /api/settings alongside Target and Goal, and folded straight into
// Savings once set.
func TestStartingBalanceRoundTripsThroughSettingsAndFoldsIntoSavings(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 4000, "category_id": alimentari.ID})

	withoutBalance := a.savings(t)

	if res := a.put(t, settingsPath, map[string]any{"savings_starting_balance_cents": int64(150000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	got := a.savings(t)
	if got.StartingBalanceCents != 150000 {
		t.Errorf("starting_balance_cents = %d, want 150000", got.StartingBalanceCents)
	}
	if want := withoutBalance.SavingsCents + 150000; got.SavingsCents != want {
		t.Errorf("savings_cents = %d, want %d (previous %d + starting balance)", got.SavingsCents, want, withoutBalance.SavingsCents)
	}

	// Setting the starting balance must not disturb Target/Goal, the other
	// two fields riding the same payload.
	if res := a.put(t, settingsPath, map[string]any{"target_cents": int64(9999)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}
	still := a.savings(t)
	if still.StartingBalanceCents != 150000 {
		t.Errorf("starting_balance_cents = %d after an unrelated settings write, want it to stay 150000", still.StartingBalanceCents)
	}
}

// The portfolio percentage math: three Holdings netting 6000/3000/1000 (total
// 10000) split 60/30/10.
func TestPortfolioPercentageAcrossMultipleHoldings(t *testing.T) {
	a := newTestApp(t)
	investments := a.investments(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)
	btc := a.createHolding(t, "BTC", holdingCrypto)
	bond := a.createHolding(t, "Rendite Stato", holdingBond)

	for _, buy := range []struct {
		holding holdingJSON
		cents   int64
	}{
		{vwce, 6000}, {btc, 3000}, {bond, 1000},
	} {
		a.addExpense(t, map[string]any{
			"occurred_on": "2026-01-10", "amount_cents": buy.cents,
			"category_id": investments.ID, "holding_id": buy.holding.ID,
		})
	}

	got := a.savings(t)
	if len(got.Holdings) != 3 {
		t.Fatalf("holdings = %+v, want 3", got.Holdings)
	}
	want := map[int64]struct {
		net     int64
		percent float64
	}{
		vwce.ID: {6000, 60}, btc.ID: {3000, 30}, bond.ID: {1000, 10},
	}
	for _, h := range got.Holdings {
		w, ok := want[h.HoldingID]
		if !ok {
			t.Fatalf("unexpected holding %+v", h)
		}
		if h.NetCents != w.net {
			t.Errorf("holding %d net_cents = %d, want %d", h.HoldingID, h.NetCents, w.net)
		}
		if h.Percent != w.percent {
			t.Errorf("holding %d percent = %v, want %v", h.HoldingID, h.Percent, w.percent)
		}
	}
}

// A fully-sold Holding nets to zero and must be dropped from the list
// entirely, not shown as a 0% row — the ticket's explicit rule, mirroring
// readBreakdown's "a zero remainder is dropped" rule for Categories.
func TestAFullySoldHoldingDropsOutOfTheBreakdown(t *testing.T) {
	a := newTestApp(t)
	investments := a.investments(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)
	btc := a.createHolding(t, "BTC", holdingCrypto)

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 10000,
		"category_id": investments.ID, "holding_id": vwce.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 5000,
		"category_id": investments.ID, "holding_id": btc.ID,
	})
	// Sell all of VWCE back out.
	a.addIncome(t, map[string]any{
		"amount_cents": 10000, "category_id": investments.ID,
		"payment_date": "2026-02-01", "holding_id": vwce.ID,
	})

	got := a.savings(t)
	if len(got.Holdings) != 1 || got.Holdings[0].HoldingID != btc.ID {
		t.Fatalf("holdings = %+v, want just BTC (VWCE fully sold, dropped)", got.Holdings)
	}
	if got.Holdings[0].Percent != 100 {
		t.Errorf("BTC percent = %v, want 100 (the only Holding left)", got.Holdings[0].Percent)
	}
}

// An unpaid sell (no payment_date) must not reduce a Holding's net
// contribution — the same ADR-0003 rule Savings itself follows, applied to
// the breakdown too.
func TestUnpaidSellDoesNotReduceHoldingNet(t *testing.T) {
	a := newTestApp(t)
	investments := a.investments(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 10000,
		"category_id": investments.ID, "holding_id": vwce.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 10000, "category_id": investments.ID, "holding_id": vwce.ID,
	})

	got := a.savings(t)
	if len(got.Holdings) != 1 || got.Holdings[0].NetCents != 10000 {
		t.Fatalf("holdings = %+v, want VWCE still at net 10000 (the sell is unpaid)", got.Holdings)
	}
}

// A Holding never bought or sold never appears at all — there is nothing to
// compute a share of nothing.
func TestAHoldingWithNoTransactionsIsNotInTheBreakdown(t *testing.T) {
	a := newTestApp(t)
	a.createHolding(t, "VWCE", holdingETF)

	got := a.savings(t)
	if len(got.Holdings) != 0 {
		t.Fatalf("holdings = %+v, want none", got.Holdings)
	}
}

// CombinedCents is Savings plus the current portfolio value — the whole
// household's stack, Savings and Holdings summed together (ticket 05).
func TestCombinedCentsIsSavingsPlusPortfolio(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	stipendio := a.category(t, "Stipendio")
	investments := a.investments(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)
	btc := a.createHolding(t, "BTC", holdingCrypto)

	a.addIncome(t, map[string]any{"amount_cents": 100000, "category_id": stipendio.ID, "payment_date": "2026-01-01"})
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 20000, "category_id": alimentari.ID})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-15", "amount_cents": 30000,
		"category_id": investments.ID, "holding_id": vwce.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-20", "amount_cents": 15000,
		"category_id": investments.ID, "holding_id": btc.ID,
	})

	got := a.savings(t)
	var portfolioCents int64
	for _, h := range got.Holdings {
		portfolioCents += h.NetCents
	}
	if want := got.SavingsCents + portfolioCents; got.CombinedCents != want {
		t.Errorf("combined_cents = %d, want %d (savings %d + portfolio %d)", got.CombinedCents, want, got.SavingsCents, portfolioCents)
	}
	if want := int64(100000-20000) + (30000 + 15000); got.CombinedCents != want {
		t.Errorf("combined_cents = %d, want %d", got.CombinedCents, want)
	}
}

// The pace the net worth target is projected at: twelve times the median
// month's saving over the trailing completed months, with an Investments buy
// left out of it (ADR-0009) and the current month — still being spent into —
// never one of them. Goal is set to something else entirely to prove real
// history wins over it once there is enough of it.
func TestYearlySavingsIsTwelveTimesTheMedianMonthsSaving(t *testing.T) {
	a := newTestApp(t)
	stipendio := a.category(t, "Stipendio")
	alimentari := a.category(t, "Alimentari")
	investments := a.investments(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)

	// Three completed months before March 2026, saving 60000, 50000 and
	// 40000 — the median is the middle one, and a mean would answer the same
	// here, so the amounts differ enough that the sort is what is tested.
	for _, m := range []struct {
		month   string
		expense int64
	}{{"2025-12", 40000}, {"2026-01", 50000}, {"2026-02", 60000}} {
		a.addIncome(t, map[string]any{"amount_cents": 100000, "category_id": stipendio.ID, "payment_date": m.month + "-05"})
		a.addExpense(t, map[string]any{"occurred_on": m.month + "-10", "amount_cents": m.expense, "category_id": alimentari.ID})
	}
	// Neither of these may move the pace: a stock buy is not a month's
	// normal spend, and March is the month in progress.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-20", "amount_cents": 500000,
		"category_id": investments.ID, "holding_id": vwce.ID,
	})
	a.addExpense(t, map[string]any{"occurred_on": "2026-03-01", "amount_cents": 999999, "category_id": alimentari.ID})

	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(1)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	got := a.savings(t)
	if want := int64(50000 * 12); got.YearlySavingsCents != want {
		t.Errorf("yearly_savings_cents = %d, want %d (12 × the median of 60000, 50000, 40000)",
			got.YearlySavingsCents, want)
	}
}

// Below Budget's own minimum history there is no median worth trusting, so
// the pace falls back to Goal — the monthly saving the household chose by
// hand — annualised. Two completed months of history is one month short.
func TestYearlySavingsFallsBackToGoalWithoutEnoughHistory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 1000, "category_id": alimentari.ID})
	a.addExpense(t, map[string]any{"occurred_on": "2026-02-10", "amount_cents": 1000, "category_id": alimentari.ID})

	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(30000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	got := a.savings(t)
	if want := int64(30000 * 12); got.YearlySavingsCents != want {
		t.Errorf("yearly_savings_cents = %d, want %d (12 × Goal)", got.YearlySavingsCents, want)
	}
}

// The target itself is a plain setting, round-tripped through the same
// /api/settings payload the starting balance uses and read back on the
// report the page projects from.
func TestNetWorthTargetRoundTripsThroughSettings(t *testing.T) {
	a := newTestApp(t)

	if got := a.savings(t); got.NetWorthTargetCents != 0 {
		t.Fatalf("net_worth_target_cents = %d on a fresh database, want 0", got.NetWorthTargetCents)
	}
	if res := a.put(t, settingsPath, map[string]any{"net_worth_target_cents": int64(50000000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}
	if got := a.savings(t); got.NetWorthTargetCents != 50000000 {
		t.Errorf("net_worth_target_cents = %d, want 50000000", got.NetWorthTargetCents)
	}
}

// A Holding entered only via a manual lot — quotes already owned before the
// household started tracking buys/sells, same as Titoli/Holdings.tsx shows
// it — has no linked Expense/Income, so net_cents must come from
// cost_adjustment_cents alone. Before this, such a Holding computed to 0 and
// was silently dropped from both the breakdown and CombinedCents.
func TestSavingsBreakdownIncludesAManualLotWithNoTransactions(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	if res := a.patch(t, holdingPath(h.ID), map[string]any{
		"manual_lot": map[string]any{"quantity": 10, "price_per_unit_cents": 5000, "replace": false},
	}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH manual_lot = %d, want 200", res.StatusCode)
	}

	got := a.savings(t)
	if len(got.Holdings) != 1 {
		t.Fatalf("holdings = %+v, want the manual-lot Holding present", got.Holdings)
	}
	want := int64(50000)
	if got.Holdings[0].NetCents != want {
		t.Errorf("net_cents = %d, want %d (cost_adjustment_cents alone)", got.Holdings[0].NetCents, want)
	}
	if got.CombinedCents != want {
		t.Errorf("combined_cents = %d, want %d to include the manual lot", got.CombinedCents, want)
	}
}

// A Holding's current value and gain/loss ride along in the breakdown too —
// the same computeHoldingFigures arithmetic Holdings.tsx's Dettagli view
// shows, not a second formula.
func TestSavingsBreakdownIncludesValueNowAndGainLoss(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	if res := a.patch(t, holdingPath(h.ID), map[string]any{
		"manual_lot": map[string]any{"quantity": 10, "price_per_unit_cents": 5000, "replace": false},
	}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH manual_lot = %d, want 200", res.StatusCode)
	}
	if res := a.patch(t, holdingPath(h.ID), map[string]any{"current_price_cents": int64(6000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH current_price_cents = %d, want 200", res.StatusCode)
	}

	got := a.savings(t)
	if len(got.Holdings) != 1 {
		t.Fatalf("holdings = %+v, want one row", got.Holdings)
	}
	row := got.Holdings[0]
	if row.ValueNowCents == nil || *row.ValueNowCents != 60000 {
		t.Errorf("value_now_cents = %v, want 60000 (10 × 6000)", row.ValueNowCents)
	}
	if row.GainLossCents == nil || *row.GainLossCents != 10000 {
		t.Errorf("gain_loss_cents = %v, want 10000 (60000 − 50000)", row.GainLossCents)
	}
	if row.GainLossPercent == nil || *row.GainLossPercent != 20 {
		t.Errorf("gain_loss_percent = %v, want 20", row.GainLossPercent)
	}
}
