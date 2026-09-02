# 01: Database, migrations, and the test harness

**What to build:** The binary opens its SQLite database, brings the schema up to date by itself, and answers a request. From here on, deploying a new binary to the Pi is all it takes to update the schema on the only copy of the data — no SSH surgery.

This ticket also establishes the testing pattern every later ticket copies, so build it as the reference.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [x] The database opens with `journal_mode = WAL` and `synchronous = NORMAL`, per the SD-card longevity concern in the README
- [x] Schema is versioned with `PRAGMA user_version` and applied by a switch statement at startup — no migration library
- [x] A fresh database reaches the current version on first run; starting twice in a row is a no-op
- [x] The app is constructed as `newApp(db, now)` with the clock injected, never read from the wall clock inside handlers
- [x] A health endpoint returns the schema version and the server's current time
- [x] `newTestApp(t)` gives a test a running app over a real SQLite database in a temp dir, migrated by the same code production uses, with a controllable clock
- [x] At least one test asserts against the health endpoint, demonstrating the pattern

## Comments

Implemented. `openDB`/`migrate`/`migrateStep` in `migrate.go`, `newApp` and `/api/health` in `app.go`, harness in `app_test.go`.

- Schema version 1 creates only the `setting` table. Later tickets bump `schemaVersion` and add their own case, per the spec's "each schema change bumps the version and adds a case". `setting` is first because it has no dependencies and ticket 03 needs it.
- The WAL/synchronous pragmas live in the DSN, not a post-open `Exec`, so they bind every pooled connection rather than whichever one ran the statement.
- `foreign_keys(ON)` and `busy_timeout(5000)` were added alongside them, beyond what this ticket asked for: the spec's `category_id` foreign key is inert without the former, and the latter is what stops a concurrent write returning `SQLITE_BUSY`.
- `migrateStep` has a `default:` that refuses a version with no case. Without it, bumping `schemaVersion` and forgetting the case would commit the version bump with no DDL and permanently mark the only copy of the data as migrated.
- `app_test.go` is the reference every later ticket copies and stays purely on the HTTP seam. `migrate_test.go` is the one deliberate exception, because "opens in WAL" and "re-running startup keeps existing rows" have no HTTP surface.
- `build.sh` now builds the package rather than `main.go` alone. Its frontend step belongs to ticket 02.
