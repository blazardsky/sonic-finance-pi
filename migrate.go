package main

import (
	"database/sql"
	"fmt"
)

// schemaVersion is the user_version a fully migrated database carries. Every
// schema change bumps this by one and adds the matching case to migrate's
// switch, so deploying a new binary to the Pi is all it takes to update the
// schema on the only copy of the data. No migration library.
const schemaVersion = 5

// Set on every pooled connection, not just the first: synchronous and
// busy_timeout are per-connection settings, so a PRAGMA exec'd after Open would
// apply only to whichever connection happened to run it. journal_mode is
// persisted in the database file itself, but is set here for a fresh one.
// WAL plus synchronous=NORMAL is the SD-card longevity pairing from the README.
const dsnPragmas = "_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"

// openDB opens the database at path and brings its schema up to date.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?"+dsnPragmas)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// migrate brings db up to schemaVersion, applying each step in its own
// transaction. Running it against an already-current database is a no-op.
func migrate(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return fmt.Errorf("reading user_version: %w", err)
	}
	if v > schemaVersion {
		return fmt.Errorf("database is at schema version %d, newer than this binary's %d", v, schemaVersion)
	}

	for ; v < schemaVersion; v++ {
		if err := migrateStep(db, v); err != nil {
			return fmt.Errorf("migrating to schema version %d: %w", v+1, err)
		}
	}
	return nil
}

// migrateStep applies the one step that takes the schema from version v to v+1.
func migrateStep(db *sql.DB, v int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	switch v {
	case 0:
		_, err = tx.Exec(`CREATE TABLE setting (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		) STRICT`)
	case 1:
		err = migrateCategories(tx)
	case 2:
		err = migrateExpenses(tx)
	case 3:
		err = migrateLists(tx)
	case 4:
		err = migrateItems(tx)
	default:
		// schemaVersion was bumped without adding a case. Refusing is the whole
		// point: committing the version bump with no DDL would leave the only
		// copy of the data permanently marked as migrated when it is not.
		err = fmt.Errorf("no migration defined")
	}
	if err != nil {
		return err
	}

	// user_version is part of the transaction, but takes no bound parameter.
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", v+1)); err != nil {
		return err
	}
	return tx.Commit()
}
