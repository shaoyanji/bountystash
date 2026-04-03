# Bountystash

Bountystash is a thin server-rendered Go app for turning messy technical asks into structured, reviewable work items.

It keeps browser use simple, exposes readable human-facing routes for curl and agents, and provides a keyboard-first TUI over the same backend seams. The system stores immutable work versions in Postgres, preserves deterministic normalization, and keeps a provenance/event trail legible without turning the product into a dashboard or a graph platform.

## v0.2.0

v0.2.0 is the first public-facing product-shape release.

It introduces:

- a landing-first homepage instead of an intake-first front door
- clearer public navigation across core surfaces
- featured examples on the homepage
- recent activity preview on the homepage
- light consistency polish across examples, review, work, and history pages
- preserved readable non-browser routes for curl and agent use
- preserved thin server-rendered architecture with no SPA/hydration drift

## Product shape

Bountystash is:

- a Go app server
- server-rendered HTML with minimal client-side behavior
- backed by Postgres / Supabase
- built around immutable `work_versions`
- deterministic in normalization and hash generation
- readable from browser, curl/agents, and TUI
- intentionally thin and legible

Bountystash is not:

- a browser-heavy SPA
- a graph database product
- a workflow engine marketplace
- an analytics dashboard
- a realtime collaboration app
- a marketplace / billing / escrow system

## Core entry modes

### Browser

Use the web app to:

- understand the product from the landing page
- inspect seeded examples
- review queued work
- browse recent activity
- create work items through the intake form
- inspect persisted work and its history

### Curl / agents

Use human-facing routes for readable output and the manifest for discovery:

- human routes default to HTML for browser-like requests
- non-browser requests default to readable markdown on supported routes
- `?format=html|md|text` remains meaningful on human-facing routes
- `/.well-known/bountystash-manifest` is the discovery surface
- `/api/*` remains the structured JSON surface

### TUI

Use the terminal client over HTTP for keyboard-first operation:

- browse examples and persisted work
- review queue items
- inspect work detail
- create drafts
- reload backend data quickly

## Main routes

### Human-facing routes

- `GET /` — landing page and intake
- `POST /draft` — normalize + validate + persist + redirect
- `GET /work/{id}` — persisted work detail
- `GET /work/{id}/history` — curated per-work history timeline
- `GET /examples/{slug}` — seeded example pages
- `GET /review` — minimal reviewer queue
- `GET /history` — system-wide recent activity ledger
- `GET /healthz` — health probe
- `GET /.well-known/bountystash-manifest` — discovery surface for curl/agents

### JSON API routes

- `GET /api/healthz`
- `GET /api/examples`
- `GET /api/examples/{slug}`
- `GET /api/review`
- `GET /api/work`
- `GET /api/work/{id}`
- `GET /api/work/{id}/history`
- `GET /api/events/recent`
- `POST /api/draft`

## Representation rules

Human-facing routes are readable first.

- Browser-like requests keep HTML on supported human-facing routes.
- Non-browser requests default to readable markdown on supported human-facing routes.
- Supported overrides on human-facing routes: `?format=html`, `?format=md`, `?format=text`
- `/api/*` stays JSON regardless of `Accept` or `?format=...`
- `/healthz` stays plain text
- The manifest is a discoverability surface, not an API parity layer

Notes:

- Human-facing routes are intended to be readable, not field-for-field mirrors of JSON responses.
- Prefer the manifest over broad scraping.
- Prefer `/api/*` over scraping when you need structured data.

## What the system stores

Bountystash turns messy asks into normalized work packets and persists:

- stable `work_items`
- immutable `work_versions`
- exact content hashes
- quotient hashes from an explicit projection
- append-only `backend_events` for lineage and operational history

The current event/history surfaces make that trail useful without turning the app into an audit platform.

## Current example entry points

Examples are currently exposed through seeded example routes such as:

- `/examples/auth-loop`
- `/examples/webhook-rfq`
- `/examples/pipeline-rfp`

There is not yet a standalone `/examples` index route. For now, the product uses seeded example entry points directly and keeps the diff surface narrow.

## Run web

1. Set `DATABASE_URL` to a reachable Postgres database.
2. Apply `db/migrations/0001_init.sql`.
3. Start the server:

```bash
go run ./cmd/web
```

Default bind is `:8080` (or `$PORT`).

Nix equivalents:

```bash
nix run .#default
```

Examples:

