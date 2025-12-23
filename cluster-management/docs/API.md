# 太初集群管理系统 API 文档

## 概述

太初集群管理系统是一个基于Go语言开发的Kubernetes集群管理平台，提供集群生命周期管理、监控、备份、扩展等功能。

**基础URL**: `http://localhost:8086/api/v1`

**认证方式**: JWT Bearer Token
- 在需要认证的接口请求头中添加: `Authorization: Bearer <token>`
- 默认管理员账号: `admin/admin` (首次启动时自动创建)
- 普通用户需要通过注册接口创建账号

## 目录

- [认证接口](#认证接口)
- [集群接口](#集群接口)
- [节点接口](#节点接口)
- [事件接口](#事件接口)
- [安全策略接口](#安全策略接口)
- [自动伸缩策略接口](#自动伸缩策略接口)
- [备份接口](#备份接口)
- [拓扑接口](#拓扑接口)
- [导入接口](#导入接口)
- [审计接口](#审计接口)
- [扩展接口](#扩展接口)
- [机器管理接口](#机器管理接口)
- [创建任务接口](#创建任务接口)
- [用户管理接口](#用户管理接口)
- [错误码说明](#错误码说明)
- [状态说明](#状态说明)

---

## 认证接口

### 用户登录

**接口地址**: `POST /api/v1/auth/login`

**描述**: 用户登录获取JWT令牌

**请求体**:
```json
{
  "username": "string",
  "password": "string"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "admin",
      "role": "admin"
    }
  }
}
```

---

### 用户注册

**接口地址**: `POST /api/v1/auth/register`

**描述**: 用户注册

**请求体**:
```json
{
  "username": "string",
  "password": "string",
  "email": "string"
}
```

---

### 刷新令牌

**接口地址**: `POST /api/v1/auth/refresh`

**描述**: 刷新JWT令牌

**请求体**:
```json
{
  "refresh_token": "string"
}
```

---

### 获取用户信息

**接口地址**: `GET /api/v1/auth/profile`

**认证**: 需要JWT令牌

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

---

### 用户登出

**接口地址**: `POST /api/v1/auth/logout`

**认证**: 需要JWT令牌

---

### 生成测试令牌

**接口地址**: `GET /api/v1/auth/token`

**描述**: 生成测试用令牌（无需认证，仅用于开发测试）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

---

## 集群接口

### 获取集群拓扑

**接口地址**: `GET /api/v1/clusters/topology`

**认证**: 需要JWT令牌

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "cluster1",
        "status": "healthy",
        "node_count": 3,
        "version": "v1.24.9"
      }
    ]
  }
}
```

---

### 创建集群

**接口地址**: `POST /api/v1/clusters`

**认证**: 需要JWT令牌

**请求体**:
```json
{
  "name": "string",
  "description": "string",
  "provider": "string",
  "region": "string",
  "version": "string",
  "node_count": 3
}
```

---

### 通过机器创建集群

**接口地址**: `POST /api/v1/clusters/create`

**认证**: 需要JWT令牌

**请求体**:
```json
{
  "cluster_name": "string",
  "machine_ids": ["id1", "id2", "id3"],
  "kubernetes_version": "v1.24.9"
}
```

---

### 获取集群列表

**接口地址**: `GET /api/v1/clusters`

**认证**: 需要JWT令牌

**查询参数**:
- `status`: 集群状态
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "cluster1",
        "status": "healthy",
        "provider": "太初",
        "region": "cn-east-1",
        "version": "v1.24.9",
        "node_count": 3,
        "created_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 获取集群详情

**接口地址**: `GET /api/v1/clusters/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

---

### 删除集群

**接口地址**: `DELETE /api/v1/clusters/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

---

### 导入集群

**接口地址**: `POST /api/v1/clusters/import`

**认证**: 需要JWT令牌

**请求体**:
```json
{
  "name": "string",
  "description": "string",
  "kubeconfig": "string (base64编码)",
  "provider": "string"
}
```

---

### 获取导入列表

**接口地址**: `GET /api/v1/clusters/imports`

**认证**: 需要JWT令牌

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "imports": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "cluster_name": "cluster1",
        "status": "completed",
        "created_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

## 节点接口

### 获取节点列表

**接口地址**: `GET /api/v1/clusters/{id}/nodes`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `type`: 节点类型 (control-plane/worker)
- `status`: 节点状态
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "nodes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "node1",
        "type": "control-plane",
        "status": "Ready",
        "cpu_cores": 20,
        "cpu_usage_percent": 45.5,
        "memory_bytes": 67155046400,
        "memory_usage_percent": 60.2,
        "pod_count": 35,
        "labels": {
          "kubernetes.io/hostname": "node1"
        }
      }
    ],
    "summary": {
      "control_plane_count": 1,
      "worker_count": 2,
      "ready_count": 3
    }
  }
}
```

---

### 获取节点详情

**接口地址**: `GET /api/v1/clusters/{id}/nodes/{nodeName}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `nodeName`: 节点名称

---

## 事件接口

### 获取事件列表

**接口地址**: `GET /api/v1/clusters/{id}/events`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `type`: 事件类型
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "events": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "type": "Normal",
        "reasonSet",
        "": "ScalingReplicamessage": "Scaled up replica set",
        "count": 1,
        "first_timestamp": "2025-01-01T00:00:00Z",
        "last_timestamp": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

## 安全策略接口

### 获取安全策略

**接口地址**: `GET /api/v1/clusters/{id}/security-policies`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "policies": [
      {
        "name": "default-deny-all",
        "type": "NetworkPolicy",
        "namespace": "default",
        "rules": [
          {
            "direction": "ingress",
            "from": [],
            "to": [],
            "ports": []
          }
        ]
      }
    ]
  }
}
```

---

## 自动伸缩策略接口

### 获取自动伸缩策略

**接口地址**: `GET /api/v1/clusters/{id}/autoscaling-policies`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "policies": [
      {
        "name": "nginx-hpa",
        "type": "HorizontalPodAutoscaler",
        "namespace": "default",
        "target": {
          "kind": "Deployment",
          "name": "nginx"
        },
        "min_replicas": 2,
        "max_replicas": 10,
        "metrics": [
          {
            "type": "Resource",
            "resource": {
              "name": "cpu",
              "target": {
                "type": "Utilization",
                "average_utilization": 70
              }
            }
          }
        ]
      }
    ]
  }
}
```

---

## 备份接口

### 备份概述

太初集群管理系统提供三种备份类型：
- **etcd备份**：备份etcd集群的数据快照，用于恢复集群状态
- **资源备份**：备份Kubernetes资源（Deployment、Service、ConfigMap等）
- **完整备份**：同时备份etcd和资源

**备份工作原理**：
- 备份执行是异步的，创建备份后立即返回备份ID
- etcd备份通过SSH连接到etcd节点，执行`etcdctl snapshot save`命令
- 系统会自动从备份计划中获取etcd配置（endpoints、证书、SSH凭证等）
- 备份文件存储在服务器的`/backups`目录下

**备份状态**：
- `pending`：备份已创建，等待执行
- `running`：备份正在执行中
- `completed`：备份成功完成
- `failed`：备份失败，可查看`error_message`字段了解错误详情

### 创建etcd备份（独立接口）

**接口地址**: `POST /api/v1/clusters/{id}/etcd/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**请求体**:
```json
{
  "backup_name": "string",
  "backup_type": "etcd",
  "retention_days": 30
}
```

**请求参数说明**:
- `backup_name`: 备份名称（必填）
- `backup_type`: 备份类型，固定为"etcd"（必填）
- `retention_days`: 保留天数（可选，默认30天）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "etcd-backup-20250101",
    "backup_name": "etcd-backup-20250101",
    "backup_type": "etcd",
    "status": "pending",
    "storage_location": "\\backups\\550e8400-e29b-41d4-a716-446655440001\\etcd-backup-20250101\\20250101-120000",
    "retention_days": 30,
    "created_by": "system",
    "snapshot_timestamp": "2025-01-01T00:00:00Z",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

**说明**:
- 备份执行是异步的，创建后状态为"pending"
- 系统会自动从备份计划中获取etcd配置（endpoints、证书、SSH凭证等）
- 备份会在集群的etcd节点上通过SSH执行etcdctl snapshot save命令
- 备份完成后状态会更新为"completed"，失败则更新为"failed"

---

### 创建资源备份（独立接口）

**接口地址**: `POST /api/v1/clusters/{id}/resources/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**请求体**:
```json
{
  "backup_name": "string",
  "backup_type": "resources",
  "retention_days": 30
}
```

**请求参数说明**:
- `backup_name`: 备份名称（必填）
- `backup_type`: 备份类型，固定为"resources"（必填）
- `retention_days`: 保留天数（可选，默认30天）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440002",
    "backup_name": "resources-backup-20250101",
    "backup_type": "resources",
    "status": "pending",
    "storage_location": "\\backups\\550e8400-e29b-41d4-a716-446655440002\\resources-backup-20250101\\20250101-120000",
    "retention_days": 30,
    "created_by": "system",
    "snapshot_timestamp": "2025-01-01T00:00:00Z",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

**说明**:
- 备份执行是异步的，创建后状态为"pending"
- 资源备份会备份集群中的所有Kubernetes资源（如Deployment、Service、ConfigMap等）
- 备份完成后状态会更新为"completed"，失败则更新为"failed"

---

### 创建完整备份（包含etcd和资源）

**接口地址**: `POST /api/v1/clusters/{id}/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**请求体**:
```json
{
  "backup_name": "string",
  "backup_type": "full",
  "retention_days": 30
}
```

---

### 获取etcd备份列表

**接口地址**: `GET /api/v1/clusters/{id}/etcd/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `status`: 备份状态
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "backups": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "backup_name": "etcd-backup-20250101",
        "backup_type": "etcd",
        "status": "completed",
        "storage_location": "\\backups\\550e8400-e29b-41d4-a716-446655440001\\etcd-backup-20250101\\20250101-120000",
        "storage_size_bytes": 894709792,
        "snapshot_timestamp": "2025-01-01T00:00:00Z",
        "created_at": "2025-01-01T00:00:00Z",
        "completed_at": "2025-01-01T00:05:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 获取资源备份列表

**接口地址**: `GET /api/v1/clusters/{id}/resources/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `status`: 备份状态
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "backups": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "backup_name": "resources-backup-20250101",
        "backup_type": "resources",
        "status": "completed",
        "storage_location": "/backups/cluster1/resources-backup-20250101",
        "storage_size_bytes": 10705659,
        "snapshot_timestamp": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 获取完整备份列表

**接口地址**: `GET /api/v1/clusters/{id}/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `status`: 备份状态
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "backups": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "backup_name": "daily-backup-20250101",
        "backup_type": "full",
        "status": "completed",
        "storage_location": "/backups/cluster1",
        "storage_size_bytes": 10737418240,
        "snapshot_timestamp": "2025-01-01T00:00:00Z",
        "created_at": "2025-01-01T00:00:00Z",
        "completed_at": "2025-01-01T01:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 获取etcd备份详情

**接口地址**: `GET /api/v1/clusters/{id}/etcd/backups/{backupId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `backupId`: 备份ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
    "backup_name": "etcd-backup-20250101",
    "backup_type": "etcd",
    "status": "completed",
    "storage_location": "\\backups\\550e8400-e29b-41d4-a716-446655440001\\etcd-backup-20250101\\20250101-120000",
    "storage_size_bytes": 894709792,
    "snapshot_timestamp": "2025-01-01T00:00:00Z",
    "retention_days": 30,
    "created_at": "2025-01-01T00:00:00Z",
    "completed_at": "2025-01-01T00:05:00Z",
    "error_message": ""
  }
}
```

**备份状态说明**:
- `pending`: 备份已创建，等待执行
- `running`: 备份正在执行中
- `completed`: 备份成功完成
- `failed`: 备份失败，可查看`error_message`字段了解错误详情

---

### 获取资源备份详情

**接口地址**: `GET /api/v1/clusters/{id}/resources/backups/{backupId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `backupId`: 备份ID

---

### 获取完整备份详情

**接口地址**: `GET /api/v1/clusters/{id}/backups/{backupId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `backupId`: 备份ID

---

### 恢复备份

**接口地址**: `POST /api/v1/clusters/{id}/backups/{backupId}/restore`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `backupId`: 备份ID

**请求体**:
```json
{
  "restore_name": "string"
}
```

---

### 获取恢复进度

**接口地址**: `GET /api/v1/clusters/{id}/backups/{backupId}/restore/{restoreId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `backupId`: 备份ID
- `restoreId`: 恢复任务ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "progress": 100,
    "started_at": "2025-01-01T00:00:00Z",
    "completed_at": "2025-01-01T01:00:00Z"
  }
}
```

---

### 删除备份

**接口地址**: `DELETE /api/v1/clusters/{id}/backups/{backupId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `backupId`: 备份ID

---

### 获取备份计划列表

**接口地址**: `GET /api/v1/clusters/{id}/backup-schedules`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "daily-backup",
      "schedule_name": "daily-backup",
      "cron_expr": "0 2 * * *",
      "cron_expression": "0 2 * * *",
      "backup_type": "etcd",
      "retention_days": 7,
      "retention_count": 7,
      "enabled": true,
      "created_by": "admin",
      "etcd_endpoints": "https://172.30.1.12:2379,https://172.30.1.14:2379,https://172.30.1.15:2379",
      "etcd_ca_cert": "/etc/ssl/etcd/ssl/ca.pem",
      "etcd_cert": "/etc/ssl/etcd/ssl/admin-node2.pem",
      "etcd_key": "/etc/ssl/etcd/ssl/admin-node2-key.pem",
      "etcd_data_dir": "/var/lib/etcd",
      "etcdctl_path": "/usr/local/bin/etcdctl",
      "ssh_username": "root",
      "ssh_password": "password",
      "etcd_deployment_type": "standalone",
      "k8s_deployment_type": "kubeadm",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### 创建备份计划

**接口地址**: `POST /api/v1/clusters/{id}/backup-schedules`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**请求体**:
```json
{
  "name": "daily-backup",
  "cron_expr": "0 2 * * *",
  "backup_type": "etcd",
  "retention_days": 7,
  "enabled": true,
  "created_by": "admin",
  "etcd_endpoints": "https://172.30.1.12:2379,https://172.30.1.14:2379,https://172.30.1.15:2379",
  "etcd_ca_cert": "/etc/ssl/etcd/ssl/ca.pem",
  "etcd_cert": "/etc/ssl/etcd/ssl/admin-node2.pem",
  "etcd_key": "/etc/ssl/etcd/ssl/admin-node2-key.pem",
  "etcd_data_dir": "/var/lib/etcd",
  "etcdctl_path": "/usr/local/bin/etcdctl",
  "ssh_username": "root",
  "ssh_password": "password",
  "etcd_deployment_type": "standalone",
  "k8s_deployment_type": "kubeadm"
}
```

**请求参数说明**:
- `name`: 备份计划名称（必填）
- `cron_expr`: Cron表达式，定义备份执行时间（必填）
- `backup_type`: 备份类型，可选值："etcd"、"resources"、"full"（必填）
- `retention_days`: 保留天数（可选，默认7天）
- `enabled`: 是否启用（可选，默认true）
- `created_by`: 创建者用户名（必填）
- `etcd_endpoints`: etcd端点地址，多个端点用逗号分隔（可选，etcd备份必填）
- `etcd_ca_cert`: etcd CA证书路径（可选，etcd备份必填）
- `etcd_cert`: etcd客户端证书路径（可选，etcd备份必填）
- `etcd_key`: etcd客户端密钥路径（可选，etcd备份必填）
- `etcd_data_dir`: etcd数据目录（可选）
- `etcdctl_path`: etcdctl命令路径（可选，默认"/usr/local/bin/etcdctl"）
- `ssh_username`: SSH用户名（可选，etcd备份必填）
- `ssh_password`: SSH密码（可选，etcd备份必填）
- `etcd_deployment_type`: etcd部署类型（可选）
- `k8s_deployment_type`: Kubernetes部署类型（可选）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "daily-backup",
    "schedule_name": "daily-backup",
    "cron_expr": "0 2 * * *",
    "cron_expression": "0 2 * * *",
    "backup_type": "etcd",
    "retention_days": 7,
    "retention_count": 7,
    "enabled": true,
    "created_by": "admin",
    "etcd_endpoints": "https://172.30.1.12:2379,https://172.30.1.14:2379,https://172.30.1.15:2379",
    "etcd_ca_cert": "/etc/ssl/etcd/ssl/ca.pem",
    "etcd_cert": "/etc/ssl/etcd/ssl/admin-node2.pem",
    "etcd_key": "/etc/ssl/etcd/ssl/admin-node2-key.pem",
    "etcd_data_dir": "/var/lib/etcd",
    "etcdctl_path": "/usr/local/bin/etcdctl",
    "ssh_username": "root",
    "ssh_password": "password",
    "etcd_deployment_type": "standalone",
    "k8s_deployment_type": "kubeadm",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

**说明**:
- 备份计划创建后会根据cron表达式自动执行备份
- etcd备份需要提供etcd相关配置（endpoints、证书、SSH凭证等）
- 系统会自动从endpoints中解析etcd节点IP，选择其中一个节点执行备份

---

### 更新备份计划

**接口地址**: `PUT /api/v1/clusters/{id}/backup-schedules/{scheduleId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `scheduleId`: 计划ID

**请求体**:
```json
{
  "cron_expr": "0 3 * * *",
  "backup_type": "etcd",
  "retention_days": 14,
  "enabled": true,
  "etcd_endpoints": "https://172.30.1.12:2379,https://172.30.1.14:2379,https://172.30.1.15:2379",
  "etcd_ca_cert": "/etc/ssl/etcd/ssl/ca.pem",
  "etcd_cert": "/etc/ssl/etcd/ssl/admin-node2.pem",
  "etcd_key": "/etc/ssl/etcd/ssl/admin-node2-key.pem",
  "etcd_data_dir": "/var/lib/etcd",
  "etcdctl_path": "/usr/local/bin/etcdctl",
  "ssh_username": "root",
  "ssh_password": "password",
  "etcd_deployment_type": "standalone",
  "k8s_deployment_type": "kubeadm"
}
```

**请求参数说明**:
- `cron_expr`: Cron表达式，定义备份执行时间（可选）
- `backup_type`: 备份类型，可选值："etcd"、"resources"、"full"（可选）
- `retention_days`: 保留天数（可选）
- `enabled`: 是否启用（可选）
- `etcd_endpoints`: etcd端点地址，多个端点用逗号分隔（可选）
- `etcd_ca_cert`: etcd CA证书路径（可选）
- `etcd_cert`: etcd客户端证书路径（可选）
- `etcd_key`: etcd客户端密钥路径（可选）
- `etcd_data_dir`: etcd数据目录（可选）
- `etcdctl_path`: etcdctl命令路径（可选）
- `ssh_username`: SSH用户名（可选）
- `ssh_password`: SSH密码（可选）
- `etcd_deployment_type`: etcd部署类型（可选）
- `k8s_deployment_type`: Kubernetes部署类型（可选）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "Backup schedule updated successfully"
  }
}
```

---

### 删除备份计划

**接口地址**: `DELETE /api/v1/clusters/{id}/backup-schedules/{scheduleId}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID
- `scheduleId`: 计划ID

---

## 拓扑接口

### 获取集群拓扑

**接口地址**: `GET /api/v1/clusters/topology`

**认证**: 需要JWT令牌

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "cluster1",
        "status": "healthy",
        "node_count": 3,
        "version": "v1.24.9",
        "provider": "太初"
      }
    ]
  }
}
```

---

## 导入接口

### 获取导入状态

**接口地址**: `GET /api/v1/imports/{importId}/status`

**认证**: 需要JWT令牌

**路径参数**:
- `importId`: 导入记录ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "cluster_id": "550e8400-e29b-41d4-a716-446655440001",
    "progress": 100,
    "logs": "Import completed successfully"
  }
}
```

---

## 审计接口

### 获取全局审计日志

**接口地址**: `GET /api/v1/audit`

**认证**: 需要JWT令牌

**查询参数**:
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "audit_logs": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "550e8400-e29b-41d4-a716-446655440001",
        "username": "admin",
        "action": "CREATE_CLUSTER",
        "resource_type": "cluster",
        "resource_id": "550e8400-e29b-41d4-a716-446655440002",
        "details": "Created cluster cluster1",
        "timestamp": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 创建审计日志

**接口地址**: `POST /api/v1/audit`

**认证**: 需要JWT令牌

**请求体**:
```json
{
  "action": "string",
  "resource_type": "string",
  "resource_id": "string",
  "details": "string"
}
```

---

### 获取集群审计日志

**接口地址**: `GET /api/v1/clusters/{id}/audit`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `page`: 页码
- `limit`: 每页数量

---

## 扩展接口

### 请求集群扩展

**接口地址**: `POST /api/v1/clusters/{id}/expansion`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**请求体**:
```json
{
  "node_count": 5,
  "instance_type": "string"
}
```

---

### 获取扩展历史

**接口地址**: `GET /api/v1/clusters/{id}/expansion/history`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "expansions": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "from_nodes": 3,
        "to_nodes": 5,
        "status": "completed",
        "created_at": "2025-01-01T00:00:00Z",
        "completed_at": "2025-01-01T01:00:00Z"
      }
    ]
  }
}
```

---

## 机器管理接口

### 创建机器

**接口地址**: `POST /api/v1/machines`

**认证**: 需要JWT令牌

**请求体**:
```json
{
  "name": "string",
  "ip": "string",
  "username": "string",
  "password": "string",
  "role": "control-plane"
}
```

---

### 获取机器列表

**接口地址**: `GET /api/v1/machines`

**认证**: 需要JWT令牌

**查询参数**:
- `status`: 机器状态
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "machines": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "machine1",
        "ip": "192.168.1.100",
        "status": "ready",
        "role": "control-plane"
      }
    ],
    "total": 1
  }
}
```

---

### 获取机器详情

**接口地址**: `GET /api/v1/machines/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 机器ID

---

### 更新机器

**接口地址**: `PUT /api/v1/machines/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 机器ID

**请求体**:
```json
{
  "name": "string",
  "description": "string"
}
```

---

### 删除机器

**接口地址**: `DELETE /api/v1/machines/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 机器ID

---

### 更新机器状态

**接口地址**: `PUT /api/v1/machines/{id}/status`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 机器ID

**请求体**:
```json
{
  "status": "ready"
}
```

---

## 创建任务接口

### 获取创建任务列表

**接口地址**: `GET /api/v1/create-tasks`

**认证**: 需要JWT令牌

**查询参数**:
- `page`: 页码
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tasks": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "cluster_name": "cluster1",
        "status": "completed",
        "progress": 100,
        "created_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 获取创建任务详情

**接口地址**: `GET /api/v1/create-tasks/{taskId}`

**认证**: 需要JWT令牌

**路径参数**:
- `taskId`: 任务ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_name": "cluster1",
    "status": "completed",
    "progress": 100,
    "logs": "Cluster created successfully",
    "created_at": "2025-01-01T00:00:00Z",
    "completed_at": "2025-01-01T01:00:00Z"
  }
}
```

---

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| -1 | 通用错误 |
| 400 | 请求参数错误 |
| 401 | 未认证或认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 状态说明

### 集群状态
- `pending`: 待创建
- `creating`: 创建中
- `running`: 运行中
- `healthy`: 健康
- `unhealthy`: 不健康
- `deleting`: 删除中
- `deleted`: 已删除

### 节点状态
- `Pending`: 待调度
- `Running`: 运行中
- `Ready`: 就绪
- `NotReady`: 未就绪
- `Unknown`: 未知状态

### 备份状态
- `pending`: 待执行
- `running`: 执行中
- `completed`: 已完成
- `failed`: 已失败

### 恢复状态
- `pending`: 待执行
- `running`: 执行中
- `completed`: 已完成
- `failed`: 已失败

---

**注意**: 本文档基于当前API版本v1生成，接口可能会随版本更新而变化。
