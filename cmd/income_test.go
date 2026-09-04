package main

import (
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

// An Income as the API hands it out. Both dates are optional and both are
// empty when absent: an Income with an empty payment_date is unpaid, which is
// the whole of ADR-0003's state. client_id is a pointer because "nobody" is a
// real answer — a birthday gift names no Client.
type incomeJSON struct {
	ID              int64  `json:"id"`
	AmountCents     int64  `json:"amount_cents"`
	CategoryID      int64  `json:"category_id"`
	ClientID        *int64 `json:"client_id"`
	Payer           string `json:"payer"`
	PaymentDate     string `json:"payment_date"`
	InvoiceSentDate string `json:"invoice_sent_date"`
	Note            string `json:"note"`
}

// incomePath addresses one Income the way the API does.
func incomePath(id int64) string {
	return "/api/incomes/" + strconv.FormatInt(id, 10)
}

func (a *testApp) incomes(t *testing.T) []incomeJSON {
	t.Helper()
	var got []incomeJSON
	if res := a.get(t, "/api/incomes", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/incomes = %d, want 200", res.StatusCode)
	}
	return got
}

// addIncome records one and returns it as the API answered, failing the test
// if the save was refused. Later tickets' totals need a stocked month, and
// this is the one place that knows how to stock it.
//
// Payer defaults here when a case does not care about it, now that ticket 08
// makes it required: the alternative is threading a payer through every one
// of this file's callers, most of which are testing something else entirely.
func (a *testApp) addIncome(t *testing.T, body map[string]any) incomeJSON {
	t.Helper()
	if _, ok := body["payer"]; !ok {
		body["payer"] = "Nicco"
	}
	var created incomeJSON
	res := a.post(t, "/api/incomes", body, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/incomes %v = %d, want 201", body, res.StatusCode)
	}
	return created
}

// freelance is the Income-side Base category, and what most of these tests
// record money under.
func (a *testApp) freelance(t *testing.T) categoryJSON {
	t.Helper()
	return a.category(t, seedFreelanceName)
}

// The whole ticket in one test: what one request saved is what a later request
// sees, down to the cent.
func TestARecordedIncomeIsThereForTheNextRequest(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)

	created := a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  freelance.ID,
		"payment_date": "2026-03-10",
	})
	if created.ID == 0 {
		t.Error("the created Income came back without an id")
	}

	got := a.incomes(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Incomes, want 1", len(got))
	}
	if !reflect.DeepEqual(got[0], created) {
		t.Errorf("listed Income = %+v, want the one that was created, %+v", got[0], created)
	}
	if got[0].AmountCents != 120000 || got[0].CategoryID != freelance.ID {
		t.Errorf("listed Income = %+v, want 120000 cents in %d", got[0], freelance.ID)
	}
}

// ADR-0003: an Income exists from the moment the invoice goes out. No payment
// date is not a missing field — it is the unpaid state, and it has to survive
// the round trip as one.
func TestAnIncomeCanBeRecordedWithNoPaymentDate(t *testing.T) {
	a := newTestApp(t)

	created := a.addIncome(t, map[string]any{
		"amount_cents":      50000,
		"category_id":       a.freelance(t).ID,
		"invoice_sent_date": "2026-03-01",
	})
	if created.PaymentDate != "" {
		t.Errorf("payment_date = %q, want empty — the Income is unpaid", created.PaymentDate)
	}

	got := a.incomes(t)
	if len(got) != 1 || got[0].PaymentDate != "" {
		t.Fatalf("listed Incomes = %+v, want one unpaid", got)
	}
	if got[0].InvoiceSentDate != "2026-03-01" {
		t.Errorf("invoice_sent_date = %q, want 2026-03-01", got[0].InvoiceSentDate)
	}
}

