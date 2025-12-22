-- Add missing fields to cluster_backups table
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS size_bytes BIGINT DEFAULT 0;
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS location TEXT;
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS error_msg TEXT;
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS created_by VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS started_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Create index on created_at for better query performance
CREATE INDEX IF NOT EXISTS idx_cluster_backups_created_at ON cluster_backups(created_at DESC);
