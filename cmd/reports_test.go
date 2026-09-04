package main

import (
	"net/http"
	"reflect"
	"strconv"
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
	Payer       string `json:"payer"`
	Client      string `json:"client"`
	IsGift      bool   `json:"is_gift"`
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
			AmountCents: 4237, Category: "Alimentari", Payer: "Nicco"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recent = %+v, want %+v", got, want)
	}
}

// The Dashboard's two lists show who the money moved with: Payer on an
// Expense, Client on an Income. An Income that names nobody, and every
// Expense, carry an empty Client — the same "names nobody" spelling
// outstanding already uses.
func TestARecentEntryNamesWhoTheMoneyMovedWith(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)
	studio := a.createClient(t, "Studio Rossi")

	spesa := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 1200,
		"category_id": alimentari.ID, "payer": "Nicco",
	})
	a.setNow(t, testClock.Add(time.Hour))
	fattura := a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": freelance.ID,
		"payment_date": "2026-03-10", "client_id": studio.ID,
	})

	got := a.recent(t)
	want := []recentJSON{
		{Direction: "income", ID: fattura.ID, Date: "2026-03-10",
			AmountCents: 150000, Category: seedFreelanceName, Client: "Studio Rossi"},
		{Direction: "expense", ID: spesa.ID, Date: "2026-03-02",
			AmountCents: 1200, Category: "Alimentari", Payer: "Nicco"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recent = %+v, want %+v", got, want)
	}
}

// The spoiler blur (spec's Gift section) reads is_gift off this list rather
// than matching Category by name — a household-editable name is not safe to
// resolve identity by. It is true for an Expense under Gift, and false for
// everything else, including an Income under Gift: Regali now applies to both
// directions, but the spoiler is an Expense-only behaviour.
func TestARecentEntryFlagsAGiftExpenseAndOnlyAGiftExpense(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	gift := a.category(t, seedRegaliName)

	plain := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 1000, "category_id": alimentari.ID,
	})
	a.setNow(t, testClock.Add(time.Hour))
	giftExpense := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-03", "amount_cents": 5000, "category_id": gift.ID,
	})
	a.setNow(t, testClock.Add(2*time.Hour))
	giftIncome := a.addIncome(t, map[string]any{
		"amount_cents": 2000, "category_id": gift.ID, "payment_date": "2026-03-04",
	})

	// Keyed by (direction, id) rather than id alone: Expense and Income ids
	// come from separate autoincrementing tables and can collide.
	byKey := map[string]recentJSON{}
	for _, e := range a.recent(t) {
		byKey[e.Direction+strconv.FormatInt(e.ID, 10)] = e
	}
	if got := byKey["expense"+strconv.FormatInt(plain.ID, 10)]; got.IsGift {
		t.Errorf("plain Expense is_gift = true, want false: %+v", got)
	}
	if got := byKey["expense"+strconv.FormatInt(giftExpense.ID, 10)]; !got.IsGift {
		t.Errorf("Gift Expense is_gift = false, want true: %+v", got)
	}
	if got := byKey["income"+strconv.FormatInt(giftIncome.ID, 10)]; got.IsGift {
		t.Errorf("Gift Income is_gift = true, want false — the spoiler is Expense-only: %+v", got)
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

// The year report as the API hands it out: the year's three numbers, and the
// twelve months under them so a screen can draw the shape of the year.
type yearJSON struct {
	Year             string      `json:"year"`
	IncomeCents      int64       `json:"income_cents"`
	ExpenseCents     int64       `json:"expense_cents"`
	NetCents         int64       `json:"net_cents"`
	ExtraIncomeCents int64       `json:"extra_income_cents"`
	Months           []monthJSON `json:"months"`
}

func (a *testApp) year(t *testing.T, year string) yearJSON {
	t.Helper()
	var got yearJSON
	if res := a.get(t, "/api/reports/year/"+year, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/year/%s = %d, want 200", year, res.StatusCode)
	}
	return got
}

// The whole of the year view: twelve months, each standing where it stands,
// and the year's own totals over them.
func TestAYearReportsItsTwelveMonthsAndTheirTotals(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 4000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 1500, "category_id": alimentari.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": a.freelance(t).ID, "payment_date": "2026-01-20",
	})

	got := a.year(t, "2026")
	if got.Year != "2026" {
		t.Errorf("year = %q, want %q", got.Year, "2026")
	}
	if len(got.Months) != 12 {
		t.Fatalf("months = %d, want 12 — a year has twelve of them whether or not anything happened",
			len(got.Months))
	}
	if got.Months[0].Month != "2026-01" || got.Months[11].Month != "2026-12" {
		t.Errorf("months run %s..%s, want 2026-01..2026-12", got.Months[0].Month, got.Months[11].Month)
	}
	if got.IncomeCents != 150000 || got.ExpenseCents != 5500 || got.NetCents != 144500 {
		t.Errorf("year totals = %+v, want 150000 in, 5500 out, 144500 net", got)
	}
	if got.Months[0].ExpenseCents != 4000 || got.Months[2].ExpenseCents != 1500 {
		t.Errorf("january = %+v, march = %+v, want 4000 and 1500 spent",
			got.Months[0], got.Months[2])
	}
	if got.Months[6].ExpenseCents != 0 || got.Months[6].IncomeCents != 0 {
		t.Errorf("july = %+v, want zeros", got.Months[6])
	}
}

