package main

import (
	"net/http"
	"testing"
)

// taxReserveHousehold seeds, clock at 2026-03-15, three months of history
// (Dec, Jan, Feb): 100000 salary, 30000 of freelance work with no Contract
// and 20000 of groceries every month; a €500 tax payment in January; and a
// Contract Jan–Jun owing 60000, so 20000 a month over April–June.
func taxReserveHousehold(t *testing.T) *testApp {
	t.Helper()
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	stipendio := a.category(t, seedStipendioName)
	freelance := a.freelance(t)
	tasse := a.category(t, seedTaxesName)
	client := a.createClient(t, "Acme")
	for _, m := range []string{"2025-12", "2026-01", "2026-02"} {
		a.addExpense(t, map[string]any{"occurred_on": m + "-10", "amount_cents": 20000, "category_id": alimentari.ID})
		a.addIncome(t, map[string]any{"amount_cents": 100000, "category_id": stipendio.ID, "payment_date": m + "-27"})
		a.addIncome(t, map[string]any{
			"amount_cents": 30000, "category_id": freelance.ID, "client_id": client.ID,
			"payment_date": m + "-15", "bollo_fattura": false,
		})
	}
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-20", "amount_cents": 50000, "category_id": tasse.ID, "tax_year": 2025})
	a.createContract(t, client.ID, map[string]any{"start_month": "2026-01", "end_month": "2026-06", "total_cents": 60000})
	return a
}

func (a *testApp) setSelfEmployed(t *testing.T, on bool) {
	t.Helper()
	if res := a.put(t, settingsPath, map[string]any{"self_employed": on}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT self_employed = %d, want 200", res.StatusCode)
	}
}

// With nobody self-employed, the forecast is exactly today's: the tax
// payment is ordinary spending (Jan 70000, Feb 20000 → median 45000) and
// nothing is reserved. April: 130000 + 20000 Contract − 45000 = 105000.
func TestTaxReserveOffIsTodaysForecast(t *testing.T) {
	a := taxReserveHousehold(t)
	got := a.headroom(t)
	april := got.Months[0]
	if april.TaxReserveCents != 0 || got.StartTaxReserveCents != 0 || got.TaxReservePercent != 0 || got.TaxReserveSource != "" {
		t.Errorf("switch off reserved something: %+v", got)
	}
	if april.ForecastSpendingCents != 45000 || april.HeadroomCents != 105000 {
		t.Errorf("April spending %d, headroom %d, want 45000 and 105000", april.ForecastSpendingCents, april.HeadroomCents)
	}
}

// Self-employed, no completed tax year: the 33% fallback on April's
// freelance money — the 20000 Contract share plus the 30000 freelance
// forecast — is 16500; salary is never reserved against. The tax payment
// leaves spending (median 20000, the reserve stands in for it): April =
// 130000 + 20000 − 20000 − 16500 = 113500. The start: February's 30000
// freelance → 9900 reserved; 130000 − 20000 − 9900 = 100100.
func TestTaxReserveFallbackRate(t *testing.T) {
	a := taxReserveHousehold(t)
	a.setSelfEmployed(t, true)

	got := a.headroom(t)
	if got.TaxReserveSource != "fallback" || got.TaxReservePercent != 33 {
		t.Errorf("rate = %v%% from %q, want 33%% from fallback", got.TaxReservePercent, got.TaxReserveSource)
	}
	april := got.Months[0]
	if april.FreelanceForecastCents != 30000 || april.TaxReserveCents != 16500 {
		t.Errorf("April freelance %d, reserve %d, want 30000 and 16500", april.FreelanceForecastCents, april.TaxReserveCents)
	}
	if april.ForecastSpendingCents != 20000 || april.HeadroomCents != 113500 {
		t.Errorf("April spending %d, headroom %d, want 20000 and 113500", april.ForecastSpendingCents, april.HeadroomCents)
	}
	if got.StartTaxReserveCents != 9900 || got.StartCents != 100100 {
		t.Errorf("start reserve %d, start %d, want 9900 and 100100", got.StartTaxReserveCents, got.StartCents)
	}
}

// The Goal buffer comes after the reserve: with Goal 120000, April's
// 113500 left after tax is under the Goal → 0. (Buffering first would give
// 130000 − 120000 = 10000, then −16500 → −6500.)
func TestTaxReserveBeforeGoal(t *testing.T) {
	a := taxReserveHousehold(t)
	a.setSelfEmployed(t, true)
	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(120000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT goal = %d, want 200", res.StatusCode)
	}
	if april := a.headroom(t).Months[0]; april.HeadroomCents != 0 {
		t.Errorf("April headroom = %d, want 0", april.HeadroomCents)
	}
}

