package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

// migrateItemStoreFTS is schema step 13 (ticket 03): two append-only FTS5
// vocabularies of every Item name and Store value ever typed, so a typeahead
// can suggest from history without a hard-enforced canonical table (ADR-0013,
// ADR-0015).
//
// tokenize='trigram', not the unicode61 default: unicode61 only ever matches
// whole tokens, optionally by prefix, so a query for "melanzana" would never
// find a stored "melanzane" — one wrong letter and the token simply differs.
// trigram indexes every overlapping 3-character run instead, and a query
// built the same way (trigramMatchQuery) shares most of its runs with a
// close-but-not-exact name, which is what lets bm25 rank it as a good match
// anyway. It covers prefix matching as a side effect, for free — a query that
// is a name's first few characters shares its leading trigrams with the full
// name — and it is case-insensitive by default, which is what "Esselunga" and
// "esselunga" converging needs. The judgment call is trigram over unicode61
// specifically for this: short, typo-prone, Italian grocery/store words,
// where whole-token matching would rarely tolerate the misspelling this
// ticket names as the point.
//
// Neither table is external-content (no `content=`/`content_rowid`): nothing
// ever looks up a name by its Item or Expense row, only the other way
// around, and a name is worth keeping as a suggestion even after the
// Expense/Item that typed it is edited or deleted — this is a vocabulary of
// names typed, not a live index of current rows. That is also why there is no
// DELETE/UPDATE trigger: insertItems and the Expense write path (expense.go)
// only ever INSERT into these tables alongside their normal write, so an
// edited Expense adds an entry rather than replacing one — the simpler of
// the two approaches the ticket named, and correct here because repeats are
// just more evidence for a name already in the vocabulary, not corruption.
// Existing rows are backfilled once, here, so typeahead is useful immediately
// on an already-populated household database rather than only after the next
// write.
func migrateItemStoreFTS(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE VIRTUAL TABLE item_name_fts USING fts5(name, tokenize='trigram')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE VIRTUAL TABLE store_fts USING fts5(store, tokenize='trigram')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO item_name_fts(name) SELECT name FROM item`); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT INTO store_fts(store) SELECT store FROM expense WHERE store != ''`)
	return err
}

// syncStoreFTS adds store to its vocabulary. Called next to every write of
// expense.store (handleCreateExpense, handlePatchExpense), in the same
// transaction, exactly like insertItems does for item names. "" is not a name
// worth suggesting, so it is skipped rather than stored.
func syncStoreFTS(tx *sql.Tx, store string) error {
	if store == "" {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO store_fts(store) VALUES (?)`, store)
	return err
}

// suggestLimit caps how many suggestions either endpoint returns. A
// typeahead dropdown only ever shows a handful; a household's few thousand
// names have no use for more.
const suggestLimit = 8

// trigramMatchQuery turns q into an FTS5 MATCH query over a trigram-tokenized
// column: every overlapping 3-rune run in q, deduplicated and OR'd together,
// so a row sharing enough of them ranks well under bm25 even when q is only
// close to what is stored (a typo) rather than a substring of it.
//
// ok is false when q has fewer than 3 runes — trigram has nothing to extract
// — and the caller falls back to a plain prefix match instead.
func trigramMatchQuery(q string) (query string, ok bool) {
	r := []rune(strings.ToLower(strings.TrimSpace(q)))
	if len(r) < 3 {
		return "", false
	}
	seen := map[string]bool{}
	var b strings.Builder
	for i := 0; i+3 <= len(r); i++ {
		tg := string(r[i : i+3])
		if seen[tg] {
			continue
		}
		seen[tg] = true
		if b.Len() > 0 {
			b.WriteString(" OR ")
		}
		// FTS5 phrase syntax, not Go's: a literal `"` inside a quoted phrase is
		// escaped by doubling it, not by a backslash — %q would use the wrong
		// convention and could turn a quote typed into the search box into a
		// malformed query.
		b.WriteByte('"')
		b.WriteString(strings.ReplaceAll(tg, `"`, `""`))
		b.WriteByte('"')
	}
	return b.String(), true
}

