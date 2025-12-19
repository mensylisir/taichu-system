-- Add storage fields to cluster_states table
ALTER TABLE cluster_states
ADD COLUMN total_storage_bytes BIGINT DEFAULT 0,
ADD COLUMN used_storage_bytes BIGINT DEFAULT 0,
ADD COLUMN storage_usage_percent DOUBLE PRECISION DEFAULT 0;

-- Add indexes for better query performance
CREATE INDEX idx_cluster_states_storage ON cluster_states(total_storage_bytes, used_storage_bytes);
