package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

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

	mux.HandleFunc("GET /api/clients", handleListClients(db))
	mux.HandleFunc("POST /api/clients", handleCreateClient(db))
	mux.HandleFunc("PATCH /api/clients/{id}", handlePatchClient(db))
	mux.HandleFunc("DELETE /api/clients/{id}", handleDeleteClient(db))

	mux.HandleFunc("GET /api/expenses", handleListExpenses(db))
	mux.HandleFunc("POST /api/expenses", handleCreateExpense(db, now))
	mux.HandleFunc("PATCH /api/expenses/{id}", handlePatchExpense(db))
	mux.HandleFunc("DELETE /api/expenses/{id}", handleDeleteExpense(db))

	mux.HandleFunc("GET "+settingsPath, handleGetLists(db))
	mux.HandleFunc("PUT "+settingsPath, handlePutLists(db))

	mux.HandleFunc("POST /api/login", handleLogin(db, now))
	mux.HandleFunc("POST /api/logout", handleLogout)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		var v int
		if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"schema_version": v,
			"now":            now().Format(time.RFC3339),
		})
	})

	return requireSession(db, now, mux)
}

// decodeJSON reads a JSON request body into dst. The cap is the point: the Pi
// has 512MB of RAM, and none of these payloads — an Expense with its Items is
// the largest — has any business being bigger than this.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	return json.NewDecoder(r.Body).Decode(dst)
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
