package model

import (
	"time"

	"github.com/google/uuid"
)

// ResourceQuota 资源配额 - 物理强制 (对应K8s ResourceQuota)
type ResourceQuota struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	EnvironmentID uuid.UUID `json:"environment_id" gorm:"type:uuid;not null;uniqueIndex;comment:'关联环境ID(1:1)'"`
	// K8s ResourceQuota 同步字段
	HardLimits    JSONMap   `json:"hard_limits" gorm:"type:jsonb;default:'{}';comment:'硬性限制(requests.cpu, requests.memory等)'"`
	Used          JSONMap   `json:"used" gorm:"type:jsonb;default:'{}';comment:'当前使用量'"`
	Status        string    `json:"status" gorm:"size:20;default:'active';comment:'状态(active/terminating)'"`
	// 同步信息
	LastSyncedAt  *time.Time `json:"last_synced_at" gorm:"comment:'最后同步时间'"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ResourceQuota) TableName() string {
	return "resource_quotas"
}

// TenantQuota 租户配额 - 逻辑约束
type TenantQuota struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex;comment:'关联租户ID(1:1)'"`
	// 配额配置
	HardLimits  JSONMap   `json:"hard_limits" gorm:"type:jsonb;default:'{}';comment:'租户总配额'"`
	Allocated   JSONMap   `json:"allocated" gorm:"type:jsonb;default:'{}';comment:'已分配给环境的配额总和'"`
	Available   JSONMap   `json:"available" gorm:"type:jsonb;default:'{}';comment:'可用配额'"`
	Status      string    `json:"status" gorm:"size:20;default:'active';comment:'状态(active/suspended)'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TenantQuota) TableName() string {
	return "tenant_quotas"
}

// ApplicationResourceSpec 应用资源规格
type ApplicationResourceSpec struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ApplicationID    uuid.UUID `json:"application_id" gorm:"type:uuid;not null;uniqueIndex;comment:'关联应用ID(1:1)'"`
	// 容器默认资源规格
	DefaultRequest   JSONMap   `json:"default_request" gorm:"type:jsonb;default:'{}';comment:'默认requests'"`
	DefaultLimit     JSONMap   `json:"default_limit" gorm:"type:jsonb;default:'{}';comment:'默认limits'"`
	// 应用规模控制
	MaxReplicas      int       `json:"max_replicas" gorm:"default:0;comment:'最大副本数'"`
	CurrentReplicas  int       `json:"current_replicas" gorm:"default:0;comment:'当前副本数'"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ApplicationResourceSpec) TableName() string {
	return "application_resource_specs"
}

// QuotaStatus 配额状态常量
const (
	QuotaStatusActive     = "active"
	QuotaStatusTerminating = "terminating"
	QuotaStatusSuspended  = "suspended"
)

// CommonQuotaKeys 常用配额键
var CommonQuotaKeys = struct {
	RequestsCPU      string
	RequestsMemory   string
	LimitsCPU        string
	LimitsMemory     string
	Pods             string
	Services         string
	Secrets          string
	ConfigMaps       string
	PersistentVolumeClaims string
}{
	RequestsCPU:      "requests.cpu",
	RequestsMemory:   "requests.memory",
	LimitsCPU:        "limits.cpu",
	LimitsMemory:     "limits.memory",
	Pods:             "pods",
	Services:         "services",
	Secrets:          "secrets",
	ConfigMaps:       "configmaps",
	PersistentVolumeClaims: "persistentvolumeclaims",
}
