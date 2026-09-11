package main

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// The Tracker is a view over Items across every Expense, grouping by
// normalized name x Store x Category to show how much the same Item has cost
// over time and where it was cheaper (CONTEXT.md's Tracker entry). It is
// never a record of its own: nothing here is written, only ever read back out
// of expense and item.
//
// Store is normalized (trimmed, lowercased) only for this grouping, per
// ADR-0013 — the stored value on expense.store is never touched.

// trackerStorePrice is one Store's figures for one (name, category) group:
// last price is the most recently occurred Expense's, ties broken by the
// higher item id (insertion order), the same tiebreak handleListExpenses uses
// for same-day entries.
type trackerStorePrice struct {
	Store          string  `json:"store"`
	LastPrice      float64 `json:"last_price"`
	LastOccurredOn string  `json:"last_occurred_on"`
	MinPrice       float64 `json:"min_price"`
	MaxPrice       float64 `json:"max_price"`
}

// trackerYearlyPrice is one calendar year's average price for a (name,
// category) group, across every Store — unlike trackerStorePrice, which is
// scoped to one.
type trackerYearlyPrice struct {
	Year         int     `json:"year"`
	AveragePrice float64 `json:"average_price"`
}

// trackerItem is one (normalized name, Category) group: last/min/max broken
// down per Store, and a yearly average that folds every Store together.
type trackerItem struct {
	Name          string               `json:"name"`
	CategoryID    int64                `json:"category_id"`
	Stores        []trackerStorePrice  `json:"stores"`
	YearlyAverage []trackerYearlyPrice `json:"yearly_average"`
}

// trackerGroupKey is the (name, category) grouping key. Store is deliberately
// not part of it: it is its own level (trackerStorePrice), one below this,
// because the yearly average needs every Store folded together while
// last/min/max needs them apart.
type trackerGroupKey struct {
	name       string
	categoryID int64
}

// trackerPriceOf is the price a Item contributes to the Tracker: price per
// unit when the Item carries a quantity, since that is what makes two
// purchases of different amounts comparable — a 2kg bag and a 500g bag of the
// same item are not "cheaper" or "pricier" by their amount_cents alone. An
// Item bought with no quantity/unit has nothing to divide by, so its
// amount_cents stands in unchanged.
func trackerPriceOf(amountCents int64, quantity sql.NullFloat64) float64 {
	if quantity.Valid {
		return float64(amountCents) / quantity.Float64
	}
	return float64(amountCents)
}

const trackerSelect = `SELECT i.id, i.name, i.amount_cents, i.category_id, i.quantity,
	e.store, e.occurred_on
	FROM item i JOIN expense e ON e.id = i.expense_id`

// handleTracker answers the Tracker view. Purely computed on read: no table
// is written, and normalizing name/Store here never writes back to item or
// expense (ADR-0013).
func handleTracker(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(trackerSelect)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// One accumulator per (name, category) group; storeAgg is keyed by
		// normalized Store inside it, and yearSum/yearCount build the yearly
		// average across every Store as rows are scanned.
		type storeAgg struct {
			lastPrice      float64
			lastOccurredOn string
			lastItemID     int64
			minPrice       float64
			maxPrice       float64
		}
		type group struct {
			stores    map[string]*storeAgg
			yearSum   map[int]float64
			yearCount map[int]int
		}
		groups := map[trackerGroupKey]*group{}

		for rows.Next() {
			var itemID, itemCategoryID, amountCents int64
			var quantity sql.NullFloat64
			var name, store, occurredOn string
			if err := rows.Scan(&itemID, &name, &amountCents, &itemCategoryID,
				&quantity, &store, &occurredOn); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}

			key := trackerGroupKey{
				name:       strings.ToLower(strings.TrimSpace(name)),
				categoryID: itemCategoryID,
			}
			normStore := strings.ToLower(strings.TrimSpace(store))
			price := trackerPriceOf(amountCents, quantity)
			year, err := strconv.Atoi(occurredOn[:len(yearLayout)])
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}

			g, ok := groups[key]
			if !ok {
				g = &group{stores: map[string]*storeAgg{}, yearSum: map[int]float64{}, yearCount: map[int]int{}}
				groups[key] = g
			}

			sa, ok := g.stores[normStore]
			if !ok {
				g.stores[normStore] = &storeAgg{
					lastPrice: price, lastOccurredOn: occurredOn, lastItemID: itemID,
					minPrice: price, maxPrice: price,
				}
			} else {
				if occurredOn > sa.lastOccurredOn || (occurredOn == sa.lastOccurredOn && itemID > sa.lastItemID) {
					sa.lastPrice, sa.lastOccurredOn, sa.lastItemID = price, occurredOn, itemID
				}
				if price < sa.minPrice {
					sa.minPrice = price
				}
				if price > sa.maxPrice {
					sa.maxPrice = price
				}
			}

			g.yearSum[year] += price
			g.yearCount[year]++
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		// The frontend maps over this, same reason handleListExpenses always
		// answers [] rather than null.
		out := make([]trackerItem, 0, len(groups))
		for key, g := range groups {
			ti := trackerItem{
				Name: key.name, CategoryID: key.categoryID,
				Stores: make([]trackerStorePrice, 0, len(g.stores)),
			}
			for store, sa := range g.stores {
				ti.Stores = append(ti.Stores, trackerStorePrice{
					Store: store, LastPrice: sa.lastPrice, LastOccurredOn: sa.lastOccurredOn,
					MinPrice: sa.minPrice, MaxPrice: sa.maxPrice,
				})
			}
			sort.Slice(ti.Stores, func(i, j int) bool { return ti.Stores[i].Store < ti.Stores[j].Store })

			ti.YearlyAverage = make([]trackerYearlyPrice, 0, len(g.yearSum))
			for year, sum := range g.yearSum {
				ti.YearlyAverage = append(ti.YearlyAverage, trackerYearlyPrice{
					Year: year, AveragePrice: sum / float64(g.yearCount[year]),
				})
			}
			sort.Slice(ti.YearlyAverage, func(i, j int) bool { return ti.YearlyAverage[i].Year < ti.YearlyAverage[j].Year })

			out = append(out, ti)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Name != out[j].Name {
				return out[i].Name < out[j].Name
			}
			return out[i].CategoryID < out[j].CategoryID
		})

		writeJSON(w, http.StatusOK, out)
	}
}
