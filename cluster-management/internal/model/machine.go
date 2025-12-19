package model

import (
	"time"

	"github.com/google/uuid"
)

// Machine 机器信息模型
// 用于管理基础设施机器，包括部署节点信息
type Machine struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name             string    `json:"name" gorm:"size:255;not null;index"`
	IPAddress        string    `json:"ip_address" gorm:"size:50;not null;index"`
	InternalAddress  string    `json:"internal_address" gorm:"size:50"`
	User             string    `json:"user" gorm:"size:100;not null"`
	Password         string    `json:"-" gorm:"size:255"` // 加密存储，不在JSON中返回
	Role             string    `json:"role" gorm:"size:50;not null"` // master/worker/etcd/registry
	Status           string    `json:"status" gorm:"size:50;default:'available'"`
	ArtifactPath     string    `json:"artifact_path" gorm:"type:text"`
	ImageRepo        string    `json:"image_repo" gorm:"size:255"`
	RegistryAddress  string    `json:"registry_address" gorm:"size:255"`
	Labels           JSONMap   `json:"labels" gorm:"type:jsonb;default:'{}'"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 返回表名
func (Machine) TableName() string {
	return "machines"
}