// Extra is the non-work slice of the year's received Income: gifts and
// reimbursements count, freelance and stipendio do not, and unpaid money is
// not extra any more than it is income (ADR-0003).
func TestAYearCountsOnlyNonWorkIncomeAsExtra(t *testing.T) {
	a := newTestApp(t)

	a.addIncome(t, map[string]any{
		"amount_cents": 300000, "category_id": a.freelance(t).ID, "payment_date": "2026-04-01",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 200000, "category_id": a.category(t, seedStipendioName).ID,
		"payment_date": "2026-04-02",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 5000, "category_id": a.category(t, "Regali").ID, "payment_date": "2026-04-03",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 3000, "category_id": a.category(t, "Rimborsi").ID, "payment_date": "2026-04-04",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 9999, "category_id": a.category(t, "Regali").ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 4000, "category_id": a.category(t, "Regali").ID, "payment_date": "2025-12-31",
	})

	got := a.year(t, "2026")
	if got.IncomeCents != 508000 {
		t.Errorf("income = %d, want 508000", got.IncomeCents)
	}
	if got.ExtraIncomeCents != 8000 {
		t.Errorf("extra = %d, want 8000 — work was counted as extra, or a gift was missed",
			got.ExtraIncomeCents)
	}
}

// A year counts its own months and nothing else — the neighbours are one tap
// away and must not be in this total.
func TestAYearCountsOnlyItsOwnMonths(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	for _, on := range []string{"2025-12-31", "2026-01-01", "2026-12-31", "2027-01-01"} {
		a.addExpense(t, map[string]any{"occurred_on": on, "amount_cents": 1000, "category_id": id})
	}

	if got := a.year(t, "2026"); got.ExpenseCents != 2000 {
		t.Errorf("2026 spend = %d, want 2000 — the neighbouring years were counted", got.ExpenseCents)
	}
}

// A year that is not a year is refused rather than summed, for the reason a
// month is: substr against "26" matches nothing, and zeros would be
// indistinguishable from a year in which nothing happened.
func TestAYearThatIsNotAYearIsRefused(t *testing.T) {
	a := newTestApp(t)

	for _, year := range []string{"26", "2026-03", "twenty", "20264", ""} {
		res := a.get(t, "/api/reports/year/"+year, nil)
		if res.StatusCode == http.StatusOK {
			t.Errorf("GET /api/reports/year/%q = 200, want a refusal", year)
		}
	}
}

