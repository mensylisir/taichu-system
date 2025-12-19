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
    provider VARCHAR(100) DEFAULT '太初',
    region VARCHAR(100),
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
    cluster_id UUID NOT NULL,
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

-- 节点详情表
CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('control-plane', 'worker', 'master')),
    status VARCHAR(20) NOT NULL,
    cpu_cores INTEGER DEFAULT 0,
    cpu_used_cores DECIMAL(10,2) DEFAULT 0,
    memory_bytes BIGINT DEFAULT 0,
    memory_used_bytes BIGINT DEFAULT 0,
    pod_count INTEGER DEFAULT 0,
    labels JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(cluster_id, name)
);

-- 集群资源使用情况表
CREATE TABLE cluster_resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    total_cpu_cores INTEGER DEFAULT 0,
    used_cpu_cores DECIMAL(10,2) DEFAULT 0,
    cpu_usage_percent DECIMAL(5,2) DEFAULT 0,
    total_memory_bytes BIGINT DEFAULT 0,
    used_memory_bytes BIGINT DEFAULT 0,
    memory_usage_percent DECIMAL(5,2) DEFAULT 0,
    total_storage_bytes BIGINT DEFAULT 0,
    used_storage_bytes BIGINT DEFAULT 0,
    storage_usage_percent DECIMAL(5,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 事件表
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    node_id UUID NULL,
    event_type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    component VARCHAR(100),
    first_timestamp TIMESTAMP WITH TIME ZONE,
    last_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    count INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 安全策略表
CREATE TABLE security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL UNIQUE,
    pod_security_standard VARCHAR(50) DEFAULT 'baseline' CHECK (pod_security_standard IN ('privileged', 'baseline', 'restricted', 'disabled')),
    network_policies_enabled BOOLEAN DEFAULT false,
    network_policies_count INTEGER DEFAULT 0,
    rbac_enabled BOOLEAN DEFAULT true,
    rbac_roles_count INTEGER DEFAULT 0,
    audit_logging_enabled BOOLEAN DEFAULT false,
    audit_logging_mode VARCHAR(20) DEFAULT 'none' CHECK (audit_logging_mode IN ('none', 'metadata', 'request', 'request-response')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 自动伸缩策略表
CREATE TABLE autoscaling_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT false,
    min_nodes INTEGER DEFAULT 1,
    max_nodes INTEGER DEFAULT 10,
    scale_up_threshold INTEGER DEFAULT 70,
    scale_down_threshold INTEGER DEFAULT 30,
    hpa_count INTEGER DEFAULT 0,
    cluster_autoscaler_enabled BOOLEAN DEFAULT false,
    vpa_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
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

CREATE INDEX idx_nodes_cluster_id ON nodes(cluster_id);
CREATE INDEX idx_nodes_name ON nodes(cluster_id, name);
CREATE INDEX idx_nodes_type ON nodes(type);
CREATE INDEX idx_nodes_status ON nodes(status);

CREATE INDEX idx_cluster_resources_cluster_id ON cluster_resources(cluster_id);
CREATE INDEX idx_cluster_resources_timestamp ON cluster_resources(cluster_id, timestamp);

CREATE INDEX idx_events_cluster_id ON events(cluster_id);
CREATE INDEX idx_events_timestamp ON events(last_timestamp);
CREATE INDEX idx_events_severity ON events(severity);

CREATE INDEX idx_security_policies_cluster_id ON security_policies(cluster_id);
CREATE INDEX idx_autoscaling_policies_cluster_id ON autoscaling_policies(cluster_id);

-- 应用触发器
CREATE TRIGGER update_clusters_updated_at
    BEFORE UPDATE ON clusters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cluster_states_updated_at
    BEFORE UPDATE ON cluster_states
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_nodes_updated_at
    BEFORE UPDATE ON nodes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cluster_resources_updated_at
    BEFORE UPDATE ON cluster_resources
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_security_policies_updated_at
    BEFORE UPDATE ON security_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_autoscaling_policies_updated_at
    BEFORE UPDATE ON autoscaling_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
