-- Create import_records table for tracking cluster import operations
CREATE TABLE IF NOT EXISTS import_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID,
    import_source VARCHAR(100) NOT NULL,
    import_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    imported_by VARCHAR(100) NOT NULL,
    validation_results JSONB DEFAULT '{}',
    imported_resources JSONB DEFAULT '{}',
    error_msg TEXT,
    completed_at TIMESTAMPTZ,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for import_records
CREATE INDEX IF NOT EXISTS idx_import_records_cluster_id ON import_records(cluster_id);
CREATE INDEX IF NOT EXISTS idx_import_records_status ON import_records(import_status);
CREATE INDEX IF NOT EXISTS idx_import_records_imported_at ON import_records(imported_at);