// suggestFrom returns up to suggestLimit distinct values of column from an
// FTS5 vocabulary table, ranked by how well they match q: typo-tolerant and
// prefix-friendly via trigramMatchQuery for a 3+ rune q, and a plain
// case-insensitive prefix match otherwise (SQLite's LIKE already folds ASCII
// case, which covers this app's Italian-alphabet names). An empty q returns
// no suggestions rather than the whole vocabulary.
//
// table and column are always one of the two constants the handlers below
// pass — never request input — so building the query with fmt.Sprintf is
// safe; FTS5 has no way to bind an identifier as a parameter.
//
// GROUP BY folds the vocabulary's repeats: the tables are append-only, so the
// same name legitimately appears more than once, and trigram's
// case-insensitivity already folds "Esselunga"/"esselunga" together.
func suggestFrom(db *sql.DB, table, column, q string) ([]string, error) {
	if strings.TrimSpace(q) == "" {
		return []string{}, nil
	}

	var rows *sql.Rows
	var err error
	if mq, ok := trigramMatchQuery(q); ok {
		// bm25()/rank can only be read directly off a MATCH against the FTS5
		// table itself — not through an aggregate like MIN(bm25(table)) — so
		// the match and its rank are read in the inner query, and only
		// grouped and ordered in the one wrapped around it.
		rows, err = db.Query(fmt.Sprintf(
			`SELECT %s FROM (SELECT %s, rank FROM %s WHERE %s MATCH ?)
			 GROUP BY %s ORDER BY MIN(rank) LIMIT ?`,
			column, column, table, table, column), mq, suggestLimit)
	} else {
		rows, err = db.Query(fmt.Sprintf(
			`SELECT DISTINCT %s FROM %s WHERE %s LIKE ? ORDER BY %s LIMIT ?`,
			column, table, column, column), strings.TrimSpace(q)+"%", suggestLimit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func handleSuggestItemNames(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		names, err := suggestFrom(db, "item_name_fts", "name", r.URL.Query().Get("q"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, names)
	}
}

func handleSuggestStores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stores, err := suggestFrom(db, "store_fts", "store", r.URL.Query().Get("q"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, stores)
	}
}

// distinctValues returns every distinct value ever typed into an FTS5
// vocabulary table, alphabetically — the Suggerimenti screen's own read,
// unlike suggestFrom's query-ranked handful, since here the household is
// looking the whole vocabulary over to decide what to prune.
func distinctValues(db *sql.DB, table, column string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf(
		`SELECT DISTINCT %s FROM %s ORDER BY %s COLLATE NOCASE`, column, table, column))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// deleteValues removes every row of an FTS5 vocabulary table matching any of
// values — every duplicate, since the table is append-only and a name typed
// a dozen times has a dozen rows. Also the corresponding "how many purchases
// used this store" clears from the Tracker's own comparison (ADR-0013): those
// still group Store by its raw text at query time, and a value pruned here
// simply becomes unsuggested going forward, never touching past Expenses.
func deleteValues(db *sql.DB, table, column string, values []string) error {
	if len(values) == 0 {
		return nil
	}
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		args[i] = v
	}
	_, err := db.Exec(fmt.Sprintf(
		`DELETE FROM %s WHERE %s IN (%s)`, table, column, strings.Join(placeholders, ", ")),
		args...)
	return err
}

func handleListStores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stores, err := distinctValues(db, "store_fts", "store")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, stores)
	}
}

func handleDeleteStores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Values []string `json:"values"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := deleteValues(db, "store_fts", "store", body.Values); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleListItemNames(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		names, err := distinctValues(db, "item_name_fts", "name")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, names)
	}
}

func handleDeleteItemNames(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Values []string `json:"values"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := deleteValues(db, "item_name_fts", "name", body.Values); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
