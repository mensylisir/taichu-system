-- 移除 kubeconfig_nonce 字段，因为加密服务已经将 nonce 前置到密文中
-- 这个迁移修复了 "cipher: message authentication failed" 错误

-- 首先移除 NOT NULL 约束
ALTER TABLE clusters ALTER COLUMN kubeconfig_nonce DROP NOT NULL;

-- 然后移除该字段
ALTER TABLE clusters DROP COLUMN IF EXISTS kubeconfig_nonce;