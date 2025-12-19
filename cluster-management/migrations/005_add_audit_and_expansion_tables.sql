-- Create audit_events table
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    username VARCHAR(100) NOT NULL,
    ip_address VARCHAR(50),
    user_agent TEXT,
    old_value JSONB DEFAULT '{}',
    new_value JSONB DEFAULT '{}',
    details JSONB DEFAULT '{}',
    result VARCHAR(20) DEFAULT 'success',
    error_msg TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for audit_events
CREATE INDEX IF NOT EXISTS idx_audit_events_cluster_id ON audit_events(cluster_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_username ON audit_events(username);

-- Create cluster_expansions table
CREATE TABLE IF NOT EXISTS cluster_expansions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    old_node_count INT NOT NULL,
    new_node_count INT NOT NULL,
    old_cpu_cores INT NOT NULL,
    new_cpu_cores INT NOT NULL,
    old_memory_gb INT NOT NULL,
    new_memory_gb INT NOT NULL,
    old_storage_gb INT NOT NULL,
    new_storage_gb INT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    reason TEXT,
    error_msg TEXT,
    details JSONB DEFAULT '{}',
    requested_by VARCHAR(100) NOT NULL,
    executed_by VARCHAR(100),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for cluster_expansions
CREATE INDEX IF NOT EXISTS idx_cluster_expansions_cluster_id ON cluster_expansions(cluster_id);
CREATE INDEX IF NOT EXISTS idx_cluster_expansions_status ON cluster_expansions(status);
CREATE INDEX IF NOT EXISTS idx_cluster_expansions_created_at ON cluster_expansions(created_at);
