package main

// The four values Spending intent can hold on a Category, Expense or
// Recurring expense (spec's Implementation Decisions → Schema) — the single-
// column enum that directly encodes the conditional tree, so "necessity, but
// also wise" is simply not representable. migrateSpendingIntent (ticket 01,
// setting.go) puts the identical CHECK on all three columns; this is the one
// Go-side list every validate() that touches spending_intent checks against,
// so a Category (ticket 02), Expense (ticket 03) and Recurring expense
// (ticket 04) validator never drift apart on what counts as valid.
const (
	spendingIntentNecessity      = "necessity"
	spendingIntentDesire         = "desire"
	spendingIntentDesireWise     = "desire_wise"
	spendingIntentDesireBullshit = "desire_bullshit"
)

var spendingIntents = []string{
	spendingIntentNecessity, spendingIntentDesire, spendingIntentDesireWise, spendingIntentDesireBullshit,
}

// validSpendingIntent reports whether v is nil (no Spending intent recorded —
// always valid, every column is nullable) or one of the four recognized
// values. Checked in validate() before the write reaches the database, so an
// unrecognized string is a 400 rather than the 500 the DB's own CHECK would
// otherwise turn it into (holding.validate()'s convention).
func validSpendingIntent(v *string) bool {
	if v == nil {
		return true
	}
	for _, s := range spendingIntents {
		if *v == s {
			return true
		}
	}
	return false
}
