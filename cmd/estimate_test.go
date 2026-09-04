package main

import (
	"net/http"
	"testing"
	"time"
)

// The yearly Estimate as the API hands it out (ticket 05, ADR-0010). Each
// *_estimate_cents is a pointer because null — not zero — is how an
// undefined ratio is spelled, the same convention taxJSON.NetPercent already
// uses.
type estimateJSON struct {
	Year                 string `json:"year"`
	Available            bool   `json:"available"`
	IncomeEstimateCents  *int64 `json:"income_estimate_cents"`
	ExpenseEstimateCents *int64 `json:"expense_estimate_cents"`
	TaxEstimateCents     *int64 `json:"tax_estimate_cents"`
}

func (a *testApp) estimate(t *testing.T, year string) estimateJSON {
	t.Helper()
	var got estimateJSON
	if res := a.get(t, "/api/reports/estimate/"+year, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/estimate/%s = %d, want 200", year, res.StatusCode)
	}
	return got
}

// The whole ticket in one test: this year's January+February against last
// year's, scaled onto last year's full twelve months — and March, the
// in-progress month at testClock (2026-03-15), must not move it. Numbers are
// chosen so the ratio (2) and the projection divide out exactly.
func TestEstimateProjectsFromLastYearsPaceAtTheSamePoint(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)

	// Last year: Jan+Feb (the same point) = 1500, full year = 6000.
	for _, e := range []struct {
		month string
		cents int64
	}{
		{"2025-01", 500}, {"2025-02", 1000}, {"2025-12", 4500},
	} {
		a.addExpense(t, map[string]any{
			"occurred_on": e.month + "-10", "amount_cents": e.cents, "category_id": alimentari.ID,
		})
	}
	for _, i := range []struct {
		month string
		cents int64
	}{
		{"2025-01", 2000}, {"2025-02", 4000}, {"2025-12", 18000},
	} {
		a.addIncome(t, map[string]any{
			"payment_date": i.month + "-10", "amount_cents": i.cents, "category_id": freelance.ID,
		})
	}

	// This year: Jan+Feb = 3000 (double last year's), March — in progress at
	// testClock — is a huge outlier that must be excluded from YTD entirely.
	for _, e := range []struct {
		month string
		cents int64
	}{
		{"2026-01", 1000}, {"2026-02", 2000}, {"2026-03", 999999},
	} {
		a.addExpense(t, map[string]any{
			"occurred_on": e.month + "-10", "amount_cents": e.cents, "category_id": alimentari.ID,
		})
	}
	for _, i := range []struct {
		month string
		cents int64
	}{
		{"2026-01", 4000}, {"2026-02", 8000}, {"2026-03", 5000000},
	} {
		a.addIncome(t, map[string]any{
			"payment_date": i.month + "-10", "amount_cents": i.cents, "category_id": freelance.ID,
		})
	}

	got := a.estimate(t, "2026")
	if got.Year != "2026" || !got.Available {
		t.Fatalf("estimate = %+v, want year 2026 and available", got)
	}
	if got.ExpenseEstimateCents == nil || *got.ExpenseEstimateCents != 12000 {
		t.Errorf("expense_estimate_cents = %v, want 12000 (6000 last year x2 pace) — "+
			"March's outlier leaked into YTD if this is far off", got.ExpenseEstimateCents)
	}
	if got.IncomeEstimateCents == nil || *got.IncomeEstimateCents != 48000 {
		t.Errorf("income_estimate_cents = %v, want 48000 (24000 last year x2 pace)", got.IncomeEstimateCents)
	}
}

// A second point in the year, to pin the completed-months cutoff down at a
// length other than two: at 2026-09-15, January through August (eight whole
// months) are complete, September is not.
func TestEstimateAtADifferentPointInTheYear(t *testing.T) {
	a := newTestApp(t)
	a.setNow(t, time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC))
	alimentari := a.category(t, "Alimentari")

	for _, month := range []string{"2025-01", "2025-02", "2025-03", "2025-04",
		"2025-05", "2025-06", "2025-07", "2025-08"} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": 500, "category_id": alimentari.ID})
	}
	// The last-year YTD-at-the-same-point is 8 x 500 = 4000. September through
	// December brings the full year to 4000 + 4x1000 = 8000.
	for _, month := range []string{"2025-09", "2025-10", "2025-11", "2025-12"} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": 1000, "category_id": alimentari.ID})
	}
	// This year's same eight months run at half the pace: 8 x 250 = 2000.
	for _, month := range []string{"2026-01", "2026-02", "2026-03", "2026-04",
		"2026-05", "2026-06", "2026-07", "2026-08"} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": 250, "category_id": alimentari.ID})
	}
	// September itself — in progress — must not count.
	a.addExpense(t, map[string]any{"occurred_on": "2026-09-05", "amount_cents": 999999, "category_id": alimentari.ID})

	got := a.estimate(t, "2026")
	if got.ExpenseEstimateCents == nil || *got.ExpenseEstimateCents != 4000 {
		t.Errorf("expense_estimate_cents = %v, want 4000 (8000 last year at half pace)", got.ExpenseEstimateCents)
	}
}

