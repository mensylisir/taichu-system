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
	clusterService       *service.ClusterService
	encryptionService    *service.EncryptionService
	worker               *worker.HealthCheckWorker
	createClusterService *service.CreateClusterService
}

type CreateClusterRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Kubeconfig  string            `json:"kubeconfig" binding:"required"`
	Labels      map[string]string `json:"labels"`
}

type CreateClusterByMachinesRequest struct {
	ClusterName   string            `json:"cluster_name" binding:"required"`
	Description   string            `json:"description"`
	MachineIDs    []uuid.UUID       `json:"machine_ids" binding:"required"`
	Kubernetes    KubernetesConfig  `json:"kubernetes"`
	Network       NetworkConfig     `json:"network"`
	ArtifactPath  string            `json:"artifact_path"`
	WithPackages  bool              `json:"with_packages"`
	AutoApprove   bool              `json:"auto_approve"`
	Labels        map[string]string `json:"labels"`
}

type KubernetesConfig struct {
	Version          string `json:"version"`
	ImageRepo        string `json:"image_repo"`
	ContainerManager string `json:"container_manager"`
}

type NetworkConfig struct {
	Plugin       string `json:"plugin"`
	PodsCIDR     string `json:"pods_cidr"`
	ServiceCIDR  string `json:"service_cidr"`
}

type CreateClusterResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
}

type CreateTaskResponse struct {
	ID              uuid.UUID `json:"id"`
	ClusterName     string    `json:"cluster_name"`
	Status          string    `json:"status"`
	Progress        int       `json:"progress"`
	CurrentStep     string    `json:"current_step"`
	CreatedAt       string    `json:"created_at"`
}

type ListClustersResponse struct {
	Clusters []ClusterSummary `json:"clusters"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
}

type ClusterSummary struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	Provider          string    `json:"provider"`
	Region            string    `json:"region"`
	Version           string    `json:"version"`
	KubernetesVersion string    `json:"kubernetes_version"`
	NodeCount         int       `json:"node_count"`
	Labels            map[string]string `json:"labels"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

func NewClusterHandler(
	clusterService *service.ClusterService,
	encryptionService *service.EncryptionService,
	worker *worker.HealthCheckWorker,
	createClusterService *service.CreateClusterService,
) *ClusterHandler {
	return &ClusterHandler{
		clusterService:       clusterService,
		encryptionService:    encryptionService,
		worker:               worker,
		createClusterService: createClusterService,
	}
}

func (h *ClusterHandler) CreateCluster(c *gin.Context) {
	var req CreateClusterRequest

	// 验证请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	// 验证必填字段
	if req.Name == "" {
		utils.Error(c, http.StatusBadRequest, "Cluster name is required")
		return
	}

	if req.Kubeconfig == "" {
		utils.Error(c, http.StatusBadRequest, "Kubeconfig is required")
		return
	}

	// 验证kubeconfig格式
	isValid, err := h.clusterService.ValidateKubeconfig(req.Kubeconfig)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Failed to validate kubeconfig: %v", err)
		return
	}
	if !isValid {
		utils.Error(c, http.StatusBadRequest, "Invalid kubeconfig format")
		return
	}

	// 加密kubeconfig
	encryptedKubeconfig, nonce, err := h.encryptionService.Encrypt(req.Kubeconfig)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to encrypt kubeconfig: %v", err)
		return
	}

	// 转换标签格式
	labels := convertLabelsToJSONMap(req.Labels)

	// 创建集群对象
	cluster := &model.Cluster{
		Name:                req.Name,
		Description:         req.Description,
		KubeconfigEncrypted: encryptedKubeconfig,
		KubeconfigNonce:     nonce,
		Labels:              labels,
		Provider:            "太初",
		Region:              "default",
		CreatedBy:           "api-user",
		EnvironmentType:     "production",
	}

	// 创建集群
	created, err := h.clusterService.Create(cluster)
	if err != nil {
		if err == service.ErrClusterExists {
			utils.Error(c, http.StatusConflict, "Cluster name already exists")
		} else {
			utils.Error(c, http.StatusInternalServerError, "Failed to create cluster: %v", err)
		}
		return
	}

	// 异步触发集群同步
	go h.worker.TriggerSync(created.ID)

	// 构建响应
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
	// 解析查询参数
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		utils.Error(c, http.StatusBadRequest, "Invalid page parameter, must be a positive integer")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid limit parameter, must be an integer")
		return
	}

	// 验证limit范围
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	status := c.Query("status")
	labelSelector := c.Query("label_selector")
	search := c.Query("search")

	// 计算偏移量
	offset := (page - 1) * limit

	// 查询集群列表
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

	// 构建响应
	response := ListClustersResponse{
		Clusters: make([]ClusterSummary, 0, len(clusters)),
		Total:    total,
		Page:     page,
		Limit:    limit,
	}

	// 转换集群数据为摘要格式
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
		ID                  uuid.UUID `json:"id"`
		Name                string    `json:"name"`
		Description         string    `json:"description"`
		Provider            string    `json:"provider"`
		Region              string    `json:"region"`
		Status              string    `json:"status"`
		Version             string    `json:"version"`
		Labels              map[string]string `json:"labels"`
		NodeCount           int       `json:"node_count"`
		TotalCPUCores       int       `json:"total_cpu_cores"`
		TotalMemoryBytes    int64     `json:"total_memory_bytes"`
		TotalStorageBytes   int64     `json:"total_storage_bytes"`
		UsedStorageBytes    int64     `json:"used_storage_bytes"`
		StorageUsagePercent float64   `json:"storage_usage_percent"`
		KubernetesVersion   string    `json:"kubernetes_version"`
		APIServerURL        string    `json:"api_server_url"`
		LastHeartbeatAt     string    `json:"last_heartbeat_at"`
		CreatedAt           string    `json:"created_at"`
		UpdatedAt           string    `json:"updated_at"`
	}{
		ID:                  cluster.Cluster.ID,
		Name:                cluster.Cluster.Name,
		Description:         cluster.Cluster.Description,
		Provider:            cluster.Cluster.Provider,
		Region:              cluster.Cluster.Region,
		Status:              cluster.State.Status,
		Version:             cluster.State.KubernetesVersion,
		Labels:              convertLabels(cluster.Cluster.Labels),
		NodeCount:           cluster.State.NodeCount,
		TotalCPUCores:       cluster.State.TotalCPUCores,
		TotalMemoryBytes:    cluster.State.TotalMemoryBytes,
		TotalStorageBytes:   cluster.State.TotalStorageBytes,
		UsedStorageBytes:    cluster.State.UsedStorageBytes,
		StorageUsagePercent: cluster.State.StorageUsagePercent,
		KubernetesVersion:   cluster.State.KubernetesVersion,
		APIServerURL:        cluster.State.APIServerURL,
		LastHeartbeatAt: func() string {
			if cluster.State.LastHeartbeatAt != nil {
				return cluster.State.LastHeartbeatAt.Format(time.RFC3339)
			}
			return ""
		}(),
		CreatedAt:           cluster.Cluster.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           cluster.Cluster.UpdatedAt.Format(time.RFC3339),
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

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	if err := h.clusterService.Delete(id.String()); err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to delete cluster: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "Cluster deleted"})
}

