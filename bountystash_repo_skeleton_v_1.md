# Bountystash repo skeleton v1

A thin Go app server with server-rendered HTML, Supabase/Postgres as source of truth, immutable work packets, and provenance middleware.

## Stack choice

- **App server:** Go
- **Router:** chi
- **HTML rendering:** `html/template`
- **Database:** Postgres / Supabase
- **Queries:** sqlc
- **Migrations:** goose or atlas
- **Auth model:** Supabase JWT verified in app middleware
- **Storage:** Supabase Storage for attachments / artifacts
- **Jobs:** DB-backed queue table first
- **Search:** Postgres FTS first, pgvector later
- **Graph / lineage:** adjacency tables in Postgres first

---

## Top-level tree

```text
bountystash/
├── .env.example
├── .gitignore
├── README.md
├── Taskfile.yml
├── flake.nix
├── flake.lock
├── docker-compose.yml
├── cmd/
│   ├── web/
│   │   └── main.go
│   └── migrate/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── config.go
│   │   └── routes.go
│   ├── auth/
│   │   ├── context.go
│   │   ├── middleware.go
│   │   └── verify.go
│   ├── db/
│   │   ├── db.go
│   │   ├── models.sql.go
│   │   ├── query.sql.go
│   │   └── queries/
│   │       ├── work_items.sql
│   │       ├── submissions.sql
│   │       ├── attestations.sql
│   │       └── users.sql
│   ├── http/
│   │   ├── handlers/
│   │   │   ├── home.go
│   │   │   ├── work_items.go
│   │   │   ├── drafts.go
│   │   │   ├── submissions.go
│   │   │   ├── examples.go
│   │   │   └── health.go
│   │   ├── forms/
│   │   │   ├── draft_form.go
│   │   │   └── submission_form.go
│   │   ├── middleware/
│   │   │   ├── request_id.go
│   │   │   ├── logging.go
│   │   │   ├── recover.go
│   │   │   └── security_headers.go
│   │   └── render/
│   │       ├── render.go
│   │       └── viewdata.go
│   ├── packets/
│   │   ├── types.go
│   │   ├── normalize.go
│   │   ├── classify.go
│   │   ├── validate.go
│   │   └── version.go
│   ├── provenance/
│   │   ├── hash.go
│   │   ├── quotient.go
│   │   ├── attest.go
│   │   ├── lineage.go
│   │   └── policy.go
│   ├── queue/
│   │   ├── jobs.go
│   │   ├── runner.go
│   │   └── verify_draft.go
│   ├── examples/
│   │   ├── loader.go
│   │   └── seeds/
│   │       ├── auth-loop.md
│   │       ├── webhook-rfq.md
│   │       └── pipeline-rfp.md
│   └── views/
│       ├── layout.tmpl
│       ├── home.tmpl
│       ├── draft_preview.tmpl
│       ├── work_item_show.tmpl
│       ├── work_item_list.tmpl
│       ├── submission_new.tmpl
│       ├── examples_show.tmpl
│       └── partials/
│           ├── nav.tmpl
│           ├── footer.tmpl
│           ├── flash.tmpl
│           └── packet.tmpl
├── db/
│   ├── migrations/
│   │   ├── 0001_init.sql
│   │   ├── 0002_work_items.sql
│   │   ├── 0003_work_versions.sql
│   │   ├── 0004_submissions.sql
│   │   ├── 0005_artifacts.sql
│   │   ├── 0006_attestations.sql
│   │   ├── 0007_lineage_edges.sql
│   │   ├── 0008_queue_jobs.sql
│   │   └── 0009_rls_policies.sql
│   ├── schema.sql
│   ├── seeds.sql
│   └── sqlc.yaml
├── pkg/
│   └── packetid/
│       └── packetid.go
├── static/
│   ├── app.css
│   ├── favicon.svg
│   └── robots.txt
├── deployments/
│   ├── fly.toml
│   ├── railway.json
│   └── nix/
│       └── module.nix
└── docs/
    ├── architecture.md
    ├── packet-contract.md
    ├── provenance-model.md
    ├── auth-model.md
    └── codex-taskboard.md
```

---

## Core modules

## `internal/packets/`

This is the product core.

Responsibilities:
- convert messy input into normalized work packets
- classify packet type: `bounty | rfq | rfp | private_security`
- validate minimum fields
- stamp immutable version metadata

Primary types:
- `DraftInput`
- `NormalizedPacket`
- `PacketVersion`

## `internal/provenance/`

This is the trust layer.

Responsibilities:
- exact content hash
- quotient / template hash projection
- attestation construction
- lineage edges between packets, submissions, and artifacts
- policy checks for cacheability / visibility

Primary types:
- `ArtifactDigest`
- `QuotientProjection`
- `Attestation`
- `LineageEdge`

## `internal/http/handlers/`

This is intentionally boring.

