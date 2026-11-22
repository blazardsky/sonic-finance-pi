package main

import (
	"database/sql"
	"net/http"
	"sort"
	"strings"
)

// The Tracker is a view over Items across every Expense, grouping by
// normalized name x Category to show how the same Item's price has moved over
// time and where it was cheaper (CONTEXT.md's Tracker entry). It is never a
// record of its own: nothing here is written, only ever read back out of
// expense and item.
//
// Name is normalized (trimmed, lowercased) only for this grouping, per
// ADR-0013 — the stored value on item.name is never touched.

// trackerPoint is one purchase of a group: the date it occurred, the price it
// was bought at (see trackerPriceOf), and what the chart's tooltip and the
// Tag view need to tell purchases apart.
type trackerPoint struct {
	OccurredOn    string  `json:"occurred_on"`
	Price         float64 `json:"price"`
	Unit          string  `json:"unit"`
	Store         string  `json:"store"`
	Discounted    bool    `json:"discounted"`
	SubcategoryID *int64  `json:"subcategory_id"`
}

// trackerItem is one (normalized name, Category) group with every purchase of
// it, oldest first. Last/min/max and the per-Tag/per-Category views are all
// read off these points by the frontend.
type trackerItem struct {
	Name       string         `json:"name"`
	CategoryID int64          `json:"category_id"`
	Points     []trackerPoint `json:"points"`
}

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

// Oldest first, ties by item id (insertion order) — the same tiebreak
// handleListExpenses uses for same-day entries.
const trackerSelect = `SELECT i.name, i.amount_cents, i.category_id, i.quantity, i.unit,
	i.discounted, i.subcategory_id, e.store, e.occurred_on
	FROM item i JOIN expense e ON e.id = i.expense_id
	ORDER BY e.occurred_on, i.id`

// handleTracker answers the Tracker view. Purely computed on read: no table
// is written, and normalizing name here never writes back to item (ADR-0013).
func handleTracker(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(trackerSelect)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		groups := map[trackerGroupKey]*trackerItem{}
		for rows.Next() {
			var name, store, occurredOn string
			var amountCents, categoryID int64
			var quantity sql.NullFloat64
			var unit sql.NullString
			var p trackerPoint
			if err := rows.Scan(&name, &amountCents, &categoryID, &quantity, &unit,
				&p.Discounted, &p.SubcategoryID, &store, &occurredOn); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			p.OccurredOn = occurredOn
			p.Price = trackerPriceOf(amountCents, quantity)
			p.Unit = unit.String
			p.Store = strings.TrimSpace(store)

			key := trackerGroupKey{name: strings.ToLower(strings.TrimSpace(name)), categoryID: categoryID}
			g, ok := groups[key]
			if !ok {
				g = &trackerItem{Name: key.name, CategoryID: categoryID}
				groups[key] = g
			}
			g.Points = append(g.Points, p)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		// The frontend maps over this, same reason handleListExpenses always
		// answers [] rather than null.
		out := make([]trackerItem, 0, len(groups))
		for _, g := range groups {
			out = append(out, *g)
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
