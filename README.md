# Sonic Finance | GO Backend API + Vite Frontend

This is the code for the Sonic Finance App, a simple expenses and icomes tracker for households (multiple people, shared space). 

## Stack

- Go 1.27
- modernc.org/sqlite
- vite
- react

## Target

Raspberry PI Zero W 1st gen | ARMv6

---

To prevent SD card corruption and prolong lifespan: Enable Write-Ahead Logging (PRAGMA journal_mode = WAL;) and PRAGMA synchronous = NORMAL; in SQLite.