// The invoice goes out, weeks pass, the money lands. The same row carries
// both dates, and the second is added by an edit rather than a re-entry.
//
// Unpaid Incomes count toward no total, so this transition is the moment an
// Income starts counting. Ticket 10 owns the sums; what is asserted here is
// the state they will filter on, in both directions.
func TestAPaymentDateIsAddedLaterAndCanBeTakenBack(t *testing.T) {
	a := newTestApp(t)
	unpaid := a.addIncome(t, map[string]any{
		"amount_cents":      80000,
		"category_id":       a.freelance(t).ID,
		"invoice_sent_date": "2026-02-20",
	})

	var paid incomeJSON
	res := a.patch(t, incomePath(unpaid.ID), map[string]any{"payment_date": "2026-03-14"}, &paid)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", incomePath(unpaid.ID), res.StatusCode)
	}
	if paid.PaymentDate != "2026-03-14" {
		t.Errorf("payment_date = %q, want 2026-03-14", paid.PaymentDate)
	}
	// The invoice-sent date is what "how long has this been sitting" is
	// measured from, so recording the payment must not wipe it.
	if paid.InvoiceSentDate != "2026-02-20" {
		t.Errorf("invoice_sent_date = %q, want it left at 2026-02-20", paid.InvoiceSentDate)
	}

	// A payment date typed onto the wrong Income has to be removable, or a
	// mistake would leave money counted that never arrived.
	var back incomeJSON
	a.patch(t, incomePath(unpaid.ID), map[string]any{"payment_date": ""}, &back)
	if back.PaymentDate != "" {
		t.Errorf("payment_date = %q after clearing it, want empty", back.PaymentDate)
	}
	if got := a.incomes(t); got[0].PaymentDate != "" {
		t.Errorf("the stored Income reads %+v, want it unpaid again", got[0])
	}
}

// Everything an Income carries beyond the amount, in one round trip. The
// Category is the "income reason" the UI words differently — it is not a
// second field — and the Payer comes from the same list an Expense uses.
func TestAnIncomeKeepsItsDetails(t *testing.T) {
	a := newTestApp(t)
	studio := a.createClient(t, "Studio Rossi")

	created := a.addIncome(t, map[string]any{
		"amount_cents":      120000,
		"category_id":       a.freelance(t).ID,
		"client_id":         studio.ID,
		"payer":             seedPayerBoth,
		"payment_date":      "2026-03-10",
		"invoice_sent_date": "2026-02-10",
		"note":              "fattura 12/2026",
	})

	got := a.incomes(t)[0]
	if got.ClientID == nil || *got.ClientID != studio.ID {
		t.Errorf("client_id = %v, want %d", got.ClientID, studio.ID)
	}
	if got.Payer != seedPayerBoth || got.Note != "fattura 12/2026" {
		t.Errorf("payer/note = %q/%q, want %q/%q", got.Payer, got.Note, seedPayerBoth, "fattura 12/2026")
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("listed Income = %+v, want the one that was created, %+v", got, created)
	}
}

// A Client is optional: "Mum" is as valid a Client as a studio, and a
// reimbursement comes from nobody worth naming at all.
func TestTheClientIsOptional(t *testing.T) {
	a := newTestApp(t)

	created := a.addIncome(t, map[string]any{
		"amount_cents": 5000,
		"category_id":  a.category(t, "Regali").ID,
	})
	if created.ClientID != nil {
		t.Errorf("client_id = %v, want null", *created.ClientID)
	}
	if got := a.incomes(t)[0]; got.ClientID != nil {
		t.Errorf("stored client_id = %v, want null", *got.ClientID)
	}
}

// A named Client can be taken off again — the Income was filed against the
// wrong one, and correcting it must not mean deleting the money.
func TestAClientCanBeClearedFromAnIncome(t *testing.T) {
	a := newTestApp(t)
	studio := a.createClient(t, "Studio Rossi")
	created := a.addIncome(t, map[string]any{
		"amount_cents": 5000,
		"category_id":  a.freelance(t).ID,
		"client_id":    studio.ID,
	})

	var got incomeJSON
	a.patch(t, incomePath(created.ID), map[string]any{"client_id": nil}, &got)
	if got.ClientID != nil {
		t.Errorf("client_id = %v after clearing it, want null", *got.ClientID)
	}
}

func TestIncomeWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	alimentari := a.category(t, "Alimentari")

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"no amount", map[string]any{"category_id": freelance.ID}},
		{"a negative amount", map[string]any{"amount_cents": -100, "category_id": freelance.ID}},
		{"no category", map[string]any{"amount_cents": 100}},
		{"a category that does not exist", map[string]any{"amount_cents": 100, "category_id": 9999}},
		{"an expense-only category", map[string]any{"amount_cents": 100, "category_id": alimentari.ID}},
		{"a client that does not exist", map[string]any{"amount_cents": 100, "category_id": freelance.ID, "client_id": 9999}},
		{"a malformed payment date", map[string]any{"amount_cents": 100, "category_id": freelance.ID, "payment_date": "10/03/2026"}},
		{"an impossible payment date", map[string]any{"amount_cents": 100, "category_id": freelance.ID, "payment_date": "2026-02-30"}},
		{"a malformed invoice-sent date", map[string]any{"amount_cents": 100, "category_id": freelance.ID, "invoice_sent_date": "2026-3-1"}},
		// Ticket 08: "whose money was it" is never left unanswered going forward.
		{"an empty payer", map[string]any{"amount_cents": 100, "category_id": freelance.ID, "payer": ""}},
		{"a whitespace payer", map[string]any{"amount_cents": 100, "category_id": freelance.ID, "payer": "   "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.post(t, "/api/incomes", tc.body, nil)
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("POST with %s = %d, want 400", tc.name, res.StatusCode)
			}
		})
	}

	if got := a.incomes(t); len(got) != 0 {
		t.Errorf("a refused write stored %d Incomes: %v", len(got), got)
	}
}

