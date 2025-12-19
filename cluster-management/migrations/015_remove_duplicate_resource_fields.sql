-- Migration 015: Remove duplicate resource fields from cluster_states table
-- These fields are duplicated in cluster_resources table and are not needed in cluster_states

BEGIN;

-- 删除 cluster_states 表中重复的资源字段
-- 这些字段在 cluster_resources 表中已经存在，cluster_states 表不需要
ALTER TABLE cluster_states
DROP COLUMN IF EXISTS total_cpu_cores,
DROP COLUMN IF EXISTS total_memory_bytes,
DROP COLUMN IF EXISTS total_storage_bytes,
DROP COLUMN IF EXISTS used_storage_bytes,
DROP COLUMN IF EXISTS storage_usage_percent;

COMMIT;
