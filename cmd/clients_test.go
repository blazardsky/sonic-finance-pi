package main

import (
	"net/http"
	"strconv"
	"testing"
)

// A Client as the API hands it out. Three fields is the whole of it: a Client
// is a name money comes from, with a switch for whether the pickers still
// offer it.
type clientJSON struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Hidden bool   `json:"hidden"`

	// Ticket 04: the default Income Category prefill, and the computed
	// all-time total earned from this Client's received Incomes.
	DefaultCategoryID *int64 `json:"default_category_id"`
	TotalEarnedCents  int64  `json:"total_earned_cents"`

	// The Payer prefill, the label-side twin of the default Category.
	DefaultPayer string `json:"default_payer"`
}

// clientPath addresses one Client the way the API does.
func clientPath(id int64) string {
	return "/api/clients/" + strconv.FormatInt(id, 10)
}

func (a *testApp) clients(t *testing.T) []clientJSON {
	t.Helper()
	var got []clientJSON
	if res := a.get(t, "/api/clients", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/clients = %d, want 200", res.StatusCode)
	}
	return got
}

// createClient adds one and returns it as the API answered, so a test that
// only needs a Client to exist is one line.
func (a *testApp) createClient(t *testing.T, name string) clientJSON {
	t.Helper()
	var got clientJSON
	res := a.post(t, "/api/clients", map[string]string{"name": name}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/clients %q = %d, want 201", name, res.StatusCode)
	}
	return got
}

// Nothing is seeded. Who the household is paid by is not something the code
// can guess, and an empty list is the honest starting point — unlike
// Categories, where thirteen guesses save an evening of typing.
func TestAFreshDatabaseHasNoClients(t *testing.T) {
	a := newTestApp(t)

	if got := a.clients(t); len(got) != 0 {
		t.Errorf("a fresh database has %d Clients, want none: %v", len(got), got)
	}
}

func TestCreatingAClientPutsItInTheList(t *testing.T) {
	a := newTestApp(t)

	created := a.createClient(t, "Mum")
	if created.ID == 0 {
		t.Error("the created Client has no id")
	}
	if created.Hidden {
		t.Error("a new Client is hidden, want visible")
	}

	got := a.clients(t)
	if len(got) != 1 || got[0] != created {
		t.Errorf("the list is %v, want just %v", got, created)
	}
}

// The id is the point of the test: an Income references a Client rather than
// copying its name, so a rename under a stable id is what makes every past
// Income display the new one. Ticket 09 asserts that end to end.
func TestRenamingAClientKeepsItsIdentity(t *testing.T) {
	a := newTestApp(t)
	before := a.createClient(t, "Studio Rosi")

	var got clientJSON
	res := a.patch(t, clientPath(before.ID), map[string]string{"name": "Studio Rossi"}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.ID != before.ID {
		t.Errorf("the renamed Client has id %d, want %d — a rename must not move the row",
			got.ID, before.ID)
	}

	list := a.clients(t)
	if len(list) != 1 || list[0].Name != "Studio Rossi" || list[0].ID != before.ID {
		t.Errorf("after the rename the list is %v, want one row %d named Studio Rossi",
			list, before.ID)
	}
}

// Hiding takes a Client out of the pickers, not out of the app: the list still
// carries it — this screen is the only place it can be brought back from — and
// an Income already pointing at it still resolves, because the row is there.
func TestHidingAClientLeavesItResolvable(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Ex Cliente")

	var hidden clientJSON
	if res := a.patch(t, clientPath(c.ID), map[string]bool{"hidden": true}, &hidden); res.StatusCode != http.StatusOK {
		t.Fatalf("hiding: PATCH = %d, want 200", res.StatusCode)
	}
	if !hidden.Hidden {
		t.Error("hidden = false after hiding")
	}

	list := a.clients(t)
	if len(list) != 1 || !list[0].Hidden || list[0].ID != c.ID {
		t.Fatalf("after hiding the list is %v, want the same row still there and hidden", list)
	}

	var shown clientJSON
	a.patch(t, clientPath(c.ID), map[string]bool{"hidden": false}, &shown)
	if shown.Hidden {
		t.Error("hidden = true after unhiding — a hidden Client can never come back")
	}
}

// A PATCH says only what it changes: hiding a Client must not blank its name.
func TestPatchingOneFieldLeavesTheOtherAlone(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Comune di Trento")

	a.patch(t, clientPath(c.ID), map[string]bool{"hidden": true}, nil)

	list := a.clients(t)
	if len(list) != 1 || list[0].Name != "Comune di Trento" {
		t.Errorf("after hiding, the list is %v, want the name untouched", list)
	}
}

func TestDeletingAClientRemovesIt(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Typo Srl")

	if res := a.delete(t, clientPath(c.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", res.StatusCode)
	}
	if got := a.clients(t); len(got) != 0 {
		t.Errorf("after the delete the list is %v, want empty", got)
	}
	// Deleting it twice is a 404 rather than a second 204: from outside, an id
	// that is gone and an id that never existed are the same thing.
	if res := a.delete(t, clientPath(c.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("deleting twice = %d, want 404", res.StatusCode)
	}
}

func TestClientWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Valido")

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"no name", http.MethodPost, "/api/clients", map[string]string{}, http.StatusBadRequest},
		{"empty name", http.MethodPost, "/api/clients", map[string]string{"name": ""}, http.StatusBadRequest},
		{"blank name", http.MethodPost, "/api/clients", map[string]string{"name": "   "}, http.StatusBadRequest},
		{"rename to nothing", http.MethodPatch, clientPath(c.ID), map[string]string{"name": " "}, http.StatusBadRequest},
		{"unknown id", http.MethodPatch, clientPath(c.ID + 999), map[string]string{"name": "X"}, http.StatusNotFound},
		{"non-numeric id", http.MethodPatch, "/api/clients/abc", map[string]string{"name": "X"}, http.StatusNotFound},
		{"delete unknown id", http.MethodDelete, clientPath(c.ID + 999), nil, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.do(t, tc.method, tc.path, tc.body, nil)
			if res.StatusCode != tc.want {
				t.Errorf("%s %s = %d, want %d", tc.method, tc.path, res.StatusCode, tc.want)
			}
		})
	}

	// None of the above landed anyway.
	if got := a.clients(t); len(got) != 1 || got[0] != c {
		t.Errorf("the list is %v, want just the one valid Client %v", got, c)
	}
}

