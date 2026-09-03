package main

import (
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// A Recurring expense as the API hands it out: an Expense's template fields,
// the day it lands on, and the window it covers. end_month is empty rather
// than null when it is still running — the window is the whole of the state,
// per ADR-0005, so there is no active flag to read here.
type recurringJSON struct {
	ID            int64  `json:"id"`
	AmountCents   int64  `json:"amount_cents"`
	CategoryID    int64  `json:"category_id"`
	Store         string `json:"store"`
	Payer         string `json:"payer"`
	PaymentMethod string `json:"payment_method"`
	Note          string `json:"note"`
	DayOfMonth    int    `json:"day_of_month"`
	StartMonth    string `json:"start_month"`
	EndMonth      string `json:"end_month"`
}

// recurringPath addresses one the way the API does.
func recurringPath(id int64) string {
	return "/api/recurring/" + strconv.FormatInt(id, 10)
}

func (a *testApp) recurrings(t *testing.T) []recurringJSON {
	t.Helper()
	var got []recurringJSON
	if res := a.get(t, "/api/recurring", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/recurring = %d, want 200", res.StatusCode)
	}
	return got
}

// addRecurring defines one and returns it as the API answered, failing the
// test if it was refused. Ticket 13 generates from these, and this is the one
// place that knows how to define one.
func (a *testApp) addRecurring(t *testing.T, body map[string]any) recurringJSON {
	t.Helper()
	var created recurringJSON
	res := a.post(t, "/api/recurring", body, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/recurring %v = %d, want 201", body, res.StatusCode)
	}
	return created
}

// rent is the shortest valid definition: the household's fixed monthly cost,
// under a Category an Expense can go in.
func (a *testApp) rent(t *testing.T, over map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"amount_cents": 85000,
		"category_id":  a.category(t, "Casa").ID,
	}
	for k, v := range over {
		body[k] = v
	}
	return body
}

// The whole ticket in one test: the rent is defined once, with every field an
// Expense carries plus the day it lands on, and what one request saved is what
// a later request sees.
func TestADefinedRecurringExpenseIsThereForTheNextRequest(t *testing.T) {
	a := newTestApp(t)
	casa := a.category(t, "Casa")

	created := a.addRecurring(t, map[string]any{
		"amount_cents":   85000,
		"category_id":    casa.ID,
		"store":          "Immobiliare Rossi",
		"payer":          "Entrambi",
		"payment_method": "Bonifico",
		"note":           "Affitto",
		"day_of_month":   5,
	})
	if created.ID == 0 {
		t.Error("the created Recurring expense came back without an id")
	}

	got := a.recurrings(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Recurring expenses, want 1", len(got))
	}
	if !reflect.DeepEqual(got[0], created) {
		t.Errorf("listed = %+v, want the one that was created, %+v", got[0], created)
	}
	// Every template field survives, because every one of them is copied onto
	// the Expenses this will produce.
	want := recurringJSON{
		ID: created.ID, AmountCents: 85000, CategoryID: casa.ID,
		Store: "Immobiliare Rossi", Payer: "Entrambi", PaymentMethod: "Bonifico",
		Note: "Affitto", DayOfMonth: 5, StartMonth: "2026-03", EndMonth: "",
	}
	if got[0] != want {
		t.Errorf("listed = %+v, want %+v", got[0], want)
	}
}

// Switching the rent on in March must not invent January and February, so a
// definition that says nothing about when it starts starts now. Nothing may
// read the wall clock for this — the Pi has no RTC.
func TestCreatingDefaultsTheStartToTheCurrentMonth(t *testing.T) {
	a := newTestApp(t)

	created := a.addRecurring(t, a.rent(t, nil))
	if created.StartMonth != "2026-03" {
		t.Errorf("start_month = %q, want 2026-03 — the injected clock's month", created.StartMonth)
	}

	// And it follows the clock rather than a constant.
	a.setNow(t, time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC))
	later := a.addRecurring(t, a.rent(t, nil))
	if later.StartMonth != "2026-11" {
		t.Errorf("start_month = %q, want 2026-11", later.StartMonth)
	}
}