// ADR-0009: Investments is excluded from both the income and the expense
// side of the Estimate, the same as Budget — a one-off stock buy or sell
// must not move either projection.
func TestEstimateExcludesInvestmentsOnBothSides(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)
	investments := a.investments(t)

	for _, month := range []string{"2025-01", "2026-01"} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": 1000, "category_id": alimentari.ID})
		a.addIncome(t, map[string]any{"payment_date": month + "-10", "amount_cents": 2000, "category_id": freelance.ID})
	}

	before := a.estimate(t, "2026")

	// A large stock buy and a large stock sale, in both years' YTD windows.
	a.addExpense(t, map[string]any{"occurred_on": "2025-01-15", "amount_cents": 500000, "category_id": investments.ID})
	a.addIncome(t, map[string]any{"payment_date": "2026-01-15", "amount_cents": 900000, "category_id": investments.ID})

	after := a.estimate(t, "2026")
	if *after.ExpenseEstimateCents != *before.ExpenseEstimateCents {
		t.Errorf("expense_estimate_cents = %v after an Investments buy, want unchanged %v",
			after.ExpenseEstimateCents, before.ExpenseEstimateCents)
	}
	if *after.IncomeEstimateCents != *before.IncomeEstimateCents {
		t.Errorf("income_estimate_cents = %v after an Investments sale, want unchanged %v",
			after.IncomeEstimateCents, before.IncomeEstimateCents)
	}
}

// A household with nothing recorded before this year has no prior year to
// project from at all (ADR-0010's named consequence): every metric is
// unavailable, not just zero.
func TestEstimateUnavailableWithNoPriorYear(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 1000, "category_id": alimentari.ID})

	got := a.estimate(t, "2026")
	if got.Available {
		t.Errorf("available = true with no prior year at all, want false: %+v", got)
	}
	if got.IncomeEstimateCents != nil || got.ExpenseEstimateCents != nil || got.TaxEstimateCents != nil {
		t.Errorf("estimate = %+v, want every metric nil with no prior year", got)
	}
}

// The unavailable case is per metric, not all-or-nothing: a metric whose
// last-year YTD-at-the-same-point is zero is nulled on its own, and a sibling
// metric with real history still answers.
func TestEstimateOmitsOnlyTheMetricWithAZeroLastYearYTD(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)

	// Income has real Jan/Feb 2025 history.
	a.addIncome(t, map[string]any{"payment_date": "2025-01-10", "amount_cents": 1000, "category_id": freelance.ID})
	a.addIncome(t, map[string]any{"payment_date": "2026-01-10", "amount_cents": 1000, "category_id": freelance.ID})

	// Expense has 2025 history, but none in Jan/Feb — only December, after the
	// YTD-at-the-same-point cutoff.
	a.addExpense(t, map[string]any{"occurred_on": "2025-12-10", "amount_cents": 5000, "category_id": alimentari.ID})
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 1000, "category_id": alimentari.ID})

	got := a.estimate(t, "2026")
	if !got.Available {
		t.Fatalf("available = false, want true — income has a usable prior year: %+v", got)
	}
	if got.IncomeEstimateCents == nil {
		t.Errorf("income_estimate_cents = nil, want a real projection")
	}
	if got.ExpenseEstimateCents != nil {
		t.Errorf("expense_estimate_cents = %v, want nil — last year's Jan/Feb expense YTD is zero",
			*got.ExpenseEstimateCents)
	}
}

// Tax is its own figure, attributed by Tax year rather than occurred_on
// (ADR-0008), and reads the same YTD cutoff as income/expense: March, in
// progress at testClock, must not count.
func TestEstimateProjectsTaxFromLastYearsPace(t *testing.T) {
	a := newTestApp(t)
	taxes := a.taxes(t)

	a.addExpense(t, map[string]any{"occurred_on": "2025-01-10", "amount_cents": 200, "category_id": taxes.ID})
	a.addExpense(t, map[string]any{"occurred_on": "2025-02-10", "amount_cents": 100, "category_id": taxes.ID})
	// Rest of 2025, after the same-point cutoff — only counts toward the
	// full-year total, not the YTD half of the ratio.
	a.addExpense(t, map[string]any{"occurred_on": "2025-11-10", "amount_cents": 900, "category_id": taxes.ID})

	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 150, "category_id": taxes.ID})
	a.addExpense(t, map[string]any{"occurred_on": "2026-02-10", "amount_cents": 150, "category_id": taxes.ID})
	// In progress this month — must not count.
	a.addExpense(t, map[string]any{"occurred_on": "2026-03-10", "amount_cents": 99999, "category_id": taxes.ID})

	got := a.estimate(t, "2026")
	// last year YTD = 300, this year YTD = 300, ratio 1; last year full = 1200.
	if got.TaxEstimateCents == nil || *got.TaxEstimateCents != 1200 {
		t.Errorf("tax_estimate_cents = %v, want 1200", got.TaxEstimateCents)
	}
}

// A year that is not a year is refused, the same as every other report
// keyed on one.
func TestAnEstimateForAYearThatIsNotAYearIsRefused(t *testing.T) {
	a := newTestApp(t)
	for _, year := range []string{"26", "2026-03", "twenty", ""} {
		res := a.get(t, "/api/reports/estimate/"+year, nil)
		if res.StatusCode == http.StatusOK {
			t.Errorf("GET /api/reports/estimate/%q = 200, want a refusal", year)
		}
	}
}
