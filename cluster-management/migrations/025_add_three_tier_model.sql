-- 三级分类模型（租户-环境-应用）数据库迁移
-- PostgreSQL 12+

-- 创建更新时间戳触发器函数（如果不存在）
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ===== 租户表 =====
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    type VARCHAR(50) NOT NULL,
    description TEXT,
    labels JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_tenant_type CHECK (type IN ('system', 'default', 'user_created')),
    CONSTRAINT chk_tenant_status CHECK (status IN ('active', 'suspended'))
);

COMMENT ON COLUMN tenants.name IS '租户名称';
COMMENT ON COLUMN tenants.display_name IS '显示名称';
COMMENT ON COLUMN tenants.type IS '租户类型(system/default/user_created)';
COMMENT ON COLUMN tenants.description IS '描述';
COMMENT ON COLUMN tenants.labels IS '标签';
COMMENT ON COLUMN tenants.status IS '状态';
COMMENT ON COLUMN tenants.is_system IS '是否为系统预定义租户';

-- ===== 环境表 =====
CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    cluster_id UUID NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    labels JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_environment_status CHECK (status IN ('active', 'inactive', 'terminating')),
    UNIQUE(cluster_id, namespace)
);

COMMENT ON COLUMN environments.namespace IS 'Kubernetes Namespace名称';
COMMENT ON COLUMN environments.display_name IS '显示名称';
COMMENT ON COLUMN environments.description IS '描述';
COMMENT ON COLUMN environments.labels IS '标签';
COMMENT ON COLUMN environments.status IS '状态';

-- ===== 应用表 =====
CREATE TABLE IF NOT EXISTS applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    labels JSONB DEFAULT '{}',
    workload_types JSONB DEFAULT '[]',
    service_names JSONB DEFAULT '[]',
    deployment_count INTEGER DEFAULT 0,
    pod_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(environment_id, name)
);

COMMENT ON COLUMN applications.name IS '应用名称';
COMMENT ON COLUMN applications.display_name IS '显示名称';
COMMENT ON COLUMN applications.description IS '描述';
COMMENT ON COLUMN applications.labels IS '标签';
COMMENT ON COLUMN applications.workload_types IS '工作负载类型列表';
COMMENT ON COLUMN applications.service_names IS '关联Service名称列表';
COMMENT ON COLUMN applications.deployment_count IS 'Deployment数量';
COMMENT ON COLUMN applications.pod_count IS 'Pod数量';

-- ===== 资源配额表 =====
CREATE TABLE IF NOT EXISTS resource_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL UNIQUE REFERENCES environments(id) ON DELETE CASCADE,
    hard_limits JSONB DEFAULT '{}',
    used JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    last_synced_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_resource_quota_status CHECK (status IN ('active', 'terminating'))
);

COMMENT ON COLUMN resource_quotas.hard_limits IS '硬性限制';
COMMENT ON COLUMN resource_quotas.used IS '当前使用量';
COMMENT ON COLUMN resource_quotas.status IS '状态';
COMMENT ON COLUMN resource_quotas.last_synced_at IS '最后同步时间';

-- ===== 租户配额表 =====
CREATE TABLE IF NOT EXISTS tenant_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    hard_limits JSONB DEFAULT '{}',
    allocated JSONB DEFAULT '{}',
    available JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_tenant_quota_status CHECK (status IN ('active', 'suspended'))
);

COMMENT ON COLUMN tenant_quotas.hard_limits IS '租户总配额';
COMMENT ON COLUMN tenant_quotas.allocated IS '已分配给环境的配额总和';
COMMENT ON COLUMN tenant_quotas.available IS '可用配额';
COMMENT ON COLUMN tenant_quotas.status IS '状态';

-- ===== 应用资源规格表 =====
CREATE TABLE IF NOT EXISTS application_resource_specs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL UNIQUE REFERENCES applications(id) ON DELETE CASCADE,
    default_request JSONB DEFAULT '{}',
    default_limit JSONB DEFAULT '{}',
    max_replicas INTEGER DEFAULT 0,
    current_replicas INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN application_resource_specs.default_request IS '默认requests';
COMMENT ON COLUMN application_resource_specs.default_limit IS '默认limits';
COMMENT ON COLUMN application_resource_specs.max_replicas IS '最大副本数';
COMMENT ON COLUMN application_resource_specs.current_replicas IS '当前副本数';

