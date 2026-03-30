# AGENTS.md

## Purpose

This repository implements **Bountystash**, a thin server-rendered application for turning messy technical asks into structured work packets (`bounty`, `rfq`, `rfp`, `private_security`), storing immutable versions, and attaching provenance metadata.

This file defines how agents should work in this repo.

The goal is **safe, incremental implementation** of the repo skeleton without drifting into speculative architecture or accidental rewrites.

0.1.7 is a backend plumbing pass that wires a single authoritative `service` layer for create/read/review flows and records every key step in an append-only `backend_events` table (intake_received, packet_normalized, work_item_created, work_version_persisted, review_queue_read, etc.). HTTP routes remain projections, not the source of truth, and the event trail is the durable history that later phases can build on. Version 0.1.8 adds a thin read surface (`GET /work/{id}/history` and `/api/work/{id}/history`) so operators can inspect the curated timeline of intake/validation/persistence events without transforming this trail into a full activity framework. Keep the human history route focused on the handful of operational events (intake accepted, validation failure, packet normalization, version persistence) and keep it cheap; the JSON route can remain close to `backend_events` so tooling still gets the raw payloads. Let the shared `service.WorkHistory` boundary own ordering and any payload summarization helpers to avoid scattering SQL across handlers.

---

## Product shape

Bountystash is:

- a **Go app server**
- with **server-rendered HTML**
- backed by **Postgres / Supabase**
- using **immutable work packets**
- with **provenance middleware**
- and **minimal client-side behavior**

Bountystash is **not yet**:

- a browser-heavy SPA
- a graph database product
- an agent runtime marketplace
- a realtime collaboration app
- an automatic CVE filing tool

Agents must preserve this shape.

0.1.4 adds a small representation boundary (HTML/markdown/plain/JSON helper) to prepare for 0.1.5 terminal/markdown responses; defaults remain HTML for browsers and JSON for APIs.
0.1.6 adds a small static manifest/discovery surface to reduce scraping pressure without changing the thin server-rendered shape.

---

## Primary implementation principles

### 1. Keep the app server thin

The browser should receive mostly HTML and CSS.
Use standard forms, normal routes, and template rendering.
Do not introduce client frameworks or hydration.

Allowed:

- Go `html/template`
- regular `GET`/`POST` routes
- iframe-targetable responses
- small CSS assets

Avoid:

- React / Next.js
- large client bundles
- heavy JS abstractions
- websocket-first assumptions

### 2. Source of truth is relational

Use Postgres / Supabase as the primary store.
Represent lineage with normal tables first.
Do not introduce Neo4j or Turso as primary data stores in the initial implementation.

Allowed:

- `work_items`
- `work_versions`
- `submissions`
- `artifacts`
- `attestations`
- `lineage_edges`
- `queue_jobs`

Avoid:

- adding a second primary database
- designing graph-native infra before relational pain is real

### 3. Packets are immutable

User input may be messy, but once normalized into a versioned packet, that version is immutable.
Mutable editing happens by creating a **new version**, not mutating prior versions in place.

Every implementation must preserve:

- stable `work_items`
- immutable `work_versions`
- exact hash per version
- quotient hash per version

### 4. Provenance is real, not decorative

Provenance is not a marketing layer.
It must correspond to concrete events in the system.

For the current 0.1.x scope, provenance minimum means:

- compute exact content hash from canonical packet data
- compute quotient hash from defined projection rules
- persist both hashes with each immutable version row

Attestation rows and lineage edges are valid future extensions, but should not be implied as already implemented unless schema and writes exist.

Do not add vague provenance features without storage and retrieval paths.

### 5. Security-sensitive flows default private

Anything in the `private_security` category must default to private handling.
Do not accidentally render private submissions as public.
The access model matters more than convenience.

---

## Agent operating style

### Work narrowly

Agents should implement one bounded change at a time.
Avoid repo-wide rewrites.
Avoid opportunistic refactors unless they unblock the requested task.

### Preserve the repo skeleton

When adding files, follow the established tree:

- `cmd/`
- `internal/app/`
- `internal/http/`
- `internal/packets/`
- `internal/provenance/`
- `internal/queue/`
- `internal/views/`
- `db/migrations/`

Do not invent parallel structures unless the existing structure clearly fails.

### Prefer boring code

Use standard library and minimal dependencies.
Prefer readable control flow and explicit types over cleverness.

Good:

- direct SQL via sqlc-generated methods
- explicit validation functions
- small handlers
- deterministic helpers

Bad:

