package main

import (
	"net/http"
	"reflect"
	"testing"
)

// The "Dati lavoratori" settings as the settings payload carries them.
type workersJSON struct {
	SelfEmployed              bool     `json:"self_employed"`
	TaxReserveFallbackPercent int64    `json:"tax_reserve_fallback_percent"`
	TaxMonths                 []int    `json:"tax_months"`
	BonusPaychecks            bool     `json:"bonus_paychecks"`
	BonusMonths               []int    `json:"bonus_months"`
	GoalCents                 int64    `json:"goal_cents"`
	Payers                    []string `json:"payers"`
}

func (a *testApp) workers(t *testing.T) workersJSON {
	t.Helper()
	var got workersJSON
	if res := a.get(t, settingsPath, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", settingsPath, res.StatusCode)
	}
	return got
}

// A fresh database: both switches off, 33%, no months.
func TestWorkerSettingsDefaults(t *testing.T) {
	got := newTestApp(t).workers(t)
	if got.SelfEmployed || got.BonusPaychecks || got.TaxReserveFallbackPercent != 33 ||
		len(got.TaxMonths) != 0 || len(got.BonusMonths) != 0 {
		t.Fatalf("defaults = %+v, want switches off, 33%%, no months", got)
	}
}

// They round-trip, months sorted and deduplicated, and a partial update of
// either side leaves the other alone.
func TestWorkerSettingsRoundTripAndPartialUpdates(t *testing.T) {
	a := newTestApp(t)
	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(50000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT goal = %d, want 200", res.StatusCode)
	}
	before := a.workers(t)

	res := a.put(t, settingsPath, map[string]any{
		"self_employed": true, "tax_reserve_fallback_percent": 30, "tax_months": []int{11, 6, 11},
		"bonus_paychecks": true, "bonus_months": []int{12, 6},
	}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT workers = %d, want 200", res.StatusCode)
	}
	got := a.workers(t)
	if !got.SelfEmployed || got.TaxReserveFallbackPercent != 30 || !reflect.DeepEqual(got.TaxMonths, []int{6, 11}) ||
		!got.BonusPaychecks || !reflect.DeepEqual(got.BonusMonths, []int{6, 12}) {
		t.Fatalf("after PUT = %+v", got)
	}
	if got.GoalCents != 50000 || !reflect.DeepEqual(got.Payers, before.Payers) {
		t.Errorf("worker settings disturbed the rest: goal %d, payers %v", got.GoalCents, got.Payers)
	}

	// The reverse: a Goal-only update leaves the worker settings alone.
	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(1)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT goal = %d, want 200", res.StatusCode)
	}
	if after := a.workers(t); !after.SelfEmployed || !reflect.DeepEqual(after.TaxMonths, []int{6, 11}) {
		t.Errorf("a Goal update disturbed the worker settings: %+v", after)
	}
}

func TestWorkerSettingsRefuseOutOfRange(t *testing.T) {
	a := newTestApp(t)
	for name, body := range map[string]map[string]any{
		"percent above 100": {"tax_reserve_fallback_percent": 101},
		"negative percent":  {"tax_reserve_fallback_percent": -1},
		"month 13":          {"tax_months": []int{13}},
		"month 0":           {"bonus_months": []int{0}},
	} {
		if res := a.put(t, settingsPath, body, nil); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: PUT = %d, want 400", name, res.StatusCode)
		}
	}
	if got := a.workers(t); got.TaxReserveFallbackPercent != 33 {
		t.Errorf("a refused PUT changed the percentage to %d", got.TaxReserveFallbackPercent)
	}
}
