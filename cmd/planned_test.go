package main

import (
	"net/http"
	"strconv"
	"testing"
)

const plannedPath = "/api/planned-purchases"

func plannedItemPath(id int64) string {
	return plannedPath + "/" + strconv.FormatInt(id, 10)
}

func (a *testApp) planned(t *testing.T) []plannedPurchase {
	t.Helper()
	var got []plannedPurchase
	if res := a.get(t, plannedPath, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", plannedPath, res.StatusCode)
	}
	return got
}

func (a *testApp) addPlanned(t *testing.T, label string, cents int64) plannedPurchase {
	t.Helper()
	var created plannedPurchase
	res := a.post(t, plannedPath, map[string]any{"label": label, "amount_cents": cents}, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST %s %q = %d, want 201", plannedPath, label, res.StatusCode)
	}
	return created
}

func plannedLabels(list []plannedPurchase) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.Label
	}
	return out
}

func assertLabels(t *testing.T, got []plannedPurchase, want ...string) {
	t.Helper()
	labels := plannedLabels(got)
	if len(labels) != len(want) {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("labels = %v, want %v", labels, want)
		}
	}
}

// New ones go to the bottom: adding never reshuffles the priorities.
func TestPlannedPurchaseNewGoesToTheBottom(t *testing.T) {
	a := newTestApp(t)
	a.addPlanned(t, "Laptop", 150000)
	a.addPlanned(t, "Divano", 80000)
	a.addPlanned(t, "Cuffie", 20000)
	assertLabels(t, a.planned(t), "Laptop", "Divano", "Cuffie")
}

func TestPlannedPurchaseUpdateAndDelete(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	p := a.addPlanned(t, "Laptop", 150000)

	var updated plannedPurchase
	res := a.put(t, plannedItemPath(p.ID), map[string]any{
		"label": "Laptop nuovo", "amount_cents": 140000, "category_id": alimentari.ID,
	}, &updated)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT = %d, want 200", res.StatusCode)
	}
	got := a.planned(t)[0]
	if got.Label != "Laptop nuovo" || got.AmountCents != 140000 || got.CategoryID == nil || *got.CategoryID != alimentari.ID {
		t.Fatalf("after update = %+v", got)
	}
	if got.Position != p.Position {
		t.Errorf("position = %d after update, want it unchanged at %d", got.Position, p.Position)
	}

	if res := a.delete(t, plannedItemPath(p.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE = %d, want 204", res.StatusCode)
	}
	if n := len(a.planned(t)); n != 0 {
		t.Fatalf("%d planned purchases left after delete, want 0", n)
	}
	if res := a.delete(t, plannedItemPath(p.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("second DELETE = %d, want 404", res.StatusCode)
	}
}

func TestPlannedPurchaseRejectsInvalidInput(t *testing.T) {
	a := newTestApp(t)
	for name, body := range map[string]map[string]any{
		"blank label":      {"label": "  ", "amount_cents": 100},
		"zero amount":      {"label": "Laptop", "amount_cents": 0},
		"negative amount":  {"label": "Laptop", "amount_cents": -5},
		"unknown category": {"label": "Laptop", "amount_cents": 100, "category_id": 99999},
	} {
		if res := a.post(t, plannedPath, body, nil); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: POST = %d, want 400", name, res.StatusCode)
		}
	}
	if n := len(a.planned(t)); n != 0 {
		t.Errorf("%d planned purchases saved from invalid input, want 0", n)
	}
}

// Up/down move one place, and refuse to move past either end.
func TestPlannedPurchaseMove(t *testing.T) {
	a := newTestApp(t)
	laptop := a.addPlanned(t, "Laptop", 150000)
	a.addPlanned(t, "Divano", 80000)
	cuffie := a.addPlanned(t, "Cuffie", 20000)

	move := func(id int64, dir string) int {
		t.Helper()
		return a.post(t, plannedItemPath(id)+"/move", map[string]string{"direction": dir}, nil).StatusCode
	}

	if code := move(cuffie.ID, "up"); code != http.StatusOK {
		t.Fatalf("move up = %d, want 200", code)
	}
	assertLabels(t, a.planned(t), "Laptop", "Cuffie", "Divano")

	if code := move(laptop.ID, "down"); code != http.StatusOK {
		t.Fatalf("move down = %d, want 200", code)
	}
	assertLabels(t, a.planned(t), "Cuffie", "Laptop", "Divano")

	if code := move(cuffie.ID, "up"); code != http.StatusBadRequest {
		t.Errorf("moving the first up = %d, want 400", code)
	}
	divano := a.planned(t)[2]
	if code := move(divano.ID, "down"); code != http.StatusBadRequest {
		t.Errorf("moving the last down = %d, want 400", code)
	}
	if code := move(divano.ID, "sideways"); code != http.StatusBadRequest {
		t.Errorf("unknown direction = %d, want 400", code)
	}
	assertLabels(t, a.planned(t), "Cuffie", "Laptop", "Divano")
}
