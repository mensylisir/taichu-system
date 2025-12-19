package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClusterResource struct {
	ID                   uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ClusterID            uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index"`
	Timestamp            time.Time `json:"timestamp" gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP;index"`
	TotalCPUCores        int       `json:"total_cpu_cores" gorm:"type:integer;default:0"`
	UsedCPUCores         float64   `json:"used_cpu_cores" gorm:"type:decimal(10,2);default:0"`
	CPUUsagePercent      float64   `json:"cpu_usage_percent" gorm:"type:decimal(5,2);default:0"`
	TotalMemoryBytes     int64     `json:"total_memory_bytes" gorm:"type:bigint;default:0"`
	UsedMemoryBytes      int64     `json:"used_memory_bytes" gorm:"type:bigint;default:0"`
	MemoryUsagePercent   float64   `json:"memory_usage_percent" gorm:"type:decimal(5,2);default:0"`
	TotalStorageBytes    int64     `json:"total_storage_bytes" gorm:"type:bigint;default:0"`
	UsedStorageBytes     int64     `json:"used_storage_bytes" gorm:"type:bigint;default:0"`
	StorageUsagePercent  float64   `json:"storage_usage_percent" gorm:"type:decimal(5,2);default:0"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Cluster Cluster `json:"-" gorm:"foreignKey:ClusterID"`
}

func (ClusterResource) TableName() string {
	return "cluster_resources"
}

type ResourceSummary struct {
	CPU    ResourceMetrics    `json:"cpu"`
	Memory ResourceMetrics    `json:"memory"`
	Storage ResourceMetrics   `json:"storage"`
	Timestamp time.Time       `json:"timestamp"`
}

type ResourceMetrics struct {
	Total         int64   `json:"total"`
	Used          int64   `json:"used"`
	UsagePercent  float64 `json:"usage_percent"`
}

func (cr ClusterResource) ToResourceSummary() ResourceSummary {
	return ResourceSummary{
		CPU: ResourceMetrics{
			Total:        int64(cr.TotalCPUCores),
			Used:         int64(cr.UsedCPUCores),
			UsagePercent: cr.CPUUsagePercent,
		},
		Memory: ResourceMetrics{
			Total:        cr.TotalMemoryBytes,
			Used:         cr.UsedMemoryBytes,
			UsagePercent: cr.MemoryUsagePercent,
		},
		Storage: ResourceMetrics{
			Total:        cr.TotalStorageBytes,
			Used:         cr.UsedStorageBytes,
			UsagePercent: cr.StorageUsagePercent,
		},
		Timestamp: cr.Timestamp,
	}
}
