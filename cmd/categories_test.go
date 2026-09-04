package main

import (
	"net/http"
	"strconv"
	"testing"
)

// A Category as the API hands it out. `base` is the protection flag: it is
// true for the Categories the code resolves by identity, and those are the
// ones rename and delete refuse.
type categoryJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AppliesTo string `json:"applies_to"`
	Hidden    bool   `json:"hidden"`
	Base      bool   `json:"base"`
	Gift      bool   `json:"gift"`
}

// categoryPath addresses one Category the way the API does.
func categoryPath(id int64) string {
	return "/api/categories/" + strconv.FormatInt(id, 10)
}

func (a *testApp) categories(t *testing.T) []categoryJSON {
	t.Helper()
	var got []categoryJSON
	if res := a.get(t, "/api/categories", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/categories = %d, want 200", res.StatusCode)
	}
	return got
}

// category finds a Category by name, or fails the test. Tests address rows by
// name rather than by id, because the seeded ids are not part of the API.
func (a *testApp) category(t *testing.T, name string) categoryJSON {
	t.Helper()
	for _, c := range a.categories(t) {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no Category named %q in %v", name, a.categories(t))
	return categoryJSON{}
}

func TestAFreshDatabaseSeedsTheTwoBaseCategories(t *testing.T) {
	a := newTestApp(t)

	freelance := a.category(t, seedFreelanceName)
	if !freelance.Base {
		t.Errorf("%s.base = false, want true — the tax summary resolves it", seedFreelanceName)
	}
	if freelance.AppliesTo != appliesIncome {
		t.Errorf("%s.applies_to = %q, want %q", seedFreelanceName, freelance.AppliesTo, appliesIncome)
	}

	taxes := a.category(t, seedTaxesName)
	if !taxes.Base {
		t.Errorf("%s.base = false, want true", seedTaxesName)
	}
	if taxes.AppliesTo != appliesExpense {
		t.Errorf("%s.applies_to = %q, want %q", seedTaxesName, taxes.AppliesTo, appliesExpense)
	}
}

// The convenience Categories exist so the household is not typing a list on
// first use, but nothing in the code resolves them, so they are not protected.
func TestSeededConvenienceCategoriesAreNotProtected(t *testing.T) {
	a := newTestApp(t)

	var protected []string
	for _, c := range a.categories(t) {
		if c.Base && c.Name != seedFreelanceName && c.Name != seedTaxesName && c.Name != seedInvestmentiName && c.Name != seedRegaliName {
			protected = append(protected, c.Name)
		}
	}
	if protected != nil {
		t.Errorf("these seeded Categories are protected but nothing resolves them: %v", protected)
	}
}

// The dormant "Investimenti" category is promoted in place, not replaced: a
// fresh database has exactly one row with code = investments, it is the same
// row that was seeded named Investimenti, and applies_to now admits both an
// Expense and an Income (ticket 01's spec).
func TestInvestimentiIsPromotedInPlaceNotDuplicated(t *testing.T) {
	a := newTestApp(t)

	investments := a.category(t, seedInvestmentiName)
	if !investments.Base {
		t.Errorf("%s.base = false, want true — it is now a protected Base category", seedInvestmentiName)
	}
	if investments.AppliesTo != appliesBoth {
		t.Errorf("%s.applies_to = %q, want %q", seedInvestmentiName, investments.AppliesTo, appliesBoth)
	}

	var named, coded int
	for _, c := range a.categories(t) {
		if c.Name == seedInvestmentiName {
			named++
		}
		if c.Base && c.Name == seedInvestmentiName {
			coded++
		}
	}
	if named != 1 {
		t.Errorf("%d Categories named %q, want exactly 1 — the promotion must not insert a second row",
			named, seedInvestmentiName)
	}
	if coded != 1 {
		t.Errorf("%d protected Categories named %q, want exactly 1", coded, seedInvestmentiName)
	}
}

// The dormant "Regali" category is promoted in place, not replaced: a fresh
// database has exactly one row with code = gift, it is the same row that was
// seeded named Regali, and applies_to now admits both an Expense and an
// Income (schema-foundation ticket's spec).
func TestRegaliIsPromotedInPlaceNotDuplicated(t *testing.T) {
	a := newTestApp(t)

	gift := a.category(t, seedRegaliName)
	if !gift.Base {
		t.Errorf("%s.base = false, want true — it is now a protected Base category", seedRegaliName)
	}
	if gift.AppliesTo != appliesBoth {
		t.Errorf("%s.applies_to = %q, want %q", seedRegaliName, gift.AppliesTo, appliesBoth)
	}

	var named, coded int
	for _, c := range a.categories(t) {
		if c.Name == seedRegaliName {
			named++
		}
		if c.Base && c.Name == seedRegaliName {
			coded++
		}
	}
	if named != 1 {
		t.Errorf("%d Categories named %q, want exactly 1 — the promotion must not insert a second row",
			named, seedRegaliName)
	}
	if coded != 1 {
		t.Errorf("%d protected Categories named %q, want exactly 1", coded, seedRegaliName)
	}
}

// The spoiler blur resolves the Gift Category by identity — from the
// frontend, since it decides on its own whether an amount blurs — so it is
// the one field, unlike `base`, published for the household's list to act on
// rather than merely grey buttons for. It is true for Regali and nothing
// else, including the other Base categories.
func TestGiftIsExposedForTheGiftCategoryOnly(t *testing.T) {
	a := newTestApp(t)

	if gift := a.category(t, seedRegaliName); !gift.Gift {
		t.Errorf("%s.gift = false, want true", seedRegaliName)
	}
	for _, name := range []string{seedFreelanceName, seedTaxesName, seedInvestmentiName, "Alimentari"} {
		if c := a.category(t, name); c.Gift {
			t.Errorf("%s.gift = true, want false", name)
		}
	}
}

// Both pickers read one list, so every Category has to say which one it
// belongs in: an expense picker must never offer "Freelance".
func TestEverySeededCategoryDeclaresWhereItApplies(t *testing.T) {
	a := newTestApp(t)

	seen := map[string]bool{}
	for _, c := range a.categories(t) {
		switch c.AppliesTo {
		case appliesExpense, appliesIncome, appliesBoth:
			seen[c.AppliesTo] = true
		default:
			t.Errorf("%s.applies_to = %q, want one of expense/income/both", c.Name, c.AppliesTo)
		}
	}
	if !seen[appliesExpense] || !seen[appliesIncome] {
		t.Errorf("the seed has to cover both pickers, got %v", seen)
	}
}

func TestCreatingACategoryPutsItInTheList(t *testing.T) {
	a := newTestApp(t)

	var created categoryJSON
	res := a.post(t, "/api/categories", map[string]any{"name": "Bici", "applies_to": appliesExpense}, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}
	if created.ID == 0 {
		t.Error("the created Category came back without an id")
	}
	if created.Base {
		t.Error("a Category the household created came back protected")
	}

	got := a.category(t, "Bici")
	if got.ID != created.ID || got.AppliesTo != appliesExpense || got.Hidden {
		t.Errorf("listed Category = %+v, want the one that was created", got)
	}
}

// Entries reference a Category rather than copying its name, so a rename is
// the whole of "fix every past entry at once". Until Expenses exist, the
// observable half is that the row itself changes under the same id.
func TestRenamingACategoryKeepsItsIdentity(t *testing.T) {
	a := newTestApp(t)
	before := a.category(t, "Alimentari")

	res := a.patch(t, categoryPath(before.ID), map[string]any{"name": "Cibo"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}

	after := a.category(t, "Cibo")
	if after.ID != before.ID {
		t.Errorf("id = %d after the rename, want %d — a rename must not be a new row", after.ID, before.ID)
	}
	for _, c := range a.categories(t) {
		if c.Name == "Alimentari" {
			t.Error("the old name is still in the list")
		}
	}
}

func TestHidingACategoryLeavesItResolvable(t *testing.T) {
	a := newTestApp(t)
	before := a.category(t, "Alimentari")

	if res := a.patch(t, categoryPath(before.ID), map[string]any{"hidden": true}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}

	// Still there, still the same row: pickers filter on hidden, reports do not.
	after := a.category(t, "Alimentari")
	if !after.Hidden {
		t.Error("hidden = false after hiding it")
	}
	if after.ID != before.ID {
		t.Errorf("id = %d after hiding, want %d", after.ID, before.ID)
	}

	// And unhiding is the same switch the other way.
	a.patch(t, categoryPath(before.ID), map[string]any{"hidden": false}, nil)
	if a.category(t, "Alimentari").Hidden {
		t.Error("hidden = true after unhiding it")
	}
}

// The tax summary resolves Freelance and Taxes by identity, Budget/Target/
// Estimate resolve Investments the same way, and the spoiler blur resolves
// Gift the same way again — so tidying the list must not be able to break
// any of them. Hiding is still allowed: quitting freelancing should not leave
// a dead option in the picker forever. See ADR-0008.
func TestBaseCategoriesRefuseRenameAndDeleteButAllowHiding(t *testing.T) {
	a := newTestApp(t)

	for _, name := range []string{seedFreelanceName, seedTaxesName, seedInvestmentiName, seedRegaliName} {
		t.Run(name, func(t *testing.T) {
			before := a.category(t, name)
			id := categoryPath(before.ID)

			if res := a.patch(t, id, map[string]any{"name": name + " x"}, nil); res.StatusCode != http.StatusConflict {
				t.Errorf("renaming = %d, want 409", res.StatusCode)
			}
			// The attempted value must actually differ from what is already
			// stored — Investments is already "both", unlike Freelance and
			// Taxes — or "did it land" is not a question this can answer.
			attempt := appliesBoth
			if before.AppliesTo == appliesBoth {
				attempt = appliesExpense
			}
			if res := a.patch(t, id, map[string]any{"applies_to": attempt}, nil); res.StatusCode != http.StatusConflict {
				t.Errorf("changing applies_to = %d, want 409", res.StatusCode)
			}
			if res := a.delete(t, id); res.StatusCode != http.StatusConflict {
				t.Errorf("deleting = %d, want 409", res.StatusCode)
			}
			if res := a.patch(t, id, map[string]any{"hidden": true}, nil); res.StatusCode != http.StatusOK {
				t.Errorf("hiding = %d, want 200", res.StatusCode)
			}

			after := a.category(t, name)
			if after.Name != name || after.AppliesTo != before.AppliesTo {
				t.Errorf("the refused edits landed anyway: %+v", after)
			}
			if !after.Hidden {
				t.Error("hiding a base Category did not take")
			}
		})
	}
}

func TestDeletingACategoryRemovesIt(t *testing.T) {
	a := newTestApp(t)
	id := categoryPath(a.category(t, "Alimentari").ID)

	if res := a.delete(t, id); res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", res.StatusCode)
	}
	for _, c := range a.categories(t) {
		if c.Name == "Alimentari" {
			t.Error("the deleted Category is still in the list")
		}
	}
	if res := a.delete(t, id); res.StatusCode != http.StatusNotFound {
		t.Errorf("deleting it twice = %d, want 404", res.StatusCode)
	}
}

func TestCategoryWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	id := categoryPath(a.category(t, "Alimentari").ID)

	cases := map[string]struct {
		method, path string
		body         any
		want         int
	}{
		"an empty name":               {"POST", "/api/categories", map[string]any{"name": "  ", "applies_to": appliesExpense}, http.StatusBadRequest},
		"a missing applies_to":        {"POST", "/api/categories", map[string]any{"name": "Bici"}, http.StatusBadRequest},
		"an unknown applies_to":       {"POST", "/api/categories", map[string]any{"name": "Bici", "applies_to": "elsewhere"}, http.StatusBadRequest},
		"a rename to nothing":         {"PATCH", id, map[string]any{"name": ""}, http.StatusBadRequest},
		"a patch of an unknown id":    {"PATCH", "/api/categories/9999", map[string]any{"name": "Bici"}, http.StatusNotFound},
		"a patch of a non-numeric id": {"PATCH", "/api/categories/abc", map[string]any{"name": "Bici"}, http.StatusNotFound},
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

// Names are trimmed on the way in, so "Bici " and "Bici" are not two rows that
// look identical in the picker.
func TestCategoryNamesAreTrimmed(t *testing.T) {
	a := newTestApp(t)

	var created categoryJSON
	a.post(t, "/api/categories", map[string]any{"name": "  Bici  ", "applies_to": appliesExpense}, &created)
	if created.Name != "Bici" {
		t.Errorf("name = %q, want %q", created.Name, "Bici")
	}
}

// A Category with Expenses in it refuses to be deleted, and says so as a 409
// rather than a 500: something is still pointing at the row, which is a
// conflict the household can act on — hide it instead — not a server fault.
// Ticket 04 left this to whoever added the foreign key; the answer lives in
// writeError, so a Client with Incomes behind it gets it for free.
func TestDeletingACategoryInUseIsRefused(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	a.addExpense(t, map[string]any{
		"occurred_on":  "2026-03-15",
		"amount_cents": 4237,
		"category_id":  alimentari.ID,
	})

	if res := a.delete(t, categoryPath(alimentari.ID)); res.StatusCode != http.StatusConflict {
		t.Errorf("DELETE a Category with an Expense in it = %d, want 409", res.StatusCode)
	}
	// And it is still there, so the Expense still resolves through it.
	if got := a.category(t, "Alimentari"); got.ID != alimentari.ID {
		t.Errorf("the refused delete moved the Category to id %d, want %d", got.ID, alimentari.ID)
	}

	// An unused Category still deletes: the guard is about what points at the
	// row, not about Categories in general.
	bici := a.createCategoryNamed(t, "Bici")
	if res := a.delete(t, categoryPath(bici.ID)); res.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE an unused Category = %d, want 204", res.StatusCode)
	}
}

// createCategoryNamed adds an expense-side Category and returns it.
func (a *testApp) createCategoryNamed(t *testing.T, name string) categoryJSON {
	t.Helper()
	var got categoryJSON
	res := a.post(t, "/api/categories", map[string]any{"name": name, "applies_to": appliesExpense}, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/categories %q = %d, want 201", name, res.StatusCode)
	}
	return got
}
