package model

import (
	"time"

	"github.com/google/uuid"
)

type Node struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ClusterID       uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index"`
	Name            string    `json:"name" gorm:"type:varchar(255);not null;index:idx_nodes_name"`
	Type            string    `json:"type" gorm:"type:varchar(50);not null;check:type IN ('control-plane','worker','master')"`
	Status          string    `json:"status" gorm:"type:varchar(20);not null;index"`
	CPUCores        int       `json:"cpu_cores" gorm:"type:integer;default:0"`
	CPUUsedCores    float64   `json:"cpu_used_cores" gorm:"type:decimal(10,2);default:0"`
	MemoryBytes     int64     `json:"memory_bytes" gorm:"type:bigint;default:0"`
	MemoryUsedBytes int64     `json:"memory_used_bytes" gorm:"type:bigint;default:0"`
	PodCount        int       `json:"pod_count" gorm:"type:integer;default:0"`
	Labels          JSONMap   `json:"labels" gorm:"type:jsonb;default:'{}'::jsonb"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// 关联
	Cluster Cluster `json:"-" gorm:"foreignKey:ClusterID"`
}

func (Node) TableName() string {
	return "nodes"
}

type NodeList []Node

func (list NodeList) ToJSON() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, node := range list {
		result = append(result, map[string]interface{}{
			"id":                node.ID,
			"cluster_id":        node.ClusterID,
			"name":              node.Name,
			"type":              node.Type,
			"status":            node.Status,
			"cpu_cores":         node.CPUCores,
			"cpu_used_cores":    node.CPUUsedCores,
			"cpu_usage_percent": calculateCPUPercent(node.CPUCores, node.CPUUsedCores),
			"memory_bytes":      node.MemoryBytes,
			"memory_used_bytes": node.MemoryUsedBytes,
			"memory_usage_percent": calculateMemoryPercent(node.MemoryBytes, node.MemoryUsedBytes),
			"pod_count":         node.PodCount,
			"labels":            node.Labels,
			"created_at":        node.CreatedAt,
			"updated_at":        node.UpdatedAt,
		})
	}
	return result
}

func calculateCPUPercent(total int, used float64) float64 {
	if total == 0 {
		return 0
	}
	return round(used/float64(total)*100, 2)
}

func calculateMemoryPercent(total int64, used int64) float64 {
	if total == 0 {
		return 0
	}
	return round(float64(used)/float64(total)*100, 2)
}

func round(val float64, precision int) float64 {
	pow := 1
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	return float64(int(val*float64(pow)+0.5)) / float64(pow)
}
