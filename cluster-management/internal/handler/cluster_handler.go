package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/internal/service/worker"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type ClusterHandler struct {
	clusterService       *service.ClusterService
	encryptionService    *service.EncryptionService
	worker               *worker.HealthCheckWorker
	createClusterService *service.CreateClusterService
	nodeService          *service.NodeService
	auditService         *service.AuditService
}

type CreateClusterRequest struct {
	Name        string            `json:"name" binding:"required,min=1,max=63"`
	Description string            `json:"description" binding:"max=500"`
	Kubeconfig  string            `json:"kubeconfig" binding:"required"`
	Labels      map[string]string `json:"labels"`
}

type CreateClusterByMachinesRequest struct {
	ClusterName  string            `json:"cluster_name" binding:"required,min=1,max=63"`
	Description  string            `json:"description" binding:"max=500"`
	MachineIDs   []uuid.UUID       `json:"machine_ids" binding:"required,min=1"`
	Kubernetes   KubernetesConfig  `json:"kubernetes"`
	Network      NetworkConfig     `json:"network"`
	ArtifactPath string            `json:"artifact_path"`
	WithPackages bool              `json:"with_packages"`
	AutoApprove  bool              `json:"auto_approve"`
	Labels       map[string]string `json:"labels"`
}

type KubernetesConfig struct {
	Version          string `json:"version" binding:"required,semver"`
	ImageRepo        string `json:"image_repo" binding:"required"`
	ContainerManager string `json:"container_manager" binding:"required,oneof=docker containerd"`
}

type NetworkConfig struct {
	Plugin      string `json:"plugin" binding:"required,oneof=flannel calico cilium weave"`
	PodsCIDR    string `json:"pods_cidr" binding:"required,cidr"`
	ServiceCIDR string `json:"service_cidr" binding:"required,cidr"`
}

type CreateClusterResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
}

type CreateTaskResponse struct {
	ID          uuid.UUID `json:"id"`
	ClusterName string    `json:"cluster_name"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	CurrentStep string    `json:"current_step"`
	CreatedAt   string    `json:"created_at"`
}

type ListClustersResponse struct {
	Clusters []ClusterSummary `json:"clusters"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
}

type ClusterSummary struct {
	ID                uuid.UUID         `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Status            string            `json:"status"`
	Provider          string            `json:"provider"`
	Region            string            `json:"region"`
	Version           string            `json:"version"`
	KubernetesVersion string            `json:"kubernetes_version"`
	NodeCount         int               `json:"node_count"`
	Labels            map[string]string `json:"labels"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

func NewClusterHandler(
	clusterService *service.ClusterService,
	encryptionService *service.EncryptionService,
	worker *worker.HealthCheckWorker,
	createClusterService *service.CreateClusterService,
	nodeService *service.NodeService,
	auditService *service.AuditService,
) *ClusterHandler {
	return &ClusterHandler{
		clusterService:       clusterService,
		encryptionService:    encryptionService,
		worker:               worker,
		createClusterService: createClusterService,
		nodeService:          nodeService,
		auditService:         auditService,
	}
}

func (h *ClusterHandler) CreateCluster(c *gin.Context) {
	var req CreateClusterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	if req.Name == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "Cluster name is required")
		return
	}

	if req.Kubeconfig == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "Kubeconfig is required")
		return
	}

	isValid, err := h.clusterService.ValidateKubeconfig(req.Kubeconfig)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to validate kubeconfig: %v", err)
		return
	}
	if !isValid {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid kubeconfig format")
		return
	}

	encryptedKubeconfig, err := h.encryptionService.Encrypt(req.Kubeconfig)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to encrypt kubeconfig: %v", err)
		return
	}

	labels := convertLabelsToJSONMap(req.Labels)

	cluster := &model.Cluster{
		Name:                req.Name,
		Description:         req.Description,
		KubeconfigEncrypted: encryptedKubeconfig,
		Labels:              labels,
		Provider:            "太初",
		Region:              "default",
		CreatedBy:           "api-user",
		EnvironmentType:     "production",
	}

	created, err := h.clusterService.Create(cluster)
	if err != nil {
		if err == service.ErrClusterExists {
			utils.Error(c, utils.ErrCodeAlreadyExists, "Cluster name already exists")
		} else {
			utils.Error(c, utils.ErrCodeInternalError, "Failed to create cluster: %v", err)
		}
		return
	}

	go h.worker.TriggerSync(created.ID)

	if h.auditService != nil {
		user := "api-user"
		h.auditService.LogClusterOperation(
			created.ID,
			constants.EventTypeCreate,
			"cluster",
			user,
			map[string]interface{}{
				"name":        created.Name,
				"description": created.Description,
				"provider":    created.Provider,
				"region":      created.Region,
			},
		)
	}

	response := CreateClusterResponse{
		ID:          created.ID,
		Name:        created.Name,
		Description: created.Description,
		Status:      "unknown",
		CreatedAt:   created.CreatedAt.Format(time.RFC3339),
	}

	utils.Success(c, http.StatusCreated, response)
}

