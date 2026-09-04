package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// A Contract as the API hands it out. The three read-time figures travel
// alongside the stored fields, per ADR-0012 — nothing here is ever fetched
// separately from the Contract itself.
type contractJSON struct {
	ID         int64  `json:"id"`
	ClientID   int64  `json:"client_id"`
	StartMonth string `json:"start_month"`
	EndMonth   string `json:"end_month"`
	TotalCents int64  `json:"total_cents"`

	ExpectedSoFarCents int64 `json:"expected_so_far_cents"`
	ReceivedCents      int64 `json:"received_cents"`
	AccountedCents     int64 `json:"accounted_cents"`
	InvoiceTargetCents int64 `json:"invoice_target_this_month_cents"`
	Overdue            bool  `json:"overdue"`
}

// contractsPath and contractPath address a Client's Contracts the way the
// API does.
func contractsPath(clientID int64) string {
	return fmt.Sprintf("/api/clients/%d/contracts", clientID)
}

func (a *testApp) contracts(t *testing.T, clientID int64) []contractJSON {
	t.Helper()
	var got []contractJSON
	res := a.get(t, contractsPath(clientID), &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", contractsPath(clientID), res.StatusCode)
	}
	return got
}

// createContract defines one and returns it as the API answered, failing the
// test if the save was refused — for the tests whose point is something else
// entirely.
func (a *testApp) createContract(t *testing.T, clientID int64, body map[string]any) contractJSON {
	t.Helper()
	var got contractJSON
	res := a.post(t, contractsPath(clientID), body, &got)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST %s %v = %d, want 201", contractsPath(clientID), body, res.StatusCode)
	}
	return got
}

func yearContract(total int64) map[string]any {
	return map[string]any{
		"start_month": "2026-01",
		"end_month":   "2026-12",
		"total_cents": total,
	}
}

func TestCreatingAContractPutsItInTheClientsList(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Studio Rossi")

	created := a.createContract(t, c.ID, yearContract(120000))
	if created.ID == 0 {
		t.Error("the created Contract has no id")
	}
	if created.ClientID != c.ID {
		t.Errorf("client_id = %d, want %d", created.ClientID, c.ID)
	}

	got := a.contracts(t, c.ID)
	if len(got) != 1 || got[0] != created {
		t.Errorf("the list is %v, want just %v", got, created)
	}
}

func TestAFreshClientHasNoContracts(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Cliente Nuovo")

	if got := a.contracts(t, c.ID); len(got) != 0 {
		t.Errorf("a fresh Client has %d Contracts, want none: %v", len(got), got)
	}
}

// The heart of the ticket: expected_so_far_cents is a straight-line share of
// elapsed time, invoice_target_this_month_cents is the remaining shortfall
// divided by the months left, and both are pinned at four points across a
// 12-month, 120000-cent Contract (10000/month if paid exactly on schedule).
func TestExpectedAndInvoiceTargetAtSeveralPointsInTheSpan(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Studio Rossi")
	a.createContract(t, c.ID, yearContract(120000))

	for _, tc := range []struct {
		name         string
		now          time.Time
		wantExpected int64
		wantInvoice  int64
		wantOverdue  bool
	}{
		// Early: the first month of the range. 1/12 elapsed, 12 months
		// (including this one) left to invoice the whole total across.
		{"early", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), 10000, 10000, false},
		// Mid: exactly half elapsed (6 of 12 months), 7 months (June through
		// December) left — nothing received yet, so the whole total is still
		// the shortfall.
		{"mid", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), 60000, 120000 / 7, false},
		// Exactly at end: the full 12/12 elapsed, capped at the total rather
		// than reading as "one month left" — this is still the last month of
		// the range, not yet overdue.
		{"exactly at end", time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), 120000, 120000, false},
		// Past end: overdue, and the whole remaining shortfall is shown
		// instead of dividing by zero or a negative month count.
		{"past end", time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC), 120000, 120000, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.setNow(t, tc.now)
			got := a.contracts(t, c.ID)[0]
			if got.ExpectedSoFarCents != tc.wantExpected {
				t.Errorf("expected_so_far_cents = %d, want %d", got.ExpectedSoFarCents, tc.wantExpected)
			}
			if got.InvoiceTargetCents != tc.wantInvoice {
				t.Errorf("invoice_target_this_month_cents = %d, want %d", got.InvoiceTargetCents, tc.wantInvoice)
			}
			if got.Overdue != tc.wantOverdue {
				t.Errorf("overdue = %v, want %v", got.Overdue, tc.wantOverdue)
			}
			if got.ReceivedCents != 0 || got.AccountedCents != 0 {
				t.Errorf("received/accounted = %d/%d, want 0/0 — no Income has been linked yet",
					got.ReceivedCents, got.AccountedCents)
			}
		})
	}
}

