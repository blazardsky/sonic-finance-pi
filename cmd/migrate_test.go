package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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

// Schema step 16 (ticket 04): category and subcategory each gain a color
// column, and every existing Category row is backfilled by snapping its
// legacy hue = (id * 137.508) % 360 to the nearest of the 8 vivid slots by
// circular distance — not left at the plain column default, which a fresh
// Subcategory insert (no legacy color to preserve) does take.
//
// Two seeded ids are checked against hand-computed expectations:
//   - id 1 (Tasse, the first row migrateCategories inserts) has legacy hue
//     1*137.508 mod 360 = 137.508. Distance to green (120.0) is 17.508 — the
//     smallest of any slot (yellow at 40.8 is 96.7 away, aqua at 158.5 is
//     21.0 away) — so it lands on green.
//   - id 9 (Abbigliamento, the 9th row) has legacy hue 9*137.508 mod 360 =
//     1237.572 mod 360 = 157.572. Distance to aqua (158.5) is 0.928 —
//     closer than blue (212.8, 55.2 away) or any other slot — so it lands
//     on aqua.
func TestCategoryColorMigrationBackfillsExistingCategoriesByHue(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "category-color.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var tasseColor string
	if err := db.QueryRow(`SELECT color FROM category WHERE code = ?`, codeTaxes).Scan(&tasseColor); err != nil {
		t.Fatalf("category.color missing or wrong shape: %v", err)
	}
	if tasseColor != "green" {
		t.Errorf("Tasse (id 1) color = %q, want %q (hue 137.508, nearest slot green at 120.0)", tasseColor, "green")
	}

	var abbigliamentoColor string
	if err := db.QueryRow(`SELECT color FROM category WHERE name = ?`, "Abbigliamento").Scan(&abbigliamentoColor); err != nil {
		t.Fatal(err)
	}
	if abbigliamentoColor != "aqua" {
		t.Errorf("Abbigliamento (id 9) color = %q, want %q (hue 157.572, nearest slot aqua at 158.5)", abbigliamentoColor, "aqua")
	}

	// Subcategory never had a legacy color: a fresh row simply takes the
	// plain column default, no snapping applied.
	res, err := db.Exec(`INSERT INTO subcategory (name, applies_to) VALUES ('Caffè', 'expense')`)
	if err != nil {
		t.Fatalf("subcategory.color missing or wrong shape: %v", err)
	}
	subID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var subColor string
	if err := db.QueryRow(`SELECT color FROM subcategory WHERE id = ?`, subID).Scan(&subColor); err != nil {
		t.Fatal(err)
	}
	if subColor != "blue-gray" {
		t.Errorf("a fresh Subcategory's color = %q, want the plain column default %q", subColor, "blue-gray")
	}

	if _, err := db.Exec(`UPDATE category SET color = 'not-a-color' WHERE code = ?`, codeTaxes); err == nil {
		t.Error("an out-of-list color was accepted, want the CHECK constraint to refuse it")
	}
}

