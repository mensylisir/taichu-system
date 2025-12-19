# 太初集群管理系统 Kubernetes 部署指南

本文档描述了如何在 Kubernetes 集群上部署太初集群管理系统。

## 目录结构

```
deploy/
├── README.md              # 本文件
├── deploy.sh              # 自动部署脚本
├── uninstall.sh           # 卸载脚本
├── configmap.yaml         # ConfigMap 配置
├── secret.yaml            # Secret 配置
├── deployment.yaml        # Deployment 配置
├── service.yaml           # Service 配置
├── ingress.yaml           # Ingress 配置
├── postgres.yaml          # PostgreSQL 部署
├── hpa.yaml               # HPA 和 VPA 配置
├── pdb.yaml               # PodDisruptionBudget
└── networkpolicy.yaml     # NetworkPolicy
```

## 前置要求

1. **Kubernetes 集群版本**: >= 1.24
2. **工具要求**:
   - kubectl >= 1.24
   - docker >= 20.10
   - openssl (用于生成随机密码)
3. **可选组件**:
   - Ingress Controller (Nginx)
   - cert-manager (用于 TLS 证书)
   - Metrics Server (用于 HPA)

## 快速开始

### 1. 自动部署

```bash
# 克隆项目后，进入 deploy 目录
cd deploy

# 执行自动部署脚本
chmod +x deploy.sh
./deploy.sh
```

部署脚本会自动完成以下步骤：
- 检查依赖工具
- 构建 Docker 镜像
- 创建命名空间
- 部署所有资源
- 等待部署完成
- 显示部署状态

### 2. 手动部署

如果需要自定义配置，可以手动部署：

```bash
# 1. 创建命名空间
kubectl create namespace taichu-system

# 2. 修改 secret.yaml 中的密码（使用 base64 编码）
echo -n "your-password" | base64

# 3. 应用所有配置
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f postgres.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml
kubectl apply -f networkpolicy.yaml

# 4. 检查状态
kubectl get pods -n taichu-system
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| DATABASE_PASSWORD | 数据库密码 | 自动生成 |
| JWT_SECRET | JWT 签名密钥 | 自动生成 |
| ENCRYPTION_KEY | 数据加密密钥 | 自动生成 |
| ADMIN_USERNAME | 管理员用户名 | admin |
| ADMIN_PASSWORD | 管理员密码 | admin |

### 资源配置

#### API Server
- **副本数**: 2
- **资源限制**:
  - CPU: 1000m
  - 内存: 1Gi
- **健康检查**: HTTP /api/v1/auth/token

#### PostgreSQL
- **存储**: 20Gi PVC
- **资源限制**:
  - CPU: 1000m
  - 内存: 2Gi
- **数据持久化**: 通过 PVC

### Ingress 配置

默认情况下，Ingress 配置使用以下域名：
- `api.taichu.example.com` - API 服务
- `taichu.example.com` - Web 界面

**注意**: 请根据实际域名修改 `ingress.yaml` 文件。

## 访问服务

### 外部访问

部署完成后，通过以下方式访问服务：

```bash
# 获取 Ingress 地址
kubectl get ingress -n taichu-system

# 或通过 Service NodePort 访问
kubectl get svc taichu-cluster-management-service -n taichu-system
```

### 内部访问

在集群内部可以通过以下方式访问：

```bash
# 直接访问 Service
kubectl exec -it <pod-name> -n taichu-system -- curl http://taichu-cluster-management-service:8081/api/v1/auth/token
```

## 监控

### HPA (水平自动伸缩)

系统配置了 HPA，会根据 CPU 和内存使用率自动伸缩：

```bash
# 查看 HPA 状态
kubectl get hpa -n taichu-system

# 查看详细配置
kubectl describe hpa taichu-cluster-management-hpa -n taichu-system
```

### VPA (垂直自动伸缩)

PostgreSQL 配置了 VPA，会自动调整资源请求：

```bash
# 查看 VPA 状态
kubectl get vpa -n taichu-system
```

### 指标监控

如果安装了 Metrics Server，可以通过以下命令查看资源使用情况：

```bash
# 查看资源使用
kubectl top pods -n taichu-system
kubectl top nodes
```

## 日志

### 查看日志

```bash
# 查看 API Server 日志
kubectl logs -f deployment/taichu-cluster-management -n taichu-system

