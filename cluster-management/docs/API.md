# 太初集群管理系统 API 文档

## 概述

太初集群管理系统是一个基于Go语言开发的Kubernetes集群管理平台，提供集群生命周期管理、监控、备份、扩展等功能。

**基础URL**: `http://localhost:8081/api/v1`

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
    "expires_in": 86400,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    }
  }
}
```

---

### 用户注册

**接口地址**: `POST /api/v1/auth/register`

**描述**: 注册新用户

**请求体**:
```json
{
  "username": "string",
  "email": "string",
  "password": "string",
  "role": "string"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### 刷新令牌

**接口地址**: `POST /api/v1/auth/refresh`

**描述**: 刷新JWT令牌

**请求体**:
```json
{
  "token": "string"
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

**描述**: 生成测试用JWT令牌（仅开发环境使用）

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

### 创建集群

**接口地址**: `POST /api/v1/clusters`

**认证**: 需要JWT令牌

**描述**: 通过kubeconfig创建集群

**请求体**:
```json
{
  "name": "string",
  "description": "string",
  "kubeconfig": "string",
  "labels": {
    "key": "value"
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-cluster",
    "description": "My Kubernetes cluster",
    "status": "unknown",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### 通过机器创建集群

**接口地址**: `POST /api/v1/clusters/create`

**认证**: 需要JWT令牌

**描述**: 通过预定义机器列表创建集群

**请求体**:
```json
{
  "cluster_name": "string",
  "description": "string",
  "machine_ids": ["uuid1", "uuid2"],
  "kubernetes": {
    "version": "string",
    "image_repo": "string",
    "container_manager": "string"
  },
  "network": {
    "plugin": "string",
    "pods_cidr": "string",
    "service_cidr": "string"
  },
  "artifact_path": "string",
  "with_packages": true,
  "auto_approve": true,
  "labels": {
    "key": "value"
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_name": "my-cluster",
    "status": "pending",
    "progress": 0,
    "current_step": "Initializing",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### 获取集群列表

**接口地址**: `GET /api/v1/clusters`

**认证**: 需要JWT令牌

**查询参数**:
- `page`: 页码（默认1）
- `limit`: 每页数量（默认20，最大100）
- `status`: 集群状态过滤
- `label_selector`: 标签选择器
- `search`: 搜索关键词

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "my-cluster",
        "description": "My Kubernetes cluster",
        "status": "healthy",
        "provider": "太初",
        "region": "default",
        "version": "v1.28.0",
        "kubernetes_version": "v1.28.0",
        "node_count": 3,
        "labels": {
          "env": "production"
        },
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

---

### 获取集群详情

**接口地址**: `GET /api/v1/clusters/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-cluster",
    "description": "My Kubernetes cluster",
    "provider": "太初",
    "region": "default",
    "status": "healthy",
    "version": "v1.28.0",
    "labels": {
      "env": "production"
    },
    "node_count": 3,
    "total_cpu_cores": 12,
    "total_memory_bytes": 51539607552,
    "total_storage_bytes": 107374182400,
    "used_storage_bytes": 21474836480,
    "storage_usage_percent": 20.0,
    "kubernetes_version": "v1.28.0",
    "api_server_url": "https://172.30.1.12:6443",
    "last_heartbeat_at": "2025-01-01T00:00:00Z",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### 删除集群

**接口地址**: `DELETE /api/v1/clusters/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "Cluster deleted"
  }
}
```

---

### 导入集群

**接口地址**: `POST /api/v1/clusters/import`

**认证**: 需要JWT令牌

**描述**: 导入现有Kubernetes集群

**请求体**:
```json
{
  "import_source": "string",
  "name": "string",
  "description": "string",
  "environment_type": "string",
  "kubeconfig": "string",
  "labels": {
    "key": "value"
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_id": "660e8400-e29b-41d4-a716-446655440001",
    "import_source": "manual",
    "import_status": "pending",
    "imported_resources": "pending",
    "imported_by": "api-user",
    "imported_at": "2025-01-01T00:00:00Z",
    "completed_at": ""
  }
}
```

---

### 获取导入列表

**接口地址**: `GET /api/v1/clusters/imports`

**认证**: 需要JWT令牌

**查询参数**:
- `import_source`: 导入源过滤
- `status`: 状态过滤

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "imports": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "cluster_id": "660e8400-e29b-41d4-a716-446655440001",
        "import_source": "manual",
        "import_status": "completed",
        "imported_resources": "pending",
        "imported_by": "api-user",
        "imported_at": "2025-01-01T00:00:00Z",
        "completed_at": "2025-01-01T00:05:00Z"
      }
    ],
    "total": 1
  }
}
```

---

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
    "import_id": "550e8400-e29b-41d4-a716-446655440000",
    "import_status": "completed",
    "progress": 100,
    "current_step": "Import completed successfully",
    "error_message": "",
    "validation_results": {
      "kubeconfig_valid": true
    }
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
- `type`: 节点类型（control-plane, worker, master）
- `status`: 节点状态
- `page`: 页码（默认1）
- `limit`: 每页数量（默认20）

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
        "status": "ready",
        "cpu_cores": 4,
        "cpu_usage_percent": 45.5,
        "memory_bytes": 17179869184,
        "memory_usage_percent": 38.2,
        "pod_count": 50,
        "labels": {
          "node-role.kubernetes.io/master": ""
        },
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
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

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "name": "node1",
    "type": "control-plane",
    "status": "ready",
    "cpu_cores": 4,
    "cpu_usage_percent": 45.5,
    "memory_bytes": 17179869184,
    "memory_usage_percent": 38.2,
    "pod_count": 50,
    "labels": {
      "node-role.kubernetes.io/master": ""
    },
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "reason": "KubeletReady",
        "message": "kubelet is posting ready status"
      }
    ],
    "addresses": [
      {
        "type": "InternalIP",
        "address": "192.168.1.10"
      }
    ],
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

## 事件接口

### 获取事件列表

**接口地址**: `GET /api/v1/clusters/{id}/events`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `severity`: 严重程度（info, warning, error, critical）
- `since`: 时间过滤（RFC3339格式）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "events": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "event_type": "Node",
        "message": "Node node1 has condition Ready",
        "severity": "info",
        "component": "kubelet",
        "count": 1,
        "first_seen": "2025-01-01T00:00:00Z",
        "last_seen": "2025-01-01T00:00:00Z",
        "created_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "summary": {
      "info_count": 1,
      "warning_count": 0,
      "error_count": 0,
      "critical_count": 0
    }
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
    "pod_security_standard": "baseline",
    "network_policies_enabled": true,
    "network_policies_count": 5,
    "rbac_enabled": true,
    "rbac_roles_count": 10,
    "audit_logging_enabled": true,
    "audit_logging_mode": "request"
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
    "enabled": true,
    "min_nodes": 1,
    "max_nodes": 10,
    "scale_up_threshold": 70,
    "scale_down_threshold": 30,
    "hpa_count": 3,
    "cluster_autoscaler_enabled": true,
    "vpa_count": 2,
    "hpa_policies": [
      {
        "name": "my-app",
        "namespace": "default",
        "min_replicas": 2,
        "max_replicas": 10,
        "current_replicas": 3
      }
    ]
  }
}
```

