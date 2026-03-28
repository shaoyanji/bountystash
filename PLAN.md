# PLAN.md

## Objective

Implement Bountystash as a thin, server-rendered Go application backed by Postgres / Supabase, with immutable work packets, concrete hash-based provenance, and a minimal reviewer-facing flow.

This plan stays intentionally narrow.

The target is not a full marketplace.
The target is a working core for:

- intake
- normalization
- immutable persistence
- hash-based provenance
- review
- controlled publishability

---

## Current baseline

The repository already has a working thin prototype with the following proven pieces:

- `GET /` renders a server-side landing page
- `GET /healthz` returns `ok`
- `POST /draft` accepts form input
- `GET /examples/{slug}` renders seeded example content
- `POST /draft` can persist to Postgres when `DATABASE_URL` is configured and the schema is applied
- `GET /work/{id}` renders a persisted work item from Postgres
- `work_items` / `work_versions` use an immutable version model
- exact hash and quotient hash are computed and stored on create
- `private_security` defaults to private visibility
- the app is buildable with Nix and deployable on Garnix

0.1.4 adds a small representation boundary (HTML/markdown/plain/JSON selection helper) to prepare for 0.1.5 non-browser terminal/markdown rendering while keeping current HTML/JSON defaults.

That means Bountystash is no longer just a stub.
It is a thin persisted prototype.
The next work should extend the core, not restart it.

---

## Success criteria

The implementation is successful for this phase when the following work end-to-end:

1. `GET /` renders a usable intake page from Go templates.
2. `POST /draft` accepts a real form submission, normalizes it, persists it, and redirects to a persisted work page.
3. `GET /work/{id}` renders the persisted current version from Postgres.
4. `GET /examples/{slug}` renders seeded examples through the same or closely related packet view layer.
5. `work_items` and `work_versions` persist correctly in Postgres.
6. exact hash and quotient hash are computed and stored for each created version.
7. `private_security` items default to private visibility.
8. a minimal review queue page exists and renders queued drafts or review candidates.

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
- large client-side SPA architecture
- broad organization/user/account systems unless forced by the review flow

---

## Phase map

### Phase 0 — repository bootstrap

Goal: make the repo runnable and structured.

Status: largely complete.

Delivered or expected:

- Go module
- top-level app structure
- Nix flake
- basic config loading
- runnable app entrypoint

Acceptance:

- app compiles
- config loads
- route wiring is coherent

Core files:

- `go.mod`
- `cmd/web/main.go`
- `internal/app/config.go`
- `internal/app/routes.go`

---

### Phase 1 — relational core

Goal: establish the minimum relational source of truth.

Status: complete for the current prototype scope.

Current minimum tables:

- `work_items`
- `work_versions`

Current acceptance:

- migration applies on empty database
- schema supports creating a work item and version
- immutable version flow is encoded structurally
- current version can be read back cleanly

Current files:

- `db/migrations/0001_init.sql`
- `db/sqlc.yaml`

Important note:

This phase is intentionally smaller than the original larger-schema vision.
Do not expand into `users`, `organizations`, `submissions`, `artifacts`, `attestations`, `lineage_edges`, or `queue_jobs` unless an actual next milestone requires them.

---

### Phase 2 — HTTP surface and rendering

Goal: provide the minimum HTML-first product surface.

Status: mostly complete.

Routes in scope:

- `GET /`
- `GET /healthz`
- `GET /examples/{slug}`
- `POST /draft`
- `GET /work/{id}`

Acceptance:

- all routes respond successfully
- templates render without client-side framework support
- landing page is a real intake surface, not just placeholder copy
- draft submission ends in a persisted show page

Current files likely involved:

- `internal/http/handlers/drafts.go`
- `internal/http/handlers/examples.go`
- `internal/views/home.tmpl`
- `internal/views/examples_show.tmpl`
- `internal/views/work_show.tmpl`

---

### Phase 3 — packet normalization

Goal: transform messy form input into deterministic packet data.

