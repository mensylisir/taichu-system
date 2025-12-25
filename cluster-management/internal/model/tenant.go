package model

import (
	"time"

	"github.com/google/uuid"
)

// Tenant 租户 - 逻辑层
type Tenant struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null;size:255;comment:'租户名称(system/default/用户自定义)'"`
	DisplayName string    `json:"display_name" gorm:"size:255;comment:'显示名称'"`
	Type        string    `json:"type" gorm:"size:50;not null;comment:'租户类型(system/default/user_created)'"`
	Description string    `json:"description" gorm:"type:text;comment:'描述'"`
	Labels      JSONMap   `json:"labels" gorm:"type:jsonb;default:'{}';comment:'标签'"`
	Status      string    `json:"status" gorm:"size:20;default:'active';comment:'状态(active/suspended)'"`
	IsSystem    bool      `json:"is_system" gorm:"default:false;comment:'是否为系统预定义租户(system/default)'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Tenant) TableName() string {
	return "tenants"
}

func (t *Tenant) BeforeCreate() error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// PredefinedTenants 预定义租户常量
var PredefinedTenants = struct {
	System  string
	Default string
}{
	System:  "system",
	Default: "default",
}

// TenantType 租户类型常量
const (
	TenantTypeSystem      = "system"
	TenantTypeDefault     = "default"
	TenantTypeUserCreated = "user_created"
)

// TenantStatus 租户状态常量
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
)
