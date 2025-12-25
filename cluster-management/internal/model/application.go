package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// StringSlice 用于存储字符串切片到数据库
type StringSlice []string

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*s = result
	return nil
}

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Application 应用 - 业务逻辑单元
type Application struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID        uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index;comment:'关联租户ID'"`
	EnvironmentID   uuid.UUID `json:"environment_id" gorm:"type:uuid;not null;index;comment:'关联环境ID'"`
	Name            string    `json:"name" gorm:"size:255;not null;comment:'应用名称'"`
	DisplayName     string    `json:"display_name" gorm:"size:255;comment:'显示名称'"`
	Description     string    `json:"description" gorm:"type:text;comment:'描述'"`
	Labels          JSONMap     `json:"labels" gorm:"type:jsonb;default:'{}';comment:'标签'"`
	// 应用聚合信息
	WorkloadTypes   StringSlice `json:"workload_types" gorm:"type:jsonb;comment:'工作负载类型列表(Deployment/StatefulSet等)'"`
	ServiceNames    StringSlice `json:"service_names" gorm:"type:jsonb;comment:'关联Service名称列表'"`
	// 资源统计
	DeploymentCount int       `json:"deployment_count" gorm:"default:0;comment:'Deployment数量'"`
	PodCount        int       `json:"pod_count" gorm:"default:0;comment:'Pod数量'"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Application) TableName() string {
	return "applications"
}

func (a *Application) BeforeCreate() error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// ApplicationLabelPriority 应用标签优先级
const (
	AppLabelKubernetesName = "app.kubernetes.io/name"
	AppLabelStandard       = "app"
	AppLabelK8sApp         = "k8s-app"
)
