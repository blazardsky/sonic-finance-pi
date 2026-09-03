package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	passwordHashKey = "password_hash"
	sessionCookie   = "sonic_session"
	sessionTTL      = 30 * 24 * time.Hour
)

// bcryptCost is the one knob worth leaving on this: DefaultCost costs a tenth
// of a second on a laptop and a slow second or so on the Pi Zero W's ARMv6,
// paid once a month when the session expires. The test suite turns it down to
// MinCost, because every test logs in.
var bcryptCost = bcrypt.DefaultCost

// ensurePassword makes sure there is a password to log in with, generating one
// on first run and returning it so the caller can show it to whoever is
// setting the Pi up. Returns "" when a password was already set: regenerating
// on every boot would lock the household out of their own data.
func ensurePassword(db *sql.DB) (string, error) {
	hash, err := getSetting(db, passwordHashKey)
	if err != nil || hash != "" {
		return "", err
	}
	pw := rand.Text()
	hashed, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		return "", err
	}
	return pw, setSetting(db, passwordHashKey, string(hashed))
}

// passwordHash reads the stored bcrypt hash, and owns the rule that a missing
// one is an error rather than an empty string: nothing downstream — checking a
// password, deriving a signing key — is safe to do without it.
func passwordHash(db *sql.DB) (string, error) {
	hash, err := getSetting(db, passwordHashKey)
	if err != nil {
		return "", err
	}
	if hash == "" {
		return "", errors.New("no password is set")
	}
	return hash, nil
}

// sessionKey derives the cookie-signing key from the stored password hash.
// There is no second secret to generate, store, or keep in sync, and changing
// the password invalidates every outstanding session — which, for one shared
// password, is the behaviour you want.
//
// ponytail: one small SELECT per API request. If that ever shows up on the Pi,
// cache the key in newApp and invalidate it when the password changes.
// Never sign with a key derived from an empty hash: it would be the same
// constant on every install, and anyone could mint a session. passwordHash
// refusing that case is what stops it.
func sessionKey(db *sql.DB) ([]byte, error) {
	hash, err := passwordHash(db)
	if err != nil {
		return nil, err
	}
	k := sha256.Sum256([]byte(hash))
	return k[:], nil
}

// A session cookie is its own expiry plus an HMAC over it, "<unix>.<hex>".
// Nothing is stored server-side, so there is no session table to grow or
// prune, and restarting the Pi does not log the household out.
func sessionValue(key []byte, expiry time.Time) string {
	exp := strconv.FormatInt(expiry.Unix(), 10)
	return exp + "." + sessionMAC(key, exp)
}

func sessionMAC(key []byte, expiry string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(expiry))
	return hex.EncodeToString(m.Sum(nil))
}

func validSession(db *sql.DB, value string, now time.Time) bool {
	exp, mac, ok := strings.Cut(value, ".")
	if !ok {
		return false
	}
	key, err := sessionKey(db)
	if err != nil {
		log.Printf("checking session: %v", err)
		return false
	}
	if !hmac.Equal([]byte(mac), []byte(sessionMAC(key, exp))) {
		return false
	}
	unix, err := strconv.ParseInt(exp, 10, 64)
	return err == nil && now.Before(time.Unix(unix, 0))
}

// requireSession gates every /api/ route except login. It wraps the whole mux
// rather than sitting on a sub-mux of authenticated routes, so a route added
// by a later ticket is protected by default — forgetting to opt in is the
// failure mode worth designing out. Static assets fall through: the SPA shell
// has to be reachable to render its own login screen.
func requireSession(db *sql.DB, now func() time.Time, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/login" {
			c, err := r.Cookie(sessionCookie)
			if err != nil || !validSession(db, c.Value, now()) {
				writeError(w, http.StatusUnauthorized, errors.New("no valid session"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func handleLogin(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Password string `json:"password"`
		}
		// The only route reachable without a session, on a box with 512MB of
		// RAM: the body it will read is capped rather than trusted.
		r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		hash, err := passwordHash(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}

		if err := issueSession(w, db, now); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// issueSession sets a fresh session cookie, signed with the current password
// hash. Login and a password change both end in exactly this, and a password
// change needs its own fresh cookie anyway — the one the caller arrived with
// was signed with the hash that just changed.
func issueSession(w http.ResponseWriter, db *sql.DB, now func() time.Time) error {
	key, err := sessionKey(db)
	if err != nil {
		return err
	}
	expiry := now().Add(sessionTTL)
	setSessionCookie(w, &http.Cookie{
		Name:    sessionCookie,
		Value:   sessionValue(key, expiry),
		Expires: expiry,
		MaxAge:  int(sessionTTL.Seconds()),
	})
	return nil
}

// handleLogout clears the cookie, which is all a stateless session can be
// asked to do: a copy of it captured beforehand stays valid until its expiry.
// Revoking one would mean a server-side session store, which for one shared
// password on a household Pi buys less than it costs — change the password and
// every session dies at once (see sessionKey).
func handleLogout(w http.ResponseWriter, r *http.Request) {
	setSessionCookie(w, &http.Cookie{Name: sessionCookie, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

// setSessionCookie fills in the attributes that must match between issuing a
// session and clearing one — a cookie only clears if they do.
//
// There is no Secure flag: the app serves plain HTTP over Tailscale, which
// supplies the transport encryption, and a Secure cookie would never be sent
// at all. See ADR-0007.
func setSessionCookie(w http.ResponseWriter, c *http.Cookie) {
	c.Path = "/"
	c.HttpOnly = true
	c.SameSite = http.SameSiteLaxMode
	http.SetCookie(w, c)
}

// minPasswordLength is the one rule on a new password. There is no complexity
// policy: one shared password on a tailnet-only Pi, typed by two people, and a
// rule they resent would be worked around with something worse.
const minPasswordLength = 8

// handleChangePassword rotates the shared password without a redeploy. The
// current one is required even though the caller already holds a session: an
// unattended phone is the realistic threat here, and being locked out of your
// own household finances by someone who walked past your desk is the outcome
// worth one extra field.
//
// Changing it invalidates every outstanding session, because sessionKey is
// derived from the hash — including the session that made this request, so a
// fresh cookie is issued before answering. Every other device is logged out,
// which is exactly what rotating a shared password is for.
func handleChangePassword(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		hash, err := passwordHash(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.CurrentPassword)); err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		// RuneCountInString, not len: len counts bytes, and an accented
		// character an Italian household actually types — è, à, ù — is
		// multiple bytes but one character. Counting bytes would silently
		// admit a password shorter than the rule this checks and the message
		// below both claim.
		if utf8.RuneCountInString(body.NewPassword) < minPasswordLength {
			writeInvalid(w, fmt.Errorf("a password needs at least %d characters", minPasswordLength))
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcryptCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := setSetting(db, passwordHashKey, string(hashed)); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if err := issueSession(w, db, now); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
