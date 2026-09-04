package main

import (
	"net/http"
	"testing"
	"time"
)

// The two lists as the API hands them out. Outstanding is money genuinely
// owed; not_yet_invoiced is the nudge — a Client who is usually billed by now
// and has not been. Neither is a record: both are views over Incomes.
type pendingJSON struct {
	Outstanding []struct {
		ID              int64  `json:"id"`
		AmountCents     int64  `json:"amount_cents"`
		Client          string `json:"client"`
		Category        string `json:"category"`
		WaitingSince    string `json:"waiting_since"`
		DaysWaiting     int    `json:"days_waiting"`
		InvoiceSentDate string `json:"invoice_sent_date"`
	} `json:"outstanding"`
	NotYetInvoiced []struct {
		ClientID int64  `json:"client_id"`
		Client   string `json:"client"`
	} `json:"not_yet_invoiced"`
}

func (a *testApp) pending(t *testing.T) pendingJSON {
	t.Helper()
	var got pendingJSON
	if res := a.get(t, "/api/pending-payments", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/pending-payments = %d, want 200", res.StatusCode)
	}
	return got
}

// notYetInvoiced is the second list as names, which is all any assertion about
// it needs: whether a Client is being chased this month.
func (a *testApp) notYetInvoiced(t *testing.T) []string {
	t.Helper()
	var names []string
	for _, c := range a.pending(t).NotYetInvoiced {
		names = append(names, c.Client)
	}
	return names
}

// The first list, whole: every unpaid Income, oldest first, with the number of
// days it has been waiting. A paid one is not owed and is not here.
func TestUnpaidIncomesAreListedOldestFirstWithTheDaysTheyHaveWaited(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	rossi := a.createClient(t, "Studio Rossi")

	a.addIncome(t, map[string]any{
		"amount_cents": 50000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-02-20",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 120000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-01-05",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 30000, "category_id": freelance.ID,
		"invoice_sent_date": "2026-03-01",
	})
	// Paid, and therefore owed by nobody.
	a.addIncome(t, map[string]any{
		"amount_cents": 90000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-01-10", "payment_date": "2026-02-01",
	})

	got := a.pending(t).Outstanding
	if len(got) != 3 {
		t.Fatalf("outstanding = %d entries, want 3 — a paid Income is not pending", len(got))
	}
	// Oldest first, and the days counted from the fixed clock: 2026-03-15.
	wantDays := []int{69, 23, 14}
	wantSince := []string{"2026-01-05", "2026-02-20", "2026-03-01"}
	for i, e := range got {
		if e.WaitingSince != wantSince[i] || e.DaysWaiting != wantDays[i] {
			t.Errorf("entry %d: waiting since %s for %d days, want %s for %d",
				i, e.WaitingSince, e.DaysWaiting, wantSince[i], wantDays[i])
		}
		if e.InvoiceSentDate != wantSince[i] {
			t.Errorf("entry %d: invoice_sent_date = %q, want %s", i, e.InvoiceSentDate, wantSince[i])
		}
	}
	if got[0].AmountCents != 120000 || got[0].Client != "Studio Rossi" || got[0].Category != seedFreelanceName {
		t.Errorf("oldest = %+v, want €1200,00 from Studio Rossi under %s", got[0], seedFreelanceName)
	}
	// An Income names nobody when nobody sent it an invoice.
	if got[2].Client != "" {
		t.Errorf("client of a clientless Income = %q, want empty", got[2].Client)
	}
}

// Recording the payment is what takes an Income off the list — the same PATCH
// that moves it into the month's totals.
func TestRecordingThePaymentTakesAnIncomeOffTheOutstandingList(t *testing.T) {
	a := newTestApp(t)
	invoice := a.addIncome(t, map[string]any{
		"amount_cents": 80000, "category_id": a.freelance(t).ID,
		"invoice_sent_date": "2026-03-01",
	})

	if got := a.pending(t).Outstanding; len(got) != 1 {
		t.Fatalf("before payment: outstanding = %d, want 1", len(got))
	}
	if res := a.patch(t, incomePath(invoice.ID), map[string]any{"payment_date": "2026-03-12"}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("recording the payment = %d, want 200", res.StatusCode)
	}
	if got := a.pending(t).Outstanding; len(got) != 0 {
		t.Errorf("after payment: outstanding = %d, want 0", len(got))
	}
}

