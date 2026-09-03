package main

import (
	"database/sql"
	"encoding/csv"
	"log"
	"net/http"
	"os"
	"strconv"
)

// handleBackup streams a consistent copy of the database. VACUUM INTO writes
// a new file without stopping other connections, which is the point: a backup
// taken while the app is running still has to open. The copy lives only for
// the length of the response — the Pi's SD card is the one copy that matters,
// and leaving a second .db in /tmp after every click would spend it twice.
func handleBackup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmp, err := os.CreateTemp("", "sonic-finance-backup-*.db")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		path := tmp.Name()
		tmp.Close()
		// VACUUM INTO will not overwrite: CreateTemp's empty file has to go
		// before SQLite will write the copy.
		if err := os.Remove(path); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer os.Remove(path)

		if _, err := db.Exec("VACUUM INTO ?", path); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.sqlite3")
		w.Header().Set("Content-Disposition", `attachment; filename="sonic-finance.db"`)
		http.ServeFile(w, r, path)
	}
}

// handleExportExpenses dumps every Expense as a row, then each of its Items
// as a row that names that Expense. Items have no life of their own — ADR-0002
// — so they do not get a file; they ride along, and expense_id is how a
// spreadsheet tells which Expense the book came from.
func handleExportExpenses(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := itemsByExpense(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		rows, err := db.Query(expenseSelect + ` ORDER BY occurred_on DESC, id DESC`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		var expenses []expense
		for rows.Next() {
			e, err := scanExpense(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			e.Items = items[e.ID]
			expenses = append(expenses, e)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		out := [][]string{{"type", "id", "expense_id", "occurred_on", "amount_cents", "category_id", "store", "payer", "payment_method", "note", "tax_year", "name"}}
		for _, e := range expenses {
			id := strconv.FormatInt(e.ID, 10)
			out = append(out, []string{
				"expense", id, "", e.OccurredOn, strconv.FormatInt(e.AmountCents, 10),
				strconv.FormatInt(e.CategoryID, 10), e.Store, e.Payer, e.PaymentMethod,
				e.Note, strconv.Itoa(e.TaxYear), "",
			})
			for _, it := range e.Items {
				out = append(out, []string{
					"item", "", id, "", strconv.FormatInt(it.AmountCents, 10),
					strconv.FormatInt(it.CategoryID, 10), "", "", "", "", "", it.Name,
				})
			}
		}
		writeCSV(w, "expenses.csv", out)
	}
}

// handleExportIncomes dumps every Income, paid and unpaid, newest first — the
// same order the list uses. An empty payment_date cell is the unpaid state,
// not a missing field; filling it in would be the bug ADR-0003 exists to catch.
func handleExportIncomes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(incomeSelect + ` ORDER BY id DESC`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		out := [][]string{{"id", "amount_cents", "category_id", "client_id", "payer", "payment_date", "invoice_sent_date", "note"}}
		for rows.Next() {
			in, err := scanIncome(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			client := ""
			if in.ClientID != nil {
				client = strconv.FormatInt(*in.ClientID, 10)
			}
			out = append(out, []string{
				strconv.FormatInt(in.ID, 10), strconv.FormatInt(in.AmountCents, 10),
				strconv.FormatInt(in.CategoryID, 10), client, in.Payer,
				in.PaymentDate, in.InvoiceSentDate, in.Note,
			})
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeCSV(w, "incomes.csv", out)
	}
}

func writeCSV(w http.ResponseWriter, filename string, records [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	cw := csv.NewWriter(w)
	if err := cw.WriteAll(records); err != nil {
		log.Printf("writing csv: %v", err)
	}
}
