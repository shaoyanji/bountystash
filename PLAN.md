# PLAN.md

## Objective

Implement the initial Bountystash skeleton as a thin server-rendered Go application backed by Postgres / Supabase, with immutable work packets, basic provenance, and a minimal review flow.

This plan is intentionally narrow.
The target is not a full marketplace.
The target is a working core for:

- intake
- packetization
- persistence
- provenance
- review
- publish

---

## Success criteria

The implementation is successful when the following work end-to-end:

1. `GET /` renders the landing page from Go templates.
2. `POST /draft` accepts a form submission and returns a rendered normalized brief preview.
3. `GET /examples/:slug` renders at least one seeded example.
4. `work_items` and `work_versions` persist correctly in Postgres.
5. exact hash and quotient hash are computed and stored for each created version.
6. a reviewer queue page exists and renders queued drafts.
7. `private_security` items default to private visibility.

---

## Non-goals for this phase

Do not implement yet:

- Neo4j integration
- Turso replication
- public API token systems
- realtime collaboration
- payments / escrow
- autonomous in-product multi-agent execution
- automatic CVE or disclosure submission flows
- advanced semantic search beyond simple scaffolding

---

## Phase map

### Phase 0 — bootstrap the repository

Goal: make the repo runnable and structured.

Deliverables:

- initialize Go module
- create top-level directory structure
- add `.env.example`
- add `README.md`
- add `Taskfile.yml`
- add `flake.nix` / `flake.lock` if using Nix dev shell
- add `docker-compose.yml` for local Postgres if desired

Acceptance:

- project tree matches the agreed skeleton
- app can compile with placeholder handlers
- config loads cleanly

Suggested files:

- `go.mod`
- `cmd/web/main.go`
- `internal/app/app.go`
- `internal/app/config.go`
- `internal/app/routes.go`

---

### Phase 1 — schema and migrations

Goal: establish the relational source of truth.

Deliverables:

- initial migrations
- sqlc config
- query files for core entities
- base indexes and constraints

Minimum tables:

- `users`
- `organizations`
- `work_items`
- `work_versions`
- `submissions`
- `artifacts`
- `attestations`
- `lineage_edges`
- `queue_jobs`

Acceptance:

- migrations apply cleanly on empty database
- schema supports creating a work item and a version
- sqlc generation succeeds

Suggested order:

1. `0001_init.sql`
2. `0002_work_items.sql`
3. `0003_work_versions.sql`
4. `0004_submissions.sql`
5. `0005_artifacts.sql`
6. `0006_attestations.sql`
7. `0007_lineage_edges.sql`
8. `0008_queue_jobs.sql`
9. `0009_rls_policies.sql`

---

### Phase 2 — HTTP server and rendering

Goal: wire the minimal web surface.

Deliverables:

- render landing page from template
- health endpoint
- example route
- draft handler route
- shared rendering helpers

Routes:

- `GET /`
- `GET /healthz`
- `GET /examples/:slug`
- `POST /draft`

Acceptance:

- all routes respond successfully
- templates render without client-side framework support
- iframe-targetable preview response works

Suggested files:

- `internal/http/handlers/home.go`
- `internal/http/handlers/examples.go`
- `internal/http/handlers/drafts.go`
- `internal/http/handlers/health.go`
- `internal/http/render/render.go`
- `internal/views/home.tmpl`
- `internal/views/draft_preview.tmpl`
- `internal/views/examples_show.tmpl`

---

### Phase 3 — packet pipeline

Goal: transform messy form input into normalized work packets.

Deliverables:

- input structs
- normalization logic
- kind classification
- validation logic
- version metadata generation

Core package:

- `internal/packets/`

Required outputs:

- normalized title
- packet kind
- scope list
- deliverables list
- acceptance list
- reward/quote/proposal model scaffold
- visibility intent

Acceptance:

- same input deterministically produces same normalized packet shape
- invalid inputs return clear validation errors
- `private_security` requests force safe defaults

Suggested files:

- `types.go`
- `normalize.go`
- `classify.go`
- `validate.go`
- `version.go`

---

### Phase 4 — persistence flow

Goal: save normalized packets as versioned work items.

Deliverables:

- create work item if needed
- create immutable work version
- update `current_version_id`
- render saved preview or show page

Acceptance:

- a form submission can create a persisted `work_item`
- each edit creates a new `work_version`
- no in-place mutation of prior versions occurs

Suggested route expansion:

- `POST /work-items`
- `GET /work-items/:id`

---

### Phase 5 — provenance layer

Goal: make packet creation produce concrete provenance artifacts.

Deliverables:

- canonical serialization helper
- exact hash computation
- quotient projection function
- quotient hash computation
- attestation record creation
- lineage edge creation where appropriate

Acceptance:

- every persisted version stores exact hash and quotient hash
- at least one attestation is written per created version
- projection rules are explicit and documented

Suggested files:

- `internal/provenance/hash.go`
- `internal/provenance/quotient.go`
- `internal/provenance/attest.go`
- `internal/provenance/lineage.go`
- `docs/provenance-model.md`

---

### Phase 6 — examples and seeded content

Goal: make the system demonstrable before real inventory exists.

Deliverables:

- seeded example briefs
- example loader
- one public example per kind category of interest

Initial examples:

- `auth-loop.md`
- `webhook-rfq.md`
- `pipeline-rfp.md`

Acceptance:

- `GET /examples/:slug` works for seeded examples
- examples render through the same packet view layer as real drafts where practical

---

### Phase 7 — review queue

Goal: establish the first reviewer-facing flow.

Deliverables:

- queue job record creation for draft verification
- worker stub or synchronous placeholder
- review page listing queued items
- draft status transitions into review flow

Acceptance:

- creating a draft can enqueue a verification job
- reviewer page renders queued entries
- `private_security` drafts appear only in appropriate private review contexts

Suggested files:

- `internal/queue/jobs.go`
- `internal/queue/runner.go`
- `internal/queue/verify_draft.go`
- `internal/http/handlers/review.go`
- `internal/views/review_queue.tmpl`

---

### Phase 8 — auth and visibility controls

Goal: add safe access control around draft/private/public flows.

Deliverables:

- Supabase JWT verification middleware
- request context user loading
- draft/private/public route guards
- RLS alignment with app behavior

Acceptance:

- unauthenticated users can only see public resources
- authenticated issuers can see their drafts/private items
- reviewer role logic is explicit and not accidental

Suggested files:

- `internal/auth/verify.go`
- `internal/auth/middleware.go`
- `internal/auth/context.go`
- `docs/auth-model.md`

---

## Parallel workstreams for Codex / subagents

These tracks can proceed largely in parallel with coordination.

### Track A — repository bootstrap

Scope:

- config
- app wiring
- route skeleton
- local dev tooling

Outputs:

- runnable app shell
- health endpoint
- base route registration

### Track B — schema and queries

Scope:

- migrations
- sqlc config
- base query definitions

Outputs:

- working database layer
- generated models / query wrappers

### Track C — packet pipeline

Scope:

- input types
- normalization
- validation
- version metadata

Outputs:

- deterministic normalized packet generation

### Track D — provenance

Scope:

- exact hashing
- quotient projection
- attestation helpers

Outputs:

- persistence-ready provenance data

### Track E — rendering and examples

Scope:

- templates
- preview rendering
- example seeding

Outputs:

- demonstrable UI without real inventory

### Track F — review flow

Scope:

- queue table usage
- reviewer list page
- initial verification worker

Outputs:

- minimal operational review loop

---

## Recommended implementation order

If a single builder is working sequentially, use this order:

1. repo bootstrap
2. schema + migrations
3. sqlc config and queries
4. route skeleton and template rendering
5. packet normalization
6. `POST /draft`
7. persistence to `work_items` and `work_versions`
8. exact and quotient hash persistence
9. example route and seeds
10. review queue page
11. auth + visibility controls

This sequence gets the visible product working early while still preserving the trust model.

---

## Acceptance gates by milestone

### Milestone 1 — visible demo

Gate:

- landing page renders
- example route works
- `POST /draft` returns a normalized preview

### Milestone 2 — persistent core

Gate:

- work items and work versions persist
- immutable version flow exists
- hashes are stored

### Milestone 3 — reviewable system

Gate:

- review queue page exists
- queue jobs are created
- private flows remain private

### Milestone 4 — authenticated internal alpha

Gate:

- auth middleware works
- issuer dashboard can be added safely
- visibility rules are enforced

---

## Deliverables checklist

### Repo / tooling

- [ ] Go module initialized
- [ ] local config scaffolded
- [ ] migration tool wired
- [ ] sqlc configured

### Database

- [ ] core tables created
- [ ] indexes and constraints added
- [ ] base seed data available

### HTTP surface

- [ ] home page route
- [ ] health route
- [ ] draft route
- [ ] example route
- [ ] work item show route

### Packet pipeline

- [ ] input struct defined
- [ ] normalization implemented
- [ ] validation implemented
- [ ] version creation implemented

### Provenance

- [ ] canonical serialization helper
- [ ] exact hash
- [ ] quotient projection
- [ ] quotient hash
- [ ] attestation row write

### Review

- [ ] queue job table in use
- [ ] review handler
- [ ] review template

### Security / access

- [ ] private-by-default logic for security intake
- [ ] auth middleware
- [ ] route guards
- [ ] RLS alignment

---

## Risks and mitigation

### Risk: architectural drift into a SPA

Mitigation:

- keep all milestone acceptance based on server-rendered HTML
- reject work that introduces heavy client frameworks without explicit approval

### Risk: provenance becomes hand-wavy

Mitigation:

- require exact and quotient hashes to be stored in DB
- require at least one attestation row in the create-version flow

### Risk: graph fascination too early

Mitigation:

- represent lineage with adjacency tables first
- defer Neo4j entirely for this phase

### Risk: private security data leaks

Mitigation:

- default `private_security` to private visibility
- make route visibility checks explicit
- align RLS with app guards

### Risk: normalization becomes inconsistent

Mitigation:

- keep deterministic packet shaping rules
- add fixture-based tests around representative inputs

---

## Testing priorities

High-priority tests:

- packet normalization determinism
- validation failures for bad input
- exact hash stability
- quotient projection stability
- work version immutability
- private visibility enforcement
- draft route happy-path rendering

Suggested test locations:

- `internal/packets/*_test.go`
- `internal/provenance/*_test.go`
- `internal/http/handlers/*_test.go`

---

## Minimal documentation to keep current

These docs should evolve with the code:

- `README.md`
- `docs/packet-contract.md`
- `docs/provenance-model.md`
- `docs/auth-model.md`
- `docs/architecture.md`

If implementation changes meaningfully, update the docs in the same workstream.

---

## Immediate next actions

Start with these concrete files:

1. `cmd/web/main.go`
2. `internal/app/config.go`
3. `internal/app/routes.go`
4. `db/migrations/0001_init.sql`
5. `db/sqlc.yaml`
6. `internal/packets/types.go`
7. `internal/http/handlers/drafts.go`
8. `internal/views/home.tmpl`

These provide the shortest path to a visible, persistent, reviewable core.

---

## Summary

This plan builds the smallest serious version of Bountystash.
It favors:

- thin HTML-first delivery
- relational clarity
- immutable packets
- concrete provenance
- private-by-default security handling

Ship the core first.
Do the fancy stuff later only if the core proves itself.