// A cost that has been running since January, entered in March. Setting the
// start back is a deliberate answer, not an accident: ADR-0005 makes the
// window what a report over an old month regenerates from, so a start in the
// past is how those months get their Expenses.
func TestTheStartMonthCanBeSetBackDeliberately(t *testing.T) {
	a := newTestApp(t)

	created := a.addRecurring(t, a.rent(t, map[string]any{"start_month": "2026-01"}))
	if created.StartMonth != "2026-01" {
		t.Errorf("start_month = %q, want 2026-01", created.StartMonth)
	}
	if got := a.recurrings(t); len(got) != 1 || got[0].StartMonth != "2026-01" {
		t.Errorf("listed = %+v, want the start left in January", got)
	}
}

// An empty end month is ongoing, and it has to survive the round trip as one:
// everything downstream asks "is this month inside the window", and a missing
// end is the answer "yes, still".
func TestAnEndMonthIsOptionalAndOngoingSurvivesTheRoundTrip(t *testing.T) {
	a := newTestApp(t)

	created := a.addRecurring(t, a.rent(t, nil))
	if created.EndMonth != "" {
		t.Errorf("end_month = %q, want empty — the Recurring expense is ongoing", created.EndMonth)
	}
	if got := a.recurrings(t); len(got) != 1 || got[0].EndMonth != "" {
		t.Errorf("listed = %+v, want one ongoing", got)
	}
}

// Deactivating is setting the end month to the current one, and nothing else:
// there is no flag, no endpoint of its own, and no row removed. ADR-0005 — the
// window is the record, so the months it did cover stay reproducible.
func TestDeactivatingIsSettingTheEndMonth(t *testing.T) {
	a := newTestApp(t)
	rec := a.addRecurring(t, a.rent(t, map[string]any{"start_month": "2025-06"}))

	var ended recurringJSON
	res := a.patch(t, recurringPath(rec.ID), map[string]any{"end_month": "2026-03"}, &ended)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", recurringPath(rec.ID), res.StatusCode)
	}
	if ended.EndMonth != "2026-03" {
		t.Errorf("end_month = %q, want 2026-03", ended.EndMonth)
	}
	// The window it did cover is untouched: ending it does not rewrite when
	// it started, or a report over 2025 would stop generating those months.
	if ended.StartMonth != "2025-06" {
		t.Errorf("start_month = %q, want it left at 2025-06", ended.StartMonth)
	}

	got := a.recurrings(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Recurring expenses, want the ended one still listed", len(got))
	}
	if got[0].EndMonth != "2026-03" {
		t.Errorf("listed end_month = %q, want 2026-03", got[0].EndMonth)
	}
}

// A window that ends before it starts covers no month at all, which is not a
// way to say "never ran" — it is a typo, and one that would silently produce
// nothing forever.
func TestAnEndMonthBeforeTheStartIsRefused(t *testing.T) {
	a := newTestApp(t)

	body := a.rent(t, map[string]any{"start_month": "2026-03", "end_month": "2026-01"})
	if res := a.post(t, "/api/recurring", body, nil); res.StatusCode != http.StatusBadRequest {
		t.Errorf("POST with an end before the start = %d, want 400", res.StatusCode)
	}

	// And an edit cannot smuggle past what a create refuses.
	rec := a.addRecurring(t, a.rent(t, map[string]any{"start_month": "2026-03"}))
	if res := a.patch(t, recurringPath(rec.ID), map[string]any{"end_month": "2026-02"}, nil); res.StatusCode != http.StatusBadRequest {
		t.Errorf("PATCH with an end before the start = %d, want 400", res.StatusCode)
	}
	if got := a.recurrings(t); got[0].EndMonth != "" {
		t.Errorf("end_month = %q after a refused edit, want it left empty", got[0].EndMonth)
	}
}