// A year view covers twelve months, so it generates twelve months' worth of
// rent: ADR-0005's "a month nobody opened at the time is not silently empty
// forever" is exactly what a year view is for.
func TestAYearGeneratesTheRecurringExpensesItsMonthsOwe(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"amount_cents": 80000, "start_month": "2026-01", "day_of_month": 5,
	}))

	// The clock is in March 2026, so the year owes three months and not
	// twelve: an Expense is money that left, and April's rent has not.
	got := a.year(t, "2026")
	if got.ExpenseCents != 240000 {
		t.Errorf("year spend = %d, want 240000 — three months of rent generated by looking",
			got.ExpenseCents)
	}
	for i, m := range got.Months {
		want := int64(0)
		if i < 3 {
			want = 80000
		}
		if m.ExpenseCents != want {
			t.Errorf("%s spend = %d, want %d", m.Month, m.ExpenseCents, want)
		}
	}

	// Idempotent, like every other read that materialises.
	if again := a.year(t, "2026"); again.ExpenseCents != 240000 {
		t.Errorf("second read = %d, want 240000 — a second look generated a second rent",
			again.ExpenseCents)
	}
}

// The tax summary as the API hands it out. net_percent is null rather than a
// number when nothing was received: a percentage of nothing is not zero.
type taxJSON struct {
	Year          string   `json:"year"`
	ReceivedCents int64    `json:"received_cents"`
	TaxPaidCents  int64    `json:"tax_paid_cents"`
	NetCents      int64    `json:"net_cents"`
	NetPercent    *float64 `json:"net_percent"`
}

func (a *testApp) tax(t *testing.T, year string) taxJSON {
	t.Helper()
	var got taxJSON
	if res := a.get(t, "/api/reports/tax/"+year, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/tax/%s = %d, want 200", year, res.StatusCode)
	}
	return got
}

// The whole ticket in one test: received X, paid Y in tax, kept Z%.
func TestTheTaxSummaryReadsReceivedPaidAndTheNetShare(t *testing.T) {
	a := newTestApp(t)

	a.addIncome(t, map[string]any{
		"amount_cents": 4000000, "category_id": a.freelance(t).ID, "payment_date": "2026-05-10",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-11-30", "amount_cents": 1000000, "category_id": a.taxes(t).ID,
	})

	got := a.tax(t, "2026")
	if got.Year != "2026" || got.ReceivedCents != 4000000 || got.TaxPaidCents != 1000000 || got.NetCents != 3000000 {
		t.Fatalf("tax summary = %+v, want 4000000 received, 1000000 paid, 3000000 net", got)
	}
	if got.NetPercent == nil || *got.NetPercent != 75 {
		t.Errorf("net_percent = %v, want 75", got.NetPercent)
	}
}

// The ticket's named case, and the reason the field exists: the balance paid
// in June 2027 is tax on 2026's income, and belongs in 2026's summary rather
// than in the year the money left the account.
func TestTaxPaidInOneYearAttributesToThePreviousYearsSummary(t *testing.T) {
	a := newTestApp(t)
	taxes := a.taxes(t)

	a.addIncome(t, map[string]any{
		"amount_cents": 5000000, "category_id": a.freelance(t).ID, "payment_date": "2026-09-30",
	})
	// Paid during 2027, on 2026's income.
	a.addExpense(t, map[string]any{
		"occurred_on": "2027-06-30", "amount_cents": 1500000,
		"category_id": taxes.ID, "tax_year": 2026,
	})
	// And a payment during 2027 that really is 2027's, left to default.
	a.addExpense(t, map[string]any{
		"occurred_on": "2027-11-30", "amount_cents": 400000, "category_id": taxes.ID,
	})

	got := a.tax(t, "2026")
	if got.ReceivedCents != 5000000 || got.TaxPaidCents != 1500000 {
		t.Errorf("2026 = %+v, want 5000000 received and 1500000 paid — attribution is by tax year", got)
	}
	if got.NetCents != 3500000 || got.NetPercent == nil || *got.NetPercent != 70 {
		t.Errorf("2026 net = %d (%v%%), want 3500000 and 70", got.NetCents, got.NetPercent)
	}

	// 2027 keeps only the payment that says it is 2027's, and the cash that
	// left in 2027 is not what either year is measured by.
	if got := a.tax(t, "2027"); got.TaxPaidCents != 400000 {
		t.Errorf("2027 paid = %d, want 400000", got.TaxPaidCents)
	}
}

