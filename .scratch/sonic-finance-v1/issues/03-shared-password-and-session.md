# 03: Shared password and session

**What to build:** The household's finances stop being readable by anything that reaches the Pi. One shared password gets you in, and you stay in for a month.

Landing this early rather than late means no test written afterwards has to be retrofitted for auth.

**Blocked by:** 01, 02

**Status:** ready-for-human

- [x] A single shared password, bcrypt-hashed in settings, no username — see ADR-0007
- [x] The password is set on first run
- [x] Logging in issues a signed session cookie lasting 30 days, `HttpOnly` and `SameSite=Lax`
- [x] Every `/api/` route except login rejects unauthenticated requests
- [x] Static assets stay reachable without a session, so the app can render its own login screen
- [x] The login screen is in Italian and reports a wrong password clearly
- [x] `newTestApp(t)` returns an authenticated client, so later tickets never think about this again
- [x] Tests cover: unauthenticated API call rejected, login then succeeds, static assets always reachable

## Comments

Implemented. `auth.go` holds the whole mechanism, `setting.go` the two key/value helpers it needed.

- **No session store.** The cookie is `<expiry unix>.<HMAC-SHA256 over that expiry>`, so there is no session table to grow or prune and restarting the Pi does not log the household out. `HttpOnly`, `SameSite=Lax`, `Path=/`, `Max-Age` 30 days. No `Secure` flag: the app serves plain HTTP over Tailscale, and a `Secure` cookie would simply never be sent (ADR-0007).
- **The signing key is derived from the password hash** — `sha256(bcrypt hash)` — rather than being a second secret to generate, store and keep in sync. It falls out of that choice that changing the password logs every session out, which for one shared password is the behaviour you want. It costs one small indexed `SELECT` per API request; marked with a `ponytail:` comment naming the cache as the upgrade path if the Pi ever notices.
- **First run generates the password** and logs it once: `first run: the shared password is X2DD25WA… — write it down, it is logged once`. `crypto/rand.Text()` does the generating, so there is no encoding code. Set-up needs no env var, no config file and no insecure window where any password would be accepted. Recovery, if the line is missed, is `DELETE FROM setting WHERE key = 'password_hash'` and a restart, which the README now documents. The log line deliberately does **not** say "change it from the settings screen": `POST /api/settings/password` is ticket 16's and does not exist yet, so that would have been a false instruction.
- **The gate wraps the whole mux** rather than sitting on a sub-mux of protected routes, so every route a later ticket adds is protected by default. Opting *out* is the deliberate act; forgetting to opt in is the failure mode worth designing out. `/api/login` is the one exemption, matched by exact path. Static assets fall through untouched. Path-normalisation bypasses were probed (`//api/health`, `/x/../api/health`, `/%61pi/health`, `/api%2fhealth`, `/API/health`, absolute-form URIs): every variant either 401s or is redirected by `ServeMux` before reaching a handler, because `ServeMux` redirects unclean paths rather than serving them.
- `/api/health` is now behind the gate like every other `/api/` route, and the frontend uses its 401 as the session check — no separate "am I logged in" endpoint.
- **Login is the only unauthenticated endpoint**, so its body is capped with `http.MaxBytesReader` at 4KB rather than trusted. The Pi Zero W has 512MB of RAM and `net/http` imposes no body limit of its own.
- **`/api/logout` clears the cookie and nothing more.** A copy captured beforehand stays valid until its expiry — inherent to a stateless session, and revoking one would mean the server-side store this design exists to avoid. Documented at `handleLogout`, and the test is named `TestLogoutClearsTheSessionCookie` so it does not claim more than it proves. Changing the password is the way to kill every session at once.
- **`bcryptCost` is a package variable.** `DefaultCost` is a tenth of a second on a laptop and a slow second or so on ARMv6, paid once a month when the session expires; the suite turns it down to `MinCost` in `TestMain`, because every test logs in. It stays a knob rather than a constant because the right cost is a property of the hardware, not of the code.
- **`newTestApp(t)` returns a logged-in client**: it runs the real first-run path, gives the client a cookie jar, and POSTs `/api/login`. `setNow(t, …)` moves the clock *and* logs back in — otherwise ticket 13's month-jumping tests would start collecting 401s for reasons unrelated to what they test — with `moveClock` underneath it for the auth tests, which are precisely about a session outliving the clock. `post(t, path, body, dst)` joins `get` on the harness.
- **Tests**: unauthenticated `/api/` rejected, login then succeeds, static assets always reachable (the ticket's three), plus wrong password rejected with no cookie, cookie attributes, logout, an expired session rejected, four flavours of tampered cookie rejected — including an expiry rewritten with the original signature, the forgery the design has to stop — and the password surviving a restart. That last one is asserted the way the household would notice it, a second app over the same database, rather than by calling `ensurePassword` twice.
- **Frontend**: `src/Login.tsx` uses shadcn's `Input`, added with `pnpm dlx shadcn@latest add input` per `web/README.md`, rather than a hand-rolled `<input>` re-deriving shadcn's focus-ring classes. Wrong password renders `Password errata. Riprova.` from `strings.ts` in a `role="alert"`; a dead server is reported separately. `App.tsx` is a three-state machine — `checking` renders neither the shell nor the login screen, so a cold load no longer flashes the authed UI before `/api/health` answers 401.
- **Verified against the real shipped binary**, not only the tests: unauthenticated `/api/health` 401, `GET /` and `GET /.gitkeep` 200, the Italian login strings present in the built bundle, wrong password 401, right password 204 with `HttpOnly; SameSite=Lax; Max-Age=2592000`, authenticated `/api/health` 200, a 20KB login body 400, a forged cookie 401, and 401 again after logout. The suite is clean under `-race` and over ten consecutive runs — one 1-in-16 flake was found and fixed on the way (a "corrupted" signature built by appending a fixed digit is the original signature when it already ends in that digit).
- **Not built, deliberately**: login rate-limiting and TLS, both deferred by the spec until the app is reachable from the public internet (ADR-0007). No `POST /api/settings/password` — ticket 16's, and the hashing it needs is four lines inside `ensurePassword` to lift out.
- **Unrelated change carried in the same commit, on request**: `build.sh` now writes both binaries into a gitignored `build/` (`build/api-armv6` for the Pi, `build/sonic-finance-pi` for this machine) instead of leaving them beside the source, and the module path was renamed to `github.com/blazardsky/sonic-finance-pi`.
