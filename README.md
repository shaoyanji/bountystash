# Bountystash

Bountystash is a thin server-rendered Go app that turns messy intake into deterministic work packets, stores immutable versions in Postgres, and persists provenance hashes.

Current product shape:

- Go app server
- HTML templates (no SPA/hydration framework)
- Postgres/Supabase relational source of truth
- Immutable `work_versions`
- Exact + quotient hash provenance
- Minimal reviewer queue surface
- Keyboard-first terminal client over HTTP (`cmd/bountystash-tui`)

## Current Milestone Scope

- `GET /` intake form
- `POST /draft` normalize + validate + persist + redirect
- `GET /work/{id}` persisted packet view
- `GET /examples/{slug}` seeded packet examples
- `GET /review` minimal reviewer queue (with private security separated)
- `GET /healthz` health probe
- JSON API for terminal client:
  - `GET /api/healthz`
  - `GET /api/examples`
  - `GET /api/examples/{slug}`
  - `GET /api/review`
  - `GET /api/work`
  - `GET /api/work/{id}`
  - `POST /api/draft`

## Run

1. Set `DATABASE_URL` to a reachable Postgres database.
2. Apply `db/migrations/0001_init.sql`.
3. Start the server with Go:

```bash
go run ./cmd/web
```

Default bind is `:8080` (or `$PORT`).

Nix equivalents:

```bash
nix run .#default
```

Terminal client:

```bash
go run ./cmd/bountystash-tui --base-url http://127.0.0.1:8080
```

Or with env fallback:

```bash
BOUNTYSTASH_BASE_URL=http://127.0.0.1:8080 go run ./cmd/bountystash-tui
```

Nix:

```bash
nix run .#tui
```

TUI keys:

- `b` browse examples + recent persisted work
- `r` review queue (private security separated)
- `c` create draft
- `Enter` inspect selected item
- `Ctrl+S` submit draft in create mode
- `Ctrl+L` reload backend data
- `?` help overlay
- `q` quit

Build and checks:

```bash
go build ./...
go test ./...
nix build .#default
nix build .#tui
nix flake check
```

If `vendorHash` changes after dependency updates, temporarily set `vendorHash = pkgs.lib.fakeHash;`, run `nix build .#default`, and replace `vendorHash` with the hash from the build error.

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