// "Rossi " and "Rossi" would look identical in a picker and be two rows, which
// is the exact problem a Client is here to solve.
func TestClientNamesAreTrimmed(t *testing.T) {
	a := newTestApp(t)

	if got := a.createClient(t, "  Rossi  "); got.Name != "Rossi" {
		t.Errorf("created name = %q, want %q", got.Name, "Rossi")
	}

	c := a.createClient(t, "Bianchi")
	var renamed clientJSON
	a.patch(t, clientPath(c.ID), map[string]string{"name": " Bianchi Srl "}, &renamed)
	if renamed.Name != "Bianchi Srl" {
		t.Errorf("renamed name = %q, want %q", renamed.Name, "Bianchi Srl")
	}
}

// A Client is not scoped to freelance work: nothing on the way in asks what
// kind of Income it is for, so "Mum" is as valid a Client as "Studio Rossi".
func TestAnyNameIsAValidClient(t *testing.T) {
	a := newTestApp(t)

	for _, name := range []string{"Mum", "Studio Rossi", "Comune di Trento", "Zia Carla"} {
		if got := a.createClient(t, name); got.Name != name {
			t.Errorf("created %q, got %q", name, got.Name)
		}
	}
	if got := a.clients(t); len(got) != 4 {
		t.Errorf("the list has %d Clients, want 4: %v", len(got), got)
	}
}

// The list is ordered so the screen is stable across reloads, and
// case-insensitively because SQLite's default collation is binary — "zia"
// would otherwise sort above every capitalised name.
func TestClientsAreListedCaseInsensitivelyByName(t *testing.T) {
	a := newTestApp(t)
	for _, name := range []string{"zia Carla", "Banca", "mamma", "Studio Rossi"} {
		a.createClient(t, name)
	}

	var got []string
	for _, c := range a.clients(t) {
		got = append(got, c.Name)
	}
	want := []string{"Banca", "mamma", "Studio Rossi", "zia Carla"}
	if len(got) != len(want) {
		t.Fatalf("the list is %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the list is %v, want %v", got, want)
		}
	}
}

// Highest earner first — the name is only the tiebreak, which the test above
// already covers for the all-zero case.
func TestClientsAreListedHighestEarnerFirst(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)

	low := a.createClient(t, "Cliente Basso")
	high := a.createClient(t, "Cliente Alto")
	zero := a.createClient(t, "Cliente Zero")

	a.addIncome(t, map[string]any{
		"amount_cents": 10000, "category_id": freelance.ID, "client_id": low.ID, "payment_date": "2026-03-10",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 90000, "category_id": freelance.ID, "client_id": high.ID, "payment_date": "2026-03-10",
	})

	var got []string
	for _, c := range a.clients(t) {
		got = append(got, c.Name)
	}
	want := []string{high.Name, low.Name, zero.Name}
	if len(got) != len(want) {
		t.Fatalf("the list is %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the list is %v, want %v", got, want)
		}
	}
}

