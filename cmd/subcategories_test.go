package main

import (
	"net/http"
	"strconv"
	"testing"
)

// A Subcategory as the API hands it out. Unlike Category, there is no Base
// flag here — nothing resolves a Subcategory by identity, so every one of
// them is the household's to rename, hide or delete freely.
type subcategoryJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AppliesTo string `json:"applies_to"`
	Hidden    bool   `json:"hidden"`
}

func subcategoryPath(id int64) string {
	return "/api/subcategories/" + strconv.FormatInt(id, 10)
}

func (a *testApp) subcategories(t *testing.T) []subcategoryJSON {
	t.Helper()
	var got []subcategoryJSON
	if res := a.get(t, "/api/subcategories", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/subcategories = %d, want 200", res.StatusCode)
	}
	return got
}

func (a *testApp) subcategory(t *testing.T, name string) subcategoryJSON {
	t.Helper()
	for _, s := range a.subcategories(t) {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("no Subcategory named %q in %v", name, a.subcategories(t))
	return subcategoryJSON{}
}

// createSubcategoryNamed adds an expense-side Subcategory and returns it.
func (a *testApp) createSubcategoryNamed(t *testing.T, name string) subcategoryJSON {
	t.Helper()
	var got subcategoryJSON
	res := a.post(t, "/api/subcategories", map[string]any{"name": name, "applies_to": appliesExpense}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/subcategories %q = %d, want 201", name, res.StatusCode)
	}
	return got
}

// Unlike Category, a fresh database seeds no Subcategories at all — there is
// no convenience list, because nothing about them is universal to every
// household the way "Alimentari" is.
func TestAFreshDatabaseHasNoSubcategories(t *testing.T) {
	a := newTestApp(t)
	if got := a.subcategories(t); len(got) != 0 {
		t.Errorf("fresh database listed %d Subcategories, want 0", len(got))
	}
}

func TestCreatingASubcategoryPutsItInTheList(t *testing.T) {
	a := newTestApp(t)

	var created subcategoryJSON
	res := a.post(t, "/api/subcategories", map[string]any{"name": "Caffè", "applies_to": appliesExpense}, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}
	if created.ID == 0 {
		t.Error("the created Subcategory came back without an id")
	}

	got := a.subcategory(t, "Caffè")
	if got.ID != created.ID || got.AppliesTo != appliesExpense || got.Hidden {
		t.Errorf("listed Subcategory = %+v, want the one that was created", got)
	}
}

func TestRenamingASubcategoryKeepsItsIdentity(t *testing.T) {
	a := newTestApp(t)
	before := a.createSubcategoryNamed(t, "Caffè")

	res := a.patch(t, subcategoryPath(before.ID), map[string]any{"name": "Bar"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	after := a.subcategory(t, "Bar")
	if after.ID != before.ID {
		t.Errorf("id = %d after the rename, want %d", after.ID, before.ID)
	}
}

func TestHidingASubcategoryLeavesItResolvable(t *testing.T) {
	a := newTestApp(t)
	before := a.createSubcategoryNamed(t, "Caffè")

	if res := a.patch(t, subcategoryPath(before.ID), map[string]any{"hidden": true}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	after := a.subcategory(t, "Caffè")
	if !after.Hidden || after.ID != before.ID {
		t.Errorf("after hiding = %+v, want hidden=true id=%d", after, before.ID)
	}
}

func TestDeletingASubcategoryRemovesIt(t *testing.T) {
	a := newTestApp(t)
	id := subcategoryPath(a.createSubcategoryNamed(t, "Caffè").ID)

	if res := a.delete(t, id); res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", res.StatusCode)
	}
	if res := a.delete(t, id); res.StatusCode != http.StatusNotFound {
		t.Errorf("deleting it twice = %d, want 404", res.StatusCode)
	}
}

func TestSubcategoryWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	id := subcategoryPath(a.createSubcategoryNamed(t, "Caffè").ID)

	cases := map[string]struct {
		method, path string
		body         any
		want         int
	}{
		"an empty name":               {"POST", "/api/subcategories", map[string]any{"name": "  ", "applies_to": appliesExpense}, http.StatusBadRequest},
		"a missing applies_to":        {"POST", "/api/subcategories", map[string]any{"name": "Asporto"}, http.StatusBadRequest},
		"an unknown applies_to":       {"POST", "/api/subcategories", map[string]any{"name": "Asporto", "applies_to": "elsewhere"}, http.StatusBadRequest},
		"a rename to nothing":         {"PATCH", id, map[string]any{"name": ""}, http.StatusBadRequest},
		"a patch of an unknown id":    {"PATCH", "/api/subcategories/9999", map[string]any{"name": "Asporto"}, http.StatusNotFound},
		"a patch of a non-numeric id": {"PATCH", "/api/subcategories/abc", map[string]any{"name": "Asporto"}, http.StatusNotFound},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res := a.do(t, c.method, c.path, c.body, nil)
			if res.StatusCode != c.want {
				t.Errorf("%s %s with %s = %d, want %d", c.method, c.path, name, res.StatusCode, c.want)
			}
		})
	}
}

// The whole point of a Subcategory: the same one pairs with different
// Categories on different Expenses, because nothing ties it to just one.
func TestASubcategoryPairsWithDifferentCategories(t *testing.T) {
	a := newTestApp(t)
	caffe := a.createSubcategoryNamed(t, "Caffè")
	alimentari := a.category(t, "Alimentari")
	svago := a.category(t, "Svago")

	e1 := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 300,
		"category_id": alimentari.ID, "subcategory_id": caffe.ID,
	})
	e2 := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-16", "amount_cents": 450,
		"category_id": svago.ID, "subcategory_id": caffe.ID,
	})

	if e1.SubcategoryID == nil || *e1.SubcategoryID != caffe.ID {
		t.Errorf("first expense subcategory_id = %v, want %d", e1.SubcategoryID, caffe.ID)
	}
	if e2.SubcategoryID == nil || *e2.SubcategoryID != caffe.ID {
		t.Errorf("second expense subcategory_id = %v, want %d", e2.SubcategoryID, caffe.ID)
	}
	if e1.CategoryID == e2.CategoryID {
		t.Fatalf("test setup mistake: both expenses landed in the same category %d", e1.CategoryID)
	}
}

// A Subcategory still referenced by an Expense refuses to be deleted, the
// same 409 a still-referenced Category answers with — the FK has no ON
// DELETE clause, and writeError turns the constraint failure into a 409.
func TestDeletingASubcategoryInUseIsRefused(t *testing.T) {
	a := newTestApp(t)
	caffe := a.createSubcategoryNamed(t, "Caffè")
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 300,
		"category_id": a.category(t, "Alimentari").ID, "subcategory_id": caffe.ID,
	})

	if res := a.delete(t, subcategoryPath(caffe.ID)); res.StatusCode != http.StatusConflict {
		t.Errorf("DELETE a Subcategory in use = %d, want 409", res.StatusCode)
	}

	unused := a.createSubcategoryNamed(t, "Asporto")
	if res := a.delete(t, subcategoryPath(unused.ID)); res.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE an unused Subcategory = %d, want 204", res.StatusCode)
	}
}
