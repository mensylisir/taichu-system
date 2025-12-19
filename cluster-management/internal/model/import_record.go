package model

import (
	"time"

	"github.com/google/uuid"
)

type ImportRecord struct {
	ID                uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID         *uuid.UUID `json:"cluster_id" gorm:"index"`
	ImportSource      string     `json:"import_source" gorm:"size:100;not null"`
	ImportStatus      string     `json:"import_status" gorm:"size:50;default:'pending'"`
	ImportedBy        string     `json:"imported_by" gorm:"size:100;not null"`
	ValidationResults JSONMap    `json:"validation_results" gorm:"type:jsonb;default:'{}'"`
	ImportedResources JSONMap    `json:"imported_resources" gorm:"type:jsonb;default:'{}'"`
	ErrorMsg          string     `json:"error_msg" gorm:"type:text"`
	ErrorMessage      string     `json:"error_message" gorm:"-"`
	CompletedAt       *time.Time `json:"completed_at"`
	ImportedAt        time.Time  `json:"imported_at" gorm:"autoCreateTime"`
}

func (ImportRecord) TableName() string {
	return "import_records"
}