// ADR-0005 answers reactivation with a new definition, not a reopened window:
// "the gap months genuinely had no payment". Clearing an end month that is set
// would tell generation to fill every month of the gap, so it is refused —
// while moving that end month, which invents nothing, is not.
func TestAnEndedRecurringExpenseCannotBeReopenedByClearingItsEnd(t *testing.T) {
	a := newTestApp(t)
	rec := a.addRecurring(t, a.rent(t, map[string]any{"start_month": "2025-01"}))
	a.patch(t, recurringPath(rec.ID), map[string]any{"end_month": "2025-06"}, nil)

	if res := a.patch(t, recurringPath(rec.ID), map[string]any{"end_month": ""}, nil); res.StatusCode != http.StatusBadRequest {
		t.Errorf("PATCH clearing the end month = %d, want 400", res.StatusCode)
	}
	if got := a.recurrings(t); got[0].EndMonth != "2025-06" {
		t.Errorf("end_month = %q after the refusal, want it left at 2025-06", got[0].EndMonth)
	}

	// An end month typed wrongly is still a typo, and moving it invents no
	// month that was skipped — so that is allowed.
	var moved recurringJSON
	res := a.patch(t, recurringPath(rec.ID), map[string]any{"end_month": "2025-09"}, &moved)
	if res.StatusCode != http.StatusOK || moved.EndMonth != "2025-09" {
		t.Errorf("moving the end month = %d, end_month %q, want 200 and 2025-09", res.StatusCode, moved.EndMonth)
	}

	// And an edit that says nothing about the end month keeps it, rather than
	// reading the omission as a clear.
	var edited recurringJSON
	a.patch(t, recurringPath(rec.ID), map[string]any{"amount_cents": 90000}, &edited)
	if edited.EndMonth != "2025-09" {
		t.Errorf("end_month = %q after an unrelated edit, want it left at 2025-09", edited.EndMonth)
	}
}

// The rent goes up. History keeps what was true at the time by ending the old
// definition and starting a new one, rather than by editing an amount that
// past months were generated from — ADR-0005's window is what makes the two
// stretches of time tell different stories.
func TestAnAmountChangeIsAnEndingAndANewDefinition(t *testing.T) {
	a := newTestApp(t)
	old := a.addRecurring(t, a.rent(t, map[string]any{
		"start_month": "2025-01", "amount_cents": 85000,
	}))

	a.patch(t, recurringPath(old.ID), map[string]any{"end_month": "2026-02"}, nil)
	a.addRecurring(t, a.rent(t, map[string]any{
		"start_month": "2026-03", "amount_cents": 90000,
	}))

	got := a.recurrings(t)
	if len(got) != 2 {
		t.Fatalf("listed %d Recurring expenses, want both the old and the new", len(got))
	}
	if got[0].AmountCents != 85000 || got[0].EndMonth != "2026-02" {
		t.Errorf("the old one = %+v, want 85000 ending 2026-02", got[0])
	}
	if got[1].AmountCents != 90000 || got[1].StartMonth != "2026-03" || got[1].EndMonth != "" {
		t.Errorf("the new one = %+v, want 90000 running from 2026-03", got[1])
	}
}

// The day of the month is what generation places the Expense on, so it has to
// be a real day. 29, 30 and 31 are all accepted and stored as meant — the
// short months are clamped where the date is built, not by refusing a rent
// that genuinely leaves on the 31st.
func TestTheDayOfTheMonthIsCheckedAndDefaultsToTheDayItWasSetUp(t *testing.T) {
	a := newTestApp(t)

	// The injected clock is the 15th.
	if created := a.addRecurring(t, a.rent(t, nil)); created.DayOfMonth != 15 {
		t.Errorf("day_of_month = %d, want 15 — the day it was set up", created.DayOfMonth)
	}
	if created := a.addRecurring(t, a.rent(t, map[string]any{"day_of_month": 31})); created.DayOfMonth != 31 {
		t.Errorf("day_of_month = %d, want 31 stored as meant", created.DayOfMonth)
	}
	// 0 is not in the list: an omitted day_of_month and an explicit 0 are the
	// same thing to a JSON decode, and 0 is not a day anyone means — so it is
	// the sentinel the default fills in, not a value to refuse.
	for _, day := range []int{-1, 32, 100} {
		body := a.rent(t, map[string]any{"day_of_month": day})
		if res := a.post(t, "/api/recurring", body, nil); res.StatusCode != http.StatusBadRequest {
			t.Errorf("POST with day_of_month %d = %d, want 400", day, res.StatusCode)
		}
	}
}

