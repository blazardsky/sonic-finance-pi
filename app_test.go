package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// testClock is the moment every test runs at. Nothing may read the wall clock:
// a test asserting on "this month" against the real clock is a flake waiting
// for the 1st of the month.
var testClock = time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

// testApp is a running app over a real SQLite database in a temp dir, migrated
// by the same code that runs on the Pi. Tests drive it over HTTP and assert on
// status codes, response bodies, and what a later request sees — never on SQL
// or unexported functions. A test should survive any refactor that leaves the
// API's behaviour unchanged; that is the point of putting the seam this high.
type testApp struct {
	*httptest.Server

	// The clock is read by the server goroutine and written by the test
	// goroutine, so it is guarded. Move it with setNow.
	mu  sync.Mutex
	now time.Time
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	a := &testApp{now: testClock}
	a.Server = httptest.NewServer(newApp(db, a.clock))
	t.Cleanup(a.Server.Close)
	return a
}

func (a *testApp) clock() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.now
}

// setNow moves the app's clock, for tests about what month it is.
func (a *testApp) setNow(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.now = now
}

// get sends a GET and, when dst is non-nil, decodes the JSON body into it.
func (a *testApp) get(t *testing.T, path string, dst any) *http.Response {
	t.Helper()
	res, err := a.Client().Get(a.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { res.Body.Close() })
	if dst != nil {
		if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
			t.Fatalf("GET %s: decoding body: %v", path, err)
		}
	}
	return res
}

func TestHealthReportsSchemaVersionAndTheInjectedClock(t *testing.T) {
	a := newTestApp(t)

	var got struct {
		SchemaVersion int    `json:"schema_version"`
		Now           string `json:"now"`
	}
	res := a.get(t, "/api/health", &got)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if got.SchemaVersion != schemaVersion {
		t.Errorf("schema_version = %d, want %d", got.SchemaVersion, schemaVersion)
	}
	if want := testClock.Format(time.RFC3339); got.Now != want {
		t.Errorf("now = %q, want %q — the handler is reading the wall clock", got.Now, want)
	}
}

func TestHealthFollowsTheClockWhenItMoves(t *testing.T) {
	a := newTestApp(t)
	later := testClock.AddDate(0, 1, 0)
	a.setNow(later)

	var got struct {
		Now string `json:"now"`
	}
	a.get(t, "/api/health", &got)

	if want := later.Format(time.RFC3339); got.Now != want {
		t.Errorf("now = %q, want %q", got.Now, want)
	}
}
