package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type ExpansionHandler struct {
	expansionService *service.ExpansionService
}

type ExpansionRequest struct {
	NewNodeCount int    `json:"new_node_count" binding:"required,min=1"`
	NewCPUCores  int    `json:"new_cpu_cores" binding:"required,min=1"`
	NewMemoryGB  int    `json:"new_memory_gb" binding:"required,min=1"`
	NewStorageGB int    `json:"new_storage_gb" binding:"required,min=1"`
	Reason       string `json:"reason" binding:"required"`
}

type ExpansionResponse struct {
	ID           uuid.UUID `json:"id"`
	ClusterID    uuid.UUID `json:"cluster_id"`
	OldNodeCount int       `json:"old_node_count"`
	NewNodeCount int       `json:"new_node_count"`
	OldCPUCores  int       `json:"old_cpu_cores"`
	NewCPUCores  int       `json:"new_cpu_cores"`
	OldMemoryGB  int       `json:"old_memory_gb"`
	NewMemoryGB  int       `json:"new_memory_gb"`
	OldStorageGB int       `json:"old_storage_gb"`
	NewStorageGB int       `json:"new_storage_gb"`
	Status       string    `json:"status"`
	Reason       string    `json:"reason"`
	RequestedBy  string    `json:"requested_by"`
	CreatedAt    string    `json:"created_at"`
}

type ExpansionHistoryResponse struct {
	Expansions []ExpansionResponse `json:"expansions"`
}

func NewExpansionHandler(expansionService *service.ExpansionService) *ExpansionHandler {
	return &ExpansionHandler{
		expansionService: expansionService,
	}
}

func (h *ExpansionHandler) RequestExpansion(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	var req ExpansionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	expansion, err := h.expansionService.RequestExpansion(
		id,
		req.NewNodeCount,
		req.NewCPUCores,
		req.NewMemoryGB,
		req.NewStorageGB,
		req.Reason,
		"api-user",
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to request expansion: %v", err)
		return
	}

	go h.expansionService.ExecuteExpansion(expansion.ID.String())

	response := ExpansionResponse{
		ID:           expansion.ID,
		ClusterID:    expansion.ClusterID,
		OldNodeCount: expansion.OldNodeCount,
		NewNodeCount: expansion.NewNodeCount,
		OldCPUCores:  expansion.OldCPUCores,
		NewCPUCores:  expansion.NewCPUCores,
		OldMemoryGB:  expansion.OldMemoryGB,
		NewMemoryGB:  expansion.NewMemoryGB,
		OldStorageGB: expansion.OldStorageGB,
		NewStorageGB: expansion.NewStorageGB,
		Status:       expansion.Status,
		Reason:       expansion.Reason,
		RequestedBy:  expansion.RequestedBy,
		CreatedAt:    expansion.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	utils.Success(c, http.StatusCreated, response)
}

func (h *ExpansionHandler) GetExpansionHistory(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	expansions, err := h.expansionService.GetExpansionHistory(id)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to get expansion history: %v", err)
		return
	}

	responses := make([]ExpansionResponse, 0, len(expansions))
	for _, expansion := range expansions {
		responses = append(responses, ExpansionResponse{
			ID:           expansion.ID,
			ClusterID:    expansion.ClusterID,
			OldNodeCount: expansion.OldNodeCount,
			NewNodeCount: expansion.NewNodeCount,
			OldCPUCores:  expansion.OldCPUCores,
			NewCPUCores:  expansion.NewCPUCores,
			OldMemoryGB:  expansion.OldMemoryGB,
			NewMemoryGB:  expansion.NewMemoryGB,
			OldStorageGB: expansion.OldStorageGB,
			NewStorageGB: expansion.NewStorageGB,
			Status:       expansion.Status,
			Reason:       expansion.Reason,
			RequestedBy:  expansion.RequestedBy,
			CreatedAt:    expansion.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response := ExpansionHistoryResponse{
		Expansions: responses,
	}

	utils.Success(c, http.StatusOK, response)
}