// An edit answers to the same rules a create does, or an edit becomes the way
// past them.
func TestIncomeEditsAreValidated(t *testing.T) {
	a := newTestApp(t)
	created := a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  a.freelance(t).ID,
		"payment_date": "2026-03-10",
	})

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"a zero amount", map[string]any{"amount_cents": 0}},
		{"an expense-only category", map[string]any{"category_id": a.category(t, "Alimentari").ID}},
		{"a client that does not exist", map[string]any{"client_id": 9999}},
		{"a malformed payment date", map[string]any{"payment_date": "domani"}},
		// Ticket 08: an edit setting the Payer empty is refused the same as a create.
		{"an empty payer", map[string]any{"payer": ""}},
		{"a whitespace payer", map[string]any{"payer": "   "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.patch(t, incomePath(created.ID), tc.body, nil)
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("PATCH with %s = %d, want 400", tc.name, res.StatusCode)
			}
		})
	}

	if got := a.incomes(t)[0]; !reflect.DeepEqual(got, created) {
		t.Errorf("a refused edit left %+v, want %+v", got, created)
	}
}

// The same partial-edit bargain the Expense screen makes: a field the body
// omits keeps what it had, so recording a payment does not blank the note.
func TestPatchingAnIncomeLeavesTheOmittedFieldsAlone(t *testing.T) {
	a := newTestApp(t)
	created := a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  a.freelance(t).ID,
		"payer":        seedPayerBoth,
		"note":         "fattura 12/2026",
	})

	var got incomeJSON
	a.patch(t, incomePath(created.ID), map[string]any{"amount_cents": 130000}, &got)
	if got.AmountCents != 130000 {
		t.Errorf("amount_cents = %d, want 130000", got.AmountCents)
	}
	if got.Payer != created.Payer || got.Note != created.Note {
		t.Errorf("payer/note = %q/%q, want them untouched at %q/%q", got.Payer, got.Note, created.Payer, created.Note)
	}
}

func TestAnIncomeCanBeDeleted(t *testing.T) {
	a := newTestApp(t)
	created := a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  a.freelance(t).ID,
	})

	if res := a.delete(t, incomePath(created.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", incomePath(created.ID), res.StatusCode)
	}
	if got := a.incomes(t); len(got) != 0 {
		t.Errorf("the list still holds %v", got)
	}
	if res := a.delete(t, incomePath(created.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("deleting it twice = %d, want 404", res.StatusCode)
	}
}

// Ticket 08's box, finished here: an Income references a Client rather than
// copying its name, so a rename is what every past Income displays from then
// on — with no rewrite of anything.
func TestRenamingAClientChangesWhatAPastIncomeDisplays(t *testing.T) {
	a := newTestApp(t)
	studio := a.createClient(t, "Studio Rosi")
	created := a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  a.freelance(t).ID,
		"client_id":    studio.ID,
	})

	a.patch(t, clientPath(studio.ID), map[string]string{"name": "Studio Rossi"}, nil)

	got := a.incomes(t)[0]
	if got.ClientID == nil || *got.ClientID != studio.ID {
		t.Fatalf("client_id = %v, want it still %d", got.ClientID, studio.ID)
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("the Income now reads %+v, want it unchanged at %+v", got, created)
	}
	// What the Income displays is the Client's current name, resolved through
	// that id — which is the half a rename is supposed to fix.
	for _, c := range a.clients(t) {
		if c.ID == studio.ID && c.Name != "Studio Rossi" {
			t.Errorf("the Client the Income points at is named %q, want %q", c.Name, "Studio Rossi")
		}
	}
}

// The other half of ticket 08's bargain: hiding takes a Client out of the
// pickers, not out of the app. A past Income still resolves it.
func TestHidingAClientLeavesItsIncomesResolvable(t *testing.T) {
	a := newTestApp(t)
	studio := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  a.freelance(t).ID,
		"client_id":    studio.ID,
	})

	if res := a.patch(t, clientPath(studio.ID), map[string]any{"hidden": true}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("hiding the Client = %d, want 200", res.StatusCode)
	}

	got := a.incomes(t)[0]
	if got.ClientID == nil || *got.ClientID != studio.ID {
		t.Errorf("client_id = %v, want it still %d", got.ClientID, studio.ID)
	}
	var found bool
	for _, c := range a.clients(t) {
		if c.ID == studio.ID {
			found = true
			if !c.Hidden {
				t.Error("the Client the Income points at is not hidden")
			}
		}
	}
	if !found {
		t.Error("the hidden Client is gone from /api/clients, so nothing can resolve the Income")
	}
}

