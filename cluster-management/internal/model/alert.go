package model

import (
	"time"

	"github.com/google/uuid"
)

type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
)

type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusIgnored  AlertStatus = "ignored"
)

type AlertType string

const (
	AlertTypeBackupFailed    AlertType = "backup_failed"
	AlertTypeScheduleFailed  AlertType = "schedule_failed"
	AlertTypeSystemError     AlertType = "system_error"
	AlertTypeResourceExhausted AlertType = "resource_exhausted"
)

type Alert struct {
	ID          uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Type        AlertType    `gorm:"type:varchar(50);not null" json:"type"`
	Severity    AlertSeverity `gorm:"type:varchar(20);not null" json:"severity"`
	Status      AlertStatus  `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	Title       string       `gorm:"type:varchar(200);not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	ResourceType string      `gorm:"type:varchar(50)" json:"resource_type"`
	ResourceID   string      `gorm:"type:varchar(100)" json:"resource_id"`
	Metadata     JSONMap      `gorm:"type:jsonb" json:"metadata"`
	ResolvedAt   *time.Time   `gorm:"type:timestamp" json:"resolved_at"`
	CreatedAt    time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Alert) TableName() string {
	return "alerts"
}