// Malformed months never reach the database, because every comparison after
// this one is a string comparison: "2026-3" sorts below "2026-01" and would
// put the window in the wrong place without failing anywhere.
func TestAMalformedMonthIsRefused(t *testing.T) {
	a := newTestApp(t)

	for _, month := range []string{"2026-3", "marzo", "2026-13", "2026-03-01"} {
		body := a.rent(t, map[string]any{"start_month": month})
		if res := a.post(t, "/api/recurring", body, nil); res.StatusCode != http.StatusBadRequest {
			t.Errorf("POST with start_month %q = %d, want 400", month, res.StatusCode)
		}
	}
}

// What this defines is Expenses, so it answers to the Expense rule: an
// income-only Category is a real row and still not somewhere a Recurring
// expense can land. The same gate an Expense and its Items pass.
func TestARecurringExpenseCannotBeDefinedInAnIncomeCategory(t *testing.T) {
	a := newTestApp(t)

	body := a.rent(t, map[string]any{"category_id": a.freelance(t).ID})
	if res := a.post(t, "/api/recurring", body, nil); res.StatusCode != http.StatusBadRequest {
		t.Errorf("POST in an income-only Category = %d, want 400", res.StatusCode)
	}
	body = a.rent(t, map[string]any{"category_id": 9999})
	if res := a.post(t, "/api/recurring", body, nil); res.StatusCode != http.StatusBadRequest {
		t.Errorf("POST in a Category that does not exist = %d, want 400", res.StatusCode)
	}
}

// One defined by mistake, before it ever produced anything, is simply removed.
func TestARecurringExpenseCanBeDeleted(t *testing.T) {
	a := newTestApp(t)
	rec := a.addRecurring(t, a.rent(t, nil))

	if res := a.delete(t, recurringPath(rec.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", recurringPath(rec.ID), res.StatusCode)
	}
	if got := a.recurrings(t); len(got) != 0 {
		t.Errorf("listed %+v after the delete, want none", got)
	}
	if res := a.delete(t, recurringPath(rec.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("DELETE of a gone Recurring expense = %d, want 404", res.StatusCode)
	}
}

// An id that names nothing is a 404 on every verb, whether it fails to parse
// or simply is not there: from outside, those are the same thing.
func TestAMissingRecurringExpenseIsANotFound(t *testing.T) {
	a := newTestApp(t)

	for _, path := range []string{"/api/recurring/9999", "/api/recurring/abc"} {
		if res := a.patch(t, path, map[string]any{"amount_cents": 100}, nil); res.StatusCode != http.StatusNotFound {
			t.Errorf("PATCH %s = %d, want 404", path, res.StatusCode)
		}
	}
}

// --- Generation on sight (ticket 13) ---------------------------------------
//
// There is no scheduler to drive in a test, and that is the point: every test
// below reads a month and asserts on what reading it produced. The clock is
// the fixed testClock throughout — 2026-03-15 — so "this month" is 2026-03 and
// every month before it is the past.

// clockOKJSON reads the one field the screens warn on.
func (a *testApp) clockOK(t *testing.T) bool {
	t.Helper()
	var got struct {
		ClockOK bool `json:"clock_ok"`
	}
	a.get(t, "/api/health", &got)
	return got.ClockOK
}

// The whole ticket in one test: nothing generated the rent, and it is there
// because somebody looked at the month. Every template field comes with it,
// because the Expense it produces has to be the one that would have been typed.
func TestReadingAMonthGeneratesTheRecurringExpensesItOwes(t *testing.T) {
	a := newTestApp(t)
	casa := a.category(t, "Casa")
	a.addRecurring(t, map[string]any{
		"amount_cents": 85000, "category_id": casa.ID,
		"store": "Immobiliare Rossi", "payer": "Entrambi",
		"payment_method": "Bonifico", "note": "Affitto",
		"day_of_month": 5, "start_month": "2026-03",
	})

	// Before anything reads the month there is no Expense at all: generation
	// is not something defining a Recurring expense did.
	if got := a.expenses(t); len(got) != 0 {
		t.Fatalf("listed %+v before any month was read, want none — defining one generated an Expense", got)
	}

	if got := a.month(t, "2026-03"); got.ExpenseCents != 85000 {
		t.Errorf("expense_cents = %d, want 85000 — the month read did not generate the rent", got.ExpenseCents)
	}

	got := a.expenses(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Expenses, want the one generated", len(got))
	}
	want := expenseJSON{
		ID: got[0].ID, OccurredOn: "2026-03-05", AmountCents: 85000, CategoryID: casa.ID,
		Store: "Immobiliare Rossi", Payer: "Entrambi", PaymentMethod: "Bonifico",
		Note: "Affitto", Items: []itemJSON{},
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("generated Expense = %+v, want %+v", got[0], want)
	}
}

// Opening a month twice does not produce two rents. The household's own screen
// does this on every navigation, so idempotence is not an edge case — it is the
// normal path.
func TestGeneratingTheSameMonthTwiceProducesOneRent(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{"day_of_month": 5}))

	first := a.month(t, "2026-03")
	second := a.month(t, "2026-03")
	if first != second {
		t.Errorf("reading the month twice gave %+v then %+v, want the same totals", first, second)
	}
	if got := a.expenses(t); len(got) != 1 {
		t.Errorf("listed %d Expenses after two reads, want 1", len(got))
	}

	// And a third read after the same month was read through the breakdown,
	// which is the other thing the month screen loads.
	a.breakdown(t, "2026-03")
	if got := a.expenses(t); len(got) != 1 {
		t.Errorf("listed %d Expenses after the breakdown too, want 1", len(got))
	}
}

