package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// testClock is the moment every test runs at. Nothing may read the wall clock:
// a test asserting on "this month" against the real clock is a flake waiting
// for the 1st of the month.
var testClock = time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

// Every test logs in, and bcrypt at DefaultCost is a tenth of a second each
// time. The Pi pays that once a month; the suite would pay it per test.
func TestMain(m *testing.M) {
	bcryptCost = bcrypt.MinCost
	m.Run()
}

// testApp is a running app over a real SQLite database in a temp dir, migrated
// by the same code that runs on the Pi, and already logged in. Tests drive it
// over HTTP and assert on status codes, response bodies, and what a later
// request sees — never on SQL or unexported functions. A test should survive
// any refactor that leaves the API's behaviour unchanged; that is the point of
// putting the seam this high.
type testApp struct {
	*httptest.Server

	// password is the one the app generated on its first run, kept for the
	// tests that log in again by hand.
	password string

	// db is here so a test can stand a second app up over the same database —
	// what a restart looks like from outside. It is not an invitation to
	// assert on SQL: that is still on the far side of the seam.
	db *sql.DB

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

	pw, err := ensurePassword(db)
	if err != nil {
		t.Fatalf("ensurePassword: %v", err)
	}

	a := &testApp{now: testClock, password: pw, db: db}
	a.Server = httptest.NewServer(newApp(db, a.clock))
	t.Cleanup(a.Server.Close)
	a.useCookies()

	a.login(t)
	return a
}

// login gets the test client a session. Called by newTestApp, so no later
// ticket's test has to think about auth.
func (a *testApp) login(t *testing.T) {
	t.Helper()
	res := a.post(t, "/api/login", map[string]string{"password": a.password}, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("logging the test client in: status = %d, want 204", res.StatusCode)
	}
}

// useCookies gives the app's client a cookie jar, so a session issued by
// /api/login is carried by every later request the test makes.
func (a *testApp) useCookies() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err) // cookiejar.New only errors on a bad option, and there is none
	}
	a.Client().Jar = jar
}

func (a *testApp) clock() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.now
}

// setNow moves the app's clock, for tests about what month it is, and logs
// back in. Sessions last 30 days, so a test that jumps a month forward would
// otherwise start collecting 401s for reasons that have nothing to do with
// what it is testing.
func (a *testApp) setNow(t *testing.T, now time.Time) {
	t.Helper()
	a.moveClock(now)
	a.login(t)
}

// moveClock moves the clock and leaves the session alone. Only the auth tests
// want this: they are about whether a session outlives the clock, which is the
// one thing setNow is designed to hide.
func (a *testApp) moveClock(now time.Time) {
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
	return a.decodeBody(t, "GET", path, res, dst)
}

// post sends body as JSON — nil for an empty request — and, when dst is
// non-nil, decodes the JSON response into it.
func (a *testApp) post(t *testing.T, path string, body, dst any) *http.Response {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("POST %s: encoding body: %v", path, err)
	}
	res, err := a.Client().Post(a.URL+path, "application/json", bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return a.decodeBody(t, "POST", path, res, dst)
}

func (a *testApp) decodeBody(t *testing.T, method, path string, res *http.Response, dst any) *http.Response {
	t.Helper()
	t.Cleanup(func() { res.Body.Close() })
	if dst != nil {
		if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
			t.Fatalf("%s %s: decoding body: %v", method, path, err)
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
	a.setNow(t, later)

	var got struct {
		Now string `json:"now"`
	}
	a.get(t, "/api/health", &got)

	if want := later.Format(time.RFC3339); got.Now != want {
		t.Errorf("now = %q, want %q", got.Now, want)
	}
}
