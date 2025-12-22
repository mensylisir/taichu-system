-- Add name column to cluster_backups table
ALTER TABLE cluster_backups ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';

-- Update existing records to have the backup_name as name
UPDATE cluster_backups SET name = COALESCE(backup_name, '') WHERE name = '';

-- Add NOT NULL constraint after updating existing records
ALTER TABLE cluster_backups ALTER COLUMN name SET NOT NULL;
