package main

import (
	"database/sql"
	"net/http"
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
// its own), and the portfolio breakdown.
type savingsReport struct {
	SavingsCents         int64              `json:"savings_cents"`
	StartingBalanceCents int64              `json:"starting_balance_cents"`
	Holdings             []holdingBreakdown `json:"holdings"`
}

func handleSavingsReport(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		savingsCents, startingBalanceCents, err := computeSavingsCents(db, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		holdings, err := readHoldingBreakdown(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, savingsReport{
			SavingsCents:         savingsCents,
			StartingBalanceCents: startingBalanceCents,
			Holdings:             holdings,
		})
	}
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
// every other Holding still standing. Same shape as readBreakdown's Category
// grouping (cmd/reports.go), applied to Holdings — buys and sells are summed
// per Holding in SQL, but the zero-drop and the percentage itself are plain
// Go arithmetic over the small handful of rows a household ever has, rather
// than SQL cleverness for its own sake.
func readHoldingBreakdown(db *sql.DB) ([]holdingBreakdown, error) {
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
		return nil, err
	}
	defer rows.Close()

	out := []holdingBreakdown{}
	var total int64
	for rows.Next() {
		var hb holdingBreakdown
		if err := rows.Scan(&hb.HoldingID, &hb.Name, &hb.Type, &hb.NetCents); err != nil {
			return nil, err
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
		return nil, err
	}
	for i := range out {
		out[i].Percent = float64(out[i].NetCents) * 100 / float64(total)
	}
	return out, nil
}
