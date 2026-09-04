# 06: Dashboard reminders

**What to build:** A household-managed list of labeled on/off Reminders on the Dashboard, for manual actions the app doesn't automate, each resetting to off at the start of the next month.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] A Reminder can be created with a short label, a household can have more than one at a time, and a Reminder can be deleted.
- [x] A Reminder can be toggled on once the labeled action is done.
- [x] A Reminder reads as off again starting on the first of the next calendar month, read off the browser's clock with no scheduler involved (same pattern as Recurring expense generation).
- [x] The Dashboard shows the current list of Reminders as toggles.
- [x] Backend tests cover a Reminder collapsing to disabled once the month rolls over, using the existing clock-controlled test pattern.

## Comments

Implemented as `cmd/reminder.go` (CRUD + read-time `collapse` against `now()`, no scheduler) with routes in `cmd/app.go`, tests in `cmd/reminder_test.go` using the existing `testApp`/`setNow` clock injection; frontend adds a self-contained `RemindersCard` (create/toggle/delete inline) to `Dashboard.tsx`, a `Reminder` type in `types.ts`, strings in `strings.ts`, and pulls in shadcn's `Switch` component via the CLI.
