package main

import (
	"net/http"
	"testing"
)

// The month report as the API hands it out: three numbers and the month they
// belong to. Money in is received Income only — an unpaid one counts toward
// nothing, which is what most of these tests are about.
type monthJSON struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
	NetCents     int64  `json:"net_cents"`
}

func (a *testApp) month(t *testing.T, month string) monthJSON {
	t.Helper()
	var got monthJSON
	if res := a.get(t, "/api/reports/month/"+month, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reports/month/%s = %d, want 200", month, res.StatusCode)
	}
	return got
}

// The whole ticket in one test: what came in, what went out, and the
// difference between them.
func TestAMonthReportsWhatCameInWhatWentOutAndTheDifference(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 4237, "category_id": alimentari.ID,
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-28", "amount_cents": 1250, "category_id": alimentari.ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 150000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-10",
	})

	got := a.month(t, "2026-03")
	want := monthJSON{Month: "2026-03", IncomeCents: 150000, ExpenseCents: 5487, NetCents: 144513}
	if got != want {
		t.Errorf("month = %+v, want %+v", got, want)
	}
}

// The assertion ticket 09 could not make, because there was no total to be
// excluded from: an unpaid Income is absent from money in, and present the
// moment it has a payment date. ADR-0003 calls the missing filter the single
// most likely bug in this codebase — this is the test that would catch it.
func TestAnUnpaidIncomeIsAbsentFromTheMonthUntilItIsPaid(t *testing.T) {
	a := newTestApp(t)
	invoice := a.addIncome(t, map[string]any{
		"amount_cents":      80000,
		"category_id":       a.freelance(t).ID,
		"invoice_sent_date": "2026-03-01",
	})

	if got := a.month(t, "2026-03"); got.IncomeCents != 0 || got.NetCents != 0 {
		t.Fatalf("unpaid: income = %d, net = %d, want 0 and 0 — money that has not arrived was counted",
			got.IncomeCents, got.NetCents)
	}

	if res := a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": "2026-03-20"}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("recording the payment = %d, want 200", res.StatusCode)
	}
	if got := a.month(t, "2026-03"); got.IncomeCents != 80000 || got.NetCents != 80000 {
		t.Errorf("paid: income = %d, net = %d, want 80000 and 80000", got.IncomeCents, got.NetCents)
	}

	// And back out again, because a payment typed onto the wrong Income is
	// taken back the same way it was recorded.
	a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": ""}, nil)
	if got := a.month(t, "2026-03"); got.IncomeCents != 0 {
		t.Errorf("payment taken back: income = %d, want 0", got.IncomeCents)
	}
}

// An Income belongs to the month its money arrived, not the month its invoice
// went out. invoice_sent_date answers "how long has this been sitting", which
// is a different question and a different ticket.
func TestAnIncomeBelongsToTheMonthItWasPaidNotTheMonthItWasInvoiced(t *testing.T) {
	a := newTestApp(t)
	a.addIncome(t, map[string]any{
		"amount_cents":      50000,
		"category_id":       a.freelance(t).ID,
		"invoice_sent_date": "2026-02-20",
		"payment_date":      "2026-03-05",
	})

	if got := a.month(t, "2026-02"); got.IncomeCents != 0 {
		t.Errorf("February income = %d, want 0 — it was counted in the month it was invoiced", got.IncomeCents)
	}
	if got := a.month(t, "2026-03"); got.IncomeCents != 50000 {
		t.Errorf("March income = %d, want 50000", got.IncomeCents)
	}
}

// The calendar month is the reporting unit, and its edges are the 1st and the
// last day: neither neighbour leaks in.
func TestAMonthCountsOnlyItsOwnEntries(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	freelance := a.freelance(t)

	for _, date := range []string{"2026-02-28", "2026-03-01", "2026-03-31", "2026-04-01"} {
		a.addExpense(t, map[string]any{
			"occurred_on": date, "amount_cents": 1000, "category_id": alimentari.ID,
		})
		a.addIncome(t, map[string]any{
			"amount_cents": 2000, "category_id": freelance.ID, "payment_date": date,
		})
	}

	got := a.month(t, "2026-03")
	want := monthJSON{Month: "2026-03", IncomeCents: 4000, ExpenseCents: 2000, NetCents: 2000}
	if got != want {
		t.Errorf("month = %+v, want %+v — a neighbouring month leaked in", got, want)
	}
}

// A month nobody spent anything in is zeros, not an error: every household
// starts there, and so does every month before the app was installed.
func TestAMonthWithNothingInItIsZeros(t *testing.T) {
	a := newTestApp(t)

	got := a.month(t, "2019-07")
	want := monthJSON{Month: "2019-07"}
	if got != want {
		t.Errorf("month = %+v, want %+v", got, want)
	}
}

// The difference is a difference, not a total: a month that spent more than it
// received says so.
func TestTheDifferenceGoesNegativeWhenMoreWentOutThanCameIn(t *testing.T) {
	a := newTestApp(t)
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-09", "amount_cents": 90000, "category_id": a.category(t, "Casa").ID,
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 25000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-09",
	})

	if got := a.month(t, "2026-03"); got.NetCents != -65000 {
		t.Errorf("net = %d, want -65000", got.NetCents)
	}
}

// Anything that is not a calendar month is refused, rather than summing a
// substring nothing matches and answering zeros as if the month were empty.
func TestAMonthThatIsNotAMonthIsRefused(t *testing.T) {
	a := newTestApp(t)

	for _, month := range []string{"2026-13", "2026-00", "2026-3", "2026", "marzo", "2026-03-15"} {
		t.Run(month, func(t *testing.T) {
			if res := a.get(t, "/api/reports/month/"+month, nil); res.StatusCode != http.StatusBadRequest {
				t.Errorf("GET /api/reports/month/%s = %d, want 400", month, res.StatusCode)
			}
		})
	}
}