// 辅助函数：转换标签格式
func convertLabelsToJSONMap(labels map[string]string) model.JSONMap {
	jsonLabels := make(model.JSONMap)
	for k, v := range labels {
		jsonLabels[k] = v
	}
	return jsonLabels
}

func (h *ClusterHandler) ListClusters(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid page parameter, must be a positive integer")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid limit parameter, must be an integer")
		return
	}

	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	status := c.Query("status")
	labelSelector := c.Query("label_selector")
	search := c.Query("search")

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
		utils.Error(c, utils.ErrCodeInternalError, "Failed to list clusters: %v", err)
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
			Provider:    cluster.Cluster.Provider,
			Region:      cluster.Cluster.Region,
			Labels:      convertLabels(cluster.Cluster.Labels),
			CreatedAt:   cluster.Cluster.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   cluster.Cluster.UpdatedAt.Format(time.RFC3339),
		}
		if cluster.State != nil {
			summary.Status = cluster.State.Status
			summary.Version = cluster.State.KubernetesVersion
			summary.KubernetesVersion = cluster.State.KubernetesVersion
			summary.NodeCount = cluster.State.NodeCount
		} else {
			summary.Status = "unknown"
			summary.Version = ""
			summary.KubernetesVersion = ""
			summary.NodeCount = 0
		}
		response.Clusters = append(response.Clusters, summary)
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *ClusterHandler) GetCluster(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	cluster, err := h.clusterService.GetClusterWithState(id.String())
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Cluster not found")
		return
	}

	var resource *model.ClusterResource
	if clusterResource, err := h.clusterService.GetClusterResourceRepo().GetLatestByClusterID(id.String()); err == nil {
		resource = clusterResource
	}

	nodesWithDetails, _, _, err := h.nodeService.ListNodes(id.String(), "", "", 1, 1000)
	if err != nil {
		log.Printf("Failed to get nodes for cluster %s: %v", id.String(), err)
		nodesWithDetails = nil
	}

	response := struct {
		ID                  uuid.UUID                  `json:"id"`
		Name                string                     `json:"name"`
		Description         string                     `json:"description"`
		Provider            string                     `json:"provider"`
		Region              string                     `json:"region"`
		Status              string                     `json:"status"`
		Version             string                     `json:"version"`
		Labels              map[string]string          `json:"labels"`
		NodeCount           int                        `json:"node_count"`
		Nodes               []*service.NodeWithDetails `json:"nodes,omitempty"`
		TotalCPUCores       int                        `json:"total_cpu_cores"`
		UsedCPUCores        float64                    `json:"used_cpu_cores"`
		CPUUsagePercent     float64                    `json:"cpu_usage_percent"`
		TotalMemoryBytes    int64                      `json:"total_memory_bytes"`
		UsedMemoryBytes     int64                      `json:"used_memory_bytes"`
		MemoryUsagePercent  float64                    `json:"memory_usage_percent"`
		TotalStorageBytes   int64                      `json:"total_storage_bytes"`
		UsedStorageBytes    int64                      `json:"used_storage_bytes"`
		StorageUsagePercent float64                    `json:"storage_usage_percent"`
		KubernetesVersion   string                     `json:"kubernetes_version"`
		APIServerURL        string                     `json:"api_server_url"`
		LastHeartbeatAt     string                     `json:"last_heartbeat_at"`
		CreatedAt           string                     `json:"created_at"`
		UpdatedAt           string                     `json:"updated_at"`
	}{
		ID:          cluster.Cluster.ID,
		Name:        cluster.Cluster.Name,
		Description: cluster.Cluster.Description,
		Provider:    cluster.Cluster.Provider,
		Region:      cluster.Cluster.Region,
		Status:      cluster.State.Status,
		Version:     cluster.State.KubernetesVersion,
		Labels:      convertLabels(cluster.Cluster.Labels),
		NodeCount:   cluster.State.NodeCount,
		Nodes:       nodesWithDetails,
		TotalCPUCores: func() int {
			if resource != nil {
				return resource.TotalCPUCores
			}
			return 0
		}(),
		UsedCPUCores: func() float64 {
			if resource != nil {
				return resource.UsedCPUCores
			}
			return 0
		}(),
		CPUUsagePercent: func() float64 {
			if resource != nil {
				return resource.CPUUsagePercent
			}
			return 0
		}(),
		TotalMemoryBytes: func() int64 {
			if resource != nil {
				return resource.TotalMemoryBytes
			}
			return 0
		}(),
		UsedMemoryBytes: func() int64 {
			if resource != nil {
				return resource.UsedMemoryBytes
			}
			return 0
		}(),
		MemoryUsagePercent: func() float64 {
			if resource != nil {
				return resource.MemoryUsagePercent
			}
			return 0
		}(),
		TotalStorageBytes: func() int64 {
			if resource != nil {
				return resource.TotalStorageBytes
			}
			return 0
		}(),
		UsedStorageBytes: func() int64 {
			if resource != nil {
				return resource.UsedStorageBytes
			}
			return 0
		}(),
		StorageUsagePercent: func() float64 {
			if resource != nil {
				return resource.StorageUsagePercent
			}
			return 0
		}(),
		KubernetesVersion: cluster.State.KubernetesVersion,
		APIServerURL:      cluster.State.APIServerURL,
		LastHeartbeatAt: func() string {
			if cluster.State.LastHeartbeatAt != nil {
				return cluster.State.LastHeartbeatAt.Format(time.RFC3339)
			}
			return ""
		}(),
		CreatedAt: cluster.Cluster.CreatedAt.Format(time.RFC3339),
		UpdatedAt: cluster.Cluster.UpdatedAt.Format(time.RFC3339),
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

func (h *ClusterHandler) DeleteCluster(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	cluster, err := h.clusterService.GetClusterWithState(id.String())
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Cluster not found")
		return
	}

	if err := h.clusterService.Delete(id.String()); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to delete cluster: %v", err)
		return
	}

	if h.auditService != nil {
		user := "api-user"
		h.auditService.LogClusterOperation(
			id,
			constants.EventTypeDelete,
			"cluster",
			user,
			map[string]interface{}{
				"name":        cluster.Cluster.Name,
				"description": cluster.Cluster.Description,
				"status":      cluster.State.Status,
			},
		)
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "Cluster deleted"})
}