// Ticket 04: a default Income Category is a plain PATCH-able field, exactly
// like hidden — a new Client starts with none, and one it is given can later
// be taken away.
func TestAClientsDefaultCategoryCanBeSetReadAndCleared(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	c := a.createClient(t, "Studio Rossi")

	if c.DefaultCategoryID != nil {
		t.Fatalf("a new Client's default category = %v, want nil", c.DefaultCategoryID)
	}

	var withDefault clientJSON
	res := a.patch(t, clientPath(c.ID), map[string]any{"default_category_id": freelance.ID}, &withDefault)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("setting a default category: PATCH = %d, want 200", res.StatusCode)
	}
	if withDefault.DefaultCategoryID == nil || *withDefault.DefaultCategoryID != freelance.ID {
		t.Fatalf("default_category_id after setting it = %v, want %d", withDefault.DefaultCategoryID, freelance.ID)
	}

	list := a.clients(t)
	if len(list) != 1 || list[0].DefaultCategoryID == nil || *list[0].DefaultCategoryID != freelance.ID {
		t.Fatalf("the list is %v, want the default category to stick", list)
	}

	var cleared clientJSON
	a.patch(t, clientPath(c.ID), map[string]any{"default_category_id": nil}, &cleared)
	if cleared.DefaultCategoryID != nil {
		t.Errorf("default_category_id after clearing it = %v, want nil", cleared.DefaultCategoryID)
	}
}

// The Payer prefill, on the same terms as the Category one above: unset on a
// new Client, PATCH-able, trimmed so it still matches a settings-list label,
// and clearable back to "". Never validated against that list — a Payer
// renamed away leaves this reading as it was, exactly like income.payer.
func TestAClientsDefaultPayerCanBeSetReadAndCleared(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Studio Rossi")

	if c.DefaultPayer != "" {
		t.Fatalf("a new Client's default payer = %q, want empty", c.DefaultPayer)
	}

	var set clientJSON
	res := a.patch(t, clientPath(c.ID), map[string]any{"default_payer": "  Nicco  "}, &set)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("setting a default payer: PATCH = %d, want 200", res.StatusCode)
	}
	if set.DefaultPayer != "Nicco" {
		t.Fatalf("default_payer after setting it = %q, want %q", set.DefaultPayer, "Nicco")
	}

	// A rename says nothing about the Payer: the partial PATCH must leave it.
	var renamed clientJSON
	a.patch(t, clientPath(c.ID), map[string]any{"name": "Studio Rossi srl"}, &renamed)
	if renamed.DefaultPayer != "Nicco" {
		t.Errorf("default_payer after a rename = %q, want it left alone", renamed.DefaultPayer)
	}

	list := a.clients(t)
	if len(list) != 1 || list[0].DefaultPayer != "Nicco" {
		t.Fatalf("the list is %v, want the default payer to stick", list)
	}

	var cleared clientJSON
	a.patch(t, clientPath(c.ID), map[string]any{"default_payer": ""}, &cleared)
	if cleared.DefaultPayer != "" {
		t.Errorf("default_payer after clearing it = %q, want empty", cleared.DefaultPayer)
	}
}

// Ticket 04's whole computation: received Incomes count, an invoice sent but
// not yet paid does not, and a Client with no Incomes at all reads as zero
// rather than erroring — the case the ticket calls out by name.
func TestTotalEarnedCountsOnlyThatClientsReceivedIncomes(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)

	paid := a.createClient(t, "Studio Rossi")
	unpaidOnly := a.createClient(t, "Cliente Lento")
	untouched := a.createClient(t, "Cliente Nuovo")

	a.addIncome(t, map[string]any{
		"amount_cents": 50000,
		"category_id":  freelance.ID,
		"client_id":    paid.ID,
		"payment_date": "2026-03-10",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 30000,
		"category_id":  freelance.ID,
		"client_id":    paid.ID,
		"payment_date": "2026-04-01",
	})
	// Invoiced but not yet paid: not money in hand, so it must not count
	// (ADR-0003) — and must not count against a different Client either.
	a.addIncome(t, map[string]any{
		"amount_cents":      99999,
		"category_id":       freelance.ID,
		"client_id":         unpaidOnly.ID,
		"invoice_sent_date": "2026-04-01",
	})

	byID := map[int64]clientJSON{}
	for _, c := range a.clients(t) {
		byID[c.ID] = c
	}

	if got := byID[paid.ID].TotalEarnedCents; got != 80000 {
		t.Errorf("total earned for %s = %d, want 80000", paid.Name, got)
	}
	if got := byID[unpaidOnly.ID].TotalEarnedCents; got != 0 {
		t.Errorf("total earned for %s = %d, want 0 — its only Income is unpaid", unpaidOnly.Name, got)
	}
	if got := byID[untouched.ID].TotalEarnedCents; got != 0 {
		t.Errorf("total earned for %s = %d, want 0 — it has no Incomes at all", untouched.Name, got)
	}
}

// total_earned_cents is computed, not stored: a PATCH claiming one must not
// be able to make it say anything but the truth.
func TestPatchingAClientCannotForgeTotalEarned(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	c := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 50000,
		"category_id":  freelance.ID,
		"client_id":    c.ID,
		"payment_date": "2026-03-10",
	})

	var got clientJSON
	a.patch(t, clientPath(c.ID),
		map[string]any{"name": "Studio Rossi Srl", "total_earned_cents": 999999999}, &got)
	if got.TotalEarnedCents != 50000 {
		t.Errorf("total_earned_cents after a PATCH claiming otherwise = %d, want the real 50000", got.TotalEarnedCents)
	}
}
