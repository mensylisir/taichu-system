-- Add deployment type fields to backup_schedules table
-- This allows the system to know how to stop/start control plane components based on deployment type

-- Add etcd deployment type field
ALTER TABLE "backup_schedules" ADD COLUMN "etcd_deployment_type" VARCHAR(50) DEFAULT 'kubexm';

-- Add kubernetes deployment type field  
ALTER TABLE "backup_schedules" ADD COLUMN "k8s_deployment_type" VARCHAR(50) DEFAULT 'kubeadm';

-- Add comments to document the deployment types
COMMENT ON COLUMN "backup_schedules"."etcd_deployment_type" IS 'etcd deployment type: kubexm (systemctl), kubeadm (manifest mv)';
COMMENT ON COLUMN "backup_schedules"."k8s_deployment_type" IS 'kubernetes deployment type: kubexm (systemctl), kubeadm (manifest mv)';