// Switching the rent on in February must not invent January, and ending it in
// March must not carry it into April. The window is the whole of the state, so
// it is the whole of what generation asks — ADR-0005.
func TestOnlyMonthsInsideTheWindowAreGenerated(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 5, "start_month": "2026-02", "end_month": "2026-03",
	}))

	for _, tc := range []struct {
		month string
		want  int64
	}{
		{"2026-01", 0},     // before the start
		{"2026-02", 85000}, // the first month it covers
		{"2026-03", 85000}, // the end month is covered, not excluded
	} {
		if got := a.month(t, tc.month); got.ExpenseCents != tc.want {
			t.Errorf("%s expense_cents = %d, want %d", tc.month, got.ExpenseCents, tc.want)
		}
	}
	if got := a.expenses(t); len(got) != 2 {
		t.Errorf("listed %d Expenses, want 2 — one per month inside the window", len(got))
	}
}

// A month nobody opened at the time is not permanently empty: the report is
// what generates it, whenever it is run. This is the reason there is no
// scheduler — the Pi was switched off in June 2025 and the answer still has to
// be right in 2026.
func TestAMonthLongPastStillGeneratesItsRent(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 5, "start_month": "2025-01",
	}))

	if got := a.month(t, "2025-06"); got.ExpenseCents != 85000 {
		t.Errorf("2025-06 expense_cents = %d, want 85000 — a month long past generated nothing", got.ExpenseCents)
	}
	got := a.expenses(t)
	if len(got) != 1 || got[0].OccurredOn != "2025-06-05" {
		t.Errorf("listed %+v, want one Expense on 2025-06-05", got)
	}
	// Reading it again is still one, twenty months after the fact.
	a.month(t, "2025-06")
	if got := a.expenses(t); len(got) != 1 {
		t.Errorf("listed %d Expenses after a second read of the old month, want 1", len(got))
	}
}

