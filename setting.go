package main

import (
	"database/sql"
	"errors"
)

// The setting table is a plain key/value store: the password hash, and later
// the Payer and Payment method lists. Keys are constants next to the code that
// owns them rather than collected here, so an area's key lives with its area.

// getSetting returns the value stored under key, or "" when there is none.
func getSetting(db *sql.DB, key string) (string, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM setting WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO setting (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
