-- 资源分类状态跟踪数据库迁移
-- PostgreSQL 12+

-- 创建更新时间戳触发器函数（如果不存在）
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ===== 资源分类记录表 =====
CREATE TABLE IF NOT EXISTS resource_classifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    tenant_id UUID,
    environment_id UUID,
    application_id UUID,
    resource_type VARCHAR(50) NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255),
    assigned_tenant VARCHAR(100),
    assigned_env VARCHAR(100),
    assigned_app VARCHAR(100),
    classification_rule VARCHAR(255),
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    classified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_classification_status CHECK (status IN ('pending', 'classified', 'failed'))
);

COMMENT ON COLUMN resource_classifications.resource_type IS '资源类型';
COMMENT ON COLUMN resource_classifications.resource_name IS '资源名称';
COMMENT ON COLUMN resource_classifications.namespace IS '命名空间';
COMMENT ON COLUMN resource_classifications.assigned_tenant IS '分配的租户名';
COMMENT ON COLUMN resource_classifications.assigned_env IS '分配的环境名';
COMMENT ON COLUMN resource_classifications.assigned_app IS '分配的应用名';
COMMENT ON COLUMN resource_classifications.classification_rule IS '分类规则';
COMMENT ON COLUMN resource_classifications.status IS '状态';
COMMENT ON COLUMN resource_classifications.error_message IS '错误信息';
COMMENT ON COLUMN resource_classifications.classified_at IS '分类时间';

-- ===== 资源分类历史表 =====
CREATE TABLE IF NOT EXISTS resource_classification_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'running',
    total_resources INTEGER DEFAULT 0,
    classified_resources INTEGER DEFAULT 0,
    failed_resources INTEGER DEFAULT 0,
    new_tenants INTEGER DEFAULT 0,
    new_environments INTEGER DEFAULT 0,
    new_applications INTEGER DEFAULT 0,
    details JSONB DEFAULT '{}',
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER DEFAULT 0,
    CONSTRAINT chk_history_status CHECK (status IN ('running', 'completed', 'failed')),
    CONSTRAINT chk_trigger_type CHECK (trigger_type IN ('import', 'manual', 'scheduled'))
);

COMMENT ON COLUMN resource_classification_history.trigger_type IS '触发类型';
COMMENT ON COLUMN resource_classification_history.status IS '状态';
COMMENT ON COLUMN resource_classification_history.total_resources IS '总资源数';
COMMENT ON COLUMN resource_classification_history.classified_resources IS '已分类资源数';
COMMENT ON COLUMN resource_classification_history.failed_resources IS '分类失败资源数';
COMMENT ON COLUMN resource_classification_history.new_tenants IS '新增租户数';
COMMENT ON COLUMN resource_classification_history.new_environments IS '新增环境数';
COMMENT ON COLUMN resource_classification_history.new_applications IS '新增应用数';
COMMENT ON COLUMN resource_classification_history.details IS '详细信息';
COMMENT ON COLUMN resource_classification_history.error_message IS '错误信息';
COMMENT ON COLUMN resource_classification_history.completed_at IS '完成时间';
COMMENT ON COLUMN resource_classification_history.duration_seconds IS '耗时秒数';

-- ===== 创建索引 =====
-- resource_classifications
CREATE INDEX IF NOT EXISTS idx_resource_classifications_cluster_id ON resource_classifications(cluster_id);
CREATE INDEX IF NOT EXISTS idx_resource_classifications_tenant_id ON resource_classifications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_resource_classifications_environment_id ON resource_classifications(environment_id);
CREATE INDEX IF NOT EXISTS idx_resource_classifications_application_id ON resource_classifications(application_id);
CREATE INDEX IF NOT EXISTS idx_resource_classifications_status ON resource_classifications(status);
CREATE INDEX IF NOT EXISTS idx_resource_classifications_resource_type ON resource_classifications(resource_type);

-- resource_classification_history
CREATE INDEX IF NOT EXISTS idx_classification_history_cluster_id ON resource_classification_history(cluster_id);
CREATE INDEX IF NOT EXISTS idx_classification_history_status ON resource_classification_history(status);
CREATE INDEX IF NOT EXISTS idx_classification_history_trigger_type ON resource_classification_history(trigger_type);
CREATE INDEX IF NOT EXISTS idx_classification_history_started_at ON resource_classification_history(started_at);

-- ===== 应用触发器 =====
-- resource_classifications
DROP TRIGGER IF EXISTS update_resource_classifications_updated_at ON resource_classifications;
CREATE TRIGGER update_resource_classifications_updated_at
    BEFORE UPDATE ON resource_classifications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===== 创建视图（用于查询分类统计）=====
-- 租户分类统计视图
CREATE OR REPLACE VIEW tenant_classification_summary AS
SELECT
    t.id as tenant_id,
    t.name as tenant_name,
    t.type as tenant_type,
    COUNT(DISTINCT rc.environment_id) as classified_environments,
    COUNT(DISTINCT rc.application_id) as classified_applications,
    COUNT(DISTINCT CASE WHEN rc.status = 'classified' THEN rc.id END) as successfully_classified,
    COUNT(DISTINCT CASE WHEN rc.status = 'failed' THEN rc.id END) as failed_classifications
FROM tenants t
LEFT JOIN resource_classifications rc ON t.id = rc.tenant_id
GROUP BY t.id, t.name, t.type;

-- 集群分类统计视图
CREATE OR REPLACE VIEW cluster_classification_summary AS
SELECT
    c.id as cluster_id,
    c.name as cluster_name,
    COUNT(DISTINCT rc.id) as total_classified_resources,
    COUNT(DISTINCT CASE WHEN rc.status = 'classified' THEN rc.id END) as successfully_classified,
    COUNT(DISTINCT CASE WHEN rc.status = 'failed' THEN rc.id END) as failed_classifications,
    MAX(rc.classified_at) as last_classified_at
FROM clusters c
LEFT JOIN resource_classifications rc ON c.id = rc.cluster_id
GROUP BY c.id, c.name;

-- 分类历史统计视图
CREATE OR REPLACE VIEW classification_history_summary AS
SELECT
    cluster_id,
    trigger_type,
    status,
    COUNT(*) as execution_count,
    AVG(duration_seconds) as avg_duration_seconds,
    MAX(started_at) as last_execution_at
FROM resource_classification_history
GROUP BY cluster_id, trigger_type, status;
