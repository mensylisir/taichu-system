-- 移除clusters表中version字段的默认值
-- 清理所有旧的1.0.0版本数据，触发重新同步

UPDATE clusters
SET version = ''
WHERE version = '1.0.0';

-- 如果ClusterState表中的KubernetesVersion为空，从集群重新获取
-- 这里只是标记，实际获取需要通过健康检查
UPDATE cluster_states
SET kubernetes_version = ''
WHERE kubernetes_version = '1.0.0' OR kubernetes_version IS NULL;
