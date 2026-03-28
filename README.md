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

Current release target: `0.1.2`.

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

## Run Web

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

## Run TUI

Default API endpoint used by the TUI:

`https://garnixmachine.main.nixconfig.shaoyanji.garnix.me/`

Run with defaults:

```bash
go run ./cmd/bountystash-tui
```

Override explicitly with flag (highest precedence):

```bash
go run ./cmd/bountystash-tui --base-url http://127.0.0.1:8080
```

Or override with environment variable (used when `--base-url` is not passed):

```bash
BOUNTYSTASH_BASE_URL=http://127.0.0.1:8080 go run ./cmd/bountystash-tui
```

Nix:

```bash
nix run .#tui
```

Show TUI build metadata:

```bash
go run ./cmd/bountystash-tui --version
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

## Build and Verify

```bash
go build ./...
go test ./...
nix build .#default
nix build .#tui
nix flake check
```

## Release Artifacts

Tagged releases (`v*`) run `.github/workflows/release.yml` and publish:

- `bountystash-tui_<version>_linux_x86_64.tar.gz`

Example:

```bash
git tag v0.1.2
git push origin v0.1.2
```

## Nix Hash Update Ritual

If Go dependencies, vendoring, or package inputs change, refresh `vendorHash` values in `flake.nix` before relying on CI/deploy.

For `.#default`:

1. Set `packages.<system>.default.vendorHash = pkgs.lib.fakeHash;`
2. Run `nix build .#default`
3. Copy the hash from the build error into `vendorHash`

For `.#tui`:

1. Set `packages.<system>.tui.vendorHash = pkgs.lib.fakeHash;`
2. Run `nix build .#tui`
3. Copy the hash from the build error into `vendorHash`

Then run the full verification set:

```bash
go test ./...
go build ./...
nix build .#default
nix build .#tui
nix flake check
```

Release/deploy checklist:

1. Update hashes if needed using the ritual above.
2. Run the full verification set locally.
3. Open/merge PR with passing CI.
4. Tag release (`vX.Y.Z`) to publish TUI artifact(s).

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
