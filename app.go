package main

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"time"
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
func writeError(w http.ResponseWriter, status int, err error) {
	log.Printf("request failed: %v", err)
	writeJSON(w, status, map[string]string{"error": http.StatusText(status)})
}
