# 生产级多集群管理系统

一个基于Go语言开发的企业级集群管理系统，支持Kubernetes集群的全生命周期管理。

## 功能特性

### 核心功能
- 🔐 **安全加密存储**: 使用AES-256-GCM加密kubeconfig，确保敏感信息安全
- 🔄 **异步健康检查**: 后台Worker定时检查集群状态，毫秒级响应列表查询
- 📊 **实时状态监控**: 监控节点数、CPU/内存/存储资源水位、Kubernetes版本
- 🎯 **高性能架构**: 连接池管理、LRU缓存、并发控制
- 🗄️ **生产级数据库**: PostgreSQL多表设计，软删除、索引优化
- 📡 **RESTful API**: 标准API设计，支持分页、筛选、搜索

### 20+ API接口
- **集群管理**: 创建、导入、列表、详情、拓扑 (5个接口)
- **节点监控**: 节点列表、节点详情 (2个接口)
- **事件管理**: 事件列表 (1个接口)
- **策略管理**: 安全策略、自动伸缩策略 (2个接口)
- **备份系统**: 创建、列表、详情、恢复、删除备份，备份计划 (6个接口)
- **审计日志**: 审计事件查询 (1个接口)
- **集群扩展**: 资源扩展、扩展历史 (2个接口)
- **导入管理**: 导入集群、导入记录、导入状态 (3个接口)

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

### 快速测试

使用提供的测试脚本：

```bash
# 启动服务器
./test/scripts/test-cluster-api.sh start

# 创建集群
./test/scripts/test-cluster-api.sh create-cluster

# 查看集群列表
./test/scripts/test-cluster-api.sh list-clusters

# 获取集群详情
./test/scripts/test-cluster-api.sh get-cluster <CLUSTER_ID>

# 查看集群拓扑
./test/scripts/test-cluster-api.sh get-topology

# 测试所有接口
./test/scripts/test-cluster-api.sh test-all

# 停止服务器
./test/scripts/test-cluster-api.sh stop
```

### 主要API接口

#### 集群管理

**创建集群**
```bash
POST /api/v1/clusters
```

**导入集群**
```bash
POST /api/v1/clusters/import
```

**获取集群列表**
```bash
GET /api/v1/clusters?page=1&limit=10&status=active&search=prod
```

**获取集群详情**
```bash
GET /api/v1/clusters/{id}
```

**获取集群拓扑**
```bash
GET /api/v1/clusters/topology
```

#### 节点监控

**获取节点列表**
```bash
GET /api/v1/clusters/{clusterId}/nodes
```

**获取节点详情**
```bash
GET /api/v1/clusters/{clusterId}/nodes/{nodeName}
```

#### 事件管理

**获取事件列表**
```bash
GET /api/v1/clusters/{clusterId}/events?type=Warning
```

#### 策略管理

**获取安全策略**
```bash
GET /api/v1/clusters/{clusterId}/security-policies
```

**获取自动伸缩策略**
```bash
GET /api/v1/clusters/{clusterId}/autoscaling-policies
```

#### 备份系统

**创建备份**
```bash
POST /api/v1/clusters/{clusterId}/backups
```

**获取备份列表**
```bash
GET /api/v1/clusters/{clusterId}/backups
```

**获取备份详情**
```bash
GET /api/v1/clusters/{clusterId}/backups/{backupId}
```

**恢复备份**
```bash
POST /api/v1/clusters/{clusterId}/backups/{backupId}/restore
```

**删除备份**
```bash
DELETE /api/v1/clusters/{clusterId}/backups/{backupId}
```

**获取备份计划**
```bash
GET /api/v1/clusters/{clusterId}/backup-schedules
```

#### 审计日志

**获取审计事件**
```bash
GET /api/v1/clusters/{clusterId}/audit?event_type=create
```

#### 集群扩展

**请求扩展**
```bash
POST /api/v1/clusters/{clusterId}/expansion
```

**获取扩展历史**
```bash
GET /api/v1/clusters/{clusterId}/expansion/history
```

#### 导入管理

**获取导入记录列表**
```bash
GET /api/v1/clusters/imports
```

**获取导入状态**
```bash
GET /api/v1/imports/{importId}/status
```

### 完整API文档

详细的API文档请参考：[API文档](docs/API.md)

## 核心设计

### 数据库设计

#### 核心表

**clusters** - 集群配置
- 存储集群元数据和加密的kubeconfig
- 支持软删除（`deleted_at`字段）
- JSONB标签字段支持灵活过滤
- UUID主键，支持分布式

**cluster_states** - 状态缓存
- 存储实时状态信息（节点数、资源统计、存储容量、心跳时间）
- 每个集群只有一个最新状态记录
- 由后台Worker异步更新

**nodes** - 节点信息
- 存储集群中所有节点的详细信息
- 包括CPU、内存、Pod数量等

**events** - 事件记录
- 存储集群中的关键事件
- 支持按类型、时间、命名空间筛选

**security_policies** - 安全策略
- 存储Pod安全策略、网络策略、RBAC策略

**autoscaling_policies** - 自动伸缩策略
- 存储HPA/VPA配置信息

**cluster_backups** - 备份记录
- 存储所有备份的元数据
- 包括备份类型、状态、创建时间等

**backup_schedules** - 备份计划
- 存储自动备份计划配置

**audit_events** - 审计日志
- 记录所有关键操作的审计信息
- 包括操作人、IP地址、操作结果等

**cluster_expansions** - 扩展记录
- 记录集群资源扩展历史
- 包括扩展前后资源对比

**import_records** - 导入记录
- 记录集群导入过程的详细信息

**cluster_resources** - 资源快照
- 存储集群资源的定期快照

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
