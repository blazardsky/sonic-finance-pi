# 02: React app into `web/`, one binary

**What to build:** Opening the app's URL serves the React frontend from the Go binary. One artifact gets copied to the Pi; there is no second thing to deploy.

The existing `sonic-finance-frontend` scaffold is copied in as a plain copy — no git history, it is an unmodified Vite scaffold — and the old repository is removed once this lands.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [x] The frontend lives in `web/` in this repository and builds into `static/`
- [x] `static/` stays gitignored apart from the committed `.gitkeep` that keeps `go:embed` compiling
- [x] `build.sh` builds the frontend, then cross-compiles for ARMv6, producing one file
- [x] The dev loop works: the Go server and the Vite dev server run side by side, with `/api` proxied
- [x] UI strings live in an Italian strings file rather than inline in components, so later tickets have somewhere to put text
- [ ] `sonic-finance-frontend` is deleted once the copy is verified — n/a, see comments

## Comments

Implemented. The frontend is `web/` (Vite + React + TypeScript + Tailwind), building into `static/`, which `main.go` already embeds.

- `sonic-finance-frontend` was not on this machine — no clone, no remote, nowhere under `/Volumes/SSD/DEV` or `$HOME`. `web/` is therefore a fresh `create-vite` `react-ts` scaffold, which is what the ticket describes that repository as holding. Nothing was deleted because nothing was there.
- `vite.config.ts` carries the two settings that make one binary work: `build.outDir: '../static'` and `server.proxy` sending `/api` to `:8080`, so the frontend uses the same relative URLs in dev as in the binary.
- `emptyOutDir` wipes the committed `static/.gitkeep`, which `//go:embed static/*` needs to compile against a clean checkout. `npm run build` recreates it, so both `./build.sh` and a bare `npm run build` leave the tree correct.
- Tailwind is in, shadcn is not. ADR-0006 names both, but shadcn copies components in on demand and there is no component yet; the first ticket that needs one adds it.
- No Go code changed: `newApp` already serves `static/` via `http.FileServer`, and SPA deep-link fallback is not needed until a router exists.
- Verified by running the built binary and curling it: `GET /` returns the built `index.html`, `GET /assets/…` returns 200, and `GET /api/health` still answers. Separately, `:5173` serves the dev app and proxies `/api/health` through to `:8080`.
