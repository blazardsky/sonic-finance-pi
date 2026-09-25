package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
)

// The "Dati lavoratori" settings (spec: .scratch/tax-reserve): whether the
// household has someone self-employed — which is what turns the Tax reserve
// (CONTEXT.md) on — the rate to reserve at until a Tax year has completed,
// and the months two kinds of reminder are shown for. They ride the settings
// payload like Target and Goal; nothing here needs a table of its own.
const (
	selfEmployedKey              = "self_employed"
	taxReserveFallbackPercentKey = "tax_reserve_fallback_percent"
	taxMonthsKey                 = "tax_months"
	bonusPaychecksKey            = "bonus_paychecks"
	bonusMonthsKey               = "bonus_months"
)

// defaultTaxReserveFallbackPercent is roughly what a self-employed household
// pays on what it earns — only a starting point, until its own completed
// Tax years give a rate of its own.
const defaultTaxReserveFallbackPercent = 33

// workers is embedded in lists, so its fields sit on the same payload.
// Switching either switch off keeps the values behind it: they come back as
// they were when it is switched on again.
type workers struct {
	SelfEmployed              bool  `json:"self_employed"`
	TaxReserveFallbackPercent int64 `json:"tax_reserve_fallback_percent"`
	TaxMonths                 []int `json:"tax_months"`
	BonusPaychecks            bool  `json:"bonus_paychecks"`
	BonusMonths               []int `json:"bonus_months"`
}

func readWorkers(db *sql.DB) (workers, error) {
	var w workers
	var err error
	if w.SelfEmployed, err = getSettingBool(db, selfEmployedKey); err != nil {
		return w, err
	}
	if w.TaxReserveFallbackPercent, err = getSettingInt(db, taxReserveFallbackPercentKey, defaultTaxReserveFallbackPercent); err != nil {
		return w, err
	}
	if w.TaxMonths, err = getMonths(db, taxMonthsKey); err != nil {
		return w, err
	}
	if w.BonusPaychecks, err = getSettingBool(db, bonusPaychecksKey); err != nil {
		return w, err
	}
	w.BonusMonths, err = getMonths(db, bonusMonthsKey)
	return w, err
}

// tidy validates what was submitted and puts the month sets in order: sorted,
// each month once. A percentage outside 0–100 or a month outside 1–12 is a
// bad request rather than something to clamp silently.
func (w *workers) tidy() error {
	if w.TaxReserveFallbackPercent < 0 || w.TaxReserveFallbackPercent > 100 {
		return errors.New("tax_reserve_fallback_percent must be between 0 and 100")
	}
	for _, months := range []*[]int{&w.TaxMonths, &w.BonusMonths} {
		if *months == nil {
			// A body sending null means "none"; answering null would leave the
			// screen with no list to look months up in.
			*months = []int{}
		}
		for _, m := range *months {
			if m < 1 || m > 12 {
				return errors.New("months must be between 1 and 12")
			}
		}
		slices.Sort(*months)
		*months = slices.Compact(*months)
	}
	return nil
}

func putWorkers(db *sql.DB, w workers) error {
	if err := putSettingBool(db, selfEmployedKey, w.SelfEmployed); err != nil {
		return err
	}
	if err := setSetting(db, taxReserveFallbackPercentKey, strconv.FormatInt(w.TaxReserveFallbackPercent, 10)); err != nil {
		return err
	}
	if err := putMonths(db, taxMonthsKey, w.TaxMonths); err != nil {
		return err
	}
	if err := putSettingBool(db, bonusPaychecksKey, w.BonusPaychecks); err != nil {
		return err
	}
	return putMonths(db, bonusMonthsKey, w.BonusMonths)
}

// getSettingInt reads a whole-number setting, or def when it has never been
// set — unlike getSettingCents, whose unset value is always 0.
func getSettingInt(db *sql.DB, key string, def int64) (int64, error) {
	v, err := getSetting(db, key)
	if err != nil || v == "" {
		return def, err
	}
	return strconv.ParseInt(v, 10, 64)
}

// A month set is stored as a JSON array of 1–12, like the label lists; never
// set reads as empty rather than null.
func getMonths(db *sql.DB, key string) ([]int, error) {
	raw, err := getSetting(db, key)
	if err != nil || raw == "" {
		return []int{}, err
	}
	out := []int{}
	return out, json.Unmarshal([]byte(raw), &out)
}

func putMonths(db *sql.DB, key string, months []int) error {
	if months == nil {
		months = []int{}
	}
	encoded, err := json.Marshal(months)
	if err != nil {
		return err
	}
	return setSetting(db, key, string(encoded))
}
