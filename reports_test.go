package main

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

// The month report as the API hands it out: three numbers and the month they
// belong to. Money in is received Income only — an unpaid one counts toward
// nothing, which is what most of these tests are about.
type monthJSON struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
	NetCents     int64  `json:"net_cents"`
}

func (a *testApp) month(t *testing.T, month string) monthJSON {
	t.Helper()
	var got monthJSON
	if res := a.get(t, "/api/reports/month/"+month, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/month/%s = %d, want 200", month, res.StatusCode)
	}
	return got
}

// The whole ticket in one test: what came in, what went out, and the
// difference between them.
func TestAMonthReportsWhatCameInWhatWentOutAndTheDifference(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 4237, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-28", "amount_cents": 1250, "category_id": alimentari.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-10",
	})

	got := a.month(t, "2026-03")
	want := monthJSON{Month: "2026-03", IncomeCents: 150000, ExpenseCents: 5487, NetCents: 144513}
	if got != want {
		t.Errorf("month = %+v, want %+v", got, want)
	}
}

// The assertion ticket 09 could not make, because there was no total to be
// excluded from: an unpaid Income is absent from money in, and present the
// moment it has a payment date. ADR-0003 calls the missing filter the single
// most likely bug in this codebase — this is the test that would catch it.
func TestAnUnpaidIncomeIsAbsentFromTheMonthUntilItIsPaid(t *testing.T) {
	a := newTestApp(t)
	invoice := a.addIncome(t, map[string]any{
		"amount_cents":      80000,
		"category_id":       a.freelance(t).ID,
		"invoice_sent_date": "2026-03-01",
	})

	if got := a.month(t, "2026-03"); got.IncomeCents != 0 || got.NetCents != 0 {
		t.Fatalf("unpaid: income = %d, net = %d, want 0 and 0 — money that has not arrived was counted",
			got.IncomeCents, got.NetCents)
	}

	if res := a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": "2026-03-20"}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("recording the payment = %d, want 200", res.StatusCode)
	}
	if got := a.month(t, "2026-03"); got.IncomeCents != 80000 || got.NetCents != 80000 {
		t.Errorf("paid: income = %d, net = %d, want 80000 and 80000", got.IncomeCents, got.NetCents)
	}

	// And back out again, because a payment typed onto the wrong Income is
	// taken back the same way it was recorded.
	a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": ""}, nil)
	if got := a.month(t, "2026-03"); got.IncomeCents != 0 {
		t.Errorf("payment taken back: income = %d, want 0", got.IncomeCents)
	}
}

// An Income belongs to the month its money arrived, not the month its invoice
// went out. invoice_sent_date answers "how long has this been sitting", which
// is a different question and a different ticket.
func TestAnIncomeBelongsToTheMonthItWasPaidNotTheMonthItWasInvoiced(t *testing.T) {
	a := newTestApp(t)
	a.addIncome(t, map[string]any{
		"amount_cents":      50000,
		"category_id":       a.freelance(t).ID,
		"invoice_sent_date": "2026-02-20",
		"payment_date":      "2026-03-05",
	})

	if got := a.month(t, "2026-02"); got.IncomeCents != 0 {
		t.Errorf("February income = %d, want 0 — it was counted in the month it was invoiced", got.IncomeCents)
	}
	if got := a.month(t, "2026-03"); got.IncomeCents != 50000 {
		t.Errorf("March income = %d, want 50000", got.IncomeCents)
	}
}

// The calendar month is the reporting unit, and its edges are the 1st and the
// last day: neither neighbour leaks in.
func TestAMonthCountsOnlyItsOwnEntries(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)

	for _, date := range []string{"2026-02-28", "2026-03-01", "2026-03-31", "2026-04-01"} {
		a.addExpense(t, map[string]any{
			"occurred_on": date, "amount_cents": 1000, "category_id": alimentari.ID,
		})
		a.addIncome(t, map[string]any{
			"amount_cents": 2000, "category_id": freelance.ID, "payment_date": date,
		})
	}

	got := a.month(t, "2026-03")
	want := monthJSON{Month: "2026-03", IncomeCents: 4000, ExpenseCents: 2000, NetCents: 2000}
	if got != want {
		t.Errorf("month = %+v, want %+v — a neighbouring month leaked in", got, want)
	}
}

// A month nobody spent anything in is zeros, not an error: every household
// starts there, and so does every month before the app was installed.
func TestAMonthWithNothingInItIsZeros(t *testing.T) {
	a := newTestApp(t)

	got := a.month(t, "2019-07")
	want := monthJSON{Month: "2019-07"}
	if got != want {
		t.Errorf("month = %+v, want %+v", got, want)
	}
}

// The difference is a difference, not a total: a month that spent more than it
// received says so.
func TestTheDifferenceGoesNegativeWhenMoreWentOutThanCameIn(t *testing.T) {
	a := newTestApp(t)
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-09", "amount_cents": 90000, "category_id": a.category(t, "Casa").ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 25000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-09",
	})

	if got := a.month(t, "2026-03"); got.NetCents != -65000 {
		t.Errorf("net = %d, want -65000", got.NetCents)
	}
}

