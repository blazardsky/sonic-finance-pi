package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUnauthenticatedAPICallsAreRejected(t *testing.T) {
	a := newTestApp(t)

	// A bare client, without the harness's session cookie.
	res, err := http.Get(a.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
}

func TestLoggingInIssuesAMonthLongSessionCookie(t *testing.T) {
	a := newTestApp(t)

	res := a.post(t, "/api/login", map[string]string{"password": a.password}, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", res.StatusCode)
	}

	var c *http.Cookie
	for _, got := range res.Cookies() {
		if got.Name == sessionCookie {
			c = got
		}
	}
	if c == nil {
		t.Fatalf("login set no %s cookie", sessionCookie)
	}
	if !c.HttpOnly {
		t.Error("session cookie is not HttpOnly")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if want := int(sessionTTL.Seconds()); c.MaxAge != want {
		t.Errorf("MaxAge = %d, want %d (30 days)", c.MaxAge, want)
	}

	// And the cookie the harness is already holding actually opens the API.
	if res := a.get(t, "/api/health", nil); res.StatusCode != http.StatusOK {
		t.Errorf("authenticated GET /api/health = %d, want 200", res.StatusCode)
	}
}

func TestWrongPasswordIsRejectedAndIssuesNoSession(t *testing.T) {
	a := newTestApp(t)

	res := a.post(t, "/api/login", map[string]string{"password": a.password + "-nope"}, nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
	for _, c := range res.Cookies() {
		if c.Name == sessionCookie && c.Value != "" {
			t.Error("a wrong password issued a session cookie")
		}
	}
}

// The SPA shell has to be able to render its own login screen, so the static
// bundle is served without a session. .gitkeep is the one asset guaranteed to
// be embedded in any checkout, built frontend or not.
func TestStaticAssetsAreReachableWithoutASession(t *testing.T) {
	a := newTestApp(t)

	for _, path := range []string{"/", "/.gitkeep"} {
		res, err := http.Get(a.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		res.Body.Close()
		if res.StatusCode == http.StatusUnauthorized {
			t.Errorf("GET %s = 401, want the static bundle to be reachable", path)
		}
	}
}

// Logout clears the cookie. It cannot revoke a copy of it taken beforehand —
// see the comment on handleLogout — so that is all this asserts.
func TestLogoutClearsTheSessionCookie(t *testing.T) {
	a := newTestApp(t)

	if res := a.post(t, "/api/logout", nil, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", res.StatusCode)
	}
	if res := a.get(t, "/api/health", nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /api/health after logout = %d, want 401", res.StatusCode)
	}
}

// A session is only worth as much as its expiry, so this is the test that the
// 30 days in the cookie's Max-Age is also enforced by the server.
func TestAnExpiredSessionIsRejected(t *testing.T) {
	a := newTestApp(t)

	a.moveClock(testClock.Add(sessionTTL).Add(time.Second))

	if res := a.get(t, "/api/health", nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /api/health past the session's expiry = %d, want 401", res.StatusCode)
	}
}

// The cookie carries its own expiry, so the signature is the only thing
// stopping a client from writing itself a longer session — or a session it was
// never issued at all.
func TestATamperedSessionIsRejected(t *testing.T) {
	a := newTestApp(t)

	valid := a.sessionCookieFromLogin(t)
	_, mac, _ := strings.Cut(valid, ".")
	farFuture := strconv.FormatInt(testClock.AddDate(10, 0, 0).Unix(), 10)

	for name, session := range map[string]string{
		"a later expiry with the original signature": farFuture + "." + mac,
		"a corrupted signature":                      flipLastChar(valid),
		"no signature at all":                        farFuture,
		"nonsense":                                   "not-a-session",
	} {
		t.Run(name, func(t *testing.T) {
			res := a.getWithSession(t, "/api/health", session)
			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("GET /api/health with %s = %d, want 401", name, res.StatusCode)
			}
		})
	}
}

// The password must survive a restart: regenerating it on every boot would
// lock the household out of their own data. Asserted the way the household
// would notice — a second app over the same database still takes the password
// the first one generated.
func TestThePasswordSurvivesARestart(t *testing.T) {
	a := newTestApp(t)

	restarted := &testApp{now: testClock, password: a.password}
	restarted.Server = httptest.NewServer(newApp(a.db, restarted.clock))
	defer restarted.Server.Close()
	restarted.useCookies()
	restarted.login(t)

	if res := restarted.get(t, "/api/health", nil); res.StatusCode != http.StatusOK {
		t.Errorf("GET /api/health after a restart = %d, want 200", res.StatusCode)
	}
}

// flipLastChar changes the final character of s. Truncating and appending a
// fixed digit instead would leave the "corrupted" signature identical to the
// original one time in sixteen.
func flipLastChar(s string) string {
	if strings.HasSuffix(s, "0") {
		return s[:len(s)-1] + "1"
	}
	return s[:len(s)-1] + "0"
}

// sessionCookieFromLogin logs in again and returns the raw cookie value, for
// the tests that need to take one apart.
func (a *testApp) sessionCookieFromLogin(t *testing.T) string {
	t.Helper()
	res := a.post(t, "/api/login", map[string]string{"password": a.password}, nil)
	for _, c := range res.Cookies() {
		if c.Name == sessionCookie {
			return c.Value
		}
	}
	t.Fatalf("login set no %s cookie", sessionCookie)
	return ""
}

// getWithSession sends a GET carrying exactly the given session cookie,
// bypassing the harness's jar.
func (a *testApp) getWithSession(t *testing.T, path, session string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, a.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

const passwordPath = settingsPath + "/password"

// Rotating the password needs no redeploy, and the session that rotated it
// keeps working — the household changes the password on the phone it is
// holding, and being logged out by the act of changing it would read as the
// app breaking.
func TestChangingThePasswordTakesEffectAndKeepsTheCallerLoggedIn(t *testing.T) {
	a := newTestApp(t)

	res := a.post(t, passwordPath, map[string]string{
		"current_password": a.password,
		"new_password":     "una password nuova",
	}, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("POST %s = %d, want 204", passwordPath, res.StatusCode)
	}

	if res := a.get(t, "/api/health", nil); res.StatusCode != http.StatusOK {
		t.Errorf("GET /api/health after changing the password = %d, want the caller still logged in", res.StatusCode)
	}

	if res := a.post(t, "/api/login", map[string]string{"password": "una password nuova"}, nil); res.StatusCode != http.StatusNoContent {
		t.Errorf("logging in with the new password = %d, want 204", res.StatusCode)
	}
	if res := a.post(t, "/api/login", map[string]string{"password": a.password}, nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("logging in with the old password = %d, want 401", res.StatusCode)
	}
}

// Every other device is logged out, which is the whole point of rotating a
// shared password: the session held by whoever you are rotating it away from
// stops working.
func TestChangingThePasswordEndsSessionsHeldElsewhere(t *testing.T) {
	a := newTestApp(t)
	elsewhere := a.sessionCookieFromLogin(t)

	a.post(t, passwordPath, map[string]string{
		"current_password": a.password,
		"new_password":     "una password nuova",
	}, nil)

	if res := a.getWithSession(t, "/api/health", elsewhere); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("a session issued before the change = %d, want 401", res.StatusCode)
	}
}

// Holding a session is not the same as knowing the password: an unattended
// phone must not be enough to lock the household out of its own finances.
func TestChangingThePasswordIsRefusedWithoutTheCurrentOne(t *testing.T) {
	a := newTestApp(t)

	for name, body := range map[string]map[string]string{
		"a wrong current password": {"current_password": a.password + "-nope", "new_password": "una password nuova"},
		"no current password":      {"new_password": "una password nuova"},
	} {
		t.Run(name, func(t *testing.T) {
			if res := a.post(t, passwordPath, body, nil); res.StatusCode != http.StatusUnauthorized {
				t.Errorf("POST %s with %s = %d, want 401", passwordPath, name, res.StatusCode)
			}
		})
	}

	if res := a.post(t, "/api/login", map[string]string{"password": a.password}, nil); res.StatusCode != http.StatusNoContent {
		t.Errorf("the old password stopped working after a refused change: %d", res.StatusCode)
	}
}

// A password short enough to guess is worth refusing at the boundary rather
// than in the form, which is the half a determined browser can skip.
func TestATooShortPasswordIsRefused(t *testing.T) {
	a := newTestApp(t)

	res := a.post(t, passwordPath, map[string]string{
		"current_password": a.password,
		"new_password":     strings.Repeat("a", minPasswordLength-1),
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST %s with a short password = %d, want 400", passwordPath, res.StatusCode)
	}

	if res := a.post(t, "/api/login", map[string]string{"password": a.password}, nil); res.StatusCode != http.StatusNoContent {
		t.Errorf("a refused change replaced the password anyway: %d", res.StatusCode)
	}
}

// The rule reads "8 characters", and an accented character an Italian
// household actually types — è, à, ù — is more than one byte. A password
// short on runes but long enough in bytes must be refused exactly like any
// other password under the limit.
func TestAPasswordWithFewerRunesThanBytesIsStillRefused(t *testing.T) {
	a := newTestApp(t)

	res := a.post(t, passwordPath, map[string]string{
		"current_password": a.password,
		"new_password":     "aaaaéé", // 6 runes, 8 bytes
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST %s with 6 runes packed into 8 bytes = %d, want 400", passwordPath, res.StatusCode)
	}
}
