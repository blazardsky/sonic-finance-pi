package main

import (
	"net/http"
	"slices"
	"testing"
)

// The two configured lists as the API hands them out: labels, in the order the
// picker should offer them.
type listsJSON struct {
	Payers                []string `json:"payers"`
	PaymentMethods        []string `json:"payment_methods"`
	SpendingIntentEnabled bool     `json:"spending_intent_enabled"`
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

// Ticket 01: the feature's single switch is off on a fresh database, round
// trips through PUT/GET, and setting it leaves every other setting on the
// same payload — the two lists included — untouched. Same pattern
// TestStartingBalanceRoundTripsThroughSettingsAndFoldsIntoSavings already
// exercises for a different field riding this payload.
func TestSpendingIntentEnabledDefaultsFalseAndRoundTripsWithoutDisturbingOtherSettings(t *testing.T) {
	a := newTestApp(t)
	before := a.lists(t)
	if before.SpendingIntentEnabled {
		t.Errorf("spending_intent_enabled = true on a fresh database, want false")
	}

	if res := a.put(t, settingsPath, map[string]any{"spending_intent_enabled": true}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	after := a.lists(t)
	if !after.SpendingIntentEnabled {
		t.Errorf("spending_intent_enabled = false after turning it on, want true")
	}
	if !slices.Equal(after.Payers, before.Payers) {
		t.Errorf("payers = %v, want the untouched %v", after.Payers, before.Payers)
	}
	if !slices.Equal(after.PaymentMethods, before.PaymentMethods) {
		t.Errorf("payment_methods = %v, want the untouched %v", after.PaymentMethods, before.PaymentMethods)
	}

	if res := a.put(t, settingsPath, map[string]any{"spending_intent_enabled": false}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}
	if got := a.lists(t); got.SpendingIntentEnabled {
		t.Errorf("spending_intent_enabled = true after turning it back off, want false")
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

// The bargain the schema makes: a Payer is label text on the Expense, not a
// reference into the list. Renaming one in Settings is therefore a change to
// what new entries can be, and to nothing already recorded — the November
// entry keeps reading "Nicco" whatever the list says in March.
//
// The same holds in both directions, because an Income carries the same field.
func TestRenamingAPayerLeavesRecordedEntriesReadingExactlyAsBefore(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	expense := a.addExpense(t, map[string]any{
		"occurred_on":    "2026-03-14",
		"amount_cents":   4210,
		"category_id":    alimentari.ID,
		"payer":          "Nicco",
		"payment_method": "Contanti",
	})
	income := a.addIncome(t, map[string]any{
		"amount_cents": 90000,
		"category_id":  a.freelance(t).ID,
		"payer":        "Nicco",
		"payment_date": "2026-03-10",
	})

	if res := a.put(t, settingsPath, map[string]any{
		"payers":          []string{"Niccolò", seedPayerBoth, seedPayerSomeoneElse},
		"payment_methods": []string{"Contante"},
	}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	if got := a.expense(t, expense.ID); got.Payer != "Nicco" || got.PaymentMethod != "Contanti" {
		t.Errorf("expense reads %q / %q, want the %q / %q it was saved with",
			got.Payer, got.PaymentMethod, "Nicco", "Contanti")
	}

	for _, got := range a.incomes(t) {
		if got.ID == income.ID && got.Payer != "Nicco" {
			t.Errorf("income reads %q, want the %q it was saved with", got.Payer, "Nicco")
		}
	}
}