// An Income with no invoice date is still waiting for money, and the day it
// was typed is the only date it has to have been waiting since.
func TestAnIncomeWithNoInvoiceDateWaitsFromTheDayItWasTyped(t *testing.T) {
	a := newTestApp(t)
	a.addIncome(t, map[string]any{"amount_cents": 40000, "category_id": a.freelance(t).ID})

	got := a.pending(t).Outstanding
	if len(got) != 1 {
		t.Fatalf("outstanding = %d, want 1", len(got))
	}
	if got[0].WaitingSince != "2026-03-15" || got[0].DaysWaiting != 0 {
		t.Errorf("waiting since %s for %d days, want 2026-03-15 for 0",
			got[0].WaitingSince, got[0].DaysWaiting)
	}
	if got[0].InvoiceSentDate != "" {
		t.Errorf("invoice_sent_date = %q, want empty — no invoice was sent", got[0].InvoiceSentDate)
	}
}

// The nudge, and the whole of it: a Client billed inside the window and not
// this month is chased, and falls in and out as the months pass.
func TestAClientFallsInAndOutOfTheNudgeAsMonthsPass(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	rossi := a.createClient(t, "Studio Rossi")

	// Billed last month, nothing this month: the invoice that was forgotten.
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-02-10", "payment_date": "2026-02-28",
	})
	if got := a.notYetInvoiced(t); len(got) != 1 || got[0] != "Studio Rossi" {
		t.Fatalf("March: nudged = %v, want [Studio Rossi]", got)
	}

	// Invoiced this month: nothing to chase, paid or not.
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-03-05",
	})
	if got := a.notYetInvoiced(t); len(got) != 0 {
		t.Fatalf("March, after invoicing: nudged = %v, want none", got)
	}

	// April: March is inside the window again, and April has nothing in it.
	a.setNow(t, time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC))
	if got := a.notYetInvoiced(t); len(got) != 1 || got[0] != "Studio Rossi" {
		t.Fatalf("April: nudged = %v, want [Studio Rossi]", got)
	}

	// July: the last invoice is four months back, outside the three-month
	// window. A Client no longer billed is not a forgotten invoice.
	a.setNow(t, time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC))
	if got := a.notYetInvoiced(t); len(got) != 0 {
		t.Fatalf("July: nudged = %v, want none — the window has moved past", got)
	}

	// June is the last month that still sees March, which is the window's edge.
	a.setNow(t, time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC))
	if got := a.notYetInvoiced(t); len(got) != 1 {
		t.Fatalf("June: nudged = %v, want [Studio Rossi] — March is the third month back", got)
	}
}

// The nudge is about invoices, so only freelance Income counts as having been
// billed: a relative who sent a gift in February is not overdue an invoice.
func TestOnlyFreelanceIncomeCountsAsHavingBeenBilled(t *testing.T) {
	a := newTestApp(t)
	mum := a.createClient(t, "Mamma")
	a.addIncome(t, map[string]any{
		"amount_cents": 20000, "category_id": a.category(t, "Regali").ID, "client_id": mum.ID,
		"invoice_sent_date": "2026-02-10", "payment_date": "2026-02-10",
	})

	if got := a.notYetInvoiced(t); len(got) != 0 {
		t.Errorf("nudged = %v, want none — a gift is not a missing invoice", got)
	}
}

// Hiding is how a Client is retired, and a retired Client is not owed an
// invoice: the app stops asking about the one the household stopped billing.
func TestAHiddenClientIsNotChased(t *testing.T) {
	a := newTestApp(t)
	rossi := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": a.freelance(t).ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-02-10", "payment_date": "2026-02-20",
	})
	if got := a.notYetInvoiced(t); len(got) != 1 {
		t.Fatalf("nudged = %v, want [Studio Rossi] before hiding", got)
	}

	if res := a.patch(t, clientPath(rossi.ID), map[string]any{"hidden": true}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("hiding the client = %d, want 200", res.StatusCode)
	}
	if got := a.notYetInvoiced(t); len(got) != 0 {
		t.Errorf("nudged = %v, want none — a hidden Client is retired", got)
	}
}

