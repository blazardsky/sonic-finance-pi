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
	Color     string `json:"color"`
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
		"an empty name":                {"POST", "/api/subcategories", map[string]any{"name": "  ", "applies_to": appliesExpense}, http.StatusBadRequest},
		"a missing applies_to":         {"POST", "/api/subcategories", map[string]any{"name": "Asporto"}, http.StatusBadRequest},
		"an unknown applies_to":        {"POST", "/api/subcategories", map[string]any{"name": "Asporto", "applies_to": "elsewhere"}, http.StatusBadRequest},
		"an out-of-set color":          {"POST", "/api/subcategories", map[string]any{"name": "Asporto", "applies_to": appliesExpense, "color": "chartreuse"}, http.StatusBadRequest},
		"a rename to nothing":          {"PATCH", id, map[string]any{"name": ""}, http.StatusBadRequest},
		"an out-of-set color on patch": {"PATCH", id, map[string]any{"color": "chartreuse"}, http.StatusBadRequest},
		"a patch of an unknown id":     {"PATCH", "/api/subcategories/9999", map[string]any{"name": "Asporto"}, http.StatusNotFound},
		"a patch of a non-numeric id":  {"PATCH", "/api/subcategories/abc", map[string]any{"name": "Asporto"}, http.StatusNotFound},
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

// Subcategory never had a color before this session, but the create/patch
// rules are exactly Category's own (spec, ticket 04).
func TestSubcategoryColorDefaultsToBlueGrayWhenOmitted(t *testing.T) {
	a := newTestApp(t)

	created := a.createSubcategoryNamed(t, "Caffè")
	if created.Color != defaultColor {
		t.Errorf("color = %q, want %q", created.Color, defaultColor)
	}
}

