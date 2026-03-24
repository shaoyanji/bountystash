BEGIN;

CREATE TABLE work_items (
    id UUID PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('bounty', 'rfq', 'rfp', 'private_security')),
    visibility TEXT NOT NULL CHECK (visibility IN ('draft', 'private', 'public', 'archived')),
    status TEXT NOT NULL CHECK (status IN ('open', 'review', 'awarded', 'closed', 'rejected')),
    current_version_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE work_versions (
    id UUID PRIMARY KEY,
    work_item_id UUID NOT NULL REFERENCES work_items(id) ON DELETE RESTRICT,
    version_number INTEGER NOT NULL CHECK (version_number > 0),
    packet JSONB NOT NULL,
    exact_hash TEXT NOT NULL CHECK (exact_hash ~ '^[0-9a-f]{64}$'),
    quotient_hash TEXT NOT NULL CHECK (quotient_hash ~ '^[0-9a-f]{64}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT work_versions_work_item_version_unique UNIQUE (work_item_id, version_number),
    CONSTRAINT work_versions_work_item_exact_hash_unique UNIQUE (work_item_id, exact_hash),
    CONSTRAINT work_versions_work_item_id_id_unique UNIQUE (work_item_id, id)
);

ALTER TABLE work_items
    ADD CONSTRAINT work_items_current_version_fk
    FOREIGN KEY (id, current_version_id)
    REFERENCES work_versions (work_item_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE OR REPLACE FUNCTION prevent_work_versions_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'work_versions rows are immutable';
END;
$$;

CREATE TRIGGER work_versions_no_update
BEFORE UPDATE ON work_versions
FOR EACH ROW
EXECUTE FUNCTION prevent_work_versions_mutation();

CREATE TRIGGER work_versions_no_delete
BEFORE DELETE ON work_versions
FOR EACH ROW
EXECUTE FUNCTION prevent_work_versions_mutation();

CREATE INDEX work_items_current_version_idx
    ON work_items (current_version_id);

CREATE INDEX work_versions_work_item_created_idx
    ON work_versions (work_item_id, created_at DESC);

COMMIT;