```bash
curl http://127.0.0.1:8080/
curl http://127.0.0.1:8080/.well-known/bountystash-manifest
curl 'http://127.0.0.1:8080/.well-known/bountystash-manifest?format=text'
curl 'http://127.0.0.1:8080/?format=md'
curl 'http://127.0.0.1:8080/?format=text'
curl http://127.0.0.1:8080/examples/auth-loop
curl http://127.0.0.1:8080/review
curl http://127.0.0.1:8080/history
curl http://127.0.0.1:8080/api/examples
curl http://127.0.0.1:8080/api/events/recent
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
- `Esc` back from inspect/create
- `Tab` / `Shift+Tab` move create focus
- `Left` / `Right` cycle create `kind` / `visibility` when focused
- `Ctrl+S` submit draft in create mode
- `Ctrl+L` reload backend data
- `?` help overlay
- `q` quit

## Build and Verify

This repo uses a thin task runner ([Taskfile.yml](Taskfile.yml)) for coherent builds, checks, and version management.

**Prerequisite:** Install [task](https://taskfile.dev) or use the raw commands below.

### Quick Start

```bash
# Run all checks (Go + Nix)
task check:all

# Build both binaries with version metadata
task build:all

# Show current version source
task version:show
```

### Go Toolchain Policy

All Go commands run through the Nix dev shell to avoid toolchain drift:

```bash
nix develop --command bash -c 'unset GOROOT; GOTOOLCHAIN=local go test ./...'
nix develop --command bash -c 'unset GOROOT; GOTOOLCHAIN=local go build ./...'
```

**Do not use host Go** for repo checks. The flake provides the Go toolchain.

### Version Source of Truth

The `VERSION` file at repo root is the single source of truth. It feeds into:

- `flake.nix` package version metadata
- Build-time ldflags injection for Go binaries
- Release tags (`git tag vX.Y.Z`)

To update version:

```bash
# 1. Edit VERSION file
echo "0.2.0" > VERSION

# 2. Sync flake.nix
task version:sync

# 3. Verify
task check:all

# 4. Tag release
git tag v0.2.0
```

### Standard Tasks

| Task | Description |
|------|-------------|
| `task check:all` | Run all Go + Nix checks |
| `task check:go` | Run Go tests and build through Nix dev shell |
| `task check:nix` | Run Nix builds and flake check |
| `task build:web` | Build web server binary with version metadata |
| `task build:tui` | Build TUI binary with version metadata |
| `task build:all` | Build both binaries |
| `task version:show` | Show current version source |
| `task version:sync` | Sync flake.nix to VERSION file |
| `task dev:shell` | Enter Nix dev shell |

### Manual Commands (No Task Runner)

If you don't have `task` installed, use these directly:

```bash
# Go checks
nix develop --command bash -c 'unset GOROOT; GOTOOLCHAIN=local go test ./...'
nix develop --command bash -c 'unset GOROOT; GOTOOLCHAIN=local go build ./...'

# Nix checks
nix build .#default
nix build .#tui
nix flake check

# Build with version
VERSION=$(cat VERSION)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT=$(git rev-parse --short HEAD)
nix develop --command bash -c 'unset GOROOT; GOTOOLCHAIN=local go build -ldflags "-X github.com/shaoyanji/bountystash/internal/version.Version=$VERSION -X github.com/shaoyanji/bountystash/internal/version.Commit=$COMMIT -X github.com/shaoyanji/bountystash/internal/version.Date=$DATE" -o ./bin/web ./cmd/web'
```

### Hash Update Ritual

When Go dependencies or Nix inputs change, update `vendorHash` in `flake.nix`:

1. Set `vendorHash = pkgs.lib.fakeHash` for the affected package
2. Run `nix build .#default` or `nix build .#tui`
3. Copy the hash from the build error
4. Replace `fakeHash` with the real hash
5. Run `task check:nix` to verify

Or use: `task hash:update:default` / `task hash:update:tui`

## Release Artifacts

Tagged releases (`v*`) run `.github/workflows/release.yml` and publish:

- `bountystash-tui_<version>_linux_x86_64.tar.gz`

Example:

```bash
git tag v0.2.0
git push origin v0.2.0
```

## Determinism and Safety Notes

- Packet normalization is deterministic and excludes runtime timestamps from hash material.
- Exact hash is computed from canonical JSON of the normalized packet.
- Quotient hash is computed from an explicit projection (`kind`, `scope`, `deliverables`, `acceptance_criteria`, `reward_model`, `visibility`).
- `private_security` intake always persists with `private` visibility.

## Non-goals

- SPA/client framework architecture
- Secondary primary datastores
- Auth/accounts/permissions systems
- Realtime collaboration
- Marketplace/billing/escrow workflows
