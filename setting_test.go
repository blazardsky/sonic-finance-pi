package main

import (
	"net/http"
	"slices"
	"testing"
)

// The two configured lists as the API hands them out: labels, in the order the
// picker should offer them.
type listsJSON struct {
	Payers         []string `json:"payers"`
	PaymentMethods []string `json:"payment_methods"`
}

func (a *testApp) lists(t *testing.T) listsJSON {
	t.Helper()
	var got listsJSON
	if res := a.get(t, settingsPath, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", settingsPath, res.StatusCode)
	}
	return got
}

// A fresh database comes with lists worth using, so that the first Expense of
// the first evening has a Payer to choose. "Both" and "Someone else" are the
// two the spec names: a shared bill and a third-party payment must not need a
// fake person invented for them.
func TestAFreshDatabaseSeedsBothLists(t *testing.T) {
	a := newTestApp(t)
	got := a.lists(t)

	for _, want := range []string{seedPayerBoth, seedPayerSomeoneElse} {
		if !slices.Contains(got.Payers, want) {
			t.Errorf("payers = %v, want it to include %q", got.Payers, want)
		}
	}
	if len(got.Payers) < 3 {
		t.Errorf("payers = %v, want a household member alongside the two shared options", got.Payers)
	}
	if len(got.PaymentMethods) != 3 {
		t.Errorf("payment_methods = %v, want cash, credit card and debit card", got.PaymentMethods)
	}
}

// A body carrying one list leaves the other alone, so a screen saving the list
// it owns cannot blank the one it does not.
func TestSavingOneListLeavesTheOtherAlone(t *testing.T) {
	a := newTestApp(t)
	before := a.lists(t)

	if res := a.put(t, settingsPath, map[string]any{"payers": []string{"Nicco", "Sara"}}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	after := a.lists(t)
	if len(after.Payers) != 2 || after.Payers[0] != "Nicco" || after.Payers[1] != "Sara" {
		t.Errorf("payers = %v, want [Nicco Sara]", after.Payers)
	}
	if len(after.PaymentMethods) != len(before.PaymentMethods) {
		t.Errorf("payment_methods = %v, want the untouched %v", after.PaymentMethods, before.PaymentMethods)
	}
}

// A list nothing can be chosen from is a picker with no options, which is the
// one edit worth refusing: the household would have no way back to a Payer.
func TestAnEmptyListIsRefused(t *testing.T) {
	a := newTestApp(t)
	before := a.lists(t)

	for name, body := range map[string]any{
		"an empty list":        map[string]any{"payers": []string{}},
		"a list of blanks":     map[string]any{"payers": []string{"", "  "}},
		"an empty method list": map[string]any{"payment_methods": []string{}},
	} {
		t.Run(name, func(t *testing.T) {
			if res := a.put(t, settingsPath, body, nil); res.StatusCode != http.StatusBadRequest {
				t.Errorf("PUT %s with %s = %d, want 400", settingsPath, name, res.StatusCode)
			}
		})
	}

	if after := a.lists(t); len(after.Payers) != len(before.Payers) {
		t.Errorf("payers = %v, want the refused edits to have changed nothing (%v)", after.Payers, before.Payers)
	}
}
