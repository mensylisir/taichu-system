-- Add missing error_msg column to import_records table
ALTER TABLE import_records ADD COLUMN IF NOT EXISTS error_msg TEXT;