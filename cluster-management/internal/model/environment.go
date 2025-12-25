package model

import (
	"time"

	"github.com/google/uuid"
)

// Environment 环境 - 物理边界 (Kubernetes Namespace)
type Environment struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index;comment:'关联租户ID'"`
	ClusterID   uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index;comment:'关联集群ID'"`
	Namespace   string    `json:"namespace" gorm:"size:255;not null;comment:'Kubernetes Namespace名称'"`
	DisplayName string    `json:"display_name" gorm:"size:255;comment:'显示名称'"`
	Description string    `json:"description" gorm:"type:text;comment:'描述'"`
	Labels      JSONMap   `json:"labels" gorm:"type:jsonb;default:'{}';comment:'标签'"`
	Status      string    `json:"status" gorm:"size:20;default:'active';comment:'状态(active/inactive/terminating)'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Environment) TableName() string {
	return "environments"
}

func (e *Environment) BeforeCreate() error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// EnvironmentStatus 环境状态常量
const (
	EnvironmentStatusActive     = "active"
	EnvironmentStatusInactive   = "inactive"
	EnvironmentStatusTerminating = "terminating"
)

// ClusterEnvironment 集群环境（旧的模型，用于兼容现有代码）
type ClusterEnvironment struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID       uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index;comment:'关联集群ID'"`
	EnvironmentName string    `json:"environment_name" gorm:"size:100;not null;comment:'环境名称'"`
	EnvironmentType string    `json:"environment_type" gorm:"size:50;not null;comment:'环境类型'"`
	TopologyType    string    `json:"topology_type" gorm:"size:50;comment:'拓扑类型'"`
	DisplayName     string    `json:"display_name" gorm:"size:255;comment:'显示名称'"`
	Description     string    `json:"description" gorm:"type:text;comment:'描述'"`
	BackupEnabled   bool      `json:"backup_enabled" gorm:"default:false;comment:'是否启用备份'"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ClusterEnvironment) TableName() string {
	return "cluster_environments"
}

func (c *ClusterEnvironment) BeforeCreate() error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