// CreateClusterByMachines 通过机器列表创建集群
func (h *ClusterHandler) CreateClusterByMachines(c *gin.Context) {
	var req CreateClusterByMachinesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	if req.ClusterName == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "Cluster name is required")
		return
	}

	if len(req.MachineIDs) == 0 {
		utils.Error(c, utils.ErrCodeValidationFailed, "At least one machine is required")
		return
	}

	createReq := service.CreateClusterRequest{
		ClusterName: req.ClusterName,
		MachineIDs:  req.MachineIDs,
		Kubernetes: service.KubernetesConfig{
			Version:          req.Kubernetes.Version,
			ImageRepo:        req.Kubernetes.ImageRepo,
			ContainerManager: req.Kubernetes.ContainerManager,
		},
		Network: service.NetworkConfig{
			Plugin:      req.Network.Plugin,
			PodsCIDR:    req.Network.PodsCIDR,
			ServiceCIDR: req.Network.ServiceCIDR,
		},
		ArtifactPath: req.ArtifactPath,
		WithPackages: req.WithPackages,
		AutoApprove:  req.AutoApprove,
	}

	task, err := h.createClusterService.CreateCluster(createReq)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to create cluster: %v", err)
		return
	}

	if h.auditService != nil {
		user := "api-user"
		h.auditService.LogClusterOperation(
			uuid.Nil,
			"create_by_machines",
			"cluster",
			user,
			map[string]interface{}{
				"cluster_name":  req.ClusterName,
				"machine_count": len(req.MachineIDs),
				"description":   req.Description,
				"task_id":       task.ID,
			},
		)
	}

	response := CreateTaskResponse{
		ID:          task.ID,
		ClusterName: task.ClusterName,
		Status:      task.Status,
		Progress:    task.Progress,
		CurrentStep: task.CurrentStep,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
	}

	utils.Success(c, http.StatusCreated, response)
}

// GetCreateTask 获取创建任务详情
func (h *ClusterHandler) GetCreateTask(c *gin.Context) {
	taskID := c.Param("taskId")

	id, err := utils.ParseUUID(taskID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid task ID")
		return
	}

	task, err := h.createClusterService.GetTask(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Task not found")
		return
	}

	response := struct {
		ID                uuid.UUID `json:"id"`
		ClusterName       string    `json:"cluster_name"`
		Status            string    `json:"status"`
		Progress          int       `json:"progress"`
		CurrentStep       string    `json:"current_step"`
		Logs              string    `json:"logs"`
		ErrorMsg          string    `json:"error_msg"`
		KubernetesVersion string    `json:"kubernetes_version"`
		NetworkPlugin     string    `json:"network_plugin"`
		StartedAt         string    `json:"started_at"`
		CompletedAt       string    `json:"completed_at"`
		CreatedAt         string    `json:"created_at"`
		UpdatedAt         string    `json:"updated_at"`
	}{
		ID:                task.ID,
		ClusterName:       task.ClusterName,
		Status:            task.Status,
		Progress:          task.Progress,
		CurrentStep:       task.CurrentStep,
		Logs:              task.Logs,
		ErrorMsg:          task.ErrorMsg,
		KubernetesVersion: task.KubernetesVersion,
		NetworkPlugin:     task.NetworkPlugin,
		StartedAt:         getTimeString(task.StartedAt),
		CompletedAt:       getTimeString(task.CompletedAt),
		CreatedAt:         task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         task.UpdatedAt.Format(time.RFC3339),
	}

	utils.Success(c, http.StatusOK, response)
}

// ListCreateTasks 获取创建任务列表
func (h *ClusterHandler) ListCreateTasks(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid page parameter")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid limit parameter (1-100)")
		return
	}

	tasks, total, err := h.createClusterService.ListTasks(page, limit)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to list tasks: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// getTimeString 转换时间指针为字符串
func getTimeString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