Status: partially complete and functioning.

Required outputs:

- normalized title
- packet kind
- scope list
- deliverables list
- acceptance criteria list
- reward/quote/proposal model field
- visibility intent
- safe default behavior for `private_security`

Acceptance:

- same input deterministically produces the same normalized packet structure
- invalid inputs can return clear validation errors
- textarea-style newlines are handled cleanly
- CLI-submitted escaped newlines may be normalized if useful, but this is secondary

Suggested package focus:

- `internal/packets/`

Near-term improvement:

- separate normalization and validation concerns more explicitly if the current single-file approach becomes hard to maintain

---

### Phase 4 — persisted create/read flow

Goal: make drafts real persisted work items.

Status: complete for initial create/read path.

Current behavior:

- `POST /draft` creates a new `work_item`
- `POST /draft` creates version `1` in `work_versions`
- `current_version_id` is set on create
- `GET /work/{id}` reads the persisted current version back

Acceptance:

- create path works against real Postgres / Supabase
- read path works against persisted data
- no in-place mutation of previous versions occurs

What is still deferred inside this phase:

- editing an existing work item into version `2+`
- authoring workflows beyond initial create
- reviewer edits / annotations

---

### Phase 5 — provenance core

Goal: keep persisted versions concrete and auditable.

Status: partially complete.

Currently in place:

- canonical packet serialization for create path
- exact hash computation
- quotient projection
- quotient hash computation
- hash persistence on `work_versions`

Acceptance for this phase:

- every persisted version stores exact hash and quotient hash
- projection rules are explicit in code
- hash behavior is deterministic

Deferred within provenance:

- separate attestation table
- lineage edges
- signed attestations
- richer provenance documents

Suggested docs:

- `docs/provenance-model.md`

---

### Phase 6 — examples and seeded content

Goal: make the product demonstrable before real inventory exists.

Status: complete at basic level.

Current examples:

- `auth-loop`
- `webhook-rfq`
- `pipeline-rfp`

Acceptance:

- `GET /examples/{slug}` works
- examples are useful for demo and development
- examples do not replace real persistence work

---

### Phase 7 — review queue

Goal: establish the first reviewer-facing operational flow.

Status: complete at minimal level.

Current implementation:

- reviewer queue page at `GET /review`
- queue rows loaded from persisted `work_items` in `open` or `review` status
- `private_security` items separated into a dedicated private section

Acceptance:

- creating a work item places it into a reviewable state (`open`)
- reviewer page renders queued items
- `private_security` items are clearly separated from standard flow

Important constraint:

Keep this phase minimal.
A simple queue page is enough.
Do not build a complex job system unless clearly needed.

Suggested files:

- `internal/http/handlers/review.go`
- `internal/views/review_queue.tmpl`

---

### Phase 8 — access control and visibility guards

Goal: add safe access control around draft/private/public behavior.

Status: deferred.

Deliverables:

- auth middleware
- request user context
- route guards
- reviewer-only visibility paths
- alignment with any future Supabase auth/RLS choices

Acceptance:

- unauthenticated users only see public resources
- private drafts stay private
- reviewer access is explicit, not accidental

Important note:

Do not start here before the review flow exists.

---

## Parallel workstreams

These can proceed in parallel if needed, but should stay tightly scoped.

### Track A — product surface

Scope:

- home page
- intake form
- persisted work item show page
- examples

Outputs:

- usable HTML-first surface

### Track B — persistence

Scope:

- migration maintenance
- DB connection
- create/read flow
- future versioned update flow

Outputs:

- reliable Postgres-backed core

### Track C — packet pipeline

Scope:

- normalization
- validation
- deterministic shaping

Outputs:

- stable packet generation

### Track D — provenance

Scope:

- canonical JSON
- exact hash
- quotient projection
- quotient hash

Outputs:

- persistence-ready provenance data

### Track E — review

Scope:

- queue state
- queue page
- reviewer-facing route

Outputs:

- minimal operational review loop

---