// X is freelance income and only freelance income: employment income arrives
// already taxed and a gift is not income, so counting either makes the
// percentage meaningless (ADR-0008).
func TestOnlyFreelanceIncomeIsCountedAsReceived(t *testing.T) {
	a := newTestApp(t)

	a.addIncome(t, map[string]any{
		"amount_cents": 3000000, "category_id": a.freelance(t).ID, "payment_date": "2026-04-01",
	})
	for _, name := range []string{seedStipendioName, "Regali", "Rimborsi", "Investimenti"} {
		a.addIncome(t, map[string]any{
			"amount_cents": 100000, "category_id": a.category(t, name).ID,
			"payment_date": "2026-04-02",
		})
	}

	if got := a.tax(t, "2026"); got.ReceivedCents != 3000000 {
		t.Errorf("received = %d, want 3000000 — something other than freelance income was counted",
			got.ReceivedCents)
	}
}

// ADR-0003's standing hazard, asked of this report: an invoice sent is not
// money received, and must be in none of these numbers until it is paid.
func TestUnpaidFreelanceIncomeIsNotReceived(t *testing.T) {
	a := newTestApp(t)
	invoice := a.addIncome(t, map[string]any{
		"amount_cents": 200000, "category_id": a.freelance(t).ID,
		"invoice_sent_date": "2026-12-20",
	})

	if got := a.tax(t, "2026"); got.ReceivedCents != 0 {
		t.Fatalf("received = %d, want 0 — an unpaid invoice was counted as income", got.ReceivedCents)
	}
	a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": "2027-01-15"}, nil)

	// And once paid, it lands in the year the money arrived: received is a
	// cash figure, unlike the tax against it.
	if got := a.tax(t, "2026"); got.ReceivedCents != 0 {
		t.Errorf("2026 received = %d, want 0 — the money arrived in 2027", got.ReceivedCents)
	}
	if got := a.tax(t, "2027"); got.ReceivedCents != 200000 {
		t.Errorf("2027 received = %d, want 200000", got.ReceivedCents)
	}
}

// Only Expenses in the tax Category count as tax paid, and a hidden one still
// does: hiding takes a Category out of the picker, not out of the report.
func TestOnlyTaxCategoryExpensesCountAsTaxPaid(t *testing.T) {
	a := newTestApp(t)
	taxes := a.taxes(t)

	a.addIncome(t, map[string]any{
		"amount_cents": 1000000, "category_id": a.freelance(t).ID, "payment_date": "2026-02-01",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-06-30", "amount_cents": 300000, "category_id": taxes.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-06-30", "amount_cents": 50000,
		"category_id": a.category(t, "Casa").ID,
	})

	if res := a.patch(t, categoryPath(taxes.ID), map[string]any{"hidden": true}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("hiding the tax category = %d, want 200", res.StatusCode)
	}
	if got := a.tax(t, "2026"); got.TaxPaidCents != 300000 {
		t.Errorf("tax paid = %d, want 300000 — hiding the Category is not deleting it", got.TaxPaidCents)
	}
}

// A percentage of nothing received is not zero percent, so there is no number
// to report: null, and the screen says nothing rather than "net 0%".
func TestAYearWithNothingReceivedHasNoPercentage(t *testing.T) {
	a := newTestApp(t)

	got := a.tax(t, "2026")
	if got.ReceivedCents != 0 || got.TaxPaidCents != 0 || got.NetCents != 0 {
		t.Errorf("empty year = %+v, want zeros", got)
	}
	if got.NetPercent != nil {
		t.Errorf("net_percent = %v, want null", *got.NetPercent)
	}

	// Tax paid against no income is a real state — a balance settled after
	// quitting freelancing — and still has no percentage.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-06-30", "amount_cents": 100000, "category_id": a.taxes(t).ID,
	})
	if got := a.tax(t, "2026"); got.NetCents != -100000 || got.NetPercent != nil {
		t.Errorf("tax with no income = %d (%v%%), want -100000 and null", got.NetCents, got.NetPercent)
	}
}

func TestATaxSummaryForAYearThatIsNotAYearIsRefused(t *testing.T) {
	a := newTestApp(t)

	for _, year := range []string{"26", "2026-03", "twenty"} {
		if res := a.get(t, "/api/reports/tax/"+year, nil); res.StatusCode == http.StatusOK {
			t.Errorf("GET /api/reports/tax/%q = 200, want a refusal", year)
		}
	}
}