// CreateClusterByMachines 通过机器列表创建集群
func (h *ClusterHandler) CreateClusterByMachines(c *gin.Context) {
	var req CreateClusterByMachinesRequest

	// 验证请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	// 验证必填字段
	if req.ClusterName == "" {
		utils.Error(c, http.StatusBadRequest, "Cluster name is required")
		return
	}

	if len(req.MachineIDs) == 0 {
		utils.Error(c, http.StatusBadRequest, "At least one machine is required")
		return
	}

	// 构建创建请求
	createReq := service.CreateClusterRequest{
		ClusterName: req.ClusterName,
		MachineIDs:  req.MachineIDs,
		Kubernetes: service.KubernetesConfig{
			Version:          req.Kubernetes.Version,
			ImageRepo:        req.Kubernetes.ImageRepo,
			ContainerManager: req.Kubernetes.ContainerManager,
		},
		Network: service.NetworkConfig{
			Plugin:       req.Network.Plugin,
			PodsCIDR:     req.Network.PodsCIDR,
			ServiceCIDR:  req.Network.ServiceCIDR,
		},
		ArtifactPath:  req.ArtifactPath,
		WithPackages:  req.WithPackages,
		AutoApprove:   req.AutoApprove,
	}

	// 创建集群任务
	task, err := h.createClusterService.CreateCluster(createReq)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to create cluster: %v", err)
		return
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

	id, err := uuid.Parse(taskID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.createClusterService.GetTask(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Task not found")
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
	// 解析查询参数
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		utils.Error(c, http.StatusBadRequest, "Invalid page parameter")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		utils.Error(c, http.StatusBadRequest, "Invalid limit parameter (1-100)")
		return
	}

	tasks, total, err := h.createClusterService.ListTasks(page, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list tasks: %v", err)
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