-- ===== 创建索引 =====
-- tenants
CREATE INDEX IF NOT EXISTS idx_tenants_type ON tenants(type);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_is_system ON tenants(is_system);

-- environments
CREATE INDEX IF NOT EXISTS idx_environments_tenant_id ON environments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_environments_cluster_id ON environments(cluster_id);
CREATE INDEX IF NOT EXISTS idx_environments_status ON environments(status);

-- applications
CREATE INDEX IF NOT EXISTS idx_applications_tenant_id ON applications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_applications_environment_id ON applications(environment_id);

-- resource_quotas
CREATE INDEX IF NOT EXISTS idx_resource_quotas_environment_id ON resource_quotas(environment_id);
CREATE INDEX IF NOT EXISTS idx_resource_quotas_status ON resource_quotas(status);
CREATE INDEX IF NOT EXISTS idx_resource_quotas_last_synced_at ON resource_quotas(last_synced_at);

-- tenant_quotas
CREATE INDEX IF NOT EXISTS idx_tenant_quotas_tenant_id ON tenant_quotas(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_quotas_status ON tenant_quotas(status);

-- application_resource_specs
CREATE INDEX IF NOT EXISTS idx_application_resource_specs_application_id ON application_resource_specs(application_id);

-- ===== 应用触发器 =====
-- tenants
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- environments
DROP TRIGGER IF EXISTS update_environments_updated_at ON environments;
CREATE TRIGGER update_environments_updated_at
    BEFORE UPDATE ON environments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- applications
DROP TRIGGER IF EXISTS update_applications_updated_at ON applications;
CREATE TRIGGER update_applications_updated_at
    BEFORE UPDATE ON applications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- resource_quotas
DROP TRIGGER IF EXISTS update_resource_quotas_updated_at ON resource_quotas;
CREATE TRIGGER update_resource_quotas_updated_at
    BEFORE UPDATE ON resource_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- tenant_quotas
DROP TRIGGER IF EXISTS update_tenant_quotas_updated_at ON tenant_quotas;
CREATE TRIGGER update_tenant_quotas_updated_at
    BEFORE UPDATE ON tenant_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- application_resource_specs
DROP TRIGGER IF EXISTS update_application_resource_specs_updated_at ON application_resource_specs;
CREATE TRIGGER update_application_resource_specs_updated_at
    BEFORE UPDATE ON application_resource_specs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===== 插入预定义租户 =====
INSERT INTO tenants (name, display_name, type, description, is_system, status)
VALUES
    ('system', '系统租户', 'system', '系统预定义租户，用于承载系统组件', true, 'active'),
    ('default', '默认租户', 'default', '系统预定义租户，用于承载现有业务组件', true, 'active')
ON CONFLICT (name) DO NOTHING;

-- ===== 创建视图（可选，用于查询统计信息）=====
-- 租户资源统计视图
CREATE OR REPLACE VIEW tenant_resource_summary AS
SELECT
    t.id as tenant_id,
    t.name as tenant_name,
    t.display_name as tenant_display_name,
    t.type as tenant_type,
    COUNT(DISTINCT e.id) as environment_count,
    COUNT(DISTINCT a.id) as application_count,
    COALESCE(SUM(a.deployment_count), 0) as total_deployments,
    COALESCE(SUM(a.pod_count), 0) as total_pods
FROM tenants t
LEFT JOIN environments e ON t.id = e.tenant_id
LEFT JOIN applications a ON e.id = a.environment_id
GROUP BY t.id, t.name, t.display_name, t.type;

-- 环境资源统计视图
CREATE OR REPLACE VIEW environment_resource_summary AS
SELECT
    e.id as environment_id,
    e.namespace,
    e.display_name as environment_display_name,
    t.name as tenant_name,
    COUNT(DISTINCT a.id) as application_count,
    COALESCE(SUM(a.deployment_count), 0) as total_deployments,
    COALESCE(SUM(a.pod_count), 0) as total_pods
FROM environments e
JOIN tenants t ON e.tenant_id = t.id
LEFT JOIN applications a ON e.id = a.environment_id
GROUP BY e.id, e.namespace, e.display_name, t.name;
