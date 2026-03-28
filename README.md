# Bountystash

Bountystash is a thin server-rendered Go app that turns messy intake into deterministic work packets, stores immutable versions in Postgres, and persists provenance hashes.

Current product shape:

- Go app server
- HTML templates (no SPA/hydration framework)
- Postgres/Supabase relational source of truth
- Immutable `work_versions`
- Exact + quotient hash provenance
- Minimal reviewer queue surface

## Current Milestone Scope

- `GET /` intake form
- `POST /draft` normalize + validate + persist + redirect
- `GET /work/{id}` persisted packet view
- `GET /examples/{slug}` seeded packet examples
- `GET /review` minimal reviewer queue (with private security separated)
- `GET /healthz` health probe

## Run

1. Set `DATABASE_URL` to a reachable Postgres database.
2. Apply `db/migrations/0001_init.sql`.
3. Start the server:

```bash
go run ./cmd/web
```

Default bind is `:8080` (or `$PORT`).

## Determinism and Safety Notes

- Packet normalization is deterministic and excludes runtime timestamps from hash material.
- Exact hash is computed from canonical JSON of the normalized packet.
- Quotient hash is computed from an explicit projection (`kind`, `scope`, `deliverables`, `acceptance_criteria`, `reward_model`, `visibility`).
- `private_security` intake always persists with `private` visibility.

## Non-goals (for this milestone)

- SPA/client framework architecture
- Secondary primary datastores
- Auth/accounts/permissions systems
- Realtime collaboration
- Marketplace/billing/escrow workflows