- unnecessary abstraction layers
- dependency-heavy helper packages
- hidden magic in middleware

### Make partial progress safely

If a task is large, implement the minimal correct slice first.
Leave clean seams for later work.
Do not pretend unfinished systems are complete.

### Document assumptions

When creating schema, routes, or packet contracts, state assumptions in comments or docs if they affect future work.
Do not silently invent semantics.

---

## Coding rules

### Go

- Keep packages cohesive and small.
- Prefer explicit constructors and functions over global mutable state.
- Return errors with context.
- Use standard `context.Context` plumbing.
- Avoid reflection-heavy patterns.
- Avoid framework-style hidden dependency injection.

### SQL / migrations

- Migrations must be incremental and reviewable.
- Use explicit constraints and indexes where justified.
- Prefer enum-like text fields with checks if practical.
- Keep schema readable.
- Do not overfit for hypothetical scale.

### Templates

- Keep templates plain and composable.
- Optimize for server-rendered clarity.
- Do not move business logic into templates.

### CSS / static assets

- Keep assets minimal.
- No framework CSS unless explicitly requested.
- Preserve the lean, document-first frontend model.

---

## Domain rules

### Work item kinds

Initial kinds:

- `bounty`
- `rfq`
- `rfp`
- `private_security`

Do not introduce more kinds without a clear product reason.

### Visibility states

Initial visibility states:

- `draft`
- `private`
- `public`
- `archived`

### Work item statuses

Initial statuses:

- `open`
- `review`
- `awarded`
- `closed`
- `rejected`

### Packet normalization

Normalization should aim to produce:

- title
- kind
- scope
- deliverables
- acceptance criteria
- reward or quote model
- visibility intent

Normalization must be deterministic enough for hashing.

### Hashing

Exact hash should be computed over a canonical serialization of the normalized packet version.
Quotient hash should be computed over an explicitly documented projection that removes distinctions treated as irrelevant for lineage or template grouping.

Do not compute hashes over ad hoc string formatting.

---

## Parallelization guidance for subagents / Codex worktrees

Safe parallel tracks:

### Track A — schema

- migrations
- sqlc config
- generated query definitions

### Track B — packet pipeline

- packet types
- normalization
- validation
- kind classification

### Track C — HTTP surface

- route wiring
- handlers
- template rendering
- forms

### Track D — provenance

- exact hash
- quotient projection
- attestation helpers
- lineage edge writes

### Track E — review flow

- queue jobs
- review page
- verification worker

Agents should avoid overlapping edits across the same files unless coordination is explicit.

---

## What not to build yet

Do **not** build the following unless explicitly requested:

- Neo4j integration
- Turso replication
- websocket/live collaboration layers
- public API token ecosystems
- third-party marketplace integrations
- automatic disclosure or CVE submission pipelines
- complex billing or escrow systems
- autonomous multi-agent orchestration inside the product runtime

The current target is:
**intake, packetization, provenance, review, publish**.

---

## Definition of done for initial implementation

A change is complete only if it:

1. preserves the thin server-rendered architecture
2. does not break the repo skeleton
3. does not weaken private/security defaults
4. keeps versioned packets immutable
5. is understandable by a human reviewer without guesswork

For milestone 1 specifically, done means:

- landing page renders
- `POST /draft` works
- example route works
- work items and versions persist
- exact and quotient hashes are stored
- one review queue route exists

---

## Change discipline

Before making larger changes, agents should ask:

- Does this preserve the product shape?
- Is this needed now, or merely interesting later?
- Can this be implemented with ordinary Postgres tables and Go code first?
- Does this increase conceptual weight more than it increases shipped capability?

If the answer is unfavorable, do the simpler thing.

---

## Preferred commit style

Use small, reviewable commits with concrete scopes.
Examples:

- `add initial work_items and work_versions migration`
- `implement draft packet normalization`
- `wire post /draft handler and preview template`
- `add exact and quotient hash persistence`

Avoid vague commits like:

- `refactor architecture`
- `improve backend`
- `add many changes`

---

## Escalation rule

If a requested task would force a major architectural change, stop and surface the tradeoff instead of silently changing the design.

Examples:

- switching from server-rendered HTML to SPA
- replacing Postgres as source of truth
- changing immutable version semantics
- exposing private security intake publicly

---

## Summary

Build Bountystash as a **thin, verifiable, relational, server-rendered system**.
Use boring tools.
Prefer correctness over novelty.
Add graph-like behavior through normal tables first.
Treat provenance as a concrete storage and policy layer.
Keep the frontend lean and the backend disciplined.
