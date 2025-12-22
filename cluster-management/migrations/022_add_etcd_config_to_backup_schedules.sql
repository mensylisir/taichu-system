-- AddEtcdConfigToBackupSchedules
ALTER TABLE "backup_schedules" ADD COLUMN "etcd_endpoints" TEXT;
ALTER TABLE "backup_schedules" ADD COLUMN "etcd_ca_cert" TEXT;
ALTER TABLE "backup_schedules" ADD COLUMN "etcd_cert" TEXT;
ALTER TABLE "backup_schedules" ADD COLUMN "etcd_key" TEXT;
ALTER TABLE "backup_schedules" ADD COLUMN "etcd_data_dir" VARCHAR(255) DEFAULT '/var/lib/etcd';
ALTER TABLE "backup_schedules" ADD COLUMN "etcdctl_path" VARCHAR(255) DEFAULT '/usr/bin/etcdctl';
ALTER TABLE "backup_schedules" ADD COLUMN "ssh_username" VARCHAR(255) DEFAULT 'root';
ALTER TABLE "backup_schedules" ADD COLUMN "ssh_password" VARCHAR(255);
