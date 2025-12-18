# 生产级多集群管理系统

一个基于Go语言开发的生产级多集群管理系统，支持Kubernetes集群的导入、管理和健康监控。

## 功能特性

- 🔐 **安全加密存储**: 使用AES-256-GCM加密kubeconfig，确保敏感信息安全
- 🔄 **异步健康检查**: 后台Worker定时检查集群状态，毫秒级响应列表查询
- 📊 **实时状态监控**: 监控节点数、CPU/内存资源水位、Kubernetes版本
- 🎯 **高性能架构**: 连接池管理、LRU缓存、并发控制
- 🗄️ **生产级数据库**: PostgreSQL双表设计（配置+状态），软删除、索引优化
- 📡 **RESTful API**: 标准API设计，支持分页、筛选、搜索

## 技术栈

- **Web框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL 12+
- **Kubernetes客户端**: client-go
- **配置管理**: Viper (YAML)
- **加密**: AES-256-GCM

## 项目结构

```
cluster-management/
├── cmd/
│   └── server/
│       └── main.go                          # 应用入口
├── internal/
│   ├── config/
│   │   └── config.go                        # 配置管理
│   ├── model/
│   │   └── cluster.go                       # 数据模型
│   ├── repository/
│   │   ├── cluster_repository.go            # 集群数据访问层
│   │   └── cluster_state_repository.go      # 状态数据访问层
│   ├── service/
│   │   ├── cluster_service.go               # 集群业务逻辑
│   │   ├── cluster_manager.go               # 连接池管理
│   │   ├── encryption_service.go            # 加密服务
│   │   └── worker/
│   │       └── health_check_worker.go       # 异步健康检查
│   ├── handler/
│   │   └── cluster_handler.go               # API处理器
│   └── utils/
│       └── errors.go                        # 错误处理
├── pkg/
│   └── utils/
│       └── response.go                      # 响应格式化
├── configs/
│   └── config.yaml                          # 配置文件
├── migrations/
│   └── 001_create_clusters_tables.sql       # 数据库迁移
├── go.mod
└── README.md
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- PostgreSQL 12+
- Kubernetes 1.20+

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 配置数据库

创建PostgreSQL数据库：

```sql
CREATE DATABASE cluster_management;
```

执行数据库迁移：

```bash
psql -U postgres -d cluster_management -f migrations/001_create_clusters_tables.sql
```

### 4. 配置应用

编辑 `configs/config.yaml`：

```yaml
database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "your_password"
  dbname: "cluster_management"

encryption:
  key: "your-32-character-encryption-key-here"

worker:
  enabled: true
  check_interval: 5m
  max_concurrency: 10
```

**重要**: 加密密钥必须是32个字符，建议使用随机生成的密钥。

### 5. 启动应用

```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 启动。

## API 文档

### 1. 导入集群 (POST /api/v1/clusters)

导入新的Kubernetes集群：

```bash
curl -X POST http://localhost:8080/api/v1/clusters \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-shanghai-01",
    "description": "上海生产环境核心集群",
    "kubeconfig": "base64_encoded_kubeconfig",
    "labels": {
      "env": "prod",
      "region": "shanghai"
    }
  }'
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "c-7382abcd",
    "name": "prod-shanghai-01",
    "description": "上海生产环境核心集群",
    "status": "unknown",
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

### 2. 获取集群列表 (GET /api/v1/clusters)

查询集群列表（支持分页和筛选）：

```bash
curl "http://localhost:8080/api/v1/clusters?page=1&limit=20&status=healthy&search=prod"
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
    "page": 1,
    "limit": 20,
    "clusters": [
      {
        "id": "c-7382abcd",
        "name": "prod-shanghai-01",
        "description": "上海生产环境核心集群",
        "status": "healthy",
        "node_count": 12,
        "version": "1.0.0",
        "labels": {
          "env": "prod",
          "region": "shanghai"
        },
        "created_at": "2024-01-01T10:00:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
      }
    ]
  }
}
```

### 3. 获取集群详情 (GET /api/v1/clusters/{id})

获取集群详细信息：

```bash
curl http://localhost:8080/api/v1/clusters/c-7382abcd
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "c-7382abcd",
    "name": "prod-shanghai-01",
    "description": "上海生产环境核心集群",
    "status": "healthy",
    "version": "1.0.0",
    "labels": {
      "env": "prod",
      "region": "shanghai"
    },
    "node_count": 12,
    "total_cpu_cores": 64,
    "total_memory_bytes": 256000000000,
    "kubernetes_version": "v1.28.3",
    "api_server_url": "https://10.0.0.1:6443",
    "last_heartbeat_at": "2024-01-01T12:00:00Z",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

