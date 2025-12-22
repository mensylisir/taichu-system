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

	// 新增字段
	RBACDetails              string    `json:"rbac_details" gorm:"type:text"`
	NetworkPolicyDetails     string    `json:"network_policy_details" gorm:"type:text"`
	PodSecurityDetails       string    `json:"pod_security_details" gorm:"type:text"`
	AuditLoggingDetails      string    `json:"audit_logging_details" gorm:"type:text"`
	CNIPlugin                string    `json:"cni_plugin" gorm:"column:cni_plugin;type:varchar(50)"`
	PodSecurityAdmissionMode string    `json:"pod_security_admission_mode" gorm:"type:varchar(20);default:'none'"`

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
	
	// 新增字段
	RBACDetails              string `json:"rbac_details"`
	NetworkPolicyDetails     string `json:"network_policy_details"`
	PodSecurityDetails       string `json:"pod_security_details"`
	AuditLoggingDetails      string `json:"audit_logging_details"`
	CNIPlugin                string `json:"cni_plugin"`
	PodSecurityAdmissionMode string `json:"pod_security_admission_mode"`
}

type SecurityPolicyDetails struct {
	PSSPolicy           string `json:"pss_policy"`
	NetworkPoliciesCount int    `json:"network_policies_count"`
	RBACRolesCount       int    `json:"rbac_roles_count"`
	AuditEnabled         bool   `json:"audit_enabled"`
	
	// 新增字段
	CNIPlugin                string `json:"cni_plugin"`
	PodSecurityAdmissionMode string `json:"pod_security_admission_mode"`
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
			CNIPlugin:           sp.CNIPlugin,
			PodSecurityAdmissionMode: sp.PodSecurityAdmissionMode,
		},
		// 新增字段
		RBACDetails:              sp.RBACDetails,
		NetworkPolicyDetails:     sp.NetworkPolicyDetails,
		PodSecurityDetails:       sp.PodSecurityDetails,
		AuditLoggingDetails:      sp.AuditLoggingDetails,
		CNIPlugin:                sp.CNIPlugin,
		PodSecurityAdmissionMode: sp.PodSecurityAdmissionMode,
	}
}