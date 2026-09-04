package main

import (
	"database/sql"
	"net/http"
	"slices"
	"strconv"
	"time"
)

// budgetPath is ticket 04's report: Budget (computed), Target and Goal
// (household-set), the three figures the Month page shows only for the
// current real month.
const budgetPath = "/api/reports/budget"

// Target and Goal's own setting keys, alongside Payers and Payment methods
// in cmd/setting.go. Two keys, not one, because CONTEXT.md keeps them
// distinct concepts: Target is an Expense ceiling, Goal is a savings figure,
// and they must be settable independently of each other.
const (
	targetCentsKey = "target_cents"
	goalCentsKey   = "savings_goal_cents"
)

// budgetTrailingMonths is how far back Budget looks — the trailing 12
// completed months, not the twelve months of a calendar year the yearly
// report already answers.
const budgetTrailingMonths = 12

// budgetMinHistoryMonths is the fewest completed months of real history
// Budget will compute a median from. Below this a "typical month" is one or
// two data points pretending to be a pattern (CONTEXT.md's Budget entry).
const budgetMinHistoryMonths = 3

// budgetReport is what the Month page reads to show all three figures at
// once. target_cents and goal_cents are always real numbers — they are just
// settings — independent of whether Budget itself is available.
type budgetReport struct {
	Available   bool  `json:"available"`
	BudgetCents int64 `json:"budget_cents"`
	TargetCents int64 `json:"target_cents"`
	GoalCents   int64 `json:"goal_cents"`
}

// handleBudgetReport answers Budget, Target and Goal together: Target's
// default-then-sticky behaviour needs Budget's own number to default from, so
// there is no way to answer either correctly from two separate endpoints.
func handleBudgetReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		budgetCents, available, err := computeBudgetCents(db, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		targetCents, err := readTargetCents(db, budgetCents)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		goalCents, err := getSettingCents(db, goalCentsKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, budgetReport{
			Available:   available,
			BudgetCents: budgetCents,
			TargetCents: targetCents,
			GoalCents:   goalCents,
		})
	}
}

// computeBudgetCents is Budget itself: the median of the trailing 12
// completed months' Expense totals, excluding Investments (ADR-0009) — the
// current in-progress month is never one of them, since it is still being
// spent into and would move the number with every Expense typed today.
//
// "Completed months of history" is bounded by the household's own data — the
// earliest month it ever recorded an Expense in — rather than by the
// calendar, which always has months before now whether or not the household
// was using the app then. A month before that start is not a real zero and
// is left out of the median entirely, never counted as one.
func computeBudgetCents(db *sql.DB, now func() time.Time) (cents int64, available bool, err error) {
	firstMonth, err := firstExpenseMonth(db)
	if err != nil || firstMonth == "" {
		return 0, false, err
	}

	var totals []int64
	for _, month := range trailingCompletedMonths(now, budgetTrailingMonths) {
		if month < firstMonth {
			continue
		}
		c, err := readExpenseCentsExcludingInvestments(db, now, month)
		if err != nil {
			return 0, false, err
		}
		totals = append(totals, c)
	}
	if len(totals) < budgetMinHistoryMonths {
		return 0, false, nil
	}
	return medianCents(totals), true, nil
}

// firstExpenseMonth is the earliest month anything was ever recorded as an
// Expense, or "" when nothing has been. There is no separate "started using
// this app on" date anywhere in the schema; an Expense's own history is the
// only honest proxy for one.
func firstExpenseMonth(db *sql.DB) (string, error) {
	var v sql.NullString
	if err := db.QueryRow(`SELECT MIN(substr(occurred_on, 1, 7)) FROM expense`).Scan(&v); err != nil {
		return "", err
	}
	return v.String, nil
}

// trailingCompletedMonths is the n calendar months immediately before the
// current one, oldest first.
func trailingCompletedMonths(now func() time.Time, n int) []string {
	// now().Format/re-parse rounds down to the month's first day, so the
	// AddDate arithmetic below only ever steps by whole months.
	cur, _ := time.Parse(monthLayout, now().Format(monthLayout))
	months := make([]string, n)
	for i := range months {
		months[n-1-i] = cur.AddDate(0, -(i + 1), 0).Format(monthLayout)
	}
	return months
}

// medianCents is Budget's only user of a median: sorted, then the middle
// value, or the two middle values averaged (rounded down) when there is no
// single middle. len(totals) is at most budgetTrailingMonths — small enough
// that sorting on every call is the whole engineering this needs.
func medianCents(totals []int64) int64 {
	sorted := slices.Clone(totals)
	slices.Sort(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// readTargetCents is Target's own read, not getSettingCents's plain one:
// absence is meaningful here in a way it is not for Goal. The first time
// anything reads Target and finds no stored value, it is set to budgetCents
// right then and stored — the ticket's "sticky" default — so a later read,
// even once Budget itself has moved, returns what was defaulted rather than
// recomputing it.
func readTargetCents(db *sql.DB, budgetCents int64) (int64, error) {
	v, err := getSetting(db, targetCentsKey)
	if err != nil {
		return 0, err
	}
	if v == "" {
		if err := setSetting(db, targetCentsKey, strconv.FormatInt(budgetCents, 10)); err != nil {
			return 0, err
		}
		return budgetCents, nil
	}
	return strconv.ParseInt(v, 10, 64)
}