# 查看 PostgreSQL 日志
kubectl logs -f statefulset/postgres -n taichu-system
```

### 日志聚合

建议配置集中式日志收集，如：
- ELK Stack
- Fluentd
- Loki

## 备份

### PostgreSQL 备份

PostgreSQL 使用 PVC 进行数据持久化。建议定期备份 PVC 数据：

```bash
# 创建快照
kubectl create -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: postgres-snapshot-$(date +%Y%m%d-%H%M%S)
  namespace: taichu-system
spec:
  source:
    persistentVolumeClaimName: postgres-pvc
EOF

# 查看快照
kubectl get volumesnapshots -n taichu-system
```

### 应用配置备份

```bash
# 备份所有配置
kubectl get all,configmap,secret,pvc -n taichu-system -o yaml > taichu-backup-$(date +%Y%m%d).yaml
```

## 故障排除

### 常见问题

#### 1. Pod 无法启动

```bash
# 检查 Pod 状态
kubectl get pods -n taichu-system

# 查看事件
kubectl describe pod <pod-name> -n taichu-system

# 查看日志
kubectl logs <pod-name> -n taichu-system --previous
```

#### 2. 数据库连接失败

```bash
# 检查 PostgreSQL 状态
kubectl get pods -l app=postgres -n taichu-system

# 检查 Service
kubectl get svc postgres-service -n taichu-system

# 测试连接
kubectl exec -it <postgres-pod> -n taichu-system -- psql -U postgres -d taichu
```

#### 3. Ingress 无法访问

```bash
# 检查 Ingress 状态
kubectl get ingress -n taichu-system

# 检查 Ingress Controller
kubectl get pods -n ingress-nginx

# 检查证书
kubectl get certificate -n taichu-system
```

### 性能优化

#### 1. 调整资源限制

```yaml
# 修改 deployment.yaml 中的资源请求和限制
resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 512Mi
```

#### 2. 调整 HPA 参数

```yaml
# 修改 hpa.yaml
metrics:
- type: Resource
  resource:
    name: cpu
    target:
      type: Utilization
      averageUtilization: 50  # 降低阈值，更早扩容
```

#### 3. 优化数据库

```yaml
# 修改 postgres.yaml 中的环境变量
env:
- name: POSTGRES_SHARED_PRELOAD_LIBRARIES
  value: "pg_stat_statements"
```

## 安全

### NetworkPolicy

系统配置了 NetworkPolicy，限制 Pod 间网络通信：
- API Server 只允许来自 Ingress Controller 的访问
- API Server 只能访问 PostgreSQL
- PostgreSQL 只允许来自 API Server 的访问

### RBAC

系统配置了适当的 RBAC 权限：
- ServiceAccount 权限最小化
- 只授予必要的 Kubernetes 资源访问权限

### Secret 管理

- 所有敏感信息存储在 Kubernetes Secret 中
- 密码使用 base64 编码存储
- 建议使用外部 Secret 管理工具（如 HashiCorp Vault）

## 卸载

### 自动卸载

```bash
# 执行卸载脚本
chmod +x uninstall.sh
./uninstall.sh
```

### 手动卸载

```bash
# 删除所有资源
kubectl delete -f .

# 删除命名空间
kubectl delete namespace taichu-system

# 清理镜像
docker rmi taichu/cluster-management:latest
```

## 升级

### 版本升级

```bash
# 1. 构建新版本镜像
docker build -t taichu/cluster-management:v1.1.0 .

# 2. 更新 Deployment 镜像
kubectl set image deployment/taichu-cluster-management api-server=taichu/cluster-management:v1.1.0 -n taichu-system

# 3. 监控升级进度
kubectl rollout status deployment/taichu-cluster-management -n taichu-system

# 4. 回滚（如需要）
kubectl rollout undo deployment/taichu-cluster-management -n taichu-system
```

## 支持

如果遇到问题，请：
1. 查看本文档的故障排除部分
2. 检查 GitHub Issues
3. 联系技术支持团队

## 许可证

版权所有 © 2025 太初系统
