package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// The spec puts one seam at the HTTP boundary, and app_test.go is the reference
// for it. This file is the deliberate exception: the spec lists migrations as a
// module under test, and "the database opens in WAL" and "re-running startup
// keeps existing rows" have no HTTP surface to assert through. Nothing else may
// reach past the seam like this — later tickets copy app_test.go, not this file.

func TestFreshDatabaseMigratesAndStartingTwiceIsANoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "twice.db")

	db, err := openDB(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO setting (key, value) VALUES ('canary', 'alive')`); err != nil {
		t.Fatalf("writing to the migrated schema: %v", err)
	}
	db.Close()

	db, err = openDB(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db.Close()

	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != schemaVersion {
		t.Errorf("user_version = %d, want %d", v, schemaVersion)
	}

	var canary string
	if err := db.QueryRow(`SELECT value FROM setting WHERE key = 'canary'`).Scan(&canary); err != nil {
		t.Fatalf("re-running startup dropped existing data: %v", err)
	}
	if canary != "alive" {
		t.Errorf("canary = %q, want %q", canary, "alive")
	}
}

// Bumping schemaVersion without adding the matching case must fail loudly. The
// alternative is committing a version bump with no DDL, which marks the only
// copy of the data as migrated when it is not.
func TestMigrationStepRefusesAVersionWithNoCase(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "gap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := migrateStep(db, schemaVersion+41); err == nil {
		t.Fatal("migrateStep succeeded for a version with no case, want an error")
	}

	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != schemaVersion {
		t.Errorf("user_version = %d after the refused step, want %d — it was not rolled back", v, schemaVersion)
	}
}

// Schema step 9: the Holding table exists, and Expense, Income and
// Recurring expense each carry a nullable holding_id that references it. This
// is the raw-SQL check migrate_test.go exists for — ticket 03 is what gives
// these columns an HTTP surface, so there is nothing to assert through
// testApp yet.
func TestInvestmentsMigrationAddsHoldingTableAndColumns(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "holdings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	res, err := db.Exec(`INSERT INTO holding (name, type) VALUES ('VWCE', 'etf')`)
	if err != nil {
		t.Fatalf("holding table missing or wrong shape: %v", err)
	}
	holdingID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	var categoryID int64
	if err := db.QueryRow(`SELECT id FROM category WHERE code = ?`, codeInvestments).Scan(&categoryID); err != nil {
		t.Fatalf("no category with code = investments: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO expense (occurred_on, amount_cents, category_id, holding_id, created_at)
		VALUES ('2026-03-15', 10000, ?, ?, '2026-03-15T00:00:00Z')`, categoryID, holdingID); err != nil {
		t.Errorf("expense.holding_id missing or wrong shape: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO income (amount_cents, category_id, holding_id, created_at)
		VALUES (10000, ?, ?, '2026-03-15T00:00:00Z')`, categoryID, holdingID); err != nil {
		t.Errorf("income.holding_id missing or wrong shape: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO recurring_expense
		(amount_cents, category_id, holding_id, day_of_month, start_month, created_at)
		VALUES (10000, ?, ?, 15, '2026-03', '2026-03-15T00:00:00Z')`, categoryID, holdingID); err != nil {
		t.Errorf("recurring_expense.holding_id missing or wrong shape: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO holding (name, type) VALUES ('bad', 'shares')`); err == nil {
		t.Error("an out-of-list type was accepted, want the CHECK constraint to refuse it")
	}
}

// Schema step 10: client.default_category_id, the new contract table,
// income.contract_id, and the new reminder table. The Gift promotion itself
// is covered by categories_test.go's TestRegaliIsPromotedInPlaceNotDuplicated
// — this test is the raw-SQL check for the shapes that have no HTTP surface
// yet (tickets 04-06 are what add one).
func TestGiftContractsRemindersMigrationAddsTablesAndColumns(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "contracts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	res, err := db.Exec(`INSERT INTO client (name) VALUES ('Acme')`)
	if err != nil {
		t.Fatal(err)
	}
	clientID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	var categoryID int64
	if err := db.QueryRow(`SELECT id FROM category WHERE code = ?`, codeGift).Scan(&categoryID); err != nil {
		t.Fatalf("no category with code = gift: %v", err)
	}

	if _, err := db.Exec(`UPDATE client SET default_category_id = ? WHERE id = ?`, categoryID, clientID); err != nil {
		t.Errorf("client.default_category_id missing or wrong shape: %v", err)
	}

	res, err = db.Exec(`INSERT INTO contract (client_id, start_month, end_month, total_cents)
		VALUES (?, '2026-01', '2026-12', 120000)`, clientID)
	if err != nil {
		t.Fatalf("contract table missing or wrong shape: %v", err)
	}
	contractID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO income (amount_cents, category_id, contract_id, created_at)
		VALUES (10000, ?, ?, '2026-03-15T00:00:00Z')`, categoryID, contractID); err != nil {
		t.Errorf("income.contract_id missing or wrong shape: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO reminder (label, enabled, set_for_month) VALUES ('Bonifico affitto', 1, '2026-03')`); err != nil {
		t.Errorf("reminder table missing or wrong shape: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO reminder (label) VALUES ('Altro promemoria')`); err != nil {
		t.Errorf("reminder table should allow a bare label with defaults: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO contract (client_id, start_month, end_month, total_cents)
		VALUES (?, '2026-12', '2026-01', 120000)`, clientID); err == nil {
		t.Error("a contract with end_month before start_month was accepted, want the CHECK constraint to refuse it")
	}
}

// Schema step 12 (ticket 01): item gains quantity/unit/discounted. An
// existing Item — meaning any row saved before this ran, which on a fresh
// database is any row inserted without the three new columns — reads back
// with quantity/unit NULL and discounted false, and the unit CHECK refuses
// anything outside the fixed list.
func TestItemPricingMigrationAddsColumnsWithSafeDefaults(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "item-pricing.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var categoryID int64
	if err := db.QueryRow(`SELECT id FROM category LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec(`INSERT INTO expense (occurred_on, amount_cents, category_id, created_at)
		VALUES ('2026-03-15', 1000, ?, '2026-03-15T00:00:00Z')`, categoryID)
	if err != nil {
		t.Fatal(err)
	}
	expenseID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	// An Item written the old way, naming none of the three new columns.
	if _, err := db.Exec(`INSERT INTO item (expense_id, name, amount_cents, category_id)
		VALUES (?, 'Pane', 200, ?)`, expenseID, categoryID); err != nil {
		t.Fatalf("item insert without the new columns failed: %v", err)
	}

	var quantity sql.NullFloat64
	var unit sql.NullString
	var discounted int
	if err := db.QueryRow(`SELECT quantity, unit, discounted FROM item WHERE expense_id = ?`, expenseID).
		Scan(&quantity, &unit, &discounted); err != nil {
		t.Fatalf("item.quantity/unit/discounted missing or wrong shape: %v", err)
	}
	if quantity.Valid || unit.Valid || discounted != 0 {
		t.Errorf("existing item read as quantity=%v unit=%v discounted=%d, want NULL, NULL, 0", quantity, unit, discounted)
	}

	if _, err := db.Exec(`INSERT INTO item (expense_id, name, amount_cents, category_id, quantity, unit)
		VALUES (?, 'Bad', 100, ?, 1, 'grams')`, expenseID, categoryID); err == nil {
		t.Error("an out-of-list unit was accepted, want the CHECK constraint to refuse it")
	}
	if _, err := db.Exec(`INSERT INTO item (expense_id, name, amount_cents, category_id, quantity, unit)
		VALUES (?, 'Bad', 100, ?, -1, 'kg')`, expenseID, categoryID); err == nil {
		t.Error("a non-positive quantity was accepted, want the CHECK constraint to refuse it")
	}
}

// The README's SD-card longevity concern.
func TestDatabaseOpensWithWALAndRelaxedSync(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "pragmas.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	var synchronous int
	if err := db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if synchronous != 1 { // 1 == NORMAL
		t.Errorf("synchronous = %d, want 1 (NORMAL)", synchronous)
	}
}
