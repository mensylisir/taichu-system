-- Add deleted_at column to cluster_resources table for GORM soft delete compatibility
-- This migration adds the missing deleted_at timestamp field that GORM expects for soft deletes

ALTER TABLE cluster_resources 
ADD COLUMN deleted_at TIMESTAMPTZ NULL;

-- Create index for deleted_at for better query performance
CREATE INDEX IF NOT EXISTS idx_cluster_resources_deleted_at ON cluster_resources(deleted_at) WHERE deleted_at IS NOT NULL;