// ADR-0012's whole point: invoicing more or less than the straight-line share
// one month makes the following months' target rise or fall to compensate,
// because it is recomputed from the shortfall every time rather than read off
// a fixed schedule.
func TestInvoiceTargetRecomputesFromWhatsAlreadyAccountedFor(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	c := a.createClient(t, "Studio Rossi")
	contract := a.createContract(t, c.ID, yearContract(120000))

	a.setNow(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	if got := a.contracts(t, c.ID)[0].InvoiceTargetCents; got != 10000 {
		t.Fatalf("January's target = %d, want 10000", got)
	}

	// Invoiced (not yet paid) 15000 in January — more than the straight-line
	// share. accounted_cents must count it even unpaid, so it is not
	// suggested for invoicing a second time.
	a.addIncome(t, map[string]any{
		"amount_cents":      15000,
		"category_id":       freelance.ID,
		"client_id":         c.ID,
		"contract_id":       contract.ID,
		"invoice_sent_date": "2026-01-20",
	})

	a.setNow(t, time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
	got := a.contracts(t, c.ID)[0]
	if got.AccountedCents != 15000 {
		t.Fatalf("accounted_cents = %d, want 15000", got.AccountedCents)
	}
	if got.ReceivedCents != 0 {
		t.Fatalf("received_cents = %d, want 0 — the Income has no payment date yet", got.ReceivedCents)
	}
	// Shortfall is 120000-15000=105000, over the 11 months from February
	// through December: 105000/11 = 9545 (floored), lower than January's
	// 10000 because January over-invoiced.
	wantTarget := int64(105000 / 11)
	if got.InvoiceTargetCents != wantTarget {
		t.Errorf("February's target = %d, want %d — it must fall after January over-invoiced",
			got.InvoiceTargetCents, wantTarget)
	}

	// Once that Income is actually paid, received_cents catches up too, and
	// accounted_cents (and so the target) is unchanged — the money was
	// already accounted for the moment it was invoiced.
	a.patch(t, incomePath(a.incomes(t)[0].ID), map[string]any{"payment_date": "2026-02-01"}, nil)
	afterPayment := a.contracts(t, c.ID)[0]
	if afterPayment.ReceivedCents != 15000 {
		t.Errorf("received_cents after payment = %d, want 15000", afterPayment.ReceivedCents)
	}
	if afterPayment.InvoiceTargetCents != wantTarget {
		t.Errorf("target after payment = %d, want unchanged %d", afterPayment.InvoiceTargetCents, wantTarget)
	}
}

// Two Contracts for the same Client are refused if their ranges overlap at
// all, but a Contract starting exactly the month a previous one ended must
// NOT be refused — the boundary the inclusive a<=d && c<=b check exists to
// get right.
func TestOverlappingContractsAreRefusedButAdjacentOnesAreNot(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Studio Rossi")
	a.createContract(t, c.ID, yearContract(120000))

	for _, tc := range []struct {
		name  string
		start string
		end   string
		want  int
	}{
		{"fully inside", "2026-06", "2026-08", http.StatusBadRequest},
		{"overlaps the start", "2025-06", "2026-01", http.StatusBadRequest},
		{"overlaps the end", "2026-12", "2027-06", http.StatusBadRequest},
		{"contains it entirely", "2025-01", "2027-01", http.StatusBadRequest},
		{"starts the month straight after it ends", "2027-01", "2027-12", http.StatusCreated},
		{"ends the month straight before it starts", "2025-01", "2025-12", http.StatusCreated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.post(t, contractsPath(c.ID), map[string]any{
				"start_month": tc.start, "end_month": tc.end, "total_cents": 1000,
			}, nil)
			if res.StatusCode != tc.want {
				t.Errorf("creating [%s,%s] = %d, want %d", tc.start, tc.end, res.StatusCode, tc.want)
			}
		})
	}
}

// A second Client's Contract in the very same months as the first Client's
// is unaffected: the overlap check is scoped per Client.
func TestOverlapCheckIsScopedToOneClient(t *testing.T) {
	a := newTestApp(t)
	rossi := a.createClient(t, "Studio Rossi")
	bianchi := a.createClient(t, "Studio Bianchi")
	a.createContract(t, rossi.ID, yearContract(120000))

	res := a.post(t, contractsPath(bianchi.ID), yearContract(60000), nil)
	if res.StatusCode != http.StatusCreated {
		t.Errorf("a second Client's Contract in the same months = %d, want 201", res.StatusCode)
	}
}

// Unlinked ("Extra") Income from a Client with a Contract must not move that
// Contract's figures at all, but still has to count toward the Client's
// all-time total earned (ticket 04) — the app must not lose track of one-off
// work just because it names no Contract.
func TestExtraIncomeIsExcludedFromContractFiguresButCountsTowardTotalEarned(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	c := a.createClient(t, "Studio Rossi")
	a.createContract(t, c.ID, yearContract(120000))
	a.setNow(t, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	before := a.contracts(t, c.ID)[0]

	// Extra: no contract_id at all.
	a.addIncome(t, map[string]any{
		"amount_cents": 5000,
		"category_id":  freelance.ID,
		"client_id":    c.ID,
		"payment_date": "2026-06-10",
	})

	after := a.contracts(t, c.ID)[0]
	if after != before {
		t.Errorf("an unlinked Income changed the Contract's figures: before %+v, after %+v", before, after)
	}

	var clients []clientJSON
	a.get(t, "/api/clients", &clients)
	if len(clients) != 1 || clients[0].TotalEarnedCents != 5000 {
		t.Errorf("total_earned_cents = %+v, want 5000 — Extra income still counts toward it", clients)
	}
}

// An Income can only link to a Contract belonging to its own Client, and a
// linked Contract that does not exist is refused the same way an unknown
// Category or Client is.
func TestAnIncomeCanOnlyLinkToItsOwnClientsContract(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	rossi := a.createClient(t, "Studio Rossi")
	bianchi := a.createClient(t, "Studio Bianchi")
	rossiContract := a.createContract(t, rossi.ID, yearContract(120000))

	res := a.post(t, "/api/incomes", map[string]any{
		"amount_cents": 5000,
		"category_id":  freelance.ID,
		"client_id":    bianchi.ID,
		"contract_id":  rossiContract.ID,
		"payer":        "Nicco",
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("linking to another Client's Contract = %d, want 400", res.StatusCode)
	}

	res = a.post(t, "/api/incomes", map[string]any{
		"amount_cents": 5000,
		"category_id":  freelance.ID,
		"client_id":    rossi.ID,
		"contract_id":  rossiContract.ID + 999,
		"payer":        "Nicco",
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("linking to an unknown Contract = %d, want 400", res.StatusCode)
	}

	// The valid link, for good measure — linking to its own Client's Contract
	// works.
	var created incomeJSON
	res = a.post(t, "/api/incomes", map[string]any{
		"amount_cents": 5000,
		"category_id":  freelance.ID,
		"client_id":    rossi.ID,
		"contract_id":  rossiContract.ID,
		"payer":        "Nicco",
	}, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("linking to its own Client's Contract = %d, want 201", res.StatusCode)
	}
	if created.ContractID == nil || *created.ContractID != rossiContract.ID {
		t.Errorf("contract_id = %v, want %d", created.ContractID, rossiContract.ID)
	}
}

func TestContractWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	c := a.createClient(t, "Studio Rossi")

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"no total", map[string]any{"start_month": "2026-01", "end_month": "2026-12"}},
		{"zero total", map[string]any{"start_month": "2026-01", "end_month": "2026-12", "total_cents": 0}},
		{"negative total", map[string]any{"start_month": "2026-01", "end_month": "2026-12", "total_cents": -1}},
		{"malformed start_month", map[string]any{"start_month": "2026-1", "end_month": "2026-12", "total_cents": 1000}},
		{"malformed end_month", map[string]any{"start_month": "2026-01", "end_month": "not-a-month", "total_cents": 1000}},
		{"end before start", map[string]any{"start_month": "2026-06", "end_month": "2026-01", "total_cents": 1000}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.post(t, contractsPath(c.ID), tc.body, nil)
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("%v = %d, want 400", tc.body, res.StatusCode)
			}
		})
	}

	// None of the above landed anyway.
	if got := a.contracts(t, c.ID); len(got) != 0 {
		t.Errorf("the list is %v, want none of the invalid writes to have landed", got)
	}

	// An unknown Client is a 404, the same as every other nested create in
	// this codebase addresses a missing parent.
	res := a.post(t, contractsPath(c.ID+999), yearContract(1000), nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("creating a Contract for an unknown Client = %d, want 404", res.StatusCode)
	}
}
