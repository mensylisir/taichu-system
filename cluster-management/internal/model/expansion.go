package model

import (
	"time"

	"github.com/google/uuid"
)

type ClusterExpansion struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID    uuid.UUID `json:"cluster_id" gorm:"index;not null"`
	OldNodeCount int       `json:"old_node_count" gorm:"not null"`
	NewNodeCount int       `json:"new_node_count" gorm:"not null"`
	OldCPUCores  int       `json:"old_cpu_cores" gorm:"not null"`
	NewCPUCores  int       `json:"new_cpu_cores" gorm:"not null"`
	OldMemoryGB  int       `json:"old_memory_gb" gorm:"not null"`
	NewMemoryGB  int       `json:"new_memory_gb" gorm:"not null"`
	OldStorageGB int       `json:"old_storage_gb" gorm:"not null"`
	NewStorageGB int       `json:"new_storage_gb" gorm:"not null"`
	Status       string    `json:"status" gorm:"size:20;default:'pending'"`
	Reason       string    `json:"reason" gorm:"type:text"`
	ErrorMsg     string    `json:"error_msg" gorm:"type:text"`
	Details      JSONMap   `json:"details" gorm:"type:jsonb"`
	RequestedBy  string    `json:"requested_by" gorm:"size:100;not null"`
	ExecutedBy   string    `json:"executed_by" gorm:"size:100"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ClusterExpansion) TableName() string {
	return "cluster_expansions"
}
