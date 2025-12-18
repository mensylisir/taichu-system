# 数据库设置指南

## PostgreSQL数据库连接信息

- **主机**: 172.30.1.12
- **端口**: 32172
- **用户**: postgres
- **密码**: Def@u1tpwd
- **数据库**: taichu

## 创建表和插入数据

请使用PostgreSQL客户端连接数据库并执行以下SQL脚本：

```bash
psql -h 172.30.1.12 -p 32172 -U postgres -d taichu -f init-db.sql
```

或者您也可以：

1. 连接到PostgreSQL数据库
2. 使用taichu数据库
3. 运行 `init-db.sql` 文件中的SQL命令

## 手动初始化步骤

如果您无法运行psql命令，可以手动在PostgreSQL客户端中执行以下SQL：

```sql
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
```

然后执行插入数据的SQL命令（详见 `init-db.sql` 文件）。