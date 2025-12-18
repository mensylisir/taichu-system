package model

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ClusterID       uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;index"`
	NodeID          *uuid.UUID `json:"node_id" gorm:"type:uuid;index"`
	EventType       string    `json:"event_type" gorm:"type:varchar(50);not null"`
	Message         string    `json:"message" gorm:"type:text;not null"`
	Severity        string    `json:"severity" gorm:"type:varchar(20);default:'info';check:severity IN ('info','warning','error','critical')"`
	Component       string    `json:"component" gorm:"type:varchar(100)"`
	FirstTimestamp  time.Time `json:"first_timestamp" gorm:"type:timestamp with time zone"`
	LastTimestamp   time.Time `json:"last_timestamp" gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP;index"`
	Count           int       `json:"count" gorm:"type:integer;default:1"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// 关联
	Cluster Cluster `json:"-" gorm:"foreignKey:ClusterID"`
	Node    *Node   `json:"-" gorm:"foreignKey:NodeID"`
}

func (Event) TableName() string {
	return "events"
}

type EventList []Event

func (list EventList) ToJSON() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, event := range list {
		result = append(result, map[string]interface{}{
			"id":                event.ID,
			"cluster_id":        event.ClusterID,
			"node_id":           event.NodeID,
			"type":              event.EventType,
			"message":           event.Message,
			"severity":          event.Severity,
			"component":         event.Component,
			"count":             event.Count,
			"first_timestamp":   event.FirstTimestamp,
			"last_timestamp":    event.LastTimestamp,
			"created_at":        event.CreatedAt,
			"updated_at":        event.UpdatedAt,
		})
	}
	return result
}

func (list EventList) ToSummary(total int64) map[string]interface{} {
	return map[string]interface{}{
		"events": list.ToJSON(),
		"total":  total,
	}
}
