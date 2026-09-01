# 03: Shared password and session

**What to build:** The household's finances stop being readable by anything that reaches the Pi. One shared password gets you in, and you stay in for a month.

Landing this early rather than late means no test written afterwards has to be retrofitted for auth.

**Blocked by:** 01, 02

**Status:** ready-for-agent

- [ ] A single shared password, bcrypt-hashed in settings, no username — see ADR-0007
- [ ] The password is set on first run
- [ ] Logging in issues a signed session cookie lasting 30 days, `HttpOnly` and `SameSite=Lax`
- [ ] Every `/api/` route except login rejects unauthenticated requests
- [ ] Static assets stay reachable without a session, so the app can render its own login screen
- [ ] The login screen is in Italian and reports a wrong password clearly
- [ ] `newTestApp(t)` returns an authenticated client, so later tickets never think about this again
- [ ] Tests cover: unauthenticated API call rejected, login then succeeds, static assets always reachable
