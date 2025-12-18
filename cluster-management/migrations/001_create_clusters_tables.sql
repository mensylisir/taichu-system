-- 创建集群管理系统数据库Schema
-- PostgreSQL 12+

-- 创建更新时间戳触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 集群配置表
CREATE TABLE clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    kubeconfig_encrypted TEXT NOT NULL,
    kubeconfig_nonce VARCHAR(32) NOT NULL,
    version VARCHAR(50) DEFAULT '1.0.0',
    labels JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    created_by VARCHAR(100) DEFAULT 'system',
    updated_by VARCHAR(100) DEFAULT 'system'
);

-- 集群状态缓存表
CREATE TABLE cluster_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'unknown' CHECK (status IN ('unknown', 'healthy', 'unhealthy', 'disconnected')),
    node_count INTEGER DEFAULT 0,
    total_cpu_cores INTEGER DEFAULT 0,
    total_memory_bytes BIGINT DEFAULT 0,
    kubernetes_version VARCHAR(50),
    api_server_url VARCHAR(255),
    last_heartbeat_at TIMESTAMP WITH TIME ZONE,
    last_sync_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sync_error TEXT,
    sync_success BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(cluster_id)
);

-- 索引优化
CREATE INDEX idx_clusters_name ON clusters(name);
CREATE INDEX idx_clusters_labels ON clusters USING GIN(labels);
CREATE INDEX idx_clusters_created_at ON clusters(created_at);
CREATE INDEX idx_clusters_deleted_at ON clusters(deleted_at) WHERE deleted_at IS NULL;

CREATE INDEX idx_cluster_states_cluster_id ON cluster_states(cluster_id);
CREATE INDEX idx_cluster_states_status ON cluster_states(status);
CREATE INDEX idx_cluster_states_last_heartbeat ON cluster_states(last_heartbeat_at);
CREATE INDEX idx_cluster_states_sync_success ON cluster_states(sync_success);

-- 应用触发器
CREATE TRIGGER update_clusters_updated_at
    BEFORE UPDATE ON clusters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cluster_states_updated_at
    BEFORE UPDATE ON cluster_states
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
