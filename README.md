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

## First run

The app generates a shared password the first time it starts and logs it once:

    first run: the shared password is HCQJS3BE… — write it down

Write it down: it is logged once and never shown again. If it is lost,
`DELETE FROM setting WHERE key = 'password_hash'` and restart to get a new one.

## Development

`./scripts/build.sh` builds the frontend into `cmd/static/`, which the binary embeds, then
compiles both binaries into the gitignored `build/`: `build/api-armv6` is the
single file to copy to the Pi, and `build/sonic-finance-pi` is the same code
for this machine. The dev loop — the Go server and the Vite dev server side by
side — is in `web/README.md`.


### Localhost testing

Run the command `pnpm --dir web dev` for the Vite app, run the command `go run ./cmd` for the Go server.

## Deploying, and TLS

See [docs/guides/PI_DEPLOY_GUIDE.md](docs/guides/PI_DEPLOY_GUIDE.md) — publishing a release deploys
itself; that file also covers the manual fallback and setting up HTTPS.
