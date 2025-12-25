package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// ApplicationHandler 应用处理器
type ApplicationHandler struct {
	applicationService  *service.ApplicationService
	constraintValidator *service.ConstraintValidator
}

// NewApplicationHandler 创建应用处理器
func NewApplicationHandler(
	applicationService *service.ApplicationService,
	constraintValidator *service.ConstraintValidator,
) *ApplicationHandler {
	return &ApplicationHandler{
		applicationService:  applicationService,
		constraintValidator: constraintValidator,
	}
}

// CreateApplication 创建应用
func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	var req service.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	if err := h.constraintValidator.ValidateApplicationCreate(req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %v", err)
		return
	}

	app, err := h.applicationService.CreateApplication(req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "创建应用失败: %v", err)
		return
	}

	utils.Success(c, http.StatusCreated, app)
}

// GetApplication 获取应用详情
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	app, err := h.applicationService.GetApplication(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "应用不存在: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, app)
}

// ListApplications 获取应用列表
func (h *ApplicationHandler) ListApplications(c *gin.Context) {
	page := utils.ParseInt(c.DefaultQuery("page", "1"), 1)
	pageSize := utils.ParseInt(c.DefaultQuery("page_size", "10"), 10)
	tenantID := c.Query("tenant_id")
	environmentID := c.Query("environment_id")

	applications, total, err := h.applicationService.ListApplications(page, pageSize, tenantID, environmentID)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取应用列表失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"items":       applications,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// UpdateApplication 更新应用
func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	var req service.UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	if err := h.constraintValidator.ValidateApplicationUpdate(id, map[string]interface{}{
		"display_name": req.DisplayName,
		"description":  req.Description,
		"labels":       req.Labels,
	}); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %v", err)
		return
	}

	app, err := h.applicationService.UpdateApplication(id, req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "更新应用失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, app)
}

// DeleteApplication 删除应用
func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	if err := h.constraintValidator.ValidateApplicationDelete(id); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "删除验证失败: %v", err)
		return
	}

	if err := h.applicationService.DeleteApplication(id); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "删除应用失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, nil)
}

// GetApplicationResourceSpec 获取应用资源规格
func (h *ApplicationHandler) GetApplicationResourceSpec(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	spec, err := h.applicationService.GetApplicationResourceSpec(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取应用资源规格失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, spec)
}

// UpdateApplicationResourceSpec 更新应用资源规格
func (h *ApplicationHandler) UpdateApplicationResourceSpec(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	var req service.UpdateApplicationResourceSpecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	spec, err := h.applicationService.UpdateApplicationResourceSpec(id, req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "更新应用资源规格失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, spec)
}

// ScaleApplication 扩缩容应用
func (h *ApplicationHandler) ScaleApplication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	var req struct {
		Replicas int `json:"replicas" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	if err := h.constraintValidator.ValidateApplicationScale(id, req.Replicas); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "扩缩容验证失败: %v", err)
		return
	}

	if err := h.applicationService.ScaleApplication(id, req.Replicas); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "扩缩容失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, nil)
}

// DiscoverApplications 发现应用
func (h *ApplicationHandler) DiscoverApplications(c *gin.Context) {
	var req service.DiscoverApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	applications, err := h.applicationService.DiscoverApplications(req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "发现应用失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, applications)
}

// AggregateApplications 聚合应用
func (h *ApplicationHandler) AggregateApplications(c *gin.Context) {
	var req service.AggregateApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	result, err := h.applicationService.AggregateApplications(req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "聚合应用失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, result)
}

// GetApplicationMetrics 获取应用指标
func (h *ApplicationHandler) GetApplicationMetrics(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "应用ID不能为空")
		return
	}

	metrics, err := h.applicationService.GetApplicationMetrics(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取应用指标失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, metrics)
}

// GetApplicationsByTenant 获取租户下的应用
func (h *ApplicationHandler) GetApplicationsByTenant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "租户ID不能为空")
		return
	}

	applications, err := h.applicationService.ListApplicationsByTenantID(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取应用列表失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, applications)
}