// A Client with money behind it is retired by hiding, not deleting. The
// foreign key refuses, and that refusal is a conflict rather than a fault:
// deleting the name off a past Income would lose what it says, and deleting
// the Income with it would lose the money.
func TestAClientWithIncomesCannotBeDeleted(t *testing.T) {
	a := newTestApp(t)
	studio := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 120000,
		"category_id":  a.freelance(t).ID,
		"client_id":    studio.ID,
	})

	if res := a.delete(t, clientPath(studio.ID)); res.StatusCode != http.StatusConflict {
		t.Fatalf("DELETE a Client with an Income = %d, want 409", res.StatusCode)
	}
	if got := a.incomes(t); len(got) != 1 || got[0].ClientID == nil {
		t.Errorf("the Income reads %+v, want it still pointing at its Client", got)
	}
}

// Deleting a Category with an Income in it is refused for the same reason, by
// the same foreign key — the Category tests assert it through an Expense.
func TestDeletingACategoryWithIncomesIsRefused(t *testing.T) {
	a := newTestApp(t)
	regali := a.category(t, "Regali")
	a.addIncome(t, map[string]any{"amount_cents": 5000, "category_id": regali.ID})

	if res := a.delete(t, categoryPath(regali.ID)); res.StatusCode != http.StatusConflict {
		t.Errorf("DELETE a Category with an Income = %d, want 409", res.StatusCode)
	}
}

// An Income can be recorded under a hidden Category: hiding shortens the
// picker, and an Income being corrected months later still belongs where it
// was filed.
func TestAnIncomeCanBeRecordedInAHiddenCategory(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	a.patch(t, categoryPath(freelance.ID), map[string]any{"hidden": true}, nil)

	a.addIncome(t, map[string]any{"amount_cents": 120000, "category_id": freelance.ID})
}

// The list sits under the form, so what was just recorded is the first thing
// on it. An Income has no single date — an unpaid one has none at all — so the
// order is the order they were entered, newest first.
func TestIncomesAreListedNewestFirst(t *testing.T) {
	a := newTestApp(t)
	id := a.freelance(t).ID

	var ids []int64
	for i := 0; i < 3; i++ {
		ids = append(ids, a.addIncome(t, map[string]any{"amount_cents": 1000, "category_id": id}).ID)
	}

	got := a.incomes(t)
	if len(got) != 3 {
		t.Fatalf("listed %d Incomes, want 3", len(got))
	}
	for i, want := range []int64{ids[2], ids[1], ids[0]} {
		if got[i].ID != want {
			t.Fatalf("ids = %d, %d, %d, want them newest first: %v", got[0].ID, got[1].ID, got[2].ID, []int64{ids[2], ids[1], ids[0]})
		}
	}
}

// Free text is trimmed on the way in, so " fattura " and "fattura" are not two
// different notes.
func TestIncomeTextIsTrimmed(t *testing.T) {
	a := newTestApp(t)

	created := a.addIncome(t, map[string]any{
		"amount_cents": 5000,
		"category_id":  a.freelance(t).ID,
		"payer":        "  " + seedPayerBoth + "  ",
		"note":         "  fattura  ",
	})
	if created.Payer != seedPayerBoth || created.Note != "fattura" {
		t.Errorf("payer/note = %q/%q, want them trimmed to %q/%q", created.Payer, created.Note, seedPayerBoth, "fattura")
	}
}

// The Payer is the label text itself, not a reference into the list it was
// chosen from — ADR-0001, and the same bargain an Expense makes. Renaming the
// list must leave what a past Income says exactly as it was.
func TestRenamingAPayerLeavesExistingIncomesReadingAsBefore(t *testing.T) {
	a := newTestApp(t)
	a.addIncome(t, map[string]any{
		"amount_cents": 5000,
		"category_id":  a.freelance(t).ID,
		"payer":        "Persona 1",
	})

	res := a.put(t, settingsPath, map[string]any{
		"payers": []string{"Nicco", "Persona 2", seedPayerBoth, seedPayerSomeoneElse},
	}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	if got := a.incomes(t)[0]; got.Payer != "Persona 1" {
		t.Errorf("payer = %q, want it still reading %q", got.Payer, "Persona 1")
	}
}
