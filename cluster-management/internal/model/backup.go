package model

import (
	"time"

	"github.com/google/uuid"
)

type ClusterBackup struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID        uuid.UUID `json:"cluster_id" gorm:"index;not null"`
	Name             string    `json:"name" gorm:"size:255;not null"`
	Description      string    `json:"description" gorm:"type:text"`
	BackupName       string    `json:"backup_name" gorm:"size:255"`
	BackupType       string    `json:"backup_type" gorm:"size:50;default:'full'"`
	Status           string    `json:"status" gorm:"size:20;default:'pending'"`
	SizeBytes        int64     `json:"size_bytes" gorm:"default:0"`
	StorageSizeBytes int64     `json:"storage_size_bytes" gorm:"default:0"`
	StorageLocation  string    `json:"storage_location" gorm:"type:text"`
	Location         string    `json:"location" gorm:"type:text"`
	RetentionDays    int       `json:"retention_days" gorm:"default:7"`
	SnapshotTimestamp *time.Time `json:"snapshot_timestamp"`
	ErrorMsg         string    `json:"error_msg" gorm:"type:text"`
	CreatedBy        string    `json:"created_by" gorm:"size:100;not null"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ClusterBackup) TableName() string {
	return "cluster_backups"
}

type BackupSchedule struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID   uuid.UUID `json:"cluster_id" gorm:"index;not null"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	CronExpr    string    `json:"cron_expr" gorm:"size:100;not null"`
	BackupType  string    `json:"backup_type" gorm:"size:50;default:'full'"`
	RetentionDays int     `json:"retention_days" gorm:"default:7"`
	Status      string    `json:"status" gorm:"size:20;default:'active'"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	LastRunAt   *time.Time `json:"last_run_at"`
	NextRunAt   *time.Time `json:"next_run_at"`
	CreatedBy   string    `json:"created_by" gorm:"size:100;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (BackupSchedule) TableName() string {
	return "backup_schedules"
}
