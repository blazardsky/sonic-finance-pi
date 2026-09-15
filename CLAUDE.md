# Sonic Finance Backend

Go backend API for the Sonic Finance App. Targets a Raspberry Pi Zero W (1st gen, ARMv6); build with `./scripts/build.sh`.

## Frontend

Always use a shadcn/ui component (`web/src/components/ui/`, or `npx shadcn add <name>` from `web/`) when one fits, rather than a native HTML element or a hand-rolled component. Ask before choosing a native element or a non-shadcn component when it's unclear which fits. The project's shadcn setup is non-default — style `radix-nova`, icon library `@remixicon/react` (not `lucide-react`) — see `web/components.json`.

## Schema changes

Tickets are written behaviourally and do not mention the database. If yours needs a table or column that does not exist, bump `schemaVersion` in `migrate.go` by one and add the matching `case` to `migrateStep` — the schema is versioned by `PRAGMA user_version`, applied at startup, with no migration library. An earlier case can never be edited afterwards, because a database already at that version will not re-run it. The table shapes are specified in the spec's Schema section.

## Agent skills

### Issue tracker

Issues live as markdown files under `.scratch/<feature>/` in this repo. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical roles, unchanged (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
