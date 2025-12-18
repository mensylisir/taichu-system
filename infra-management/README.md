# Infrastructure Management Backend

Go语言实现的微服务后端，提供虚拟机、存储和防火墙服务的数据管理。

## 系统要求

- Go 1.21+
- PostgreSQL 13+

## 数据库配置

PostgreSQL连接信息：
- 主机：172.30.1.12
- 端口：32172
- 用户：postgres
- 密码：Def@u1tpwd
- 数据库：railcloud

## 安装和运行

1. 安装依赖：

```bash
go mod tidy
```

2. 初始化数据库（运行 `init-db.sql` 文件中的SQL脚本）

3. 设置环境变量（可选）：

```bash
export DB_HOST=172.30.1.12
export DB_PORT=32172
export DB_USER=postgres
export DB_PASSWORD=Def@u1tpwd
export DB_NAME=railcloud
export SERVER_PORT=8080
```

4. 运行服务：

```bash
go run cmd/server/main.go
```

服务将在端口 8080 上启动。

## API端点

- `GET /api/v1/vms` - 获取虚拟机列表
- `GET /api/v1/storages` - 获取存储列表
- `GET /api/v1/firewall-rules` - 获取防火墙规则列表

## 项目结构

```
infra-management/
├── cmd/server/          # 应用程序入口
├── internal/
│   ├── api/             # API处理器
│   ├── config/          # 配置管理
│   ├── db/              # 数据库连接
│   ├── models/          # 数据模型
│   └── routes/          # 路由配置
├── init-db.sql          # 数据库初始化脚本
└── README.md
```

## 数据模型

### 虚拟机 (vms)
- 名称、操作系统、状态、CPU、内存、存储、IP地址、集群、创建时间

### 存储 (storages)
- 名称、类型、容量、IOPS、状态、挂载到、创建时间

### 防火墙规则 (firewall_rules)
- 名称、协议、端口、源IP、目标IP、操作、状态、创建时间