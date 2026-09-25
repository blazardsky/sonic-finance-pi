package main

import (
	"database/sql"
	"strconv"
	"time"
)

// The Tax reserve (CONTEXT.md): with someone in the household self-employed,
// Headroom sets aside a share of each month's freelance Income for the tax
// that will be paid on it the following year.

const (
	taxReserveFromHistory  = "history"
	taxReserveFromFallback = "fallback"
)

// taxReserveRate is the household's own rate, in basis points (hundredths of
// a percent, so 33% is 3300 and integer arithmetic stays exact): the median,
// over completed Tax years, of tax paid for the year ÷ freelance Income
// received in it — the tax summary's own two figures. A Tax year Y is
// completed once Y+1 is over, since tax on Y's Income is paid during Y+1
// (ADR-0008); it counts only with both figures on record, because a year with
// no tax Expenses is far more likely unrecorded than tax-free. With no such
// year, the fallback percentage the household set.
//
// The months of Y and Y+1 are generated first, for the reason the tax summary
// generates its own: a tax payment can be a Recurring expense, and nobody may
// have opened the month it lands in.
func taxReserveRate(db *sql.DB, now func() time.Time, firstMonth string, fallbackPercent int64) (basisPoints int64, source string, err error) {
	// A Tax year can start before the first Expense: tax on Y is paid in Y+1,
	// so the first thing recorded may be that payment, with Y's Income alone
	// before it.
	var firstIncome sql.NullString
	if err := db.QueryRow(`SELECT MIN(substr(payment_date, 1, 7)) FROM income
		WHERE payment_date IS NOT NULL`).Scan(&firstIncome); err != nil {
		return 0, "", err
	}
	if firstIncome.Valid && firstIncome.String < firstMonth {
		firstMonth = firstIncome.String
	}
	first, err := strconv.Atoi(firstMonth[:4])
	if err != nil {
		return 0, "", err
	}
	var ratios []int64
	for y := first; y <= now().Year()-2; y++ {
		for _, month := range append(monthsOf(strconv.Itoa(y)), monthsOf(strconv.Itoa(y+1))...) {
			if err := materialise(db, now, month); err != nil {
				return 0, "", err
			}
		}
		received, paid, err := taxYearFigures(db, y)
		if err != nil {
			return 0, "", err
		}
		if received > 0 && paid > 0 {
			ratios = append(ratios, paid*10000/received)
		}
	}
	if len(ratios) == 0 {
		return fallbackPercent * 100, taxReserveFromFallback, nil
	}
	return medianCents(ratios), taxReserveFromHistory, nil
}

// reserveCents is the reserve on base at a rate in basis points, rounded to
// the nearest cent.
func reserveCents(base, basisPoints int64) int64 {
	return (base*basisPoints + 5000) / 10000
}
