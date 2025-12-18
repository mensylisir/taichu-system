-- Create machines table
CREATE TABLE IF NOT EXISTS machines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    ip_address VARCHAR(50) NOT NULL,
    internal_address VARCHAR(50),
    username VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('master', 'worker', 'etcd', 'registry')),
    status VARCHAR(50) DEFAULT 'available' CHECK (status IN ('available', 'in-use', 'deploying', 'maintenance', 'offline')),
    artifact_path TEXT,
    image_repo VARCHAR(255),
    registry_address VARCHAR(255),
    labels JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for machines
CREATE INDEX IF NOT EXISTS idx_machines_name ON machines(name);
CREATE INDEX IF NOT EXISTS idx_machines_ip_address ON machines(ip_address);
CREATE INDEX IF NOT EXISTS idx_machines_role ON machines(role);
CREATE INDEX IF NOT EXISTS idx_machines_status ON machines(status);
CREATE INDEX IF NOT EXISTS idx_machines_created_at ON machines(created_at);
