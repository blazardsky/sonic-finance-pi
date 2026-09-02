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
