# Sonic Finance | GO Backend API

This is the code for the go backend used for the Sonic Finance App.
Sonic Finance is a simple expenses and icomes tracker for households (multiple people, shared space)

## Stack

- Go 1.27
- modernc.org/sqlite

## Target

Raspberry PI Zero W 1st gen | ARMv6

---

To prevent SD card corruption and prolong lifespan: Enable Write-Ahead Logging (PRAGMA journal_mode = WAL;) and PRAGMA synchronous = NORMAL; in SQLite.