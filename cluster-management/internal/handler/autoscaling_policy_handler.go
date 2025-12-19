package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type AutoscalingPolicyHandler struct {
	autoscalingPolicyService *service.AutoscalingPolicyService
}

type AutoscalingPolicyResponse struct {
	Enabled                   bool   `json:"enabled"`
	MinNodes                  int    `json:"min_nodes"`
	MaxNodes                  int    `json:"max_nodes"`
	ScaleUpThreshold          int    `json:"scale_up_threshold"`
	ScaleDownThreshold        int    `json:"scale_down_threshold"`
	HPACount                  int    `json:"hpa_count"`
	ClusterAutoscalerEnabled  bool   `json:"cluster_autoscaler_enabled"`
	VPACount                  int    `json:"vpa_count"`
	HPAPolicies               []HPAPolicy `json:"hpa_policies,omitempty"`
}

type HPAPolicy struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	MinReplicas     int    `json:"min_replicas"`
	MaxReplicas     int    `json:"max_replicas"`
	CurrentReplicas int    `json:"current_replicas"`
}

func NewAutoscalingPolicyHandler(autoscalingPolicyService *service.AutoscalingPolicyService) *AutoscalingPolicyHandler {
	return &AutoscalingPolicyHandler{
		autoscalingPolicyService: autoscalingPolicyService,
	}
}

func (h *AutoscalingPolicyHandler) GetAutoscalingPolicy(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	policy, err := h.autoscalingPolicyService.GetAutoscalingPolicy(id.String())
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Autoscaling policy not found")
		return
	}

	hpaPolicies := make([]HPAPolicy, 0)
	for _, hpa := range policy.HPAPolicies {
		hpaPolicies = append(hpaPolicies, HPAPolicy{
			Name:            hpa.Name,
			Namespace:       hpa.Namespace,
			MinReplicas:     hpa.MinReplicas,
			MaxReplicas:     hpa.MaxReplicas,
			CurrentReplicas: hpa.CurrentReplicas,
		})
	}

	response := AutoscalingPolicyResponse{
		Enabled:                  policy.Enabled,
		MinNodes:                 policy.MinNodes,
		MaxNodes:                 policy.MaxNodes,
		ScaleUpThreshold:         policy.ScaleUpThreshold,
		ScaleDownThreshold:       policy.ScaleDownThreshold,
		HPACount:                 policy.HPACount,
		ClusterAutoscalerEnabled: policy.ClusterAutoscalerEnabled,
		VPACount:                 policy.VPACount,
		HPAPolicies:              hpaPolicies,
	}

	utils.Success(c, http.StatusOK, response)
}
