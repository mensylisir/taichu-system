package model

import (
	"time"

	"github.com/google/uuid"
)

type AutoscalingPolicy struct {
	ID                     uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ClusterID              uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;uniqueIndex"`
	Enabled                bool      `json:"enabled" gorm:"type:boolean;default:false"`
	MinNodes               int       `json:"min_nodes" gorm:"type:integer;default:1"`
	MaxNodes               int       `json:"max_nodes" gorm:"type:integer;default:10"`
	ScaleUpThreshold       int       `json:"scale_up_threshold" gorm:"type:integer;default:70"`
	ScaleDownThreshold     int       `json:"scale_down_threshold" gorm:"type:integer;default:30"`
	HPACount               int       `json:"hpa_count" gorm:"type:integer;default:0"`
	ClusterAutoscalerEnabled bool   `json:"cluster_autoscaler_enabled" gorm:"type:boolean;default:false"`
	VPACount               int       `json:"vpa_count" gorm:"type:integer;default:0"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`

	// 关联
	Cluster Cluster `json:"-" gorm:"foreignKey:ClusterID"`
}

func (AutoscalingPolicy) TableName() string {
	return "autoscaling_policies"
}

type AutoscalingPolicySummary struct {
	Enabled                bool   `json:"enabled"`
	MinNodes               int    `json:"min_nodes"`
	MaxNodes               int    `json:"max_nodes"`
	ScaleUpThreshold       int    `json:"scale_up_threshold"`
	ScaleDownThreshold     int    `json:"scale_down_threshold"`
	HPACount               int    `json:"hpa_count"`
	ClusterAutoscalerEnabled bool `json:"cluster_autoscaler_enabled"`
	VPACount               int    `json:"vpa_count"`
	HPAPolicies            []HPAPolicy `json:"hpa_policies"`
	ClusterAutoscaler      ClusterAutoscalerInfo `json:"cluster_autoscaler"`
	VPAPolicies            []VPAPolicy `json:"vpa_policies"`
}

type HPAPolicy struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	MinReplicas int  `json:"min_replicas"`
	MaxReplicas int  `json:"max_replicas"`
	CurrentReplicas int `json:"current_replicas"`
	TargetCPUUtilizationPercentage *int `json:"target_cpu_utilization_percentage"`
}

type ClusterAutoscalerInfo struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type VPAPolicy struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UpdateMode string `json:"update_mode"`
	MinReplicas *int  `json:"min_replicas"`
	MaxReplicas *int  `json:"max_replicas"`
}

func (ap AutoscalingPolicy) ToAutoscalingPolicySummary() AutoscalingPolicySummary {
	return AutoscalingPolicySummary{
		Enabled:                   ap.Enabled,
		MinNodes:                  ap.MinNodes,
		MaxNodes:                  ap.MaxNodes,
		ScaleUpThreshold:          ap.ScaleUpThreshold,
		ScaleDownThreshold:        ap.ScaleDownThreshold,
		HPACount:                  ap.HPACount,
		ClusterAutoscalerEnabled:  ap.ClusterAutoscalerEnabled,
		VPACount:                  ap.VPACount,
		HPAPolicies:               []HPAPolicy{},
		ClusterAutoscaler: ClusterAutoscalerInfo{
			Enabled: ap.ClusterAutoscalerEnabled,
			Status:  "Running",
		},
		VPAPolicies: []VPAPolicy{},
	}
}
