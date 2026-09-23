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

func months(running ...int64) []headroomMonth {
	names := nextMonths(func() time.Time { return testClock }, len(running))
	out := make([]headroomMonth, len(running))
	for i, r := range running {
		out[i] = headroomMonth{Month: names[i], RunningCents: r}
	}
	return out
}

func purchases(amounts ...int64) []plannedPurchase {
	out := make([]plannedPurchase, len(amounts))
	for i, a := range amounts {
		out[i] = plannedPurchase{ID: int64(i + 1), AmountCents: a, Position: int64(i + 1)}
	}
	return out
}

// Priority decides who gets Headroom first, and a placed purchase uses up
// its amount from its month on.
func TestPlaceInPriorityOrder(t *testing.T) {
	// Running 100, 200, 300, 400, 500, 600.
	got := place(months(100, 200, 300, 400, 500, 600), purchases(150, 150), 0)
	// The first fits from May (200); after it the balance is 100, 50, 150,
	// 250, 350, 450, so the second fits from June (exactly 150).
	if got[0].Month != "2026-05" || got[1].Month != "2026-06" {
		t.Fatalf("placed in %q and %q, want 2026-05 and 2026-06", got[0].Month, got[1].Month)
	}
}

// One that doesn't fit consumes nothing: the smaller one after it still
// lands, and the gap is measured against the balance at the end of the
// horizon after higher-priority placements.
func TestPlaceSkipsWithoutBlocking(t *testing.T) {
	got := place(months(100, 200, 300, 400, 500, 600), purchases(100, 5000, 50), 1000)
	if got[0].Month != "2026-04" {
		t.Errorf("first placed in %q, want 2026-04", got[0].Month)
	}
	if got[1].Month != "" {
		t.Fatalf("the 5000 one placed in %q, want it not to fit", got[1].Month)
	}
	// 600 - 100 (the first) = 500 left at the end, so 4500 missing.
	if got[1].MissingCents != 4500 {
		t.Errorf("missing = %d, want 4500", got[1].MissingCents)
	}
	if got[1].SavingsCover {
		t.Error("savings 1000 reported as covering a 4500 gap")
	}
	// The first used all of April, so the 50 one lands in May, where the
	// balance (100) covers it. Blocked by the skipped one, it would land nowhere.
	if got[2].Month != "2026-05" {
		t.Errorf("the 50 one placed in %q, want 2026-05: the skipped one must not block it", got[2].Month)
	}
}

func TestPlaceSavingsCoverTheGap(t *testing.T) {
	got := place(months(100, 100, 100, 100, 100, 100), purchases(400), 300)
	if got[0].Month != "" || got[0].MissingCents != 300 || !got[0].SavingsCover {
		t.Fatalf("got %+v, want not placed, 300 missing, covered by 300 savings", got[0])
	}
}

// A later negative month must not push the account below zero after
// buying: the purchase waits until the balance holds from then on.
func TestPlaceWaitsOutALaterDip(t *testing.T) {
	// Running 300, 300, 100 (a bad month), 200, 300, 400.
	got := place(months(300, 300, 100, 200, 300, 400), purchases(250), 0)
	if got[0].Month != "2026-08" {
		t.Fatalf("placed in %q, want 2026-08, the first month the balance stays at 250 or more", got[0].Month)
	}
}

// With no history the report is unavailable and places nothing, even with
// Planned purchases saved.
func TestHeadroomPlacementsEmptyWhenUnavailable(t *testing.T) {
	a := newTestApp(t)
	a.addPlanned(t, "Laptop", 150000)
	got := a.headroom(t)
	if got.Available || len(got.Placements) != 0 {
		t.Fatalf("got %+v, want unavailable with no placements", got)
	}
}

// End to end: the endpoint places the saved list with the household's real
// Savings.
func TestHeadroomReportPlacesThePlannedPurchases(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	stipendio := a.category(t, seedStipendioName)
	for _, m := range []string{"2025-12", "2026-01", "2026-02"} {
		a.addExpense(t, map[string]any{"occurred_on": m + "-10", "amount_cents": 100000, "category_id": alimentari.ID})
		a.addIncome(t, map[string]any{"amount_cents": 150000, "category_id": stipendio.ID, "payment_date": m + "-27"})
	}
	// Headroom 50000 a month, so running 50000 ... 300000. Savings: 3 x 50000.
	laptop := a.addPlanned(t, "Laptop", 120000)
	car := a.addPlanned(t, "Auto", 500000)

	got := a.headroom(t).Placements
	if len(got) != 2 {
		t.Fatalf("got %d placements, want 2", len(got))
	}
	if got[0].PlannedID != laptop.ID || got[0].Month != "2026-06" {
		t.Errorf("laptop = %+v, want placed in 2026-06 (running 150000 covers 120000)", got[0])
	}
	// 300000 - 120000 = 180000 left, so 320000 missing; Savings 150000 don't cover it.
	if got[1].PlannedID != car.ID || got[1].Month != "" || got[1].MissingCents != 320000 || got[1].SavingsCover {
		t.Errorf("car = %+v, want not placed, 320000 missing, not covered", got[1])
	}
}
