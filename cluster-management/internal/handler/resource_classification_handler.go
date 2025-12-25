package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/internal/service/worker"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// ResourceClassificationHandler 资源分类处理器
type ResourceClassificationHandler struct {
	clusterRepo               *repository.ClusterRepository
	tenantRepo                *repository.TenantRepository
	environmentRepo           *repository.EnvironmentRepository
	applicationRepo           *repository.ApplicationRepository
	quotaRepo                 *repository.QuotaRepository
	resourceClassificationWorker *worker.ResourceClassificationWorker
	quotaService              *service.QuotaService
	tenantQuotaController     *service.TenantQuotaController
}

// NewResourceClassificationHandler 创建资源分类处理器
func NewResourceClassificationHandler(
	clusterRepo *repository.ClusterRepository,
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	resourceClassificationWorker *worker.ResourceClassificationWorker,
	quotaService *service.QuotaService,
	tenantQuotaController *service.TenantQuotaController,
) *ResourceClassificationHandler {
	return &ResourceClassificationHandler{
		clusterRepo:               clusterRepo,
		tenantRepo:                tenantRepo,
		environmentRepo:           environmentRepo,
		applicationRepo:           applicationRepo,
		quotaRepo:                 quotaRepo,
		resourceClassificationWorker: resourceClassificationWorker,
		quotaService:              quotaService,
		tenantQuotaController:     tenantQuotaController,
	}
}

// TriggerClassificationRequest 触发分类请求
type TriggerClassificationRequest struct {
	ClusterID string `json:"cluster_id" binding:"required"`
}

// TriggerClassification 手动触发资源分类
// @Summary 手动触发资源分类
// @Description 对指定集群的未分类资源进行自动分类
// @Tags 资源分类
// @Accept json
// @Produce json
// @Param request body TriggerClassificationRequest true "请求参数"
// @Success 200 {object} Response "成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 500 {object} Response "内部错误"
// @Router /api/v1/resource-classification/trigger [post]
func (h *ResourceClassificationHandler) TriggerClassification(c *gin.Context) {
	var req TriggerClassificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %s", err.Error())
		return
	}

	cluster, err := h.clusterRepo.GetByID(req.ClusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "集群不存在: %s", err.Error())
		return
	}

	if err := h.resourceClassificationWorker.TriggerClassification(req.ClusterID); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "触发分类失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"cluster_id":   cluster.ID.String(),
		"cluster_name": cluster.Name,
		"status":       "triggered",
	})
}

// GetClassificationStatus 获取分类状态
// @Summary 获取资源分类状态
// @Description 获取指定集群的资源分类统计信息
// @Tags 资源分类
// @Produce json
// @Param cluster_id query string true "集群ID"
// @Success 200 {object} Response "成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 500 {object} Response "内部错误"
// @Router /api/v1/resource-classification/status [get]
func (h *ResourceClassificationHandler) GetClassificationStatus(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		utils.Error(c, utils.ErrCodeInvalidInput, "集群ID不能为空")
		return
	}

	_, err := h.clusterRepo.GetByID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "集群不存在: %s", err.Error())
		return
	}

	tenants, err := h.tenantRepo.List()
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取租户列表失败: %s", err.Error())
		return
	}

	envs, err := h.environmentRepo.ListByClusterID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取环境列表失败: %s", err.Error())
		return
	}

	apps, err := h.applicationRepo.List()
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取应用列表失败: %s", err.Error())
		return
	}

	tenantStats := make(map[string]interface{})
	for _, tenant := range tenants {
		envCount := 0
		appCount := 0
		for _, env := range envs {
			if env.TenantID == tenant.ID {
				envCount++
				for _, app := range apps {
					if app.EnvironmentID == env.ID {
						appCount++
					}
				}
			}
		}
		tenantStats[tenant.Name] = gin.H{
			"tenant_id":       tenant.ID.String(),
			"tenant_name":     tenant.Name,
			"tenant_type":     tenant.Type,
			"environment_count": envCount,
			"application_count": appCount,
		}
	}

	quotaStats := make(map[string]interface{})
	for _, env := range envs {
		quota, err := h.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if err == nil {
			quotaStats[env.Namespace] = gin.H{
				"environment_id": env.ID.String(),
				"namespace":      env.Namespace,
				"has_quota":      true,
				"quota_status":   quota.Status,
			}
		} else {
			quotaStats[env.Namespace] = gin.H{
				"environment_id": env.ID.String(),
				"namespace":      env.Namespace,
				"has_quota":      false,
			}
		}
	}

	utils.Success(c, http.StatusOK, gin.H{
		"cluster_id":        clusterID,
		"total_tenants":     len(tenants),
		"total_environments": len(envs),
		"total_applications": len(apps),
		"tenant_stats":      tenantStats,
		"quota_stats":       quotaStats,
	})
}

// GetTenantQuotaStatus 获取租户配额状态
// @Summary 获取租户配额状态
// @Description 获取指定租户的配额使用情况
// @Tags 配额管理
// @Produce json
// @Param tenant_id query string true "租户ID"
// @Success 200 {object} Response "成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 500 {object} Response "内部错误"
// @Router /api/v1/quota/tenant-status [get]
func (h *ResourceClassificationHandler) GetTenantQuotaStatus(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		utils.Error(c, utils.ErrCodeInvalidInput, "租户ID不能为空")
		return
	}

	status, err := h.tenantQuotaController.GetTenantQuotaStatus(tenantID)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取租户配额状态失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, status)
}

// ValidateEnvironmentQuota 验证环境配额
// @Summary 验证环境配额设置
// @Description 验证为环境设置的配额是否超过租户限制
// @Tags 配额管理
// @Accept json
// @Produce json
// @Param request body ValidateEnvironmentQuotaRequest true "请求参数"
// @Success 200 {object} Response "成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 500 {object} Response "内部错误"
// @Router /api/v1/quota/validate [post]
func (h *ResourceClassificationHandler) ValidateEnvironmentQuota(c *gin.Context) {
	var req ValidateEnvironmentQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %s", err.Error())
		return
	}

	result, err := h.tenantQuotaController.ValidateEnvironmentQuota(req.TenantID, req.RequestedQuota)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "验证配额失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, result)
}

// ValidateEnvironmentQuotaRequest 验证环境配额请求
type ValidateEnvironmentQuotaRequest struct {
	TenantID       string         `json:"tenant_id" binding:"required"`
	RequestedQuota map[string]interface{} `json:"requested_quota" binding:"required"`
}

// SyncResourceQuota 同步资源配额
// @Summary 同步资源配额
// @Description 手动同步指定环境的ResourceQuota
// @Tags 配额管理
// @Accept json
// @Produce json
// @Param request body SyncResourceQuotaRequest true "请求参数"
// @Success 200 {object} Response "成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 500 {object} Response "内部错误"
// @Router /api/v1/quota/sync [post]
func (h *ResourceClassificationHandler) SyncResourceQuota(c *gin.Context) {
	var req SyncResourceQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %s", err.Error())
		return
	}

	if err := h.quotaService.SyncResourceQuota(req.EnvironmentID); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "同步配额失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"environment_id": req.EnvironmentID,
		"status":         "synced",
	})
}

// SyncResourceQuotaRequest 同步资源配额请求
type SyncResourceQuotaRequest struct {
	EnvironmentID string `json:"environment_id" binding:"required"`
}