// A tax payment can be a Recurring expense like any other — an instalment
// paid every month — and the summary must not lose it. Generation copies
// template fields and knows nothing about a Tax year, so the row it writes has
// none: what makes it count is the summary applying the same default to a
// NULL column that the form applies to a typed Expense.
// One Category's share of one day's spend, as the daily reports hand it out.
type dailyLineJSON struct {
	Day         string `json:"day"`
	CategoryID  int64  `json:"category_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

func (a *testApp) daily(t *testing.T) []dailyLineJSON {
	t.Helper()
	var got []dailyLineJSON
	if res := a.get(t, "/api/reports/daily", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/daily = %d, want 200", res.StatusCode)
	}
	return got
}

func (a *testApp) monthDaily(t *testing.T, month string) []dailyLineJSON {
	t.Helper()
	var got []dailyLineJSON
	if res := a.get(t, "/api/reports/month/"+month+"/daily", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/month/%s/daily = %d, want 200", month, res.StatusCode)
	}
	return got
}

// The rolling chart's whole ticket: two days, two Categories, each its own
// line — and money in has no place in it, the same as the month breakdown.
func TestTheDailyReportBreaksTheRollingWindowDownByDayAndCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")

	// testClock is 2026-03-15, so both of these fall inside the trailing
	// 30-day window the rolling report answers with.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-14", "amount_cents": 2000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 5000, "category_id": casa.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-15",
	})

	got := a.daily(t)
	want := []dailyLineJSON{
		{Day: "2026-03-14", CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 2000},
		{Day: "2026-03-15", CategoryID: casa.ID, Category: "Casa", AmountCents: 5000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("daily = %+v, want %+v", got, want)
	}
}

// The window is trailing and fixed: an Expense outside the last 30 days,
// counted from the clock, does not appear.
func TestTheDailyReportDropsAnythingOutsideTheRollingWindow(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	// testClock is 2026-03-15, so 30 days back is 2026-02-14 — one day
	// earlier falls outside the window and must not appear.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-13", "amount_cents": 1000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-14", "amount_cents": 2000, "category_id": alimentari.ID,
	})

	got := a.daily(t)
	want := []dailyLineJSON{
		{Day: "2026-02-14", CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 2000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("daily = %+v, want %+v — the window did not stop 30 days back", got, want)
	}
}

// The case ADR-0002 exists for, asked of the daily report: a grocery run with
// a book in it puts the book under Svago on the day it was bought, and the
// remainder — never "uncategorised" — under Alimentari the same day.
func TestTheDailyReportSplitsAPartiallyItemisedExpenseAcrossCategories(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	svago := a.category(t, "Svago")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-11", "amount_cents": 6200, "category_id": alimentari.ID,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago.ID},
		},
	})

	got := a.daily(t)
	want := []dailyLineJSON{
		{Day: "2026-03-11", CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 4800},
		{Day: "2026-03-11", CategoryID: svago.ID, Category: "Svago", AmountCents: 1400},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("daily = %+v, want %+v", got, want)
	}
}

// Investments are not excluded here — ADR-0009: a stock buy reads exactly
// like any other Category's day, unlike the Budget/Estimate reports which do
// exclude it.
func TestTheDailyReportDoesNotExcludeInvestments(t *testing.T) {
	a := newTestApp(t)
	investments := a.category(t, "Investimenti")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-10", "amount_cents": 30000, "category_id": investments.ID,
	})

	got := a.daily(t)
	want := []dailyLineJSON{
		{Day: "2026-03-10", CategoryID: investments.ID, Category: "Investimenti", AmountCents: 30000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("daily = %+v, want %+v — an Investments Expense was excluded", got, want)
	}
}

// The month-scoped daily report's whole ticket: every day of the named month,
// by Category, and nothing from a neighbouring month.
func TestTheMonthDailyReportBreaksOneMonthDownByDayAndCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 4000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-31", "amount_cents": 90000, "category_id": casa.ID,
	})
	// A neighbouring month must not leak in, on either edge.
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-28", "amount_cents": 1000, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-04-01", "amount_cents": 1000, "category_id": alimentari.ID,
	})

	got := a.monthDaily(t, "2026-03")
	want := []dailyLineJSON{
		{Day: "2026-03-02", CategoryID: alimentari.ID, Category: "Alimentari", AmountCents: 4000},
		{Day: "2026-03-31", CategoryID: casa.ID, Category: "Casa", AmountCents: 90000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("month daily = %+v, want %+v", got, want)
	}
}

// A month-scoped daily report generates the month's Recurring expenses first,
// the same as every other month read (ADR-0005) — read independently of
// /api/reports/month/{month}, this is its own path to materialise from.
func TestTheMonthDailyReportGeneratesTheMonthsRecurringExpenses(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"amount_cents": 80000, "start_month": "2026-03", "day_of_month": 5,
	}))

	got := a.monthDaily(t, "2026-03")
	if len(got) != 1 || got[0].AmountCents != 80000 || got[0].Day != "2026-03-05" {
		t.Errorf("month daily = %+v, want the generated rent on 2026-03-05", got)
	}
}

// A month that is not a month is refused, the same as the month report.
func TestAMonthDailyReportForAMonthThatIsNotAMonthIsRefused(t *testing.T) {
	a := newTestApp(t)

	for _, month := range []string{"2026-13", "2026-3", "2026", "marzo"} {
		if res := a.get(t, "/api/reports/month/"+month+"/daily", nil); res.StatusCode != http.StatusBadRequest {
			t.Errorf("GET /api/reports/month/%s/daily = %d, want 400", month, res.StatusCode)
		}
	}
}

// An empty range has an empty list either way, and it has to marshal as []
// rather than null: the screen maps over it.
func TestADailyReportWithNothingInItIsAnEmptyList(t *testing.T) {
	a := newTestApp(t)

	if got := a.daily(t); len(got) != 0 {
		t.Errorf("daily = %+v, want empty", got)
	}
	var raw *[]dailyLineJSON
	a.get(t, "/api/reports/daily", &raw)
	if raw == nil {
		t.Error("daily came back as null, want []")
	}

	if got := a.monthDaily(t, "2019-07"); len(got) != 0 {
		t.Errorf("month daily = %+v, want empty", got)
	}
}

func TestARecurringTaxPaymentIsCountedByTheSummary(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"amount_cents": 30000, "category_id": a.taxes(t).ID,
		"start_month": "2026-01", "day_of_month": 16,
	}))

	// The clock is in March 2026, so three instalments have been paid.
	if got := a.tax(t, "2026"); got.TaxPaidCents != 90000 {
		t.Errorf("tax paid = %d, want 90000 — a generated tax payment was not counted",
			got.TaxPaidCents)
	}

	// And a typed Tax year still overrides it: the instalment paid in January
	// 2027 that settles 2026's balance belongs to 2026. Reading 2027 is what
	// generates it — there is no scheduler (ADR-0005).
	a.setNow(t, time.Date(2027, 2, 1, 10, 0, 0, 0, time.UTC))
	a.tax(t, "2027")

	var january expenseJSON
	for _, e := range a.expenses(t) {
		if e.OccurredOn == "2027-01-16" {
			january = e
		}
	}
	if january.ID == 0 {
		t.Fatalf("no generated Expense on 2027-01-16 in %v", a.expenses(t))
	}
	a.patch(t, expensePath(january.ID), map[string]any{"tax_year": 2026}, nil)

	// Twelve instalments now belong to 2026 — the clock is past it, so reading
	// the year filled in every month of it — plus the January 2027 payment
	// moved back to it.
	if got := a.tax(t, "2026"); got.TaxPaidCents != 390000 {
		t.Errorf("tax paid for 2026 = %d, want 390000 — twelve instalments plus the one moved back",
			got.TaxPaidCents)
	}
	// 2027 keeps the instalment that says nothing, and not the one moved back.
	if got := a.tax(t, "2027"); got.TaxPaidCents != 30000 {
		t.Errorf("tax paid for 2027 = %d, want 30000 — January was moved to 2026", got.TaxPaidCents)
	}
}
