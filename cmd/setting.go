package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// The setting table is a plain key/value store: the password hash, and the
// Payer and Payment method lists. Keys are constants next to the code that
// owns them rather than collected here, so an area's key lives with its area.

// getSetting returns the value stored under key, or "" when there is none.
func getSetting(db *sql.DB, key string) (string, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM setting WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO setting (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// getSettingCents reads a setting stored as a whole number of cents,
// defaulting to 0 when it has never been set. Unlike Target (cmd/budget.go),
// nothing here has a stickiness rule of its own to protect — this is the
// plain reader any cents-shaped setting can use.
func getSettingCents(db *sql.DB, key string) (int64, error) {
	v, err := getSetting(db, key)
	if err != nil || v == "" {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func putSettingCents(db *sql.DB, key string, cents int64) error {
	return setSetting(db, key, strconv.FormatInt(cents, 10))
}

// settingsPath is the endpoint the spec's API section gives the editable
// settings: today the two label lists, read together by the Expense form and
// written together by the settings screen.
const settingsPath = "/api/settings"

const (
	payersKey         = "payers"
	paymentMethodsKey = "payment_methods"
)

// The two Payers that are not people. A shared bill and a payment somebody
// outside the household made must not need a fake person invented for them.
const (
	seedPayerBoth        = "Entrambi"
	seedPayerSomeoneElse = "Qualcun altro"
)

// What a fresh database's lists hold. The two people are placeholders on
// purpose — they are the household's own names to type, and the settings
// screen is where they type them.
var (
	seedPayers         = []string{"Nicco", "Sofi", seedPayerBoth, seedPayerSomeoneElse}
	seedPaymentMethods = []string{"Contanti", "Carta di credito", "Bancomat"}
)

// lists is the pair of short configured label lists the Expense form picks
// from. They are labels and nothing else — ADR-0001: an Expense stores the
// text, not a reference, so what is here shapes new entries and never touches
// old ones.
type lists struct {
	Payers         []string `json:"payers"`
	PaymentMethods []string `json:"payment_methods"`

	// Target and Goal (ticket 04) ride the same payload rather than a second
	// endpoint. Target's default-then-sticky behaviour lives in
	// cmd/budget.go, the one place with a Budget to default it from — this
	// struct only round-trips whatever is already stored, same as the two
	// lists above.
	TargetCents int64 `json:"target_cents"`
	GoalCents   int64 `json:"goal_cents"`
}

// migrateLists is schema step 3: the seed values for both lists. Seeding
// inside the migration is what makes them the household's own from then on —
// it runs once, so nothing a later edit removes ever comes back.
func migrateLists(tx *sql.Tx) error {
	for _, l := range []struct {
		key    string
		values []string
	}{{payersKey, seedPayers}, {paymentMethodsKey, seedPaymentMethods}} {
		encoded, err := json.Marshal(l.values)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO setting (key, value) VALUES (?, ?)`, l.key, string(encoded)); err != nil {
			return err
		}
	}
	return nil
}

// A list is stored as a JSON array rather than separated text because a Payer
// is free-form once the household renames it, and any separator worth choosing
// is a character somebody could type.
func getList(db *sql.DB, key string) ([]string, error) {
	raw, err := getSetting(db, key)
	if err != nil {
		return nil, err
	}
	var out []string
	return out, json.Unmarshal([]byte(raw), &out)
}

func putList(db *sql.DB, key string, values []string) error {
	encoded, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return setSetting(db, key, string(encoded))
}

func readLists(db *sql.DB) (lists, error) {
	var l lists
	var err error
	if l.Payers, err = getList(db, payersKey); err != nil {
		return l, err
	}
	if l.PaymentMethods, err = getList(db, paymentMethodsKey); err != nil {
		return l, err
	}
	if l.TargetCents, err = getSettingCents(db, targetCentsKey); err != nil {
		return l, err
	}
	l.GoalCents, err = getSettingCents(db, goalCentsKey)
	return l, err
}

func handleGetLists(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, err := readLists(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, l)
	}
}

// handlePutLists replaces the lists. Decoding onto what is stored is what
// makes it forgiving: a key the body omits is left at the value it was read
// with, and one it sends replaces that list entirely.
func handlePutLists(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, err := readLists(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := decodeJSON(w, r, &l); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		l.Payers = trimmedLabels(l.Payers)
		l.PaymentMethods = trimmedLabels(l.PaymentMethods)
		// A list nothing can be chosen from is the one edit worth refusing:
		// it leaves a picker with no options and no way back to a Payer.
		if len(l.Payers) == 0 || len(l.PaymentMethods) == 0 {
			writeError(w, http.StatusBadRequest, errors.New("a list needs at least one entry"))
			return
		}

		if err := putList(db, payersKey, l.Payers); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := putList(db, paymentMethodsKey, l.PaymentMethods); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := putSettingCents(db, targetCentsKey, l.TargetCents); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := putSettingCents(db, goalCentsKey, l.GoalCents); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, l)
	}
}

// trimmedLabels tidies a submitted list: trimmed, and without the blank rows
// an empty input on the settings screen would otherwise add.
func trimmedLabels(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
