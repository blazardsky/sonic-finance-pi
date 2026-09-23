package main

import (
	"net/http"
	"testing"
	"time"
)

func (a *testApp) headroom(t *testing.T) headroomReport {
	t.Helper()
	var got headroomReport
	if res := a.get(t, headroomPath, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", headroomPath, res.StatusCode)
	}
	return got
}

func assertHeadroom(t *testing.T, got headroomReport, want []headroomMonth) {
	t.Helper()
	if !got.Available {
		t.Fatal("available = false, want true")
	}
	if len(got.Months) != len(want) {
		t.Fatalf("got %d months, want %d: %+v", len(got.Months), len(want), got.Months)
	}
	for i := range want {
		if got.Months[i] != want[i] {
			t.Errorf("month %d = %+v, want %+v", i, got.Months[i], want[i])
		}
	}
}

// The whole formula in one household, clock at 2026-03-15, history since
// 2025-12 (three completed months: Dec, Jan, Feb):
//
//   - typical Income: Stipendio 200000 every month; the Contract Income and
//     the Investments sale are left out → median 200000
//   - typical non-recurring spending: 50000 / 70000 / 60000+10000
//     Investments → median 70000; the rent Expenses the Recurring expense
//     generated in Jan and Feb are left out
//   - Contract Jan–Jun, 120000 total, 30000 received → 90000 owed over
//     Apr–Jun = 30000 a month, nothing after June
//   - Recurring: rent 85000 ongoing, gym 5000 ending in May
//   - Goal 40000
func TestHeadroomProjectsTheNextSixMonths(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")
	stipendio := a.category(t, seedStipendioName)
	investments := a.investments(t)

	// The rent, generated in January and February (each month materialises
	// when a report reads it).
	a.setNow(t, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	a.addRecurring(t, map[string]any{"amount_cents": 85000, "category_id": casa.ID, "day_of_month": 5})
	a.get(t, "/api/reports/month/2026-01", nil)
	a.setNow(t, time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
	a.get(t, "/api/reports/month/2026-02", nil)
	a.setNow(t, testClock)

	gym := a.addRecurring(t, map[string]any{"amount_cents": 5000, "category_id": casa.ID, "day_of_month": 1})
	if res := a.patch(t, recurringPath(gym.ID), map[string]any{"end_month": "2026-05"}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("ending the gym = %d, want 200", res.StatusCode)
	}

	for _, e := range []struct {
		day   string
		cents int64
		cat   int64
	}{
		{"2025-12-10", 50000, alimentari.ID},
		{"2026-01-10", 70000, alimentari.ID},
		{"2026-02-10", 60000, alimentari.ID},
		{"2026-02-12", 10000, investments.ID},
	} {
		a.addExpense(t, map[string]any{"occurred_on": e.day, "amount_cents": e.cents, "category_id": e.cat})
	}
	for _, day := range []string{"2025-12-27", "2026-01-27", "2026-02-27"} {
		a.addIncome(t, map[string]any{"amount_cents": 200000, "category_id": stipendio.ID, "payment_date": day})
	}

	client := a.createClient(t, "Acme")
	contract := a.createContract(t, client.ID, map[string]any{
		"start_month": "2026-01", "end_month": "2026-06", "total_cents": 120000,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 30000, "category_id": a.freelance(t).ID, "client_id": client.ID,
		"contract_id": contract.ID, "payment_date": "2026-01-31", "bollo_fattura": false,
	})
	vwce := a.createHolding(t, "VWCE", holdingETF)
	a.addIncome(t, map[string]any{
		"amount_cents": 50000, "category_id": investments.ID, "holding_id": vwce.ID, "payment_date": "2026-02-20",
	})

	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(40000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT goal = %d, want 200", res.StatusCode)
	}

	// 130000 typical; +30000 Contract through June; −85000 rent; −5000 gym
	// through May; −40000 Goal.
	assertHeadroom(t, a.headroom(t), []headroomMonth{
		{"2026-04", 30000, 30000},
		{"2026-05", 30000, 60000},
		{"2026-06", 35000, 95000},
		{"2026-07", 5000, 100000},
		{"2026-08", 5000, 105000},
		{"2026-09", 5000, 110000},
	})
}

// A month that costs more than it brings shows negative and eats into what
// the earlier months accumulated.
func TestHeadroomNegativeMonthLowersTheRunningTotal(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	stipendio := a.category(t, seedStipendioName)
	for _, m := range []string{"2025-12", "2026-01", "2026-02"} {
		a.addExpense(t, map[string]any{"occurred_on": m + "-10", "amount_cents": 100000, "category_id": alimentari.ID})
		a.addIncome(t, map[string]any{"amount_cents": 150000, "category_id": stipendio.ID, "payment_date": m + "-27"})
	}
	// Typical 50000 a month; a heavy 80000 Recurring expense running only
	// through April makes April 50000 − 80000 = −30000.
	heavy := a.addRecurring(t, map[string]any{"amount_cents": 80000, "category_id": alimentari.ID, "day_of_month": 1})
	if res := a.patch(t, recurringPath(heavy.ID), map[string]any{"end_month": "2026-04"}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("ending it = %d, want 200", res.StatusCode)
	}

	assertHeadroom(t, a.headroom(t), []headroomMonth{
		{"2026-04", -30000, -30000},
		{"2026-05", 50000, 20000},
		{"2026-06", 50000, 70000},
		{"2026-07", 50000, 120000},
		{"2026-08", 50000, 170000},
		{"2026-09", 50000, 220000},
	})
}

// Budget's rule: fewer than three months of history, no projection.
func TestHeadroomUnavailableWithoutThreeMonthsOfHistory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 1000, "category_id": alimentari.ID})

	got := a.headroom(t)
	if got.Available || len(got.Months) != 0 {
		t.Fatalf("got %+v, want unavailable with no months", got)
	}
}