// Schema step 21 (ticket 01): category, expense and recurring_expense each
// gain a nullable spending_intent column with the identical CHECK. Tickets
// 02-04 are what give it an HTTP surface — this is the raw-SQL check for the
// shape that has none yet, the same convention
// TestInvestmentsMigrationAddsHoldingTableAndColumns already follows.
func TestSpendingIntentMigrationAddsColumnsWithCheck(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "spending-intent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var categoryID int64
	if err := db.QueryRow(`SELECT id FROM category LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`UPDATE category SET spending_intent = 'desire_wise' WHERE id = ?`, categoryID); err != nil {
		t.Errorf("category.spending_intent missing or wrong shape: %v", err)
	}

	res, err := db.Exec(`INSERT INTO expense (occurred_on, amount_cents, category_id, spending_intent, created_at)
		VALUES ('2026-03-15', 1000, ?, 'necessity', '2026-03-15T00:00:00Z')`, categoryID)
	if err != nil {
		t.Fatalf("expense.spending_intent missing or wrong shape: %v", err)
	}
	expenseID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var spendingIntent sql.NullString
	if err := db.QueryRow(`SELECT spending_intent FROM expense WHERE id = ?`, expenseID).Scan(&spendingIntent); err != nil {
		t.Fatal(err)
	}
	if !spendingIntent.Valid || spendingIntent.String != "necessity" {
		t.Errorf("expense.spending_intent = %v, want %q", spendingIntent, "necessity")
	}

	if _, err := db.Exec(`INSERT INTO recurring_expense
		(amount_cents, category_id, spending_intent, day_of_month, start_month, created_at)
		VALUES (10000, ?, 'desire_bullshit', 15, '2026-03', '2026-03-15T00:00:00Z')`, categoryID); err != nil {
		t.Errorf("recurring_expense.spending_intent missing or wrong shape: %v", err)
	}
	// A row that leaves it unnamed reads back NULL — no value invents a
	// classification nobody picked.
	if _, err := db.Exec(`INSERT INTO recurring_expense
		(amount_cents, category_id, day_of_month, start_month, created_at)
		VALUES (10000, ?, 15, '2026-03', '2026-03-15T00:00:00Z')`, categoryID); err != nil {
		t.Errorf("recurring_expense.spending_intent should allow NULL: %v", err)
	}

	if _, err := db.Exec(`UPDATE category SET spending_intent = 'not-a-value' WHERE id = ?`, categoryID); err == nil {
		t.Error("an out-of-list spending_intent was accepted, want the CHECK constraint to refuse it")
	}
}

// A database an older binary already migrated partway needs its data
// protected before the new binary's migrateStep cases touch it.
func TestMigrateBacksUpAnExistingDatabaseBeforeMigrating(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.db")

	db, err := openDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO setting (key, value) VALUES ('canary', 'alive')`); err != nil {
		t.Fatal(err)
	}
	// Simulate a database an older binary left one migration behind.
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion-1)); err != nil {
		t.Fatal(err)
	}

	if err := backupBeforeMigrate(db, path); err != nil {
		t.Fatal(err)
	}
	db.Close()

	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatalf("reading backups dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d backup files, want 1", len(entries))
	}

	backup, err := sql.Open("sqlite", filepath.Join(dir, "backups", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()

	var v int
	if err := backup.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != schemaVersion-1 {
		t.Errorf("backup user_version = %d, want %d", v, schemaVersion-1)
	}
	var canary string
	if err := backup.QueryRow(`SELECT value FROM setting WHERE key = 'canary'`).Scan(&canary); err != nil {
		t.Fatalf("backup missing the data it was meant to protect: %v", err)
	}
	if canary != "alive" {
		t.Errorf("canary = %q, want %q", canary, "alive")
	}
}

// Only the maxBackups most recent snapshots survive — old ones are for
// falling back past one bad migration, not an ever-growing archive.
func TestPruneBackupsKeepsOnlyTheMostRecent(t *testing.T) {
	dir := t.TempDir()
	base := "sonic.db"
	names := []string{
		base + "-v12-20260101-000000.db",
		base + "-v13-20260102-000000.db",
		base + "-v14-20260103-000000.db",
		base + "-v15-20260104-000000.db",
	}
	for i, name := range names {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		// One second apart and strictly increasing with the filename order,
		// so "most recent" is unambiguous regardless of write speed.
		mtime := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	// A file for a different database in the same dir must survive untouched.
	other := filepath.Join(dir, "other.db-v15-20260101-000000.db")
	if err := os.WriteFile(other, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := pruneBackups(dir, base); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{names[2], names[3], "other.db-v15-20260101-000000.db"}
	if len(got) != len(want) {
		t.Fatalf("dir contains %v, want %v", got, want)
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			if g == w {
				found = true
			}
		}
		if !found {
			t.Errorf("dir contains %v, missing %q", got, w)
		}
	}
}

// A brand new database has nothing worth protecting yet, so no backup file
// should show up next to it.
func TestBackupBeforeMigrateSkipsAFreshDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.db")

	db, err := sql.Open("sqlite", "file:"+path+"?"+dsnPragmas)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := backupBeforeMigrate(db, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "backups")); !os.IsNotExist(err) {
		t.Error("backup dir created for a fresh database, want none")
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
