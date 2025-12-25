package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// EnvironmentHandler 环境处理器
type EnvironmentHandler struct {
	environmentService  *service.EnvironmentService
	constraintValidator *service.ConstraintValidator
	quotaInheritance    *service.QuotaInheritance
}

// NewEnvironmentHandler 创建环境处理器
func NewEnvironmentHandler(
	environmentService *service.EnvironmentService,
	constraintValidator *service.ConstraintValidator,
	quotaInheritance *service.QuotaInheritance,
) *EnvironmentHandler {
	return &EnvironmentHandler{
		environmentService:  environmentService,
		constraintValidator: constraintValidator,
		quotaInheritance:    quotaInheritance,
	}
}

// CreateEnvironment 创建环境
func (h *EnvironmentHandler) CreateEnvironment(c *gin.Context) {
	var req service.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	if err := h.constraintValidator.ValidateEnvironmentCreate(req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %v", err)
		return
	}

	env, err := h.environmentService.CreateEnvironment(req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "创建环境失败: %v", err)
		return
	}

	utils.Success(c, http.StatusCreated, env)
}

// GetEnvironment 获取环境详情
func (h *EnvironmentHandler) GetEnvironment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	env, err := h.environmentService.GetEnvironment(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "环境不存在: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, env)
}

// ListEnvironments 获取环境列表
func (h *EnvironmentHandler) ListEnvironments(c *gin.Context) {
	page := utils.ParseInt(c.DefaultQuery("page", "1"), 1)
	pageSize := utils.ParseInt(c.DefaultQuery("page_size", "10"), 10)
	tenantID := c.Query("tenant_id")
	clusterID := c.Query("cluster_id")

	environments, total, err := h.environmentService.ListEnvironments(page, pageSize, tenantID, clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取环境列表失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"items":       environments,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// UpdateEnvironment 更新环境
func (h *EnvironmentHandler) UpdateEnvironment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	var req service.UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	if err := h.constraintValidator.ValidateEnvironmentUpdate(id, map[string]interface{}{
		"display_name": req.DisplayName,
		"description":   req.Description,
		"labels":       req.Labels,
	}); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %v", err)
		return
	}

	env, err := h.environmentService.UpdateEnvironment(id, req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "更新环境失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, env)
}

// DeleteEnvironment 删除环境
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	if err := h.constraintValidator.ValidateEnvironmentDelete(id); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "删除验证失败: %v", err)
		return
	}

	if err := h.environmentService.DeleteEnvironment(id); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "删除环境失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, nil)
}

// GetEnvironmentQuota 获取环境配额
func (h *EnvironmentHandler) GetEnvironmentQuota(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	quota, err := h.environmentService.GetEnvironmentQuota(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取环境配额失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, quota)
}

// UpdateEnvironmentQuota 更新环境配额
func (h *EnvironmentHandler) UpdateEnvironmentQuota(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	var req service.UpdateEnvironmentQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	quota, err := h.environmentService.UpdateEnvironmentQuota(id, req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "更新环境配额失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, quota)
}

// InheritQuota 继承配额
func (h *EnvironmentHandler) InheritQuota(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	inheritedQuota, err := h.quotaInheritance.InheritQuotaFromTenant(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "继承配额失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, inheritedQuota)
}

// GetEnvironmentApplications 获取环境下的应用列表
func (h *EnvironmentHandler) GetEnvironmentApplications(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	applications, err := h.environmentService.GetEnvironmentApplications(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取应用列表失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, applications)
}

// GetNamespaceInfo 获取命名空间信息
func (h *EnvironmentHandler) GetNamespaceInfo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	info, err := h.environmentService.GetNamespaceInfo(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取命名空间信息失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, info)
}

// SyncResourceQuota 同步资源配额
func (h *EnvironmentHandler) SyncResourceQuota(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "环境ID不能为空")
		return
	}

	if _, err := h.environmentService.SyncResourceQuota(id); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "同步配额失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, nil)
}

// parseInt 解析整数
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var result int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		}
	}
	return result
}