---

## 备份接口

### 创建备份

**接口地址**: `POST /api/v1/clusters/{id}/backups`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**请求体**:
```json
{
  "backup_name": "string",
  "backup_type": "string",
  "retention_days": 30
}
```

---

### 获取备份列表

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

### 获取备份详情

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
    "restore_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "in_progress",
    "progress": 65.5,
    "current_step": "Restoring etcd",
    "start_time": "2025-01-01T00:00:00Z",
    "estimated_time": 1800
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
    "environments": [
      {
        "type": "production",
        "count": 3,
        "clusters": [
          {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "prod-cluster-1",
            "status": "healthy",
            "node_count": 5,
            "version": "v1.28.0"
          }
        ],
        "total_node_count": 15,
        "healthy_clusters": 3,
        "unhealthy_clusters": 0
      }
    ],
    "summary": {
      "total_clusters": 3,
      "total_environments": 2,
      "total_nodes": 18,
      "healthy_clusters": 3
    }
  }
}
```

---

## 审计接口

### 获取审计事件列表

**接口地址**: `GET /api/v1/clusters/{id}/audit`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 集群ID

**查询参数**:
- `page`: 页码
- `limit`: 每页数量
- `event_type`: 事件类型
- `action`: 操作类型
- `user`: 用户名
- `result`: 结果
- `start_time`: 开始时间
- `end_time`: 结束时间

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "events": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "event_type": "cluster",
        "action": "create",
        "resource": "cluster",
        "resource_id": "550e8400-e29b-41d4-a716-446655440001",
        "user": "admin",
        "ip_address": "192.168.1.100",
        "old_value": {},
        "new_value": {
          "name": "my-cluster",
          "status": "active"
        },
        "details": {},
        "result": "success",
        "timestamp": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

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
  "new_node_count": 5,
  "new_cpu_cores": 20,
  "new_memory_gb": 64,
  "new_storage_gb": 500,
  "reason": "业务增长需要"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "cluster_id": "660e8400-e29b-41d4-a716-446655440001",
    "old_node_count": 3,
    "new_node_count": 5,
    "old_cpu_cores": 12,
    "new_cpu_cores": 20,
    "old_memory_gb": 48,
    "new_memory_gb": 64,
    "old_storage_gb": 300,
    "new_storage_gb": 500,
    "status": "pending",
    "reason": "业务增长需要",
    "requested_by": "api-user",
    "created_at": "2025-01-01T00:00:00Z"
  }
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
        "cluster_id": "660e8400-e29b-41d4-a716-446655440001",
        "old_node_count": 3,
        "new_node_count": 5,
        "old_cpu_cores": 12,
        "new_cpu_cores": 20,
        "old_memory_gb": 48,
        "new_memory_gb": 64,
        "old_storage_gb": 300,
        "new_storage_gb": 500,
        "status": "completed",
        "reason": "业务增长需要",
        "requested_by": "api-user",
        "created_at": "2025-01-01T00:00:00Z"
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
  "ip_address": "string",
  "internal_address": "string",
  "user": "string",
  "password": "string",
  "role": "master|worker|etcd|registry",
  "artifact_path": "string",
  "image_repo": "string",
  "registry_address": "string",
  "labels": {
    "key": "value"
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "machine-1",
    "ip_address": "192.168.1.10",
    "internal_address": "10.0.0.10",
    "user": "root",
    "role": "master",
    "status": "available",
    "labels": {
      "rack": "rack1"
    },
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### 获取机器列表

**接口地址**: `GET /api/v1/machines`

**认证**: 需要JWT令牌

**查询参数**:
- `page`: 页码
- `limit`: 每页数量
- `status`: 状态过滤
- `role`: 角色过滤

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "machines": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "machine-1",
        "ip_address": "192.168.1.10",
        "internal_address": "10.0.0.10",
        "user": "root",
        "role": "master",
        "status": "available",
        "labels": {
          "rack": "rack1"
        },
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10
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

**请求体**: 同创建机器接口

---

### 删除机器

**接口地址**: `DELETE /api/v1/machines/{id}`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 机器ID

---

###**接口地址**: `PUT /api/v1/machines 更新机器状态
/{id}/status`

**认证**: 需要JWT令牌

**路径参数**:
- `id`: 机器ID

**请求体**:
```json
{
  "status": "available|unavailable|maintenance|error",
  "message": "状态变更说明"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "maintenance",
    "message": "计划维护中",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

## 用户管理接口

### 获取用户列表

**接口地址**: `GET /api/v1/users`

**认证**: 需要JWT令牌（需要管理员权限）

**查询参数**:
- `page`: 页码（默认1）
- `limit`: 每页数量（默认20，最大100）
- `role`: 角色过滤（admin, user）
- `search`: 搜索关键词（搜索用户名和邮箱）

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "users": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "admin",
        "email": "admin@example.com",
        "role": "admin",
        "created_at": "2025-01-01T00:00:00Z",
        "last_login_at": "2025-01-01T12:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

---

### 获取用户详情

**接口地址**: `GET /api/v1/users/{id}`

**认证**: 需要JWT令牌（需要管理员权限或用户本人）

**路径参数**:
- `id`: 用户ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z",
    "last_login_at": "2025-01-01T12:00:00Z"
  }
}
```

---

### 更新用户信息

**接口地址**: `PUT /api/v1/users/{id}`

**认证**: 需要JWT令牌（需要管理员权限或用户本人）

**路径参数**:
- `id`: 用户ID

**请求体**:
```json
{
  "email": "newemail@example.com",
  "role": "user|admin"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "email": "newemail@example.com",
    "role": "user",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### 删除用户

**接口地址**: `DELETE /api/v1/users/{id}`

**认证**: 需要JWT令牌（需要管理员权限）

**路径参数**:
- `id`: 用户ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "User deleted successfully"
  }
}
```

---

### 修改密码

**接口地址**: `PUT /api/v1/users/{id}/password`

**认证**: 需要JWT令牌（需要管理员权限或用户本人）

**路径参数**:
- `id`: 用户ID

**请求体**:
```json
{
  "old_password": "旧密码",
  "new_password": "新密码"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "Password updated successfully"
  }
}
```

---

## 错误码说明

| 错误码 | 说明 | 示例场景 |
|--------|------|----------|
| 0 | 成功 | 请求处理成功 |
| 400 | 请求参数错误 | 缺少必需参数、参数格式错误 |
| 401 | 未授权 | 未提供token、token无效或已过期 |
| 403 | 禁止访问 | 权限不足，无法访问资源 |
| 404 | 资源不存在 | 请求的集群、节点等不存在 |
| 409 | 资源冲突 | 创建同名集群、用户名已存在等 |
| 422 | 请求无法处理 | 参数验证失败 |
| 500 | 服务器内部错误 | 数据库连接失败、内部服务异常 |

**错误响应格式**:
```json
{
  "code": 400,
  "message": "Invalid request parameter: name is required",
  "data": null
}
```

---

## 状态说明

### 集群状态

| 状态 | 说明 |
|------|------|
| unknown | 未知状态，可能是新创建的集群 |
| healthy | 集群健康，所有组件正常运行 |
| unhealthy | 集群不健康，部分组件异常 |
| pending | 集群创建中，等待初始化完成 |
| error | 集群错误，无法正常工作 |
| maintenance | 集群维护中，暂时不可用 |

### 节点状态

| 状态 | 说明 |
|------|------|
| ready | 节点就绪，可以接受Pod调度 |
| not_ready | 节点未就绪，可能存在异常 |
| unknown | 节点状态未知，kubelet可能未响应 |

### 备份状态

| 状态 | 说明 |
|------|------|
| pending | 备份任务已创建，等待执行 |
| in_progress | 备份正在进行中 |
| completed | 备份成功完成 |
| failed | 备份失败 |
| deleted | 备份已删除 |

### 扩展状态

| 状态 | 说明 |
|------|------|
| pending | 扩展请求已提交，等待处理 |
| in_progress | 扩展正在进行中 |
| completed | 扩展成功完成 |
| failed | 扩展失败 |
| cancelled | 扩展已取消 |

### 机器状态

| 状态 | 说明 |
|------|------|
| available | 机器可用，可用于创建集群 |
| unavailable | 机器不可用，可能存在故障 |
| maintenance | 机器维护中，暂时不可用 |
| error | 机器错误，需要人工干预 |
| in_use | 机器已被集群使用 |

### 导入状态

| 状态 | 说明 |
|------|------|
| pending | 导入任务已创建，等待执行 |
| in_progress | 导入正在进行中 |
| completed | 导入成功完成 |
| failed | 导入失败 |
| cancelled | 导入已取消 |
  "status": "available|in-use|deploying|maintenance|offline"
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
        "cluster_name": "my-cluster",
        "status": "completed",
        "progress": 100,
        "current_step": "Cluster created successfully",
        "kubernetes_version": "v1.28.0",
        "network_plugin": "calico",
        "started_at": "2025-01-01T00:00:00Z",
        "completed_at": "2025-01-01T00:30:00Z",
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10
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
    "cluster_name": "my-cluster",
    "status": "completed",
    "progress": 100,
    "current_step": "Cluster created successfully",
    "logs": "...",
    "error_msg": "",
    "kubernetes_version": "v1.28.0",
    "network_plugin": "calico",
    "started_at": "2025-01-01T00:00:00Z",
    "completed_at": "2025-01-01T00:30:00Z",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:30:00Z"
  }
}
```

