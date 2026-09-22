package main

import (
	"fmt"
	"net/http"
	"testing"
)

// One (month, category) line of the full yearly report's breakdown — the
// grouped query ADR-0011 asks for instead of twelve per-month calls.
type monthCategoryLineJSON struct {
	Month       string `json:"month"`
	CategoryID  int64  `json:"category_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

// One month's Spending intent split (ticket 05) — the line chart's own row
// shape, four cents buckets already grouped server-side.
type spendingIntentMonthJSON struct {
	Month          string `json:"month"`
	NecessityCents int64  `json:"necessity_cents"`
	DesireCents    int64  `json:"desire_cents"`
	WiseCents      int64  `json:"wise_cents"`
	BullshitCents  int64  `json:"bullshit_cents"`
}

type fullYearJSON struct {
	Year                     string                    `json:"year"`
	ByMonth                  []monthCategoryLineJSON   `json:"by_month"`
	SpendingIntentByMonth    []spendingIntentMonthJSON `json:"spending_intent_by_month"`
	ExpenseExcludingTaxCents int64                     `json:"expense_excluding_tax_cents"`
	SavingsAtStartCents      int64                     `json:"savings_at_start_cents"`
	MedianExpenseCents       int64                     `json:"median_expense_cents"`
	MedianIncomeCents        int64                     `json:"median_income_cents"`
	MedianNetCents           int64                     `json:"median_net_cents"`
}

func (a *testApp) fullYear(t *testing.T, year string) fullYearJSON {
	t.Helper()
	var got fullYearJSON
	if res := a.get(t, "/api/reports/year/"+year+"/full", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/year/%s/full = %d, want 200", year, res.StatusCode)
	}
	return got
}

// The grouped query's whole ticket: every (month, category) pair with real
// spend in it, across the year, from one endpoint — not the month screen's
// breakdown called twelve times.
func TestTheFullYearReportBreaksEveryMonthDownByCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 4000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-20", "amount_cents": 1000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-05", "amount_cents": 90000, "category_id": casa.ID,
	})
	// A neighbouring year must not leak in.
	a.addExpense(t, map[string]any{
		"occurred_on": "2025-01-10", "amount_cents": 500000, "category_id": alimentari.ID,
	})
	// Money in has no place in an Expense breakdown, same as the month's own.
	a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": a.freelance(t).ID, "payment_date": "2026-01-15",
	})

	got := a.fullYear(t, "2026")
	want := []monthCategoryLineJSON{
		{Month: "2026-01", CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 5000},
		{Month: "2026-02", CategoryID: casa.ID, Category: "Casa", AmountCents: 90000},
	}
	if len(got.ByMonth) != len(want) {
		t.Fatalf("by_month = %+v, want %+v", got.ByMonth, want)
	}
	for i, line := range want {
		if got.ByMonth[i] != line {
			t.Errorf("by_month[%d] = %+v, want %+v", i, got.ByMonth[i], line)
		}
	}
}

// The same attribution rule the month breakdown uses (ADR-0002): an itemised
// line goes to the Item's own Category, and the grouped query must not lose
// that just because it now spans a whole year.
func TestTheFullYearBreakdownAttributesItemsAcrossMonths(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	svago := a.category(t, "Svago")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-11", "amount_cents": 6200, "category_id": alimentari.ID,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago.ID},
		},
	})

	got := a.fullYear(t, "2026")
	want := []monthCategoryLineJSON{
		{Month: "2026-03", CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 4800},
		{Month: "2026-03", CategoryID: svago.ID, Category: "Svago", AmountCents: 1400},
	}
	if len(got.ByMonth) != len(want) {
		t.Fatalf("by_month = %+v, want %+v", got.ByMonth, want)
	}
	for i, line := range want {
		if got.ByMonth[i] != line {
			t.Errorf("by_month[%d] = %+v, want %+v", i, got.ByMonth[i], line)
		}
	}
}

// Ticket 05's own grouped query: an Expense's amount lands in the right
// month and the right bucket — necessity_cents for a plain Necessity,
// desire_cents for a plain Desire (and nowhere else), and both desire_cents
// and wise_cents/bullshit_cents for a refined Desire — the spec's "counted
// from desire_wise/desire_bullshit rows only, a plain desire counts toward
// desire_cents but neither of the other two" rule, plus a same-month sum.
func TestSpendingIntentByMonthSumsExpensesIntoTheRightBuckets(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 4000, "category_id": alimentari.ID,
		"spending_intent": "necessity",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-20", "amount_cents": 1000, "category_id": alimentari.ID,
		"spending_intent": "desire",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-25", "amount_cents": 2000, "category_id": alimentari.ID,
		"spending_intent": "desire_wise",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-28", "amount_cents": 500, "category_id": alimentari.ID,
		"spending_intent": "desire_bullshit",
	})
	// Unclassified: must contribute to none of the four buckets.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-30", "amount_cents": 999999, "category_id": alimentari.ID,
	})
	// A different month, kept separate from January's row.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-05", "amount_cents": 3000, "category_id": alimentari.ID,
		"spending_intent": "necessity",
	})
	// Income never carries a Spending intent — confirms it cannot leak in.
	a.addIncome(t, map[string]any{
		"amount_cents": 500000, "category_id": a.freelance(t).ID, "payment_date": "2026-01-15",
	})

	got := a.fullYear(t, "2026")
	want := []spendingIntentMonthJSON{
		{Month: "2026-01", NecessityCents: 4000, DesireCents: 1000 + 2000 + 500, WiseCents: 2000, BullshitCents: 500},
		{Month: "2026-02", NecessityCents: 3000, DesireCents: 0, WiseCents: 0, BullshitCents: 0},
	}
	if len(got.SpendingIntentByMonth) != len(want) {
		t.Fatalf("spending_intent_by_month = %+v, want %+v", got.SpendingIntentByMonth, want)
	}
	for i, line := range want {
		if got.SpendingIntentByMonth[i] != line {
			t.Errorf("spending_intent_by_month[%d] = %+v, want %+v", i, got.SpendingIntentByMonth[i], line)
		}
	}
}

// A month with nothing classified at all has no row — the spec's story 21
// (a flat/empty line, not a misleading zero) starts with the backend simply
// never answering a row for it.
func TestAMonthWithNothingClassifiedHasNoSpendingIntentRow(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-04-10", "amount_cents": 1500, "category_id": alimentari.ID,
	})

	got := a.fullYear(t, "2026")
	if len(got.SpendingIntentByMonth) != 0 {
		t.Errorf("spending_intent_by_month = %+v, want empty — nothing was classified", got.SpendingIntentByMonth)
	}
}

// An empty year has an empty breakdown, marshalled as [] rather than null —
// the same bargain every other list report here makes.
func TestAnEmptyYearHasAnEmptyFullYearBreakdown(t *testing.T) {
	a := newTestApp(t)

	if got := a.fullYear(t, "2019"); len(got.ByMonth) != 0 {
		t.Errorf("by_month = %+v, want empty", got.ByMonth)
	}
	var raw struct {
		ByMonth               *[]monthCategoryLineJSON   `json:"by_month"`
		SpendingIntentByMonth *[]spendingIntentMonthJSON `json:"spending_intent_by_month"`
	}
	a.get(t, "/api/reports/year/2019/full", &raw)
	if raw.ByMonth == nil {
		t.Error("by_month came back as null, want []")
	}
	if raw.SpendingIntentByMonth == nil {
		t.Error("spending_intent_by_month came back as null, want []")
	}
}

// The tax-excluded total: what was actually spent living, separate from what
// was paid the state.
func TestTheFullYearReportExcludesTaxesFromItsExpenseTotal(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 100000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-06-15", "amount_cents": 30000, "category_id": a.taxes(t).ID,
	})

	got := a.fullYear(t, "2026")
	if got.ExpenseExcludingTaxCents != 100000 {
		t.Errorf("expense_excluding_tax_cents = %d, want 100000 — tax was counted", got.ExpenseExcludingTaxCents)
	}
}

// testClock is 2026-03-15 (reports_test.go's convention), so 2026's completed
// months are January and February only — March is still in progress.
//
// The whole distinction from Budget in one test: a fully completed past year
// medians over all twelve of its months, not "the last 12 months from today".
func TestTheFullYearMedianIsScopedToThatYearsOwnCompletedMonths(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	// 2025 is entirely in the past relative to testClock: all twelve months
	// count. Sorted expense totals: mostly 1000, one month at 100000 —
	// median should land on 1000, not be dragged toward a mean.
	for m := 1; m <= 12; m++ {
		cents := int64(1000)
		if m == 12 {
			cents = 100000
		}
		a.addExpense(t, map[string]any{
			"occurred_on":  fmt.Sprintf("2025-%02d-10", m),
			"amount_cents": cents, "category_id": alimentari.ID,
		})
	}

	got := a.fullYear(t, "2025")
	if got.MedianExpenseCents != 1000 {
		t.Errorf("median_expense_cents = %d, want 1000 (median of eleven 1000s and one 100000)",
			got.MedianExpenseCents)
	}

	// The current year (2026) has only January and February completed —
	// March, in progress under testClock, must not be counted, and neither
	// may any later month.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 2000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-10", "amount_cents": 4000, "category_id": alimentari.ID,
	})
	// In progress: must be excluded from the median even though it has spend.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-10", "amount_cents": 999999, "category_id": alimentari.ID,
	})

	got2026 := a.fullYear(t, "2026")
	if want := int64((2000 + 4000) / 2); got2026.MedianExpenseCents != want {
		t.Errorf("2026 median_expense_cents = %d, want %d (Jan/Feb only, March excluded)",
			got2026.MedianExpenseCents, want)
	}
}

// A year with no completed months yet — the current year before any month of
// it has finished — medians over nothing rather than panicking or answering
// a stale figure.
func TestTheFullYearMedianIsZeroWithNoCompletedMonths(t *testing.T) {
	a := newTestApp(t)
	got := a.fullYear(t, "2027")
	if got.MedianExpenseCents != 0 || got.MedianIncomeCents != 0 || got.MedianNetCents != 0 {
		t.Errorf("a future year's medians = %+v, want all zero", got)
	}
}

// Savings at the start of the year: the existing all-time formula
// (cmd/savings.go's computeSavingsCents), cut off at December 31st of the
// prior year instead of running through today.
func TestSavingsAtStartOfYearIsCutOffAtThePriorDecember31st(t *testing.T) {
	a := newTestApp(t)
	stipendio := a.category(t, "Stipendio")
	alimentari := a.category(t, "Alimentari")

	// Before 2026 began: 50000 in, 10000 out — this is what "start of 2026"
	// should read.
	a.addIncome(t, map[string]any{
		"amount_cents": 50000, "category_id": stipendio.ID, "payment_date": "2025-06-01",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2025-07-01", "amount_cents": 10000, "category_id": alimentari.ID,
	})
	// During 2026 itself — must not be counted in "start of year".
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-15", "amount_cents": 5000, "category_id": alimentari.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 20000, "category_id": stipendio.ID, "payment_date": "2026-02-01",
	})

	got := a.fullYear(t, "2026")
	if want := int64(50000 - 10000); got.SavingsAtStartCents != want {
		t.Errorf("savings_at_start_cents = %d, want %d — 2026's own activity leaked in",
			got.SavingsAtStartCents, want)
	}

	// The all-time figure, for contrast, does include everything.
	allTime := a.savings(t)
	if want := int64(50000 - 10000 - 5000 + 20000); allTime.SavingsCents != want {
		t.Errorf("all-time savings_cents = %d, want %d", allTime.SavingsCents, want)
	}
}

// A year that is not a year is refused, the same as every other report
// keyed by one.
func TestTheFullYearReportRefusesAYearThatIsNotAYear(t *testing.T) {
	a := newTestApp(t)

	for _, year := range []string{"26", "2026-03", "twenty"} {
		if res := a.get(t, "/api/reports/year/"+year+"/full", nil); res.StatusCode == http.StatusOK {
			t.Errorf("GET /api/reports/year/%q/full = 200, want a refusal", year)
		}
	}
}
