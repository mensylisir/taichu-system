-- Add description column to cluster_backups table
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS description TEXT;
