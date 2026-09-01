# 01: Database, migrations, and the test harness

**What to build:** The binary opens its SQLite database, brings the schema up to date by itself, and answers a request. From here on, deploying a new binary to the Pi is all it takes to update the schema on the only copy of the data — no SSH surgery.

This ticket also establishes the testing pattern every later ticket copies, so build it as the reference.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] The database opens with `journal_mode = WAL` and `synchronous = NORMAL`, per the SD-card longevity concern in the README
- [ ] Schema is versioned with `PRAGMA user_version` and applied by a switch statement at startup — no migration library
- [ ] A fresh database reaches the current version on first run; starting twice in a row is a no-op
- [ ] The app is constructed as `newApp(db, now)` with the clock injected, never read from the wall clock inside handlers
- [ ] A health endpoint returns the schema version and the server's current time
- [ ] `newTestApp(t)` gives a test a running app over a real SQLite database in a temp dir, migrated by the same code production uses, with a controllable clock
- [ ] At least one test asserts against the health endpoint, demonstrating the pattern