func TestSubcategoryColorRoundTripsOnPatch(t *testing.T) {
	a := newTestApp(t)
	before := a.createSubcategoryNamed(t, "Caffè")

	res := a.patch(t, subcategoryPath(before.ID), map[string]any{"color": "aqua"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if got := a.subcategory(t, "Caffè").Color; got != "aqua" {
		t.Errorf("color = %q after the patch, want %q", got, "aqua")
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

// replace_with reassigns every referencing Expense and Recurring expense onto
// the replacement Subcategory, leaving each row's own Category untouched, and
// then removes the old Subcategory — mirroring TestReplaceWithReassignsEveryReferenceOnDelete
// in categories_test.go.
func TestDeletingASubcategoryWithReplaceWithReassignsEveryReference(t *testing.T) {
	a := newTestApp(t)
	caffe := a.createSubcategoryNamed(t, "Caffè")
	bar := a.createSubcategoryNamed(t, "Bar")
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")

	e := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 300,
		"category_id": alimentari.ID, "subcategory_id": caffe.ID,
	})
	r := a.addRecurring(t, a.rent(t, map[string]any{
		"category_id": casa.ID, "subcategory_id": caffe.ID,
	}))

	if res := a.delete(t, subcategoryPath(caffe.ID)+"?replace_with="+strconv.FormatInt(bar.ID, 10)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE ?replace_with = %d, want 204", res.StatusCode)
	}

	gotExpenses := a.expenses(t)
	var gotExpense expenseJSON
	for _, x := range gotExpenses {
		if x.ID == e.ID {
			gotExpense = x
		}
	}
	if gotExpense.SubcategoryID == nil || *gotExpense.SubcategoryID != bar.ID {
		t.Errorf("expense subcategory_id = %v, want %d", gotExpense.SubcategoryID, bar.ID)
	}
	if gotExpense.CategoryID != alimentari.ID {
		t.Errorf("expense category_id = %d, want untouched %d", gotExpense.CategoryID, alimentari.ID)
	}

	gotRecurrings := a.recurrings(t)
	var gotRecurring recurringJSON
	for _, x := range gotRecurrings {
		if x.ID == r.ID {
			gotRecurring = x
		}
	}
	if gotRecurring.SubcategoryID == nil || *gotRecurring.SubcategoryID != bar.ID {
		t.Errorf("recurring subcategory_id = %v, want %d", gotRecurring.SubcategoryID, bar.ID)
	}
	if gotRecurring.CategoryID != casa.ID {
		t.Errorf("recurring category_id = %d, want untouched %d", gotRecurring.CategoryID, casa.ID)
	}

	if res := a.delete(t, subcategoryPath(caffe.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("the old Subcategory is still there: DELETE again = %d, want 404", res.StatusCode)
	}
}

// clear=true nulls out subcategory_id on every referencing row without
// touching category_id — the "remove the tag instead" choice Category has no
// equivalent to.
func TestDeletingASubcategoryWithClearNullsBothColumnsWithoutTouchingCategory(t *testing.T) {
	a := newTestApp(t)
	caffe := a.createSubcategoryNamed(t, "Caffè")
	alimentari := a.category(t, "Alimentari")
	casa := a.category(t, "Casa")

	e := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 300,
		"category_id": alimentari.ID, "subcategory_id": caffe.ID,
	})
	r := a.addRecurring(t, a.rent(t, map[string]any{
		"category_id": casa.ID, "subcategory_id": caffe.ID,
	}))

	if res := a.delete(t, subcategoryPath(caffe.ID)+"?clear=true"); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE ?clear=true = %d, want 204", res.StatusCode)
	}

	gotExpenses := a.expenses(t)
	var gotExpense expenseJSON
	for _, x := range gotExpenses {
		if x.ID == e.ID {
			gotExpense = x
		}
	}
	if gotExpense.SubcategoryID != nil {
		t.Errorf("expense subcategory_id = %v, want nil", gotExpense.SubcategoryID)
	}
	if gotExpense.CategoryID != alimentari.ID {
		t.Errorf("expense category_id = %d, want untouched %d", gotExpense.CategoryID, alimentari.ID)
	}

	gotRecurrings := a.recurrings(t)
	var gotRecurring recurringJSON
	for _, x := range gotRecurrings {
		if x.ID == r.ID {
			gotRecurring = x
		}
	}
	if gotRecurring.SubcategoryID != nil {
		t.Errorf("recurring subcategory_id = %v, want nil", gotRecurring.SubcategoryID)
	}
	if gotRecurring.CategoryID != casa.ID {
		t.Errorf("recurring category_id = %d, want untouched %d", gotRecurring.CategoryID, casa.ID)
	}

	if res := a.delete(t, subcategoryPath(caffe.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("the old Subcategory is still there: DELETE again = %d, want 404", res.StatusCode)
	}
}

// replace_with pointing at a nonexistent or wrong-side Subcategory is
// refused with 400 — the same rule an Expense's own subcategory_id answers to.
func TestReplaceWithAWrongSideOrNonexistentSubcategoryIsRefused(t *testing.T) {
	a := newTestApp(t)
	caffe := a.createSubcategoryNamed(t, "Caffè") // expense-side
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 300,
		"category_id": a.category(t, "Alimentari").ID, "subcategory_id": caffe.ID,
	})

	var incomeSide subcategoryJSON
	res := a.post(t, "/api/subcategories", map[string]any{"name": "Bonus", "applies_to": appliesIncome}, &incomeSide)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST an income-side Subcategory = %d, want 201", res.StatusCode)
	}

	if res := a.delete(t, subcategoryPath(caffe.ID)+"?replace_with="+strconv.FormatInt(incomeSide.ID, 10)); res.StatusCode != http.StatusBadRequest {
		t.Errorf("DELETE ?replace_with=<wrong side> = %d, want 400", res.StatusCode)
	}
	if res := a.delete(t, subcategoryPath(caffe.ID)+"?replace_with=9999"); res.StatusCode != http.StatusBadRequest {
		t.Errorf("DELETE ?replace_with=<nonexistent> = %d, want 400", res.StatusCode)
	}

	// Still there and still in use, untouched by the refused attempts.
	if res := a.delete(t, subcategoryPath(caffe.ID)); res.StatusCode != http.StatusConflict {
		t.Errorf("bare DELETE after refused replace_with attempts = %d, want 409", res.StatusCode)
	}
}
