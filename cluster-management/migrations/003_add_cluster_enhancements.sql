-- 新增集群管理增强功能相关表
-- PostgreSQL 12+

-- 创建更新时间戳触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 集群备份表
CREATE TABLE IF NOT EXISTS cluster_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    backup_name VARCHAR(255) NOT NULL,
    backup_type VARCHAR(50) NOT NULL CHECK (backup_type IN ('full', 'etcd', 'resources', 'scheduled')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'restoring')),
    storage_location TEXT NOT NULL,
    storage_size_bytes BIGINT DEFAULT 0,
    snapshot_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    retention_days INT DEFAULT 30,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT
);

-- 集群环境拓扑表
CREATE TABLE IF NOT EXISTS cluster_environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    environment_type VARCHAR(50) NOT NULL CHECK (environment_type IN ('production', 'staging', 'testing', 'development', 'custom')),
    environment_name VARCHAR(100) NOT NULL,
    topology_type VARCHAR(50) DEFAULT 'standard',
    description TEXT,
    custom_labels JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id, environment_type)
);

-- 集群导入记录表
CREATE TABLE IF NOT EXISTS import_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID REFERENCES clusters(id) ON DELETE SET NULL,
    import_source VARCHAR(50) NOT NULL CHECK (import_source IN ('kubeconfig', 'terraform', 'cloud-provider', 'manual', 'api')),
    import_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    validation_results JSONB DEFAULT '{}',
    imported_resources JSONB DEFAULT '{}',
    imported_by VARCHAR(100) NOT NULL,
    imported_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- 备份计划表
CREATE TABLE IF NOT EXISTS backup_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    schedule_name VARCHAR(255) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    backup_type VARCHAR(50) NOT NULL,
    retention_count INT DEFAULT 7,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id, schedule_name)
);

-- 修改现有clusters表，添加新字段
ALTER TABLE clusters
ADD COLUMN IF NOT EXISTS environment_type VARCHAR(50) DEFAULT 'production',
ADD COLUMN IF NOT EXISTS topology_type VARCHAR(50) DEFAULT 'standard',
ADD COLUMN IF NOT EXISTS backup_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS last_backup_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS import_source VARCHAR(50);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_cluster_backups_cluster_id ON cluster_backups(cluster_id);
CREATE INDEX IF NOT EXISTS idx_cluster_backups_status ON cluster_backups(status);
CREATE INDEX IF NOT EXISTS idx_cluster_backups_created_at ON cluster_backups(created_at);

CREATE INDEX IF NOT EXISTS idx_cluster_environments_cluster_id ON cluster_environments(cluster_id);
CREATE INDEX IF NOT EXISTS idx_cluster_environments_type ON cluster_environments(environment_type);

CREATE INDEX IF NOT EXISTS idx_import_records_cluster_id ON import_records(cluster_id);
CREATE INDEX IF NOT EXISTS idx_import_records_status ON import_records(import_status);
CREATE INDEX IF NOT EXISTS idx_import_records_imported_at ON import_records(imported_at);

CREATE INDEX IF NOT EXISTS idx_backup_schedules_cluster_id ON backup_schedules(cluster_id);
CREATE INDEX IF NOT EXISTS idx_backup_schedules_enabled ON backup_schedules(enabled);

CREATE INDEX IF NOT EXISTS idx_clusters_environment_type ON clusters(environment_type);
CREATE INDEX IF NOT EXISTS idx_clusters_topology_type ON clusters(topology_type);
CREATE INDEX IF NOT EXISTS idx_clusters_backup_enabled ON clusters(backup_enabled);

-- 应用触发器
DROP TRIGGER IF EXISTS update_cluster_backups_updated_at ON cluster_backups;
CREATE TRIGGER update_cluster_backups_updated_at
    BEFORE UPDATE ON cluster_backups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_cluster_environments_updated_at ON cluster_environments;
CREATE TRIGGER update_cluster_environments_updated_at
    BEFORE UPDATE ON cluster_environments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_import_records_updated_at ON import_records;
CREATE TRIGGER update_import_records_updated_at
    BEFORE UPDATE ON import_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_backup_schedules_updated_at ON backup_schedules;
CREATE TRIGGER update_backup_schedules_updated_at
    BEFORE UPDATE ON backup_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
