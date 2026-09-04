package main

import "database/sql"

// migrateGiftContractsReminders is schema step 10: the "Regali" category
// promoted in place to the new protected Gift base category, a Client's
// optional default Income category, the new Contract table an Income can
// point at, and the new Reminder table — bundled into one step because a
// database with some of these and not the others is not a state any of the
// four feature tickets built on this one can build on. No user-facing
// behaviour yet: that is every one of those tickets, not this one.
func migrateGiftContractsReminders(tx *sql.Tx) error {
	// Updated in place, never inserted again — same reasoning and the same
	// `code IS NULL` defense-in-depth guard as migrateInvestments's promotion
	// of "Investimenti".
	if _, err := tx.Exec(`UPDATE category SET applies_to = ?, code = ? WHERE name = ? AND code IS NULL`,
		appliesBoth, codeGift, seedRegaliName); err != nil {
		return err
	}

	// Only ever a prefill for the Income form's category picker, never
	// enforced — an Income from this Client can still pick any Income
	// category. A plain reference, so hiding or deleting the category is
	// unaffected; nothing here forces it to stay an Income-appliable one.
	if _, err := tx.Exec(`ALTER TABLE client ADD COLUMN default_category_id INTEGER REFERENCES category(id)`); err != nil {
		return err
	}

	// start_month/end_month are both required, unlike Recurring's open-ended
	// end_month — a Contract always names the range it covers. Plain string
	// comparison on YYYY-MM, same as Recurring's own range.
	if _, err := tx.Exec(`CREATE TABLE contract (
		id           INTEGER PRIMARY KEY,
		client_id    INTEGER NOT NULL REFERENCES client(id),
		start_month  TEXT NOT NULL,
		end_month    TEXT NOT NULL CHECK (end_month >= start_month),
		total_cents  INTEGER NOT NULL CHECK (total_cents > 0)
	) STRICT`); err != nil {
		return err
	}

	// A plain reference, like income.client_id: a Contract with Incomes linked
	// to it refuses deletion rather than silently orphaning the money.
	if _, err := tx.Exec(`ALTER TABLE income ADD COLUMN contract_id INTEGER REFERENCES contract(id)`); err != nil {
		return err
	}

	// set_for_month CHECKs against '' for the reason every other optional month
	// column here does: NULL is the only spelling of "not currently set".
	_, err := tx.Exec(`CREATE TABLE reminder (
		id             INTEGER PRIMARY KEY,
		label          TEXT NOT NULL,
		enabled        INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
		set_for_month  TEXT CHECK (set_for_month <> '')
	) STRICT`)
	return err
}
