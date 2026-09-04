# 06: Dashboard reminders

**What to build:** A household-managed list of labeled on/off Reminders on the Dashboard, for manual actions the app doesn't automate, each resetting to off at the start of the next month.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] A Reminder can be created with a short label, a household can have more than one at a time, and a Reminder can be deleted.
- [ ] A Reminder can be toggled on once the labeled action is done.
- [ ] A Reminder reads as off again starting on the first of the next calendar month, read off the browser's clock with no scheduler involved (same pattern as Recurring expense generation).
- [ ] The Dashboard shows the current list of Reminders as toggles.
- [ ] Backend tests cover a Reminder collapsing to disabled once the month rolls over, using the existing clock-controlled test pattern.
