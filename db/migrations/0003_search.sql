-- Migration 0003: Add full-text search support for work_versions
BEGIN;

-- Add a generated column for full-text search vector on packet content
ALTER TABLE work_versions 
ADD COLUMN IF NOT EXISTS search_vector tsvector 
GENERATED ALWAYS AS (
    to_tsvector('english', 
        COALESCE(packet->>'title', '') || ' ' ||
        array_to_string(COALESCE(packet->'scope', '[]')::jsonb#>>'{}', ' ') || ' ' ||
        array_to_string(COALESCE(packet->'deliverables', '[]')::jsonb#>>'{}', ' ') || ' ' ||
        array_to_string(COALESCE(packet->'acceptance_criteria', '[]')::jsonb#>>'{}', ' ')
    )
) STORED;

-- Create GIN index for efficient full-text search
CREATE INDEX IF NOT EXISTS work_versions_search_vector_idx 
ON work_versions USING GIN (search_vector);

-- Add a comment explaining the search functionality
COMMENT ON COLUMN work_versions.search_vector IS 'Full-text search vector for packet title, scope, deliverables, and acceptance criteria';

COMMIT;
