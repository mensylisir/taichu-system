package model

import (
	"time"

	"github.com/google/uuid"
)

// ResourceClassification 资源分类记录
type ResourceClassification struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID   uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index;comment:'关联集群ID'"`
	TenantID    *uuid.UUID `json:"tenant_id" gorm:"type:uuid;index;comment:'关联租户ID'"`
	EnvironmentID *uuid.UUID `json:"environment_id" gorm:"type:uuid;index;comment:'关联环境ID'"`
	ApplicationID *uuid.UUID `json:"application_id" gorm:"type:uuid;index;comment:'关联应用ID'"`
	ResourceType string    `json:"resource_type" gorm:"size:50;not null;comment:'资源类型(namespace/deployment/service等)'"`
	ResourceName string    `json:"resource_name" gorm:"size:255;not null;comment:'资源名称'"`
	Namespace    string    `json:"namespace" gorm:"size:255;comment:'命名空间'"`
	// 分类信息
	AssignedTenant   string    `json:"assigned_tenant" gorm:"size:100;comment:'分配的租户名'"`
	AssignedEnv      string    `json:"assigned_env" gorm:"size:100;comment:'分配的环境名'"`
	AssignedApp      string    `json:"assigned_app" gorm:"size:100;comment:'分配的应用名'"`
	ClassificationRule string  `json:"classification_rule" gorm:"size:255;comment:'分类规则'"`
	// 状态信息
	Status       string    `json:"status" gorm:"size:20;default:'pending';comment:'状态(pending/classified/failed)'"`
	ErrorMessage string    `json:"error_message" gorm:"type:text;comment:'错误信息'"`
	// 时间信息
	ClassifiedAt *time.Time `json:"classified_at" gorm:"comment:'分类时间'"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ResourceClassification) TableName() string {
	return "resource_classifications"
}

// ClassificationStatus 分类状态常量
const (
	ClassificationStatusPending   = "pending"
	ClassificationStatusClassified = "classified"
	ClassificationStatusFailed    = "failed"
)

// ResourceClassificationHistory 资源分类历史
type ResourceClassificationHistory struct {
	ID                  uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID           uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index;comment:'关联集群ID'"`
	TriggerType         string    `json:"trigger_type" gorm:"size:50;not null;comment:'触发类型(import/manual/scheduled)'"`
	Status              string    `json:"status" gorm:"size:20;default:'running';comment:'状态(running/completed/failed)'"`
	// 统计信息
	TotalResources     int       `json:"total_resources" gorm:"default:0;comment:'总资源数'"`
	ClassifiedResources int      `json:"classified_resources" gorm:"default:0;comment:'已分类资源数'"`
	FailedResources    int       `json:"failed_resources" gorm:"default:0;comment:'分类失败资源数'"`
	NewTenants         int       `json:"new_tenants" gorm:"default:0;comment:'新增租户数'"`
	NewEnvironments    int       `json:"new_environments" gorm:"default:0;comment:'新增环境数'"`
	NewApplications    int       `json:"new_applications" gorm:"default:0;comment:'新增应用数'"`
	// 详细信息
	Details            JSONMap   `json:"details" gorm:"type:jsonb;default:'{}';comment:'详细信息'"`
	ErrorMessage       string    `json:"error_message" gorm:"type:text;comment:'错误信息'"`
	// 时间信息
	StartedAt          time.Time `json:"started_at" gorm:"autoCreateTime"`
	CompletedAt        *time.Time `json:"completed_at" gorm:"comment:'完成时间'"`
	DurationSeconds    int       `json:"duration_seconds" gorm:"default:0;comment:'耗时秒数'"`
}

func (ResourceClassificationHistory) TableName() string {
	return "resource_classification_history"
}

// ClassificationHistoryStatus 分类历史状态常量
const (
	ClassificationHistoryStatusRunning   = "running"
	ClassificationHistoryStatusCompleted = "completed"
	ClassificationHistoryStatusFailed    = "failed"
)

// ClassificationTriggerType 触发类型常量
const (
	ClassificationTriggerTypeImport    = "import"
	ClassificationTriggerTypeManual    = "manual"
	ClassificationTriggerTypeScheduled = "scheduled"
)
