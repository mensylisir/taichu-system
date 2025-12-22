-- 添加安全策略检测的详细字段
ALTER TABLE security_policies 
ADD COLUMN rbac_details TEXT,
ADD COLUMN network_policy_details TEXT,
ADD COLUMN pod_security_details TEXT,
ADD COLUMN audit_logging_details TEXT,
ADD COLUMN cni_plugin VARCHAR(50),
ADD COLUMN pod_security_admission_mode VARCHAR(20) DEFAULT 'none';