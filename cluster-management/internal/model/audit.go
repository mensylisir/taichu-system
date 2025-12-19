package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditEvent struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID  uuid.UUID `json:"cluster_id" gorm:"index"`
	EventType  string    `json:"event_type" gorm:"size:50;not null"`
	Action     string    `json:"action" gorm:"size:100;not null"`
	Resource   string    `json:"resource" gorm:"size:100;not null"`
	ResourceID string    `json:"resource_id" gorm:"size:255"`
	User       string    `json:"user" gorm:"size:100;not null"`
	IPAddress  string    `json:"ip_address" gorm:"size:50"`
	UserAgent  string    `json:"user_agent" gorm:"type:text"`
	OldValue   JSONMap   `json:"old_value" gorm:"type:jsonb"`
	NewValue   JSONMap   `json:"new_value" gorm:"type:jsonb"`
	Details    JSONMap   `json:"details" gorm:"type:jsonb"`
	Result     string    `json:"result" gorm:"size:20;default:'success'"`
	ErrorMsg   string    `json:"error_msg" gorm:"type:text"`
	Timestamp  time.Time `json:"timestamp" gorm:"autoCreateTime;index"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (AuditEvent) TableName() string {
	return "audit_events"
}
