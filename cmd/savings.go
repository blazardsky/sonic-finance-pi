package main

import (
	"database/sql"
	"net/http"
	"time"
)

// savingsPath is ticket 07's report: the computed, ledger-free Savings
// figure, the one-time starting balance it already folds in, and the
// portfolio breakdown — one request, so the Savings page loads in one round
// trip (spec's Implementation Decisions).
const savingsPath = "/api/reports/savings"

// savingsStartingBalanceCentsKey is the one new setting this ticket adds,
// alongside Target and Goal's (cmd/budget.go) in the same lists struct
// (cmd/setting.go) — a single stored value, round-tripped through the
// existing PUT /api/settings, no new endpoint.
const savingsStartingBalanceCentsKey = "savings_starting_balance_cents"

// netWorthTargetCentsKey is the total the household wants Savings plus its
// portfolio to reach — another cents-shaped setting on the same lists payload
// (cmd/setting.go), with no default-then-sticky behaviour of its own: 0 means
// nothing has been set, and the page then has no target to project toward.
const netWorthTargetCentsKey = "net_worth_target_cents"

// holdingBreakdown is one Holding's share of the portfolio: what was put in
// net of what was taken out, as a percentage of the total across every other
// Holding still standing. A Holding fully sold off nets to zero and is never
// one of these rows (readHoldingBreakdown drops it, rather than list it at
// 0%).
type holdingBreakdown struct {
	HoldingID int64   `json:"holding_id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	NetCents  int64   `json:"net_cents"`
	Percent   float64 `json:"percent"`
}

// savingsReport is the Savings page's one request: the computed figure, the
// starting balance it already includes (so the page can show and edit it on
// its own), the portfolio breakdown, and CombinedCents — Savings plus the
// current portfolio value, so the page can show what the household holds
// altogether (ticket 05).
type savingsReport struct {
	SavingsCents         int64              `json:"savings_cents"`
	StartingBalanceCents int64              `json:"starting_balance_cents"`
	Holdings             []holdingBreakdown `json:"holdings"`
	CombinedCents        int64              `json:"combined_cents"`

	// The household's net worth target and the pace it is being approached
	// at, so the page can say when CombinedCents would reach it. The target
	// is a plain setting; the pace is computed — see
	// computeYearlySavingsCents.
	NetWorthTargetCents int64 `json:"net_worth_target_cents"`
	YearlySavingsCents  int64 `json:"yearly_savings_cents"`
}

func handleSavingsReport(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		savingsCents, startingBalanceCents, err := computeSavingsCents(db, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		holdings, portfolioCents, err := readHoldingBreakdown(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		targetCents, err := getSettingCents(db, netWorthTargetCentsKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		yearlySavingsCents, err := computeYearlySavingsCents(db, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, savingsReport{
			SavingsCents:         savingsCents,
			StartingBalanceCents: startingBalanceCents,
			Holdings:             holdings,
			CombinedCents:        savingsCents + portfolioCents,
			NetWorthTargetCents:  targetCents,
			YearlySavingsCents:   yearlySavingsCents,
		})
	}
}

// computeYearlySavingsCents is how fast Savings is actually growing: twelve
// times the median month's saving over Budget's own trailing window, with
// Investments excluded on both sides for the same reason Budget excludes
// them (ADR-0009) — a lumpy stock buy is no more a normal month's saving
// than it is a normal month's spend, and Savings itself is defined without
// them anyway (CONTEXT.md).
//
// Below Budget's minimum history a median is one or two months pretending to
// be a pattern, so the household's Goal — the monthly saving it chose by hand
// — stands in for it instead. Same window, same minimum and the same median
// primitive as computeBudgetCents, on the Income-minus-Expense side rather
// than the Expense one.
func computeYearlySavingsCents(db *sql.DB, now func() time.Time) (int64, error) {
	firstMonth, err := firstExpenseMonth(db)
	if err != nil {
		return 0, err
	}

	var nets []int64
	for _, month := range trailingCompletedMonths(now, budgetTrailingMonths) {
		if firstMonth == "" || month < firstMonth {
			continue
		}
		incomeCents, err := readIncomeCentsExcludingInvestments(db, now, month)
		if err != nil {
			return 0, err
		}
		expenseCents, err := readExpenseCentsExcludingInvestments(db, now, month)
		if err != nil {
			return 0, err
		}
		nets = append(nets, incomeCents-expenseCents)
	}
	if len(nets) < budgetMinHistoryMonths {
		goalCents, err := getSettingCents(db, goalCentsKey)
		return goalCents * 12, err
	}
	return medianCents(nets) * 12, nil
}

// computeSavingsCents is Savings itself (CONTEXT.md): cumulative received
// Income minus Expense, excluding Investments, since the household started
// using the app, plus the one-time starting balance. Unscoped by month —
// unlike readExpenseCentsExcludingInvestments/
// readIncomeCentsExcludingInvestments (cmd/reports.go, cmd/estimate.go),
// which exist to answer "one month's normal spend" and would need folding
// over every month back to the beginning of history — a plain direct query
// is the more honest way to ask "all of it, up to some point" than looping
// those two.
//
// cutoff is "" for the Savings page's own all-time read, or a YYYY-MM-DD date
// to bound both sums by (occurred_on/payment_date <= cutoff) — the yearly
// report's "Savings as it stood on December 31st of the prior year" (ticket
// 07). One function with an optional bound rather than a near-duplicate: the
// two reads are the same formula at different points in time, not two
// different formulas, and the starting balance is added either way — it is a
// fact about before the app existed, true at every cutoff after it.
//
// ponytail: no materialise call here, the same bargain handleRecentEntries
// (cmd/reports.go) already makes for an all-time, non-month-scoped read — a
// Recurring expense not yet generated for the month in progress is a gap
// every other all-time view already has, not a new one this ticket
// introduces. The yearly report's cutoff caller materialises its own months
// itself (handleFullYearReport, cmd/yearreport.go) before ever calling this.
func computeSavingsCents(db *sql.DB, cutoff string) (savingsCents, startingBalanceCents int64, err error) {
	var incomeCents, expenseCents int64
	// The ADR-0003 filter: an Income only counts once it has a payment date.
	incomeQuery := `SELECT COALESCE(SUM(i.amount_cents), 0) FROM income i
		JOIN category c ON c.id = i.category_id
		WHERE i.payment_date IS NOT NULL AND IFNULL(c.code, '') != ?`
	incomeArgs := []any{codeInvestments}
	expenseQuery := `SELECT COALESCE(SUM(e.amount_cents), 0) FROM expense e
		JOIN category c ON c.id = e.category_id
		WHERE IFNULL(c.code, '') != ?`
	expenseArgs := []any{codeInvestments}
	if cutoff != "" {
		incomeQuery += ` AND i.payment_date <= ?`
		incomeArgs = append(incomeArgs, cutoff)
		expenseQuery += ` AND e.occurred_on <= ?`
		expenseArgs = append(expenseArgs, cutoff)
	}

	if err = db.QueryRow(incomeQuery, incomeArgs...).Scan(&incomeCents); err != nil {
		return 0, 0, err
	}
	if err = db.QueryRow(expenseQuery, expenseArgs...).Scan(&expenseCents); err != nil {
		return 0, 0, err
	}
	if startingBalanceCents, err = getSettingCents(db, savingsStartingBalanceCentsKey); err != nil {
		return 0, 0, err
	}
	return incomeCents - expenseCents + startingBalanceCents, startingBalanceCents, nil
}

// readHoldingBreakdown is the portfolio percentage breakdown: each Holding's
// net contribution (buys minus sells) as a percentage of the total across
// every other Holding still standing, plus that total itself (ticket 05's
// CombinedCents is Savings plus this, so the caller reuses it rather than
// re-summing the same slice). Same shape as readBreakdown's Category
// grouping (cmd/reports.go), applied to Holdings — buys and sells are summed
// per Holding in SQL, but the zero-drop and the percentage itself are plain
// Go arithmetic over the small handful of rows a household ever has, rather
// than SQL cleverness for its own sake.
func readHoldingBreakdown(db *sql.DB) ([]holdingBreakdown, int64, error) {
	rows, err := db.Query(`SELECT h.id, h.name, h.type,
			COALESCE(buys.cents, 0) - COALESCE(sells.cents, 0) AS net_cents
		FROM holding h
		LEFT JOIN (SELECT holding_id, SUM(amount_cents) AS cents FROM expense
			WHERE holding_id IS NOT NULL GROUP BY holding_id) buys ON buys.holding_id = h.id
		LEFT JOIN (SELECT holding_id, SUM(amount_cents) AS cents FROM income
			WHERE holding_id IS NOT NULL AND payment_date IS NOT NULL
			GROUP BY holding_id) sells ON sells.holding_id = h.id
		ORDER BY h.name COLLATE NOCASE`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []holdingBreakdown{}
	var total int64
	for rows.Next() {
		var hb holdingBreakdown
		if err := rows.Scan(&hb.HoldingID, &hb.Name, &hb.Type, &hb.NetCents); err != nil {
			return nil, 0, err
		}
		// A Holding never bought or fully sold back out nets to zero and is
		// dropped entirely — the ticket's rule, not shown as a 0% row.
		if hb.NetCents == 0 {
			continue
		}
		out = append(out, hb)
		total += hb.NetCents
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	for i := range out {
		out[i].Percent = float64(out[i].NetCents) * 100 / float64(total)
	}
	return out, total, nil
}
