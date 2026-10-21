package main

import (
	"testing"
)

// ADR-0017: a refunded Expense still counts as spent in the month total — the
// money left — but is left out of Savings and the Budget/Estimate basis, and
// the month and year reports name the refunded slice separately rather than
// netting it away.
func TestARefundedExpenseCountsAsSpentButNotAsCost(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	plain := a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 10000, "category_id": alimentari.ID,
	})
	if plain.Refunded {
		t.Fatal("a plain Expense came back refunded")
	}
	refunded := a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-20", "amount_cents": 5000, "category_id": alimentari.ID,
		"refunded": true,
	})
	if !refunded.Refunded {
		t.Fatal("refunded did not round-trip through create")
	}

	// The month counts both: the money did leave the account.
	if got := a.month(t, "2026-01"); got.ExpenseCents != 15000 || got.RefundedCents != 5000 {
		t.Errorf("month = %+v, want expense 15000, refunded 5000", got)
	}
	// The year still totals both, and names the refunded slice.
	if got := a.year(t, "2026"); got.ExpenseCents != 15000 || got.RefundedCents != 5000 {
		t.Errorf("year = %+v, want expense 15000, refunded 5000", got)
	}
	// Savings is a planning figure: only the plain Expense was really spent.
	if got := a.savings(t); got.SavingsCents != -10000 {
		t.Errorf("savings = %d, want -10000 (the refunded Expense excluded)", got.SavingsCents)
	}

	// PATCH decodes onto the loaded row, so the flag is settable and clearable
	// like every other Expense field.
	var patched expenseJSON
	if res := a.patch(t, expensePath(plain.ID), map[string]any{"refunded": true}, &patched); res.StatusCode != 200 {
		t.Fatalf("PATCH refunded = %d, want 200", res.StatusCode)
	}
	if !patched.Refunded {
		t.Fatal("PATCH refunded = false, want true")
	}
	var cleared expenseJSON
	a.patch(t, expensePath(plain.ID), map[string]any{"refunded": false}, &cleared)
	if cleared.Refunded {
		t.Fatal("PATCH refunded = true, want false")
	}
}

// Marking the refund received closes it without moving any figure, and
// clearing the flag clears the receipt with it.
func TestReceivingARefundClosesItWithoutChangingFigures(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	e := a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-20", "amount_cents": 5000, "category_id": alimentari.ID,
		"refunded": true,
	})
	if e.RefundReceivedOn != nil {
		t.Fatalf("a new refunded Expense came back received on %s", *e.RefundReceivedOn)
	}

	var got expenseJSON
	if res := a.patch(t, expensePath(e.ID), map[string]any{"refund_received_on": "2026-03-01"}, &got); res.StatusCode != 200 {
		t.Fatalf("PATCH refund_received_on = %d, want 200", res.StatusCode)
	}
	if got.RefundReceivedOn == nil || *got.RefundReceivedOn != "2026-03-01" {
		t.Fatalf("refund_received_on = %v, want 2026-03-01", got.RefundReceivedOn)
	}
	// No longer "to be refunded", so the month's refunded line drops it.
	if m := a.month(t, "2026-01"); m.ExpenseCents != 5000 || m.RefundedCents != 0 {
		t.Errorf("month = %+v, want expense 5000, refunded 0", m)
	}
	if s := a.savings(t); s.SavingsCents != 0 {
		t.Errorf("savings = %d, want 0 (receiving a refund changes no figure)", s.SavingsCents)
	}

	if res := a.patch(t, expensePath(e.ID), map[string]any{"refund_received_on": "2026-3-1"}, nil); res.StatusCode != 400 {
		t.Errorf("PATCH bad date = %d, want 400", res.StatusCode)
	}

	a.patch(t, expensePath(e.ID), map[string]any{"refunded": false}, &got)
	if got.RefundReceivedOn != nil {
		t.Errorf("clearing refunded left refund_received_on = %s", *got.RefundReceivedOn)
	}
}