## Recommended implementation order from here

From the current repo state, use this order:

1. keep persisted `POST /draft` / `GET /work/{id}` stable
2. clean up normalization and validation edge cases
3. add tests around create/read and hashing determinism
4. harden review queue behavior and copy consistency
5. wire persisted `DATABASE_URL` into deployment secrets
6. add auth / visibility guards only when reviewer access is ready
7. expand provenance only if the review flow truly needs it

---

## Acceptance gates by milestone

### Milestone 1 — visible persisted prototype

Gate:

- landing page renders
- examples route works
- `POST /draft` persists
- `GET /work/{id}` reads persisted data
- hashes are stored

### Milestone 2 — reviewable core

Gate:

- review queue page exists
- queueable work items can be listed for review
- private security items remain private by default

### Milestone 3 — controlled internal alpha

Gate:

- auth middleware exists
- visibility guards are enforced
- reviewer flows are explicit

---

## Deliverables checklist

### Repo / tooling

- [x] Go module initialized
- [x] local config scaffolded
- [x] Nix flake added
- [x] migration workflow documented clearly
- [x] vendor-hash refresh workflow documented clearly

### Database

- [x] `work_items` table created
- [x] `work_versions` table created
- [x] indexes and constraints added
- [x] migration applies cleanly
- [ ] follow-on review queue schema added only if needed

### HTTP surface

- [x] home page route
- [x] health route
- [x] draft route
- [x] example route
- [x] work item show route
- [x] review queue route

### Packet pipeline

- [x] input struct defined
- [x] normalization implemented
- [x] validation made explicit
- [x] version create flow implemented

### Provenance

- [x] canonical serialization helper
- [x] exact hash
- [x] quotient projection
- [x] quotient hash
- [ ] attestation row write
- [ ] lineage edge write

### Review

- [x] queue state in use
- [x] review handler
- [x] review template

### Security / access

- [x] private-by-default logic for security intake
- [ ] auth middleware
- [ ] route guards
- [ ] RLS alignment

---

## Risks and mitigation

### Risk: architectural drift into a SPA

Mitigation:

- keep milestone acceptance based on server-rendered HTML
- reject heavy client frameworks unless explicitly approved

### Risk: schema inflation too early

Mitigation:

- keep the schema minimal until a real milestone forces expansion
- avoid adding speculative tables

### Risk: provenance becomes hand-wavy

Mitigation:

- keep exact and quotient hashes persisted on create
- document projection rules clearly

### Risk: private security data leaks

Mitigation:

- default `private_security` to private visibility
- keep future visibility checks explicit
- do not expose reviewer/private paths casually

### Risk: environment churn slows progress

Mitigation:

- keep the app runnable with direct Go commands
- keep Nix-specific chores separate from app design work
- automate vendor-hash refresh later instead of letting it block roadmap clarity

---

## Testing priorities

High-priority tests:

- packet normalization determinism
- exact hash stability
- quotient projection stability
- work version immutability
- create/read DB happy path
- missing work item 404 behavior
- private visibility defaulting
- draft route happy path rendering

Suggested test locations:

- `internal/packets/*_test.go`
- `internal/http/handlers/*_test.go`

---

## Minimal documentation to keep current

These docs should evolve with the code:

- `README.md`
- `PLAN.md`
- `AGENTS.md`

Add these later only when they become real:

- `docs/auth-model.md`
- `docs/review-flow.md`

---

## Immediate next actions

From the current state, the shortest useful path is:

1. add tests around persisted create/read and hash determinism
2. harden validation and copy consistency
3. wire persisted `DATABASE_URL` into deployment secrets
4. deploy the persisted version cleanly
5. only then add auth / visibility controls

---

## Summary

Bountystash now has a real thin core:

- HTML-first
- Postgres-backed
- immutable versioned packets
- stored hashes
- private-by-default security intake behavior

The next work should deepen the core, not widen the scope.

Ship the reviewable persisted system first.
Do the fancy stuff later only if the core proves itself.
