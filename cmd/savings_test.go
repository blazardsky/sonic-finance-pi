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