// Anything that is not a calendar month is refused, rather than summing a
// substring nothing matches and answering zeros as if the month were empty.
func TestAMonthThatIsNotAMonthIsRefused(t *testing.T) {
	a := newTestApp(t)

	for _, month := range []string{"2026-13", "2026-00", "2026-3", "2026", "marzo", "2026-03-15"} {
		t.Run(month, func(t *testing.T) {
			if res := a.get(t, "/api/reports/month/"+month, nil); res.StatusCode != http.StatusBadRequest {
				t.Errorf("GET /api/reports/month/%s = %d, want 400", month, res.StatusCode)
			}
		})
	}
}

// One Category's share of a month's spend, as the breakdown hands it out. The
// name comes with the id because the screen showing this has no other reason
// to hold the whole Category list.
type categoryLineJSON struct {
	CategoryID  int64  `json:"category_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

// The breakdown rides on the same response as the totals, and is decoded on
// its own because a slice makes a struct incomparable — and the three numbers
// above are worth comparing whole.
func (a *testApp) breakdown(t *testing.T, month string) []categoryLineJSON {
	t.Helper()
	var got struct {
		ByCategory []categoryLineJSON `json:"by_category"`
	}
	if res := a.get(t, "/api/reports/month/"+month, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/month/%s = %d, want 200", month, res.StatusCode)
	}
	return got.ByCategory
}

// The first half of the ticket: the month stops being three numbers and says
// where the money went, biggest share first — which is the order the question
// is asked in.
func TestTheMonthBreaksItsSpendDownByCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 4000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-09", "amount_cents": 1500, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-20", "amount_cents": 90000, "category_id": casa.ID,
	})
	// Money in has no place in a breakdown of where money went.
	a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-10",
	})

	got := a.breakdown(t, "2026-03")
	want := []categoryLineJSON{
		{CategoryID: casa.ID, Category: "Casa", AmountCents: 90000},
		{CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 5500},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("by_category = %+v, want %+v", got, want)
	}
}

// The case the ticket names: a €62 grocery shop with a €14 book in it is €14
// under Svago and €48 — the remainder, never "uncategorised" — under
// Alimentari. ADR-0002 in one assertion.
func TestAPartiallyItemisedExpenseSplitsAcrossTwoCategories(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	svago := a.category(t, "Svago")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-11", "amount_cents": 6200, "category_id": alimentari.ID,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago.ID},
		},
	})

	got := a.breakdown(t, "2026-03")
	want := []categoryLineJSON{
		{CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 4800},
		{CategoryID: svago.ID, Category: "Svago", AmountCents: 1400},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("by_category = %+v, want %+v", got, want)
	}

	// And the itemisation did not shrink the shop: the Expense's own amount is
	// still the authoritative one.
	if m := a.month(t, "2026-03"); m.ExpenseCents != 6200 {
		t.Errorf("expense_cents = %d, want 6200", m.ExpenseCents)
	}
}

// An Expense its Items account for entirely leaves no remainder, and a
// remainder of nothing is not a line: a zero under Casa would read as money
// spent there.
func TestAFullyItemisedExpenseLeavesNoLineUnderItsOwnCategory(t *testing.T) {
	a := newTestApp(t)
	casa := a.category(t, "Casa")
	svago := a.category(t, "Svago")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-11", "amount_cents": 2000, "category_id": casa.ID,
		"items": []map[string]any{
			{"name": "Cinema", "amount_cents": 2000, "category_id": svago.ID},
		},
	})

	got := a.breakdown(t, "2026-03")
	want := []categoryLineJSON{{CategoryID: svago.ID, Category: "Svago", AmountCents: 2000}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("by_category = %+v, want %+v", got, want)
	}
}

// The invariant the breakdown lives or dies by, over the mix that makes it
// hard: an unitemised Expense, a partly itemised one, a fully itemised one,
// and two Items under the same Category as each other.
func TestTheBreakdownAddsUpToTheMonthsSpendWhateverTheMix(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	svago := a.category(t, "Svago")
	casa := a.category(t, "Casa")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-01", "amount_cents": 3300, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-05", "amount_cents": 6200, "category_id": alimentari.ID,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago.ID},
			{"name": "Rivista", "amount_cents": 600, "category_id": svago.ID},
		},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-28", "amount_cents": 80000, "category_id": casa.ID,
		"items": []map[string]any{
			{"name": "Affitto", "amount_cents": 80000, "category_id": casa.ID},
		},
	})
	// A neighbouring month must not leak into the breakdown either.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-04-01", "amount_cents": 5000, "category_id": casa.ID,
	})

	var sum int64
	for _, line := range a.breakdown(t, "2026-03") {
		sum += line.AmountCents
	}
	if want := a.month(t, "2026-03").ExpenseCents; sum != want {
		t.Errorf("the breakdown adds up to %d, want %d — the month's own total", sum, want)
	}
	if sum != 89500 {
		t.Errorf("the breakdown adds up to %d, want 89500", sum)
	}
}

// A month nobody spent anything in has an empty breakdown, and it has to
// marshal as [] rather than null: the screen maps over it.
func TestAMonthWithNothingInItHasAnEmptyBreakdown(t *testing.T) {
	a := newTestApp(t)

	if got := a.breakdown(t, "2019-07"); len(got) != 0 {
		t.Errorf("by_category = %+v, want empty", got)
	}
	var raw struct {
		ByCategory *[]categoryLineJSON `json:"by_category"`
	}
	a.get(t, "/api/reports/month/2019-07", &raw)
	if raw.ByCategory == nil {
		t.Error("by_category came back as null, want []")
	}
}

// One entry as the home screen's list reads it: which way the money moved,
// when, how much, and under what name.
type recentJSON struct {
	Direction   string `json:"direction"`
	ID          int64  `json:"id"`
	Date        string `json:"date"`
	AmountCents int64  `json:"amount_cents"`
	Category    string `json:"category"`
}

func (a *testApp) recent(t *testing.T) []recentJSON {
	t.Helper()
	var got []recentJSON
	if res := a.get(t, "/api/reports/recent", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/recent = %d, want 200", res.StatusCode)
	}
	return got
}

// The second half of the ticket: what was just typed is the first thing on the
// home screen, whichever direction the money went.
func TestTheMostRecentEntriesAreListedNewestTypedFirst(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)

	spesa := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 4237, "category_id": alimentari.ID,
	})
	a.setNow(t, testClock.Add(time.Hour))
	fattura := a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": freelance.ID, "payment_date": "2026-03-10",
	})

	got := a.recent(t)
	want := []recentJSON{
		{Direction: "income", ID: fattura.ID, Date: "2026-03-10",
			AmountCents: 150000, Category: seedFreelanceName},
		{Direction: "expense", ID: spesa.ID, Date: "2026-03-02",
			AmountCents: 4237, Category: "Alimentari"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recent = %+v, want %+v", got, want)
	}
}

// "Recent" is when it was typed, not when it happened — the point of the list
// is confirming what you just entered, and a receipt found in a coat pocket is
// exactly the entry you most want to see land.
func TestAnEntryTypedNowForAnOldDateIsStillAtTheTop(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-14", "amount_cents": 1000, "category_id": alimentari.ID,
	})
	a.setNow(t, testClock.Add(time.Hour))
	backdated := a.addExpense(t, map[string]any{
		"occurred_on": "2025-11-02", "amount_cents": 2500, "category_id": alimentari.ID,
	})

	got := a.recent(t)
	if len(got) == 0 || got[0].ID != backdated.ID {
		t.Errorf("recent = %+v, want the backdated entry first (id %d)", got, backdated.ID)
	}
}

// An unpaid Income is listed — the list is a confirmation of what was typed,
// not a total — but it carries no date, because money that has not arrived has
// no day it arrived on. That empty date is the unpaid state, and it is what
// keeps an €800 invoice from reading as €800 received directly under a money-in
// total that correctly excludes it. ADR-0003, on the home screen.
//
// The day the invoice went out is deliberately not shown here: "how long has
// this been sitting" is ticket 14's question.
func TestAnUnpaidIncomeIsListedWithNoDate(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)

	invoice := a.addIncome(t, map[string]any{
		"amount_cents": 80000, "category_id": freelance.ID, "invoice_sent_date": "2026-03-01",
	})

	got := a.recent(t)
	if len(got) != 1 {
		t.Fatalf("recent = %+v, want 1 entry", got)
	}
	if got[0].Date != "" {
		t.Errorf("the unpaid Income reads %q, want no date at all", got[0].Date)
	}

	// And it gains one the moment the money lands, which is the same moment it
	// starts counting toward the month.
	a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": "2026-03-20"}, nil)
	if got := a.recent(t); got[0].Date != "2026-03-20" {
		t.Errorf("once paid the Income reads %q, want 2026-03-20", got[0].Date)
	}
}

// The list is the last few, not the household's whole history: it sits on a
// phone screen under three totals.
func TestTheRecentListIsCappedAtAFewEntries(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	for i := 0; i < recentLimit+3; i++ {
		a.addExpense(t, map[string]any{
			"occurred_on": "2026-03-02", "amount_cents": 100 + i, "category_id": alimentari.ID,
		})
	}

	if got := a.recent(t); len(got) != recentLimit {
		t.Errorf("recent has %d entries, want %d", len(got), recentLimit)
	}
}

// A household that has typed nothing yet gets [], not null: the screen maps
// over it.
func TestAHouseholdThatHasTypedNothingHasAnEmptyRecentList(t *testing.T) {
	a := newTestApp(t)

	var got *[]recentJSON
	a.get(t, "/api/reports/recent", &got)
	if got == nil {
		t.Fatal("recent came back as null, want []")
	}
	if len(*got) != 0 {
		t.Errorf("recent = %+v, want empty", *got)
	}
}
