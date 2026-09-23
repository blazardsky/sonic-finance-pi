package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// debugEnabled gates the extra diagnostic logging this turns on: every
// /api/ request's method, path, status and duration, and — on a decode
// failure — the raw body that didn't parse. A code-level knob rather than an
// env var: it's a debug aid to flip and rebuild, not deployment config.
const debugEnabled = true

// logRequests logs each /api/ request once it completes, when debugEnabled —
// silent otherwise, exactly today's behavior. Static asset serving (GET /)
// is deliberately left out: every page load fetches a handful of those, and
// they never fail in the way this exists to catch.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !debugEnabled || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("debug: %s %s -> %d (%s)", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// statusWriter is the one thing http.ResponseWriter doesn't hand back on its
// own — what status a handler actually wrote — captured for logRequests.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(status int) {
	sw.status = status
	sw.ResponseWriter.WriteHeader(status)
}

// newApp builds the whole HTTP surface. The clock is injected: no handler may
// read the wall clock, because half the recurring logic is "which month is it"
// and the Pi has no RTC.
func newApp(db *sql.DB, now func() time.Time) http.Handler {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // the files are embedded, so this can only be a build mistake
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("GET /api/categories", handleListCategories(db))
	mux.HandleFunc("POST /api/categories", handleCreateCategory(db))
	mux.HandleFunc("PATCH /api/categories/{id}", handlePatchCategory(db))
	mux.HandleFunc("DELETE /api/categories/{id}", handleDeleteCategory(db))

	mux.HandleFunc("GET /api/subcategories", handleListSubcategories(db))
	mux.HandleFunc("POST /api/subcategories", handleCreateSubcategory(db))
	mux.HandleFunc("PATCH /api/subcategories/{id}", handlePatchSubcategory(db))
	mux.HandleFunc("DELETE /api/subcategories/{id}", handleDeleteSubcategory(db))

	mux.HandleFunc("GET /api/clients", handleListClients(db))
	mux.HandleFunc("POST /api/clients", handleCreateClient(db))
	mux.HandleFunc("PATCH /api/clients/{id}", handlePatchClient(db))
	mux.HandleFunc("DELETE /api/clients/{id}", handleDeleteClient(db))

	mux.HandleFunc("GET /api/clients/{id}/contracts", handleListClientContracts(db, now))
	mux.HandleFunc("POST /api/clients/{id}/contracts", handleCreateClientContract(db, now))
	mux.HandleFunc("PATCH /api/clients/{id}/contracts/{contractId}", handlePatchClientContract(db, now))

	mux.HandleFunc("GET /api/expenses", handleListExpenses(db))
	mux.HandleFunc("POST /api/expenses", handleCreateExpense(db, now))
	mux.HandleFunc("PATCH /api/expenses/{id}", handlePatchExpense(db))
	mux.HandleFunc("DELETE /api/expenses/{id}", handleDeleteExpense(db))

	mux.HandleFunc("GET /api/items/suggest", handleSuggestItemNames(db))
	mux.HandleFunc("GET /api/stores/suggest", handleSuggestStores(db))
	mux.HandleFunc("GET /api/items/names", handleListItemNames(db))
	mux.HandleFunc("DELETE /api/items/names", handleDeleteItemNames(db))
	mux.HandleFunc("GET /api/stores", handleListStores(db))
	mux.HandleFunc("DELETE /api/stores", handleDeleteStores(db))

	mux.HandleFunc("GET /api/incomes", handleListIncomes(db))
	mux.HandleFunc("POST /api/incomes", handleCreateIncome(db, now))
	mux.HandleFunc("PATCH /api/incomes/{id}", handlePatchIncome(db))
	mux.HandleFunc("DELETE /api/incomes/{id}", handleDeleteIncome(db))

	mux.HandleFunc("GET /api/holdings", handleListHoldings(db))
	mux.HandleFunc("POST /api/holdings", handleCreateHolding(db))
	mux.HandleFunc("PATCH /api/holdings/{id}", handlePatchHolding(db))

	mux.HandleFunc("GET /api/recurring", handleListRecurring(db))
	mux.HandleFunc("POST /api/recurring", handleCreateRecurring(db, now))
	mux.HandleFunc("PATCH /api/recurring/{id}", handlePatchRecurring(db))
	mux.HandleFunc("DELETE /api/recurring/{id}", handleDeleteRecurring(db))

	mux.HandleFunc("GET /api/reports/month/{month}", handleMonthReport(db, now))
	mux.HandleFunc("GET /api/reports/month/{month}/daily", handleMonthDailyReport(db, now))
	mux.HandleFunc("GET /api/reports/year/{year}", handleYearReport(db, now))
	mux.HandleFunc("GET /api/reports/year/{year}/full", handleFullYearReport(db, now))
	mux.HandleFunc("GET /api/reports/tax/{year}", handleTaxSummary(db, now))
	mux.HandleFunc("GET /api/reports/estimate/{year}", handleEstimateReport(db, now))
	mux.HandleFunc("GET /api/reports/daily", handleDailyReport(db, now))
	mux.HandleFunc("GET /api/reports/recent", handleRecentEntries(db))
	mux.HandleFunc("GET "+budgetPath, handleBudgetReport(db, now))
	mux.HandleFunc("GET "+savingsPath, handleSavingsReport(db, now))
	mux.HandleFunc("GET /api/pending-payments", handlePendingPayments(db, now))
	mux.HandleFunc("GET /api/tracker", handleTracker(db))

	mux.HandleFunc("GET /api/reminders", handleListReminders(db, now))
	mux.HandleFunc("POST /api/reminders", handleCreateReminder(db))
	mux.HandleFunc("PATCH /api/reminders/{id}", handlePatchReminder(db, now))
	mux.HandleFunc("DELETE /api/reminders/{id}", handleDeleteReminder(db))

	mux.HandleFunc("GET "+headroomPath, handleHeadroomReport(db, now))
	mux.HandleFunc("GET /api/planned-purchases", handleListPlanned(db))
	mux.HandleFunc("POST /api/planned-purchases", handleCreatePlanned(db))
	mux.HandleFunc("PUT /api/planned-purchases/{id}", handleUpdatePlanned(db))
	mux.HandleFunc("DELETE /api/planned-purchases/{id}", handleDeletePlanned(db))
	mux.HandleFunc("POST /api/planned-purchases/{id}/move", handleMovePlanned(db))

	mux.HandleFunc("GET "+settingsPath, handleGetLists(db))
	mux.HandleFunc("PUT "+settingsPath, handlePutLists(db))
	mux.HandleFunc("POST "+settingsPath+"/password", handleChangePassword(db, now))

	mux.HandleFunc("GET /api/backup", handleBackup(db))
	mux.HandleFunc("GET /api/export/expenses.csv", handleExportExpenses(db))
	mux.HandleFunc("GET /api/export/incomes.csv", handleExportIncomes(db))

	mux.HandleFunc("POST /api/login", handleLogin(db, now))
	mux.HandleFunc("POST /api/logout", handleLogout)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		var v int
		if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// clock_ok is what the screens warn on. The Pi has no RTC, so a boot
		// without network time is a real state the household has to be told
		// about: nothing is being generated, their month is short a rent, and
		// the fix is to wait for NTP rather than to retype anything.
		writeJSON(w, http.StatusOK, map[string]any{
			"schema_version": v,
			"now":            now().Format(time.RFC3339),
			"clock_ok":       clockSane(now),
		})
	})

	return logRequests(requireSession(db, now, mux))
}

