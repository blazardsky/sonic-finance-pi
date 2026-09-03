package main

import (
	"bytes"
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// A backup has to be a database the app itself can open: that is the whole
// point of downloading one. Opening the bytes as a second app and reading
// back what was written is how the test knows the file is a real copy,
// without looking at SQL or at the temporary file VACUUM INTO wrote.
func TestABackupIsAWorkingCopyOfTheDatabase(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 4237, "category_id": alimentari.ID,
	})
	received := a.addIncome(t, map[string]any{
		"amount_cents": 120000, "category_id": a.freelance(t).ID, "payment_date": "2026-03-10",
	})

	res := a.get(t, "/api/backup", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	disp := res.Header.Get("Content-Disposition")
	if !strings.Contains(disp, "attachment") || !strings.Contains(disp, ".db") {
		t.Errorf("Content-Disposition = %q, want an attached .db file", disp)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("SQLite format 3")) {
		t.Fatalf("body is not a SQLite database")
	}

	path := filepath.Join(t.TempDir(), "restored.db")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := openDB(path)
	if err != nil {
		t.Fatalf("opening the backup: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	restored := &testApp{now: testClock, password: a.password, db: db}
	restored.Server = httptest.NewServer(newApp(db, restored.clock))
	t.Cleanup(restored.Close)
	restored.useCookies()
	restored.login(t)

	got := restored.expenses(t)
	if len(got) != 1 || got[0].ID != logged.ID || got[0].AmountCents != 4237 {
		t.Errorf("expenses in the backup = %+v, want the logged 4237 cent Expense", got)
	}
	incomes := restored.incomes(t)
	if len(incomes) != 1 || incomes[0].ID != received.ID || incomes[0].AmountCents != 120000 {
		t.Errorf("incomes in the backup = %+v, want the received 120000 cent Income", incomes)
	}
}

func readCSV(t *testing.T, res *http.Response) [][]string {
	t.Helper()
	records, err := csv.NewReader(res.Body).ReadAll()
	if err != nil {
		t.Fatalf("reading CSV: %v", err)
	}
	return records
}

// The spreadsheet escape hatch: an Expense is one row, and each Item is
// another that names its Expense. A note with a comma is the check that this
// is real CSV and not joined text.
func TestExpenseCSVPutsItemsOnTheirOwnRows(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	svago := a.category(t, "Svago")
	e := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari.ID,
		"store": "Coop", "payer": "Nicco", "payment_method": "Contanti",
		"note": "spesa, con virgola",
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago.ID},
		},
	})

	res := a.get(t, "/api/export/expenses.csv", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if disp := res.Header.Get("Content-Disposition"); !strings.Contains(disp, "expenses.csv") {
		t.Errorf("Content-Disposition = %q, want expenses.csv", disp)
	}

	got := readCSV(t, res)
	id, cat, itemCat := strconv.FormatInt(e.ID, 10), strconv.FormatInt(alimentari.ID, 10), strconv.FormatInt(svago.ID, 10)
	want := [][]string{
		{"type", "id", "expense_id", "occurred_on", "amount_cents", "category_id", "store", "payer", "payment_method", "note", "tax_year", "name"},
		{"expense", id, "", "2026-03-15", "6200", cat, "Coop", "Nicco", "Contanti", "spesa, con virgola", "0", ""},
		{"item", "", id, "", "1400", itemCat, "", "", "", "", "", "Libro"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("csv = %#v, want %#v", got, want)
	}
}

// An Income dump is one row per Income, paid or not. The empty payment_date
// is the unpaid state — ADR-0003 — and has to survive as an empty cell, not a
// made-up date, or a spreadsheet would count money that has not arrived.
func TestIncomeCSVIsAFlatDump(t *testing.T) {
	a := newTestApp(t)
	freelance := a.freelance(t)
	client := a.createClient(t, "Studio Rosi")
	paid := a.addIncome(t, map[string]any{
		"amount_cents": 120000, "category_id": freelance.ID,
		"payer": "Nicco", "payment_date": "2026-03-10", "note": "marzo",
	})
	unpaid := a.addIncome(t, map[string]any{
		"amount_cents": 50000, "category_id": freelance.ID,
		"client_id": client.ID, "invoice_sent_date": "2026-03-01",
	})

	res := a.get(t, "/api/export/incomes.csv", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if disp := res.Header.Get("Content-Disposition"); !strings.Contains(disp, "incomes.csv") {
		t.Errorf("Content-Disposition = %q, want incomes.csv", disp)
	}

	got := readCSV(t, res)
	cat := strconv.FormatInt(freelance.ID, 10)
	want := [][]string{
		{"id", "amount_cents", "category_id", "client_id", "payer", "payment_date", "invoice_sent_date", "note"},
		{strconv.FormatInt(unpaid.ID, 10), "50000", cat, strconv.FormatInt(client.ID, 10), "", "", "2026-03-01", ""},
		{strconv.FormatInt(paid.ID, 10), "120000", cat, "", "Nicco", "2026-03-10", "", "marzo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("csv = %#v, want %#v", got, want)
	}
}