// An unpaid Income from a Client billed inside the window is still an invoice
// that was sent, so the household is not told to send it again — it is on the
// first list, waiting, which is a different thing to chase.
func TestAnUnpaidInvoiceThisMonthIsNotAlsoAMissingOne(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	rossi := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-02-10", "payment_date": "2026-02-20",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": freelance.ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-03-02",
	})

	got := a.pending(t)
	if len(got.Outstanding) != 1 {
		t.Errorf("outstanding = %d, want 1 — March's invoice is unpaid", len(got.Outstanding))
	}
	if len(got.NotYetInvoiced) != 0 {
		t.Errorf("nudged = %+v, want none — March's invoice was sent", got.NotYetInvoiced)
	}
}

// Nothing pending is the state a household starts in, and both lists have to
// marshal as [] rather than null: the screen maps over them.
func TestPendingPaymentsAreTwoEmptyListsWhenNothingIsPending(t *testing.T) {
	a := newTestApp(t)
	got := a.pending(t)
	if got.Outstanding == nil || got.NotYetInvoiced == nil {
		t.Errorf("pending = %+v, want two empty lists rather than null", got)
	}
}

// The days waiting are calendar days between two dates, and a date carries no
// timezone. A Pi set to Italian time counted one day fewer than a Pi on UTC
// until the two sides of the subtraction were anchored the same way — this is
// the test that would have caught it before the real binary did.
func TestDaysWaitingDoNotDependOnTheServersTimezone(t *testing.T) {
	a := newTestApp(t)
	a.addIncome(t, map[string]any{
		"amount_cents": 80000, "category_id": a.freelance(t).ID,
		"invoice_sent_date": "2026-08-01",
	})

	rome := time.FixedZone("CEST", 2*60*60)
	a.setNow(t, time.Date(2026, 9, 3, 7, 39, 0, 0, rome))

	got := a.pending(t).Outstanding
	if len(got) != 1 {
		t.Fatalf("outstanding = %d, want 1", len(got))
	}
	if got[0].DaysWaiting != 33 {
		t.Errorf("days waiting = %d, want 33 — 1 August to 3 September", got[0].DaysWaiting)
	}
}

// A gift is not an invoice in either direction. One arriving inside the window
// must not start a nudge — and one arriving this month must not clear it: a
// relative sending money is no evidence that a Client was billed. This is the
// ticket's "limited to freelance-category income" read as one rule covering
// both halves of the list rather than only its history.
func TestAGiftThisMonthDoesNotClearTheNudge(t *testing.T) {
	a := newTestApp(t)
	rossi := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": a.freelance(t).ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-02-10", "payment_date": "2026-02-20",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 5000, "category_id": a.category(t, "Regali").ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-03-04", "payment_date": "2026-03-04",
	})

	if got := a.notYetInvoiced(t); len(got) != 1 {
		t.Errorf("not yet invoiced = %v, want [Studio Rossi] — a gift is not an invoice", got)
	}
}

// A clock the Pi cannot believe is not a refusal to answer: what is unpaid is
// unpaid whatever the clock says. What it costs is the arithmetic around it —
// the window contains nothing, so the nudge is empty, and the days floor at
// zero rather than counting backwards. clock_ok is what says why.
func TestAnUnbelievableClockStillAnswersWhatIsOwed(t *testing.T) {
	a := newTestApp(t)
	rossi := a.createClient(t, "Studio Rossi")
	a.addIncome(t, map[string]any{
		"amount_cents": 100000, "category_id": a.freelance(t).ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-02-10", "payment_date": "2026-02-20",
	})
	a.addIncome(t, map[string]any{
		"amount_cents": 80000, "category_id": a.freelance(t).ID, "client_id": rossi.ID,
		"invoice_sent_date": "2026-03-01",
	})

	// What a Zero W reads before NTP answers.
	a.setNow(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
	if a.clockOK(t) {
		t.Fatal("clock_ok = true at 1970 — the guard the screens warn on is not tripping")
	}

	got := a.pending(t)
	if len(got.Outstanding) != 1 {
		t.Fatalf("outstanding = %d, want 1 — an unpaid Income is unpaid whatever the clock says",
			len(got.Outstanding))
	}
	if got.Outstanding[0].DaysWaiting != 0 {
		t.Errorf("days waiting = %d, want 0 — nothing waits a negative number of days",
			got.Outstanding[0].DaysWaiting)
	}
	if len(got.NotYetInvoiced) != 0 {
		t.Errorf("not yet invoiced = %+v, want none — 1970's window holds no Income",
			got.NotYetInvoiced)
	}
}