// decodeJSON reads a JSON request body into dst. The cap is the point: the Pi
// has 512MB of RAM, and none of these payloads — an Expense with its Items is
// the largest — has any business being bigger than this.
//
// On a decode failure, the raw body is logged when debugEnabled — this is
// the one case "add error logs with context" exists for: the error Go's own
// json package returns already names the offending field (e.g. "cannot
// unmarshal string into Go struct field ...category_id of type int64"), but
// not what was actually sent, which is what turns "something broke" into
// "the category picker sent an empty string again."
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		if debugEnabled {
			log.Printf("debug: %s %s failed to decode body: %s", r.Method, r.URL.Path, bytes.TrimSpace(body))
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("writing response: %v", err)
	}
}

// writeError logs the detail and tells the client only the status. The
// underlying error can name tables and columns, which is not something to hand
// out over an endpoint reachable by anyone on the tailnet.
//
// A foreign key refusing is the one database error that is not the server's
// fault: it means something is still pointing at the row, which is a conflict
// the client can act on rather than a fault it can only retry. Correcting the
// status here rather than at each delete is the point — every db.Exec error in
// the app already passes through this one function, and a Category with
// Expenses in it and a Client with Incomes in it are the same refusal.
func writeError(w http.ResponseWriter, status int, err error) {
	var sqlErr *sqlite.Error
	if errors.As(err, &sqlErr) && sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
		status = http.StatusConflict
	}
	log.Printf("request failed: %v", err)
	writeJSON(w, status, map[string]string{"error": http.StatusText(status)})
}

// writeInvalid is the deliberate opposite: a refusal this codebase itself
// worded, handed to the client as written. These sentences name the rule that
// was broken and nothing else — no table, no column — so there is nothing in
// them to withhold, and a client that gets a 400 can say which rule it broke.
//
// The screens do not read it yet: they refuse the same cases themselves, in
// Italian, before sending. This is what the API answers, and what its tests
// hold it to.
func writeInvalid(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}
