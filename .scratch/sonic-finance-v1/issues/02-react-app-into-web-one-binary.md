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
- [ ] `sonic-finance-frontend` is deleted once the copy is verified — copy verified, deletion awaiting the maintainer

## Comments

Implemented. `sonic-finance-frontend` is copied into `web/` as a plain copy — no `.git`, no `node_modules` — and builds into `static/`, which `main.go` already embeds.

- The scaffold arrived as-is: Vite + React + TypeScript + Tailwind v4 + shadcn/ui (`components.json`, `radix-nova` style, remixicon), Geist/Noto Serif via fontsource, a theme provider, eslint + prettier, and pnpm. Nothing was swapped out.
- Four edits on top of it, and no others: `vite.config.ts` gained `build.outDir: '../static'`, `emptyOutDir`, and a `/api` proxy to `:8080`; `public/.gitkeep` was added; `index.html` became `lang="it"` with the app's title; `src/App.tsx` now reads its text from the new `src/strings.ts` and pings `/api/health`.
- `emptyOutDir` wipes the committed `static/.gitkeep`, which `//go:embed static/*` needs to compile against a clean checkout. Keeping it in `public/` means every build copies it back, so a bare `vite build` from an IDE or CI cannot break the Go build.
- `build.sh` runs `pnpm install --frozen-lockfile` and `pnpm build` before cross-compiling, so the lockfile is what gets built.
- `eslint.config.js` now ignores `src/components/ui` and `../static`. The scaffold arrived with `pnpm lint` already failing on shadcn's generated `button.tsx` (`react-refresh/only-export-components`); generated components are not ours to lint, and a repo whose lint is red on a clean checkout is a signal nobody reads.
- No Go code changed: `newApp` already serves `static/` via `http.FileServer`. SPA deep-link fallback is deferred until a router exists — ticket 03's login screen renders at `/`, but `/login` will 404 until it is added.
- Verified against the real artifact: `./build.sh` from an empty `static/` produces one ARMv6 ELF; running it, `GET /` returns the built shell with `lang="it"`, hashed JS and woff2 assets return 200, and `/api/health` still answers. Separately, `:5173` serves the dev app and proxies `/api/health` through to `:8080`.
- The last box is unticked deliberately: the copy is verified, but `sonic-finance-frontend` has one commit and no remote, so deleting it destroys the only copy. That is the maintainer's call, not the agent's.