// Next month's rent has not been paid, and an Expense is money that left. A
// household looking ahead must not see it in a total — and must see it the
// month it arrives, without anything else changing.
func TestNothingIsGeneratedBeyondTheCurrentMonth(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 5, "start_month": "2026-01",
	}))

	if got := a.month(t, "2026-04"); got.ExpenseCents != 0 {
		t.Errorf("next month's expense_cents = %d, want 0 — the rent has not been paid yet", got.ExpenseCents)
	}
	if got := a.month(t, "2026-03"); got.ExpenseCents != 85000 {
		t.Errorf("this month's expense_cents = %d, want 85000", got.ExpenseCents)
	}

	// April becomes the current month, and only then does it fill in.
	a.setNow(t, time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC))
	if got := a.month(t, "2026-04"); got.ExpenseCents != 85000 {
		t.Errorf("April's expense_cents once April arrived = %d, want 85000", got.ExpenseCents)
	}
}

// A rent that leaves on the 31st leaves on the 30th in April and the 28th in
// February — the day of the month is what was meant, and the month's length is
// what is possible. A date that overflowed into the next month would put the
// rent in a month it was not paid in, and no report would find it.
func TestTheGeneratedDateIsClampedToTheMonthsLength(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 31, "start_month": "2024-01",
	}))
	a.setNow(t, time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC))

	for month, want := range map[string]string{
		"2026-01": "2026-01-31", // a month that really has a 31st
		"2026-04": "2026-04-30",
		"2026-02": "2026-02-28",
		"2024-02": "2024-02-29", // and February when it has 29
	} {
		if got := a.month(t, month); got.ExpenseCents != 85000 {
			t.Fatalf("%s expense_cents = %d, want 85000", month, got.ExpenseCents)
		}
		if !hasExpenseOn(a.expenses(t), want) {
			t.Errorf("no generated Expense on %s after reading %s; got %+v",
				want, month, datesOf(a.expenses(t)))
		}
	}
}

// A generated Expense is indistinguishable in use from a typed one: the same
// PATCH corrects it when the rent went out a day late, and the same DELETE
// removes it. No second set of rules, per the ticket and story 19.
func TestAGeneratedExpenseIsEditedAndDeletedLikeATypedOne(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{"day_of_month": 5}))
	a.month(t, "2026-03")

	generated := a.expenses(t)[0]
	var edited expenseJSON
	res := a.patch(t, expensePath(generated.ID), map[string]any{
		"occurred_on": "2026-03-07", "amount_cents": 86000, "note": "aumento",
	}, &edited)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH of a generated Expense = %d, want 200", res.StatusCode)
	}
	if edited.OccurredOn != "2026-03-07" || edited.AmountCents != 86000 || edited.Note != "aumento" {
		t.Errorf("edited = %+v, want the correction to have stuck", edited)
	}
	// And the edit is not undone by the next look at the month: what is
	// already there for that month is what generation counts.
	if got := a.month(t, "2026-03"); got.ExpenseCents != 86000 {
		t.Errorf("expense_cents = %d after re-reading the month, want the edited 86000", got.ExpenseCents)
	}

	if res := a.delete(t, expensePath(generated.ID)); res.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE of a generated Expense = %d, want 204", res.StatusCode)
	}
}

// The month the rent was not paid stays the month the rent was not paid.
// Generation is otherwise a pure function of the window, so a delete has to
// write the skip down — and only for that month, because the months either
// side were paid.
func TestDeletingAGeneratedExpenseKeepsItDeleted(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 5, "start_month": "2026-01",
	}))
	a.month(t, "2026-02")

	generated := a.expenses(t)[0]
	if res := a.delete(t, expensePath(generated.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", res.StatusCode)
	}

	if got := a.month(t, "2026-02"); got.ExpenseCents != 0 {
		t.Errorf("February's expense_cents = %d after the delete, want 0 — the rent came back", got.ExpenseCents)
	}
	if got := a.expenses(t); len(got) != 0 {
		t.Errorf("listed %+v, want none — a skipped month regenerated", got)
	}
	// The skip is that month's and nobody else's.
	if got := a.month(t, "2026-03"); got.ExpenseCents != 85000 {
		t.Errorf("March's expense_cents = %d, want 85000 — the skip leaked into another month", got.ExpenseCents)
	}
	// And deleting a typed Expense skips nothing, because there is nothing
	// that would ever regenerate it.
	typed := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-20", "amount_cents": 1200, "category_id": a.category(t, "Casa").ID,
	})
	a.delete(t, expensePath(typed.ID))
	if got := a.month(t, "2026-03"); got.ExpenseCents != 85000 {
		t.Errorf("March's expense_cents = %d after a typed Expense was deleted, want 85000", got.ExpenseCents)
	}
}

