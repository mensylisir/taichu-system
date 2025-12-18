-- Add updated_at column to import_records table for GORM compatibility
-- This migration adds the missing updated_at timestamp field that GORM expects

ALTER TABLE import_records 
ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Create index for updated_at for better query performance
CREATE INDEX IF NOT EXISTS idx_import_records_updated_at ON import_records(updated_at);