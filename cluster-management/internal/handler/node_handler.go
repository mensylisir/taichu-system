package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type NodeHandler struct {
	nodeService *service.NodeService
}

type NodeSummary struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	CPUCores          int       `json:"cpu_cores"`
	CPUUsagePercent   float64   `json:"cpu_usage_percent"`
	MemoryBytes       int64     `json:"memory_bytes"`
	MemoryUsagePercent float64  `json:"memory_usage_percent"`
	PodCount          int       `json:"pod_count"`
	Labels            map[string]string `json:"labels"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

type NodeListResponse struct {
	Nodes   []NodeSummary                `json:"nodes"`
	Total   int64                        `json:"total"`
	Summary service.NodeSummaryStats `json:"summary"`
}

type NodeSummaryStats struct {
	ControlPlaneCount int `json:"control_plane_count"`
	WorkerCount       int `json:"worker_count"`
	ReadyCount        int `json:"ready_count"`
}

type NodeDetailResponse struct {
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	CPUCores          int       `json:"cpu_cores"`
	CPUUsagePercent   float64   `json:"cpu_usage_percent"`
	MemoryBytes       int64     `json:"memory_bytes"`
	MemoryUsagePercent float64  `json:"memory_usage_percent"`
	PodCount          int       `json:"pod_count"`
	Labels            map[string]string `json:"labels"`
	Conditions        []NodeCondition `json:"conditions"`
	Addresses         []NodeAddress   `json:"addresses"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

type NodeCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type NodeAddress struct {
	Type  string `json:"type"`
	Address string `json:"address"`
}

func NewNodeHandler(nodeService *service.NodeService) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
	}
}

func (h *NodeHandler) ListNodes(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	nodeType := c.Query("type")
	status := c.Query("status")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	nodes, total, summary, err := h.nodeService.ListNodes(id.String(), nodeType, status, page, limit)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to list nodes: %v", err)
		return
	}

	nodeSummaries := make([]NodeSummary, 0, len(nodes))
	for _, node := range nodes {
		nodeSummaries = append(nodeSummaries, NodeSummary{
			ID:                node.ID,
			Name:              node.Name,
			Type:              node.Type,
			Status:            node.Status,
			CPUCores:          node.CPUCores,
			CPUUsagePercent:   float64(node.CPUUsedCores) / float64(node.CPUCores) * 100,
			MemoryBytes:       node.MemoryBytes,
			MemoryUsagePercent: float64(node.MemoryUsedBytes) / float64(node.MemoryBytes) * 100,
			PodCount:          node.PodCount,
			Labels:            convertLabels(node.Labels),
			CreatedAt:         node.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:         node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response := NodeListResponse{
		Nodes:   nodeSummaries,
		Total:   total,
		Summary: *summary,
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *NodeHandler) GetNode(c *gin.Context) {
	clusterID := c.Param("id")
	nodeName := c.Param("nodeName")

	clusterUUID, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	node, err := h.nodeService.GetNode(clusterUUID.String(), nodeName)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Node not found")
		return
	}

	response := NodeDetailResponse{
		Name:              node.Name,
		Type:              node.Type,
		Status:            node.Status,
		CPUCores:          node.CPUCores,
		CPUUsagePercent:   float64(node.CPUUsedCores) / float64(node.CPUCores) * 100,
		MemoryBytes:       node.MemoryBytes,
		MemoryUsagePercent: float64(node.MemoryUsedBytes) / float64(node.MemoryBytes) * 100,
		PodCount:          node.PodCount,
		Labels:            convertLabels(node.Labels),
		CreatedAt:         node.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	utils.Success(c, http.StatusOK, response)
}
