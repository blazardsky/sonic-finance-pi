package main

import (
	"net/http"
	"strconv"
	"testing"
)

// A Holding as the API hands it out.
type holdingJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// holdingPath addresses one Holding the way the API does.
func holdingPath(id int64) string {
	return "/api/holdings/" + strconv.FormatInt(id, 10)
}

func (a *testApp) holdings(t *testing.T) []holdingJSON {
	t.Helper()
	var got []holdingJSON
	if res := a.get(t, "/api/holdings", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/holdings = %d, want 200", res.StatusCode)
	}
	return got
}

// createHolding adds one and returns it as the API answered, so a test that
// only needs a Holding to exist is one line.
func (a *testApp) createHolding(t *testing.T, name, typ string) holdingJSON {
	t.Helper()
	var got holdingJSON
	res := a.post(t, "/api/holdings", map[string]string{"name": name, "type": typ}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/holdings %q/%q = %d, want 201", name, typ, res.StatusCode)
	}
	return got
}

// Nothing is seeded: which stocks, ETFs, crypto or bonds a household holds is
// not something the code can guess.
func TestAFreshDatabaseHasNoHoldings(t *testing.T) {
	a := newTestApp(t)

	if got := a.holdings(t); len(got) != 0 {
		t.Errorf("a fresh database has %d Holdings, want none: %v", len(got), got)
	}
}

func TestCreatingAHoldingPutsItInTheList(t *testing.T) {
	a := newTestApp(t)

	created := a.createHolding(t, "VWCE", holdingETF)
	if created.ID == 0 {
		t.Error("the created Holding has no id")
	}

	got := a.holdings(t)
	if len(got) != 1 || got[0] != created {
		t.Errorf("the list is %v, want just %v", got, created)
	}
}

// The portfolio breakdown (ticket 03) groups by id, so a rename must not move
// the row.
func TestRenamingAHoldingKeepsItsIdentity(t *testing.T) {
	a := newTestApp(t)
	before := a.createHolding(t, "BTC", holdingCrypto)

	var got holdingJSON
	res := a.patch(t, holdingPath(before.ID), map[string]string{"name": "Bitcoin"}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.ID != before.ID {
		t.Errorf("the renamed Holding has id %d, want %d", got.ID, before.ID)
	}

	list := a.holdings(t)
	if len(list) != 1 || list[0].Name != "Bitcoin" || list[0].ID != before.ID {
		t.Errorf("after the rename the list is %v, want one row %d named Bitcoin", list, before.ID)
	}
}

// A PATCH can change the type alone, the same partial bargain a Client's
// rename makes.
func TestChangingAHoldingsType(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "Rendite Stato", holdingStock)

	var got holdingJSON
	res := a.patch(t, holdingPath(h.ID), map[string]string{"type": holdingBond}, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH = %d, want 200", res.StatusCode)
	}
	if got.Type != holdingBond {
		t.Errorf("type = %q, want %q", got.Type, holdingBond)
	}
	if got.Name != "Rendite Stato" {
		t.Errorf("changing the type changed the name: %q", got.Name)
	}
}

func TestHoldingWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	h := a.createHolding(t, "VWCE", holdingETF)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"no name", http.MethodPost, "/api/holdings", map[string]string{"type": holdingETF}, http.StatusBadRequest},
		{"blank name", http.MethodPost, "/api/holdings", map[string]string{"name": "  ", "type": holdingETF}, http.StatusBadRequest},
		{"unknown type", http.MethodPost, "/api/holdings", map[string]string{"name": "X", "type": "shares"}, http.StatusBadRequest},
		{"missing type", http.MethodPost, "/api/holdings", map[string]string{"name": "X"}, http.StatusBadRequest},
		{"rename to nothing", http.MethodPatch, holdingPath(h.ID), map[string]string{"name": " ", "type": holdingETF}, http.StatusBadRequest},
		{"unknown id", http.MethodPatch, holdingPath(h.ID + 999), map[string]string{"name": "X", "type": holdingETF}, http.StatusNotFound},
		{"non-numeric id", http.MethodPatch, "/api/holdings/abc", map[string]string{"name": "X", "type": holdingETF}, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.do(t, tc.method, tc.path, tc.body, nil)
			if res.StatusCode != tc.want {
				t.Errorf("%s %s = %d, want %d", tc.method, tc.path, res.StatusCode, tc.want)
			}
		})
	}

	if got := a.holdings(t); len(got) != 1 || got[0] != h {
		t.Errorf("the list is %v, want just the one valid Holding %v", got, h)
	}
}

// "VWCE " and "VWCE" would look identical yet be two rows, which is exactly
// what would silently split one holding's percentage across two — CONTEXT.md.
func TestHoldingNamesAreTrimmed(t *testing.T) {
	a := newTestApp(t)

	if got := a.createHolding(t, "  VWCE  ", holdingETF); got.Name != "VWCE" {
		t.Errorf("created name = %q, want %q", got.Name, "VWCE")
	}
}

func TestHoldingsAreListedCaseInsensitivelyByName(t *testing.T) {
	a := newTestApp(t)
	for _, name := range []string{"zcash", "Bitcoin", "amazon", "VWCE"} {
		a.createHolding(t, name, holdingOther)
	}

	var got []string
	for _, h := range a.holdings(t) {
		got = append(got, h.Name)
	}
	want := []string{"amazon", "Bitcoin", "VWCE", "zcash"}
	if len(got) != len(want) {
		t.Fatalf("the list is %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the list is %v, want %v", got, want)
		}
	}
}