// Completed tax years give the household's own rate: 2023 (35000 tax on
// 100000 received → 35%) and 2024 (25000 on 100000 → 25%), median 30%.
// 2025's 40% is ignored — its taxes are still being paid in 2026.
func TestTaxReserveRateFromCompletedTaxYears(t *testing.T) {
	a := taxReserveHousehold(t)
	a.setSelfEmployed(t, true)
	tasse := a.category(t, seedTaxesName)
	freelance := a.freelance(t)
	client := a.createClient(t, "Beta")
	for _, y := range []struct {
		received, paidOn string
		taxYear          int
		tax              int64
	}{
		{"2023-05-10", "2024-06-16", 2023, 35000},
		{"2024-05-10", "2025-06-16", 2024, 25000},
		{"2025-05-10", "2026-02-16", 2025, 40000},
	} {
		a.addIncome(t, map[string]any{
			"amount_cents": 100000, "category_id": freelance.ID, "client_id": client.ID,
			"payment_date": y.received, "bollo_fattura": false,
		})
		a.addExpense(t, map[string]any{"occurred_on": y.paidOn, "amount_cents": y.tax, "category_id": tasse.ID, "tax_year": y.taxYear})
	}

	got := a.headroom(t)
	if got.TaxReserveSource != "history" || got.TaxReservePercent != 30 {
		t.Errorf("rate = %v%% from %q, want 30%% from history", got.TaxReservePercent, got.TaxReserveSource)
	}
}

// While the reserve applies, tax payments leave last month's actual leftover
// and Tasse Recurring expenses leave the Recurring due: a €400 tax payment in
// February and a €100 Tasse Recurring expense. Off, both count: start
// 130000 - (20000 + 40000) = 70000, April's Recurring 10000. On: start
// 130000 - 20000 - 9900 reserve = 100100, April's Recurring 0.
func TestTaxReserveLeavesTaxesOutOfTheStartAndRecurring(t *testing.T) {
	a := taxReserveHousehold(t)
	tasse := a.category(t, seedTaxesName)
	a.addExpense(t, map[string]any{"occurred_on": "2026-02-16", "amount_cents": 40000, "category_id": tasse.ID, "tax_year": 2025})
	a.addRecurring(t, map[string]any{"amount_cents": 10000, "category_id": tasse.ID, "day_of_month": 16})

	off := a.headroom(t)
	if off.StartCents != 70000 || off.Months[0].RecurringCents != 10000 {
		t.Errorf("off: start %d, April recurring %d, want 70000 and 10000", off.StartCents, off.Months[0].RecurringCents)
	}
	a.setSelfEmployed(t, true)
	on := a.headroom(t)
	if on.StartCents != 100100 || on.Months[0].RecurringCents != 0 {
		t.Errorf("on: start %d, April recurring %d, want 100100 and 0", on.StartCents, on.Months[0].RecurringCents)
	}
}

// A completed year with Income but no tax on record is more likely
// unrecorded than tax-free, so it doesn't count; neither does this year,
// whose tax is paid next year. Nothing counts, so the fallback.
func TestTaxReserveIgnoresYearsWithoutTaxAndThisYear(t *testing.T) {
	a := taxReserveHousehold(t)
	a.setSelfEmployed(t, true)
	tasse := a.category(t, seedTaxesName)
	freelance := a.freelance(t)
	client := a.createClient(t, "Beta")
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": freelance.ID, "client_id": client.ID,
		"payment_date": "2024-05-10", "bollo_fattura": false,
	})
	a.addExpense(t, map[string]any{"occurred_on": "2026-03-02", "amount_cents": 90000, "category_id": tasse.ID, "tax_year": 2026})

	if got := a.headroom(t); got.TaxReserveSource != "fallback" || got.TaxReservePercent != 33 {
		t.Errorf("rate = %v%% from %q, want 33%% from fallback", got.TaxReservePercent, got.TaxReserveSource)
	}
}

// A 0% rate reserves nothing, so tax payments must stay in spending rather
// than vanish: exactly the switched-off forecast.
func TestTaxReserveAtZeroPercentIsNoReserve(t *testing.T) {
	a := taxReserveHousehold(t)
	if res := a.put(t, settingsPath, map[string]any{"self_employed": true, "tax_reserve_fallback_percent": 0}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT = %d, want 200", res.StatusCode)
	}
	got := a.headroom(t)
	if got.TaxReserveSource != "" || got.Months[0].ForecastSpendingCents != 45000 || got.Months[0].HeadroomCents != 105000 {
		t.Errorf("source %q, April spending %d, headroom %d, want empty, 45000 and 105000",
			got.TaxReserveSource, got.Months[0].ForecastSpendingCents, got.Months[0].HeadroomCents)
	}
}
