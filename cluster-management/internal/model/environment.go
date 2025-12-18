package model

import (
	"time"

	"github.com/google/uuid"
)

type ClusterEnvironment struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID     uuid.UUID `json:"cluster_id" gorm:"index;not null"`
	EnvType       string    `json:"env_type" gorm:"size:50;not null"`
	Namespace     string    `json:"namespace" gorm:"size:255;not null"`
	DisplayName   string    `json:"display_name" gorm:"size:255"`
	Description   string    `json:"description" gorm:"type:text"`
	EnvironmentType string  `json:"environment_type" gorm:"size:50;default:'production'"`
	TopologyType  string    `json:"topology_type" gorm:"size:50;default:'standard'"`
	BackupEnabled bool      `json:"backup_enabled" gorm:"default:false"`
	Labels        JSONMap   `json:"labels" gorm:"type:jsonb;default:'{}'"`
	Status        string    `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ClusterEnvironment) TableName() string {
	return "cluster_environments"
}