## 核心设计

### 数据库设计

#### clusters 表（集群配置）
- 存储集群元数据和加密的kubeconfig
- 支持软删除（`deleted_at`字段）
- JSONB标签字段支持灵活过滤
- UUID主键，支持分布式

#### cluster_states 表（状态缓存）
- 存储实时状态信息（节点数、资源统计、心跳时间）
- 每个集群只有一个最新状态记录
- 由后台Worker异步更新

### 核心架构

1. **分层架构**: Model → Repository → Service → Handler
2. **连接池管理**: ClusterManager管理kubernetes.Clientset
   - 懒加载：仅在需要时创建客户端
   - LRU缓存：限制最大客户端数，自动清理过期连接
3. **异步同步**: Worker定时轮询集群状态
   - 并发控制：使用信号量限制并发数
   - 错误处理：记录失败状态，支持重试
4. **安全加密**: AES-256-GCM加密kubeconfig
   - 每个kubeconfig使用随机nonce
   - 密钥通过配置管理（建议环境变量）

### 性能优化

- **连接池**: 限制最大客户端数100，LRU清理机制
- **并发控制**: Worker最大并发10，避免过载
- **数据库优化**: 索引、JSONB字段、软删除查询优化
- **懒加载**: Kubernetes客户端仅在需要时创建
- **状态缓存**: 列表查询直接读取cluster_states表，避免实时API调用

## 配置说明

### Worker配置

```yaml
worker:
  enabled: true              # 是否启用Worker
  check_interval: 5m         # 健康检查间隔（默认5分钟）
  max_concurrency: 10        # 最大并发数
  retry_attempts: 3          # 重试次数
  retry_delay: 30s           # 重试延迟
```

### ClusterManager配置

```yaml
cluster_manager:
  client_timeout: 30s        # 客户端超时时间
  max_clients: 100          # 最大客户端数
  cleanup_interval: 30m     # 清理间隔
```

## 最佳实践

### 1. kubeconfig准备

导入集群前，确保kubeconfig：
- 包含有效的集群、用户和上下文信息
- 可以独立访问Kubernetes API（无交互式认证）
- Base64编码后传递

### 2. 安全建议

- **加密密钥**: 使用环境变量传递加密密钥，不要硬编码
- **数据库**: 配置SSL连接，限制访问IP
- **API安全**: 在生产环境中添加认证中间件（建议JWT或OAuth2）
- **网络安全**: 使用HTTPS，配置防火墙规则

### 3. 性能调优

- **数据库连接池**: 根据并发量调整`max_open_conns`
- **Worker并发**: 根据集群规模和K8s响应时间调整`max_concurrency`
- **监控**: 建议集成Prometheus监控指标

## 后续迭代

- [ ] 添加用户认证和权限控制
- [ ] 支持多云提供商（阿里云、AWS、腾讯云）
- [ ] 集成Prometheus监控指标
- [ ] 添加集群操作API（删除、更新、重新同步）
- [ ] 支持集群分组和标签管理
- [ ] 添加集群事件日志

## 许可证

MIT

## 贡献

欢迎提交Issue和Pull Request！

## 联系方式

如有问题，请创建Issue或联系维护者。