// The bug the review found: a generated Expense moved to another month left
// the month it came from looking unpaid, and the next look at it produced a
// second rent for a payment that happened once. Generation asks whether that
// (Recurring expense, month) has an Expense — after the move it does not — so
// the move has to write the same skip a delete does.
func TestMovingAGeneratedExpenseDoesNotLeaveTheOldMonthToRegenerate(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 5, "start_month": "2026-01",
	}))
	a.month(t, "2026-02")
	generated := a.expenses(t)[0]

	// The rent went out late enough to land in March.
	var moved expenseJSON
	res := a.patch(t, expensePath(generated.ID), map[string]any{"occurred_on": "2026-03-07"}, &moved)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH moving a generated Expense = %d, want 200", res.StatusCode)
	}

	if got := a.month(t, "2026-02"); got.ExpenseCents != 0 {
		t.Errorf("February's expense_cents = %d after the rent was moved out, want 0 — a second rent was generated", got.ExpenseCents)
	}
	if got := a.month(t, "2026-03"); got.ExpenseCents != 85000 {
		t.Errorf("March's expense_cents = %d, want the 85000 that was moved there", got.ExpenseCents)
	}
	if got := a.expenses(t); len(got) != 1 {
		t.Fatalf("listed %+v, want exactly one rent — one payment happened", datesOf(got))
	}

	// And moving it back is the same story in the other direction: still one
	// rent, and March does not fill the hole it left.
	a.patch(t, expensePath(generated.ID), map[string]any{"occurred_on": "2026-02-05"}, nil)
	a.month(t, "2026-02")
	a.month(t, "2026-03")
	if got := a.expenses(t); len(got) != 1 {
		t.Errorf("listed %+v after moving it back, want one rent", datesOf(got))
	}
}

// Story 72: the Pi has no real-time clock, so a boot without network time
// reads 1970. Generating against that writes a rent into a month no report
// will ever look at, and there is no way back; generating nothing is undone by
// the next read once NTP has answered. So it refuses, and says so where the
// UI can see it.
func TestAClockBeforeTheFloorRefusesToGenerate(t *testing.T) {
	a := newTestApp(t)
	a.addRecurring(t, a.rent(t, map[string]any{
		"day_of_month": 5, "start_month": "2026-01",
	}))

	a.setNow(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
	if a.clockOK(t) {
		t.Error("clock_ok = true at 1970 — the UI has nothing to warn on")
	}
	// The read still answers: what was typed is still the truth, and refusing
	// to generate is not refusing to report.
	if got := a.month(t, "2026-01"); got.ExpenseCents != 0 {
		t.Errorf("expense_cents = %d with the clock unset, want 0 — it generated against a wrong clock", got.ExpenseCents)
	}
	if got := a.expenses(t); len(got) != 0 {
		t.Fatalf("listed %+v with the clock unset, want none", got)
	}

	// Once the clock is right, the next read generates everything the window
	// owed — nothing was lost by refusing.
	a.setNow(t, testClock)
	if !a.clockOK(t) {
		t.Error("clock_ok = false at the test clock, which is after the floor")
	}
	if got := a.month(t, "2026-01"); got.ExpenseCents != 85000 {
		t.Errorf("expense_cents = %d once the clock was right, want 85000", got.ExpenseCents)
	}
}

// hasExpenseOn and datesOf keep the clamping test readable: it asserts that a
// date is among what was generated, not on the order of a growing list.
func hasExpenseOn(expenses []expenseJSON, date string) bool {
	for _, e := range expenses {
		if e.OccurredOn == date {
			return true
		}
	}
	return false
}

func datesOf(expenses []expenseJSON) []string {
	out := []string{}
	for _, e := range expenses {
		out = append(out, e.OccurredOn)
	}
	return out
}
