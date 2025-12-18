CREATE TABLE IF NOT EXISTS vms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    os VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    cpu VARCHAR(50) NOT NULL,
    memory VARCHAR(50) NOT NULL,
    storage VARCHAR(50) NOT NULL,
    ip VARCHAR(50) NOT NULL,
    cluster VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS storages (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    capacity VARCHAR(50) NOT NULL,
    iops VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    mounted_to VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS firewall_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    protocol VARCHAR(20) NOT NULL,
    port VARCHAR(20) NOT NULL,
    source_ip VARCHAR(50) NOT NULL,
    target_ip VARCHAR(50) NOT NULL,
    action VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO vms (name, os, status, cpu, memory, storage, ip, cluster, created_at)
VALUES
    ('vm-web-01', 'Ubuntu 22.04 LTS', '运行中', '4 核', '8 GB', '100 GB SSD', '10.0.1.10', '生产集群', '2024-11-15'),
    ('vm-db-01', 'CentOS 8', '运行中', '8 核', '32 GB', '500 GB SSD', '10.0.1.20', '生产集群', '2024-11-10'),
    ('vm-cache-01', 'Ubuntu 22.04 LTS', '已停止', '2 核', '4 GB', '50 GB SSD', '10.0.1.30', '测试集群', '2024-11-20'),
    ('vm-dev-01', 'CentOS 7', '运行中', '4 核', '16 GB', '200 GB SSD', '10.0.1.40', '测试集群', '2024-11-18'),
    ('vm-prod-02', 'Ubuntu 20.04 LTS', '运行中', '16 核', '64 GB', '1 TB SSD', '10.0.1.50', '生产集群', '2024-11-12'),
    ('vm-test-01', 'CentOS 9', '运行中', '2 核', '4 GB', '80 GB SSD', '10.0.2.10', '测试集群', '2024-11-22'),
    ('vm-middleware-01', 'Ubuntu 20.04 LTS', '运行中', '8 核', '16 GB', '300 GB SSD', '10.0.2.20', '生产集群', '2024-11-21'),
    ('vm-backup-01', 'CentOS 7', '已停止', '4 核', '8 GB', '500 GB HDD', '10.0.2.30', '备份集群', '2024-11-19'),
    ('vm-api-01', 'Ubuntu 22.04 LTS', '运行中', '4 核', '8 GB', '100 GB SSD', '10.0.2.40', '生产集群', '2024-11-23'),
    ('vm-log-01', 'CentOS 8', '运行中', '2 核', '4 GB', '200 GB HDD', '10.0.2.50', '日志集群', '2024-11-24'),
    ('vm-monitor-01', 'Ubuntu 20.04 LTS', '运行中', '2 核', '4 GB', '50 GB SSD', '10.0.2.60', '监控集群', '2024-11-25'),
    ('vm-devops-01', 'CentOS 7', '运行中', '4 核', '16 GB', '200 GB SSD', '10.0.2.70', '开发集群', '2024-11-26'),
    ('vm-security-01', 'Ubuntu 22.04 LTS', '运行中', '4 核', '8 GB', '100 GB SSD', '10.0.2.80', '安全集群', '2024-11-27'),
    ('vm-analytics-01', 'CentOS 8', '运行中', '8 核', '32 GB', '500 GB SSD', '10.0.2.90', '数据分析集群', '2024-11-28'),
    ('vm-batch-01', 'Ubuntu 20.04 LTS', '已停止', '16 核', '64 GB', '1 TB SSD', '10.0.3.10', '批处理集群', '2024-11-29');

INSERT INTO storages (name, type, capacity, iops, status, mounted_to, created_at)
VALUES
    ('ssd-web-01', 'SSD', '100 GB', '10,000', '已挂载', 'vm-web-01', '2024-11-15'),
    ('ssd-db-01', 'SSD', '500 GB', '20,000', '已挂载', 'vm-db-01', '2024-11-10'),
    ('ssd-cache-01', 'SSD', '100 GB', '10,000', '未挂载', NULL, '2024-11-20'),
    ('hdd-backup-01', 'HDD', '2 TB', '5,000', '已挂载', 'vm-dev-01', '2024-11-18'),
    ('ssd-prod-02', 'SSD', '1 TB', '50,000', '已挂载', 'vm-prod-02', '2024-11-12'),
    ('ssd-test-01', 'SSD', '80 GB', '8,000', '已挂载', 'vm-test-01', '2024-11-22'),
    ('ssd-middleware-01', 'SSD', '300 GB', '15,000', '已挂载', 'vm-middleware-01', '2024-11-21'),
    ('hdd-backup-02', 'HDD', '500 GB', '3,000', '已挂载', 'vm-backup-01', '2024-11-19'),
    ('ssd-api-01', 'SSD', '100 GB', '10,000', '已挂载', 'vm-api-01', '2024-11-23'),
    ('hdd-log-01', 'HDD', '200 GB', '2,000', '已挂载', 'vm-log-01', '2024-11-24'),
    ('ssd-monitor-01', 'SSD', '50 GB', '5,000', '已挂载', 'vm-monitor-01', '2024-11-25'),
    ('ssd-devops-01', 'SSD', '200 GB', '15,000', '已挂载', 'vm-devops-01', '2024-11-26'),
    ('ssd-security-01', 'SSD', '100 GB', '10,000', '已挂载', 'vm-security-01', '2024-11-27'),
    ('ssd-analytics-01', 'SSD', '500 GB', '20,000', '已挂载', 'vm-analytics-01', '2024-11-28'),
    ('ssd-batch-01', 'SSD', '1 TB', '50,000', '已挂载', 'vm-batch-01', '2024-11-29');

INSERT INTO firewall_rules (name, protocol, port, source_ip, target_ip, action, status, created_at)
VALUES
    ('allow-http', 'TCP', '80', '0.0.0.0/0', '10.0.1.10', '允许', '启用', '2024-11-15'),
    ('allow-https', 'TCP', '443', '0.0.0.0/0', '10.0.1.10', '允许', '启用', '2024-11-15'),
    ('allow-ssh', 'TCP', '22', '10.0.0.0/8', '10.0.1.0/24', '允许', '启用', '2024-11-10'),
    ('allow-db', 'TCP', '5432', '10.0.1.0/24', '10.0.1.20', '允许', '启用', '2024-11-10'),
    ('deny-all', 'TCP', '*', '0.0.0.0/0', '10.0.1.40', '拒绝', '禁用', '2024-11-18'),
    ('allow-test', 'TCP', '8080', '10.0.0.0/8', '10.0.2.10', '允许', '启用', '2024-11-22'),
    ('allow-middleware', 'TCP', '9090', '10.0.0.0/8', '10.0.2.20', '允许', '启用', '2024-11-21'),
    ('deny-backup', 'TCP', '*', '0.0.0.0/0', '10.0.2.30', '拒绝', '禁用', '2024-11-19'),
    ('allow-api', 'TCP', '3000', '10.0.0.0/8', '10.0.2.40', '允许', '启用', '2024-11-23'),
    ('allow-log', 'TCP', '514', '10.0.0.0/8', '10.0.2.50', '允许', '启用', '2024-11-24'),
    ('allow-monitor', 'TCP', '9090', '10.0.0.0/8', '10.0.2.60', '允许', '启用', '2024-11-25'),
    ('allow-devops', 'TCP', '22', '10.0.0.0/8', '10.0.2.70', '允许', '启用', '2024-11-26'),
    ('allow-security', 'TCP', '8443', '10.0.0.0/8', '10.0.2.80', '允许', '启用', '2024-11-27'),
    ('allow-analytics', 'TCP', '5432', '10.0.0.0/8', '10.0.2.90', '允许', '启用', '2024-11-28'),
    ('allow-batch', 'TCP', '*', '10.0.0.0/8', '10.0.3.10', '拒绝', '禁用', '2024-11-29');