---

## 错误响应格式

所有接口的错误响应都遵循统一格式：

```json
{
  "code": 1,
  "message": "错误描述",
  "data": null
}
```

**常见错误码**:
- `0`: 成功
- `1`: 通用错误
- `400`: 请求参数错误
- `401`: 未认证或令牌无效
- `403`: 权限不足
- `404`: 资源不存在
- `409`: 资源冲突（如集群名称已存在）
- `500`: 服务器内部错误

---

## 数据类型说明

### 时间格式
所有时间字段使用RFC3339格式，例如：`2025-01-01T00:00:00Z`

### UUID格式
所有ID字段使用UUID v4格式，例如：`550e8400-e29b-41d4-a716-446655440000`

### 标签格式
标签为键值对，JSON格式：
```json
{
  "key1": "value1",
  "key2": "value2"
}
```

### 状态值
- **集群状态**: `unknown`, `healthy`, `unhealthy`, `disconnected`
- **节点状态**: `ready`, `notready`, `unknown`
- **机器状态**: `available`, `in-use`, `deploying`, `maintenance`, `offline`
- **事件严重程度**: `info`, `warning`, `error`, `critical`
- **任务状态**: `pending`, `running`, `completed`, `failed`

---

## 注意事项

1. **认证**: 除登录、注册、刷新令牌、获取用户信息、登出和生成测试令牌接口外，所有接口都需要JWT认证。

2. **分页**: 所有列表接口都支持分页，使用`page`和`limit`参数。

3. **过滤**: 大部分列表接口支持通过查询参数进行过滤。

4. **异步操作**: 集群创建、导入、备份、恢复、扩展等操作是异步的，返回的是任务ID，需要通过相应接口查询进度。

5. **权限**: 不同角色的用户拥有不同的API访问权限。

---

## 联系我们

如有疑问，请联系系统管理员。
