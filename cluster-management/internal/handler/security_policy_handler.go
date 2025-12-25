package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type SecurityPolicyHandler struct {
	securityPolicyService *service.SecurityPolicyService
}

type SecurityPolicyResponse struct {
	PodSecurityStandard     string `json:"pod_security_standard"`
	NetworkPoliciesEnabled  bool   `json:"network_policies_enabled"`
	NetworkPoliciesCount    int    `json:"network_policies_count"`
	RBACEnabled             bool   `json:"rbac_enabled"`
	RBACRolesCount          int    `json:"rbac_roles_count"`
	AuditLoggingEnabled     bool   `json:"audit_logging_enabled"`
	AuditLoggingMode        string `json:"audit_logging_mode"`
	
	// 新增字段
	RBACDetails              string `json:"rbac_details"`
	NetworkPolicyDetails     string `json:"network_policy_details"`
	PodSecurityDetails       string `json:"pod_security_details"`
	AuditLoggingDetails      string `json:"audit_logging_details"`
	CNIPlugin                string `json:"cni_plugin"`
	PodSecurityAdmissionMode string `json:"pod_security_admission_mode"`
}

func NewSecurityPolicyHandler(securityPolicyService *service.SecurityPolicyService) *SecurityPolicyHandler {
	return &SecurityPolicyHandler{
		securityPolicyService: securityPolicyService,
	}
}

func (h *SecurityPolicyHandler) GetSecurityPolicy(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeInvalidInput, "Invalid cluster ID")
		return
	}

	policy, err := h.securityPolicyService.GetSecurityPolicy(c.Request.Context(), id.String())
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Security policy not found")
		return
	}

	response := SecurityPolicyResponse{
		PodSecurityStandard:     policy.PodSecurityStandard,
		NetworkPoliciesEnabled:  policy.NetworkPoliciesEnabled,
		NetworkPoliciesCount:    policy.NetworkPoliciesCount,
		RBACEnabled:             policy.RBACEnabled,
		RBACRolesCount:          policy.RBACRolesCount,
		AuditLoggingEnabled:     policy.AuditLoggingEnabled,
		AuditLoggingMode:        policy.AuditLoggingMode,
		RBACDetails:              policy.RBACDetails,
		NetworkPolicyDetails:     policy.NetworkPolicyDetails,
		PodSecurityDetails:       policy.PodSecurityDetails,
		AuditLoggingDetails:      policy.AuditLoggingDetails,
		CNIPlugin:                policy.CNIPlugin,
		PodSecurityAdmissionMode: policy.PodSecurityAdmissionMode,
	}

	utils.Success(c, http.StatusOK, response)
}