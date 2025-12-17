package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/internal/service/worker"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type ClusterHandler struct {
	clusterService    *service.ClusterService
	encryptionService *service.EncryptionService
	worker           *worker.HealthCheckWorker
}

type CreateClusterRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Kubeconfig  string            `json:"kubeconfig" binding:"required"`
	Labels      map[string]string `json:"labels"`
	Version     string            `json:"version" binding:"max=50"`
}

type CreateClusterResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
}

type ListClustersResponse struct {
	Clusters []ClusterSummary `json:"clusters"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
}

type ClusterSummary struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	NodeCount   int       `json:"node_count"`
	Version     string    `json:"version"`
	Labels      map[string]string `json:"labels"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func NewClusterHandler(
	clusterService *service.ClusterService,
	encryptionService *service.EncryptionService,
	worker *worker.HealthCheckWorker,
) *ClusterHandler {
	return &ClusterHandler{
		clusterService:    clusterService,
		encryptionService: encryptionService,
		worker:           worker,
	}
}

func (h *ClusterHandler) CreateCluster(c *gin.Context) {
	var req CreateClusterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	isValid, err := h.clusterService.ValidateKubeconfig(req.Kubeconfig)
	if err != nil || !isValid {
		utils.Error(c, http.StatusBadRequest, "Invalid kubeconfig: %v", err)
		return
	}

	encryptedKubeconfig, nonce, err := h.encryptionService.Encrypt(req.Kubeconfig)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to encrypt kubeconfig: %v", err)
		return
	}

	cluster := &model.Cluster{
		Name:                req.Name,
		Description:         req.Description,
		KubeconfigEncrypted: encryptedKubeconfig,
		KubeconfigNonce:     nonce,
		Labels:              model.JSONMap(req.Labels),
		Version:             req.Version,
		CreatedBy:           "api-user",
	}

	created, err := h.clusterService.Create(cluster)
	if err != nil {
		if err == service.ErrClusterExists {
			utils.Error(c, http.StatusConflict, "Cluster name already exists")
		} else {
			utils.Error(c, http.StatusInternalServerError, "Failed to create cluster: %v", err)
		}
		return
	}

	go h.worker.TriggerSync(created.ID)

	response := CreateClusterResponse{
		ID:          created.ID,
		Name:        created.Name,
		Description: created.Description,
		Status:      "unknown",
		CreatedAt:   created.CreatedAt.Format(time.RFC3339),
	}

	utils.Success(c, http.StatusCreated, response)
}

func (h *ClusterHandler) ListClusters(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	labelSelector := c.Query("label_selector")
	search := c.Query("search")

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	clusters, total, err := h.clusterService.ListClusters(service.ListClustersParams{
		Page:          page,
		Limit:         limit,
		Offset:        offset,
		Status:        status,
		LabelSelector: labelSelector,
		Search:        search,
	})

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list clusters: %v", err)
		return
	}

	response := ListClustersResponse{
		Clusters: make([]ClusterSummary, 0, len(clusters)),
		Total:    total,
		Page:     page,
		Limit:    limit,
	}

	for _, cluster := range clusters {
		summary := ClusterSummary{
			ID:          cluster.Cluster.ID,
			Name:        cluster.Cluster.Name,
			Description: cluster.Cluster.Description,
			Status:      cluster.State.Status,
			NodeCount:   cluster.State.NodeCount,
			Version:     cluster.Cluster.Version,
			Labels:      convertLabels(cluster.Cluster.Labels),
			CreatedAt:   cluster.Cluster.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   cluster.Cluster.UpdatedAt.Format(time.RFC3339),
		}
		response.Clusters = append(response.Clusters, summary)
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *ClusterHandler) GetCluster(c *gin.Context) {
	clusterID := c.Param("cluster_id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	cluster, err := h.clusterService.GetClusterWithState(id.String())
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Cluster not found")
		return
	}

	response := struct {
		ID                uuid.UUID `json:"id"`
		Name              string    `json:"name"`
		Description       string    `json:"description"`
		Status            string    `json:"status"`
		Version           string    `json:"version"`
		Labels            map[string]string `json:"labels"`
		NodeCount         int       `json:"node_count"`
		TotalCPUCores     int       `json:"total_cpu_cores"`
		TotalMemoryBytes  int64     `json:"total_memory_bytes"`
		KubernetesVersion string    `json:"kubernetes_version"`
		APIServerURL      string    `json:"api_server_url"`
		LastHeartbeatAt   string    `json:"last_heartbeat_at"`
		CreatedAt         string    `json:"created_at"`
		UpdatedAt         string    `json:"updated_at"`
	}{
		ID:                cluster.Cluster.ID,
		Name:              cluster.Cluster.Name,
		Description:       cluster.Cluster.Description,
		Status:            cluster.State.Status,
		Version:           cluster.Cluster.Version,
		Labels:            convertLabels(cluster.Cluster.Labels),
		NodeCount:         cluster.State.NodeCount,
		TotalCPUCores:     cluster.State.TotalCPUCores,
		TotalMemoryBytes:  cluster.State.TotalMemoryBytes,
		KubernetesVersion: cluster.State.KubernetesVersion,
		APIServerURL:      cluster.State.APIServerURL,
		LastHeartbeatAt:   cluster.State.LastHeartbeatAt.Format(time.RFC3339),
		CreatedAt:         cluster.Cluster.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         cluster.Cluster.UpdatedAt.Format(time.RFC3339),
	}

	utils.Success(c, http.StatusOK, response)
}

func convertLabels(labels model.JSONMap) map[string]string {
	result := make(map[string]string)
	for k, v := range labels {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}
