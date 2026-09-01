# 02: React app into `web/`, one binary

**What to build:** Opening the app's URL serves the React frontend from the Go binary. One artifact gets copied to the Pi; there is no second thing to deploy.

The existing `sonic-finance-frontend` scaffold is copied in as a plain copy — no git history, it is an unmodified Vite scaffold — and the old repository is removed once this lands.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] The frontend lives in `web/` in this repository and builds into `static/`
- [ ] `static/` stays gitignored apart from the committed `.gitkeep` that keeps `go:embed` compiling
- [ ] `build.sh` builds the frontend, then cross-compiles for ARMv6, producing one file
- [ ] The dev loop works: the Go server and the Vite dev server run side by side, with `/api` proxied
- [ ] UI strings live in an Italian strings file rather than inline in components, so later tickets have somewhere to put text
- [ ] `sonic-finance-frontend` is deleted once the copy is verified
