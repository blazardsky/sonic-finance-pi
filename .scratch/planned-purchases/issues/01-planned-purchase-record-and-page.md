# 01: Planned purchase record and page

**What to build:** A new "Acquisti programmati" page in the nav, holding the household's priority-ordered list of Planned purchases. Add, edit and delete go through the actions sidebar, like every other record; up/down controls on each row change the priority. No projection yet: this ticket makes the list exist and persist.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] `schemaVersion` bumps from 22 to 23 with the `planned_purchase` table from the spec's Schema section
- [x] A Planned purchase has a label (required), an amount (required, > 0) and an optional Category
- [x] "Acquisti programmati" appears in the nav and opens the page
- [x] The page lists Planned purchases in priority order; a new one goes to the bottom
- [x] Up/down controls move a Planned purchase one place; the first can't move up, the last can't move down; the order survives a reload
- [x] Edit and delete work through the actions sidebar and row actions, like the other record pages
- [x] Tests: create/update/delete round-trip; new rows go to the bottom; reorder swaps positions and rejects moving past either end; invalid amount/label rejected
