# React SPA in this repo, built into the binary

The frontend is a Vite + React + Tailwind + shadcn app living in `web/`, built into `static/` and embedded in the Go binary. One artifact is copied to the Pi.

Server-rendered Go templates would have been roughly half the code — no JSON API to version, no models written twice in Go and TypeScript, no node build step. We chose React anyway: this is a personal tool maintained by one person who works comfortably in that stack, and a tool its author finds unpleasant to extend stops being extended. The cost is accepted deliberately, not overlooked.

`static/` is build output and gitignored; `static/.gitkeep` is committed because `//go:embed static/*` fails to compile against an empty directory.
