package main

import (
	"net/http"
	"strconv"
	"testing"
)

// A Reminder as the API hands it out. set_for_month is deliberately not part
// of this shape: it is bookkeeping for the collapse rule, not something a
// screen needs to read (ticket 06).
type reminderJSON struct {
	ID      int64  `json:"id"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

func reminderPath(id int64) string {
	return "/api/reminders/" + strconv.FormatInt(id, 10)
}

func (a *testApp) reminders(t *testing.T) []reminderJSON {
	t.Helper()
	var got []reminderJSON
	if res := a.get(t, "/api/reminders", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/reminders = %d, want 200", res.StatusCode)
	}
	return got
}

func (a *testApp) addReminder(t *testing.T, label string) reminderJSON {
	t.Helper()
	var created reminderJSON
	res := a.post(t, "/api/reminders", map[string]string{"label": label}, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/reminders %q = %d, want 201", label, res.StatusCode)
	}
	return created
}

// TestReminderCollapsesOnceTheMonthRolls is the ticket's whole point, written
// first: enabling a Reminder stamps the current month, and a later read in a
// new month has to read it as off again with nothing else touching the row —
// same clock-controlled shape as materialise (ADR-0005).
func TestReminderCollapsesOnceTheMonthRolls(t *testing.T) {
	a := newTestApp(t)
	rem := a.addReminder(t, "Paid the rent transfer")

	var toggled reminderJSON
	res := a.patch(t, reminderPath(rem.ID), map[string]bool{"enabled": true}, &toggled)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", reminderPath(rem.ID), res.StatusCode)
	}
	if !toggled.Enabled {
		t.Fatal("enabled = false right after toggling on")
	}

	// Still the same month: it should still read as on.
	list := a.reminders(t)
	if !list[0].Enabled {
		t.Error("enabled = false within the same month it was toggled on")
	}

	// The clock rolls into next month with nothing else touching the row.
	a.setNow(t, testClock.AddDate(0, 1, 0))

	list = a.reminders(t)
	if len(list) != 1 {
		t.Fatalf("got %d reminders, want 1", len(list))
	}
	if list[0].Enabled {
		t.Error("enabled = true a month after it was set — should have collapsed to off")
	}
	if list[0].Label != "Paid the rent transfer" {
		t.Errorf("label = %q, want %q — collapsing must not touch it", list[0].Label, "Paid the rent transfer")
	}
}

func TestCreateListDeleteReminder(t *testing.T) {
	a := newTestApp(t)

	if got := a.reminders(t); len(got) != 0 {
		t.Fatalf("got %d reminders on a fresh household, want 0", len(got))
	}

	first := a.addReminder(t, "Paid the rent transfer")
	if first.Enabled {
		t.Error("a freshly created reminder reads as enabled, want off")
	}
	second := a.addReminder(t, "Paid the gym membership")

	list := a.reminders(t)
	if len(list) != 2 {
		t.Fatalf("got %d reminders, want 2", len(list))
	}

	res := a.delete(t, reminderPath(first.ID))
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", reminderPath(first.ID), res.StatusCode)
	}

	list = a.reminders(t)
	if len(list) != 1 || list[0].ID != second.ID {
		t.Fatalf("got %+v after deleting the first, want only %+v left", list, second)
	}
}

// TestCreateReminderRejectsBlankLabel matches how every other create in this
// codebase refuses an empty name (client.validate, holding.validate).
func TestCreateReminderRejectsBlankLabel(t *testing.T) {
	a := newTestApp(t)
	res := a.post(t, "/api/reminders", map[string]string{"label": "   "}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/reminders with a blank label = %d, want 400", res.StatusCode)
	}
}

// TestToggleReminderOff proves disabling is a plain flip, with no month logic
// standing in the way — the row was never claiming to be on, and turning it
// off should not require an on month to already be current.
func TestToggleReminderOff(t *testing.T) {
	a := newTestApp(t)
	rem := a.addReminder(t, "Paid the rent transfer")
	a.patch(t, reminderPath(rem.ID), map[string]bool{"enabled": true}, nil)

	var toggled reminderJSON
	res := a.patch(t, reminderPath(rem.ID), map[string]bool{"enabled": false}, &toggled)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", reminderPath(rem.ID), res.StatusCode)
	}
	if toggled.Enabled {
		t.Error("enabled = true right after toggling off")
	}
}

// TestRenameReminderLeavesTheToggleAlone is why the PATCH body is pointers: a
// rename carries no enabled field, and must not read as "and turn it off".
func TestRenameReminderLeavesTheToggleAlone(t *testing.T) {
	a := newTestApp(t)
	rem := a.addReminder(t, "Paid the rent transfer")
	a.patch(t, reminderPath(rem.ID), map[string]bool{"enabled": true}, nil)

	var renamed reminderJSON
	res := a.patch(t, reminderPath(rem.ID), map[string]string{"label": "  Rent transfer  "}, &renamed)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", reminderPath(rem.ID), res.StatusCode)
	}
	if renamed.Label != "Rent transfer" {
		t.Errorf("label = %q, want %q trimmed", renamed.Label, "Rent transfer")
	}
	if !renamed.Enabled {
		t.Error("enabled = false after a rename that never mentioned it")
	}

	list := a.reminders(t)
	if list[0].Label != "Rent transfer" || !list[0].Enabled {
		t.Errorf("list reads %+v after the rename, want the new label still enabled", list[0])
	}

	// And a blank rename is refused the same way a blank create is.
	if res := a.patch(t, reminderPath(rem.ID), map[string]string{"label": " "}, nil); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH %s with a blank label = %d, want 400", reminderPath(rem.ID), res.StatusCode)
	}
}