Responsibilities:
- parse forms
- call packet normalization
- persist draft / version
- render HTML or iframe-targetable fragments

## `internal/queue/`

First async jobs:
- draft verification
- duplicate / similar-template lookup
- provenance recomputation
- attachment inspection

---

## First routes

```text
GET  /                      -> landing page
GET  /healthz               -> health
GET  /examples/:slug        -> rendered example brief
POST /draft                 -> generate normalized draft preview
POST /work-items            -> persist draft as work item
GET  /work-items/:id        -> show work item
GET  /work-items            -> list public work items
GET  /work-items/:id/edit   -> edit latest mutable draft state
POST /work-items/:id/publish -> publish work item
POST /work-items/:id/private -> mark private / security intake
POST /work-items/:id/submissions -> create submission
GET  /dashboard             -> issuer dashboard
GET  /review                -> reviewer queue
```

---

## Initial DB model

## `work_items`

One stable identity per item.

Fields:
- `id`
- `issuer_user_id`
- `org_id`
- `kind` (`bounty`, `rfq`, `rfp`, `private_security`)
- `visibility` (`draft`, `private`, `public`, `archived`)
- `current_version_id`
- `status` (`open`, `review`, `awarded`, `closed`, `rejected`)
- `created_at`
- `updated_at`

## `work_versions`

Immutable packet versions.

Fields:
- `id`
- `work_item_id`
- `version_no`
- `title`
- `raw_input`
- `normalized_packet_json`
- `acceptance_json`
- `reward_model_json`
- `exact_hash`
- `quotient_hash`
- `created_by_user_id`
- `created_at`

## `submissions`

Fields:
- `id`
- `work_item_id`
- `submitter_user_id`
- `version_id`
- `status`
- `submission_packet_json`
- `created_at`

## `artifacts`

Fields:
- `id`
- `owner_type`
- `owner_id`
- `storage_key`
- `mime_type`
- `byte_size`
- `exact_hash`
- `created_at`

## `attestations`

Fields:
- `id`
- `subject_type`
- `subject_id`
- `predicate_type`
- `statement_json`
- `signer_type`
- `signer_ref`
- `created_at`

## `lineage_edges`

Fields:
- `id`
- `parent_type`
- `parent_id`
- `child_type`
- `child_id`
- `edge_type`
- `created_at`

## `queue_jobs`

Fields:
- `id`
- `job_type`
- `payload_json`
- `status`
- `run_after`
- `attempts`
- `last_error`
- `created_at`
- `updated_at`

---

## Example packet contract

```json
{
  "kind": "bounty",
  "title": "Fix auth redirect loop",
  "scope": [
    "reproduce issue",
    "patch middleware",
    "add regression coverage"
  ],
  "deliverables": [
    "patch or PR",
    "test coverage",
    "maintainer notes"
  ],
  "acceptance": [
    "redirect loop removed",
    "protected routes behave correctly",
    "CI passes"
  ],
  "reward_model": {
    "type": "fixed",
    "amount": 600,
    "currency": "EUR"
  },
  "visibility": "draft"
}
```

---

## Suggested README sections

- What Bountystash is
- Why the app server is thin
- Why packets are immutable
- Why graph logic is postponed into tables first
- How Supabase fits in
- Local setup
- Running migrations
- Running the app
- How Codex should work on the repo

---

## Codex taskboard

## Track 1 — schema + migrations
1. create initial Postgres schema
2. add sqlc config and generated queries
3. add RLS policies for draft/private/public items

## Track 2 — packet pipeline
4. implement `DraftInput -> NormalizedPacket`
5. add validation and kind classification
6. add immutable version creation

## Track 3 — HTTP server
7. wire `GET /`, `POST /draft`, `GET /examples/:slug`
8. render preview packet HTML from templates

## Track 4 — provenance
9. implement exact hash computation
10. implement quotient projection + quotient hash
11. write attestation rows when versions are created

## Track 5 — review + queue
12. create queue worker for draft verification
13. add reviewer queue page
14. add duplicate / similar example lookup

---

## First milestone

The first milestone is complete when:
- landing page renders from Go templates
- `POST /draft` accepts a form and returns a rendered preview
- `work_items` and `work_versions` persist correctly
- exact hash and quotient hash are stored
- one example route works
- one reviewer queue page exists

---

## Files to ask Codex to generate first

1. `db/migrations/0001_init.sql`
2. `db/sqlc.yaml`
3. `internal/packets/types.go`
4. `internal/packets/normalize.go`
5. `internal/provenance/hash.go`
6. `internal/http/handlers/drafts.go`
7. `internal/views/home.tmpl`
8. `cmd/web/main.go`

---

## What not to build yet

Do not build yet:
- Neo4j integration
- Turso replication
- public API tokens
- browser-heavy client app
- realtime collaboration
- agent marketplace runtime
- automatic CVE submission flows

Keep v1 narrow: intake, packetization, provenance, review, publish.

