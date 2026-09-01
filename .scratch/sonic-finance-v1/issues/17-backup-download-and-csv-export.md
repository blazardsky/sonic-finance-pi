# 17: Backup download and CSV export

**What to build:** Three years of expenses stop living on one SD card with no copy. Backing up becomes a button, and leaving this app is always possible — the escape hatch the spreadsheet never had.

**Blocked by:** 06, 09

**Status:** ready-for-agent

- [ ] A download button streams a consistent copy of the database, taken with `VACUUM INTO` so it is safe while the app is running
- [ ] The temporary copy is removed after streaming
- [ ] CSV export for Expenses and for Incomes
- [ ] Items appear in the expense CSV on their own rows, referencing their Expense
- [ ] Copying the backup off the Pi stays manual; scheduled off-device backups are out of scope
