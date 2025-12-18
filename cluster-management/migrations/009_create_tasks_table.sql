-- Create create_tasks table for tracking cluster creation tasks
CREATE TABLE IF NOT EXISTS create_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID,
    task_type VARCHAR(50) NOT NULL DEFAULT 'create_cluster',
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    progress INTEGER DEFAULT 0,
    current_step VARCHAR(255),
    machine_ids JSONB DEFAULT '[]',
    config_content TEXT,
    kubeconfig_path TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for create_tasks
CREATE INDEX IF NOT EXISTS idx_create_tasks_cluster_id ON create_tasks(cluster_id);
CREATE INDEX IF NOT EXISTS idx_create_tasks_status ON create_tasks(status);
CREATE INDEX IF NOT EXISTS idx_create_tasks_created_at ON create_tasks(created_at);
