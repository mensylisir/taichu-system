package model

import (
	"time"

	"github.com/google/uuid"
)

type SecurityPolicy struct {
	ID                      uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ClusterID               uuid.UUID `json:"cluster_id" gorm:"type:uuid;not null;uniqueIndex"`
	PodSecurityStandard     string    `json:"pod_security_standard" gorm:"type:varchar(50);default:'baseline';check:pod_security_standard IN ('privileged','baseline','restricted','disabled')"`
	NetworkPoliciesEnabled  bool      `json:"network_policies_enabled" gorm:"type:boolean;default:false"`
	NetworkPoliciesCount    int       `json:"network_policies_count" gorm:"type:integer;default:0"`
	RBACEnabled             bool      `json:"rbac_enabled" gorm:"type:boolean;default:true"`
	RBACRolesCount          int       `json:"rbac_roles_count" gorm:"type:integer;default:0"`
	AuditLoggingEnabled     bool      `json:"audit_logging_enabled" gorm:"type:boolean;default:false"`
	AuditLoggingMode        string    `json:"audit_logging_mode" gorm:"type:varchar(20);default:'none';check:audit_logging_mode IN ('none','metadata','request','request-response')"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	// 关联
	Cluster Cluster `json:"-" gorm:"foreignKey:ClusterID"`
}

func (SecurityPolicy) TableName() string {
	return "security_policies"
}

type SecurityPolicySummary struct {
	PodSecurityStandard     string `json:"pod_security_standard"`
	NetworkPoliciesEnabled  bool   `json:"network_policies_enabled"`
	NetworkPoliciesCount    int    `json:"network_policies_count"`
	RBACEnabled             bool   `json:"rbac_enabled"`
	RBACRolesCount          int    `json:"rbac_roles_count"`
	AuditLoggingEnabled     bool   `json:"audit_logging_enabled"`
	AuditLoggingMode        string `json:"audit_logging_mode"`
	Details                 SecurityPolicyDetails `json:"details"`
}

type SecurityPolicyDetails struct {
	PSSPolicy           string `json:"pss_policy"`
	NetworkPoliciesCount int    `json:"network_policies_count"`
	RBACRolesCount       int    `json:"rbac_roles_count"`
	AuditEnabled         bool   `json:"audit_enabled"`
}

func (sp SecurityPolicy) ToSecurityPolicySummary() SecurityPolicySummary {
	return SecurityPolicySummary{
		PodSecurityStandard:    sp.PodSecurityStandard,
		NetworkPoliciesEnabled: sp.NetworkPoliciesEnabled,
		NetworkPoliciesCount:   sp.NetworkPoliciesCount,
		RBACEnabled:            sp.RBACEnabled,
		RBACRolesCount:         sp.RBACRolesCount,
		AuditLoggingEnabled:    sp.AuditLoggingEnabled,
		AuditLoggingMode:       sp.AuditLoggingMode,
		Details: SecurityPolicyDetails{
			PSSPolicy:           sp.PodSecurityStandard,
			NetworkPoliciesCount: sp.NetworkPoliciesCount,
			RBACRolesCount:       sp.RBACRolesCount,
			AuditEnabled:         sp.AuditLoggingEnabled,
		},
	}
}
