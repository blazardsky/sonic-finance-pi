# Frontend

Vite + React + TypeScript + Tailwind + shadcn/ui. Builds into
`../cmd/static/`, which the Go binary embeds — see ADR-0006. Package manager
is pnpm.

## Dev loop

Two servers side by side:

    go run ./cmd          # :8080, serves the API and the last build of static/
    pnpm --dir web dev    # :5173, serves the frontend with hot reload

Open `:5173`. Vite proxies `/api` to `:8080`, so the frontend uses the same
relative URLs in dev as in the binary.

## Production build

`./scripts/build.sh` at the repo root does both halves. `cmd/static/` is gitignored
apart from `.gitkeep`: Vite empties `outDir`, and `//go:embed static/*` will not
compile against an empty directory, so `public/.gitkeep` is copied back in by
every build.

UI strings go in `src/strings.ts`, not inline in components.

## Adding components

    pnpm --dir web dlx shadcn@latest add <component>

Components land in `src/components/ui`. Import them via the `@/` alias.
