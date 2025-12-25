package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// TenantHandler 租户处理器
type TenantHandler struct {
	tenantService *service.TenantService
	constraintValidator *service.ConstraintValidator
}

// NewTenantHandler 创建租户处理器
func NewTenantHandler(
	tenantService *service.TenantService,
	constraintValidator *service.ConstraintValidator,
) *TenantHandler {
	return &TenantHandler{
		tenantService:     tenantService,
		constraintValidator: constraintValidator,
	}
}

// CreateTenant 创建租户
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req service.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %v", err)
		return
	}

	if err := h.constraintValidator.ValidateTenantCreate(req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %v", err)
		return
	}

	tenant, err := h.tenantService.CreateTenant(req)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "创建租户失败: %v", err)
		return
	}

	utils.Success(c, http.StatusCreated, tenant)
}

// GetTenant 获取租户详情
func (h *TenantHandler) GetTenant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "租户ID不能为空",
		})
		return
	}

	tenant, err := h.tenantService.GetTenant(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "租户不存在",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    tenant,
	})
}

// ListTenants 获取租户列表
func (h *TenantHandler) ListTenants(c *gin.Context) {
	// 获取查询参数
	page := utils.ParseInt(c.DefaultQuery("page", "1"), 1)
	pageSize := utils.ParseInt(c.DefaultQuery("page_size", "10"), 10)
	tenantType := c.Query("type")

	// 获取租户列表
	tenants, total, err := h.tenantService.ListTenantsPaginated(page, pageSize, tenantType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取租户列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"message": "获取成功",
		"data": gin.H{
			"items":       tenants,
			"total":       total,
			"page":        int64(page),
			"page_size":   int64(pageSize),
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// UpdateTenant 更新租户
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "租户ID不能为空",
		})
		return
	}

	var req service.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 验证更新请求
	updates := make(map[string]interface{})
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Labels != nil {
		updates["labels"] = req.Labels
	}

	if err := h.constraintValidator.ValidateTenantUpdate(id, updates); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    422,
			"message": "验证失败",
			"error":   err.Error(),
		})
		return
	}

	// 更新租户
	tenant, err := h.tenantService.UpdateTenant(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新租户失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    tenant,
	})
}

// DeleteTenant 删除租户
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "租户ID不能为空",
		})
		return
	}

	// 验证删除请求
	if err := h.constraintValidator.ValidateTenantDelete(id); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    422,
			"message": "删除验证失败",
			"error":   err.Error(),
		})
		return
	}

	// 删除租户
	if err := h.tenantService.DeleteTenant(id); err != nil {
		if err == service.ErrSystemTenantCannotBeDeleted {
			utils.Error(c, utils.ErrCodeForbidden, "系统租户不能删除")
			return
		}
		utils.Error(c, utils.ErrCodeInternalError, "删除租户失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// GetTenantQuota 获取租户配额
func (h *TenantHandler) GetTenantQuota(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "租户ID不能为空")
		return
	}

	quota, err := h.tenantService.GetTenantQuota(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "获取租户配额失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, quota)
}

// UpdateTenantQuota 更新租户配额
func (h *TenantHandler) UpdateTenantQuota(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "租户ID不能为空",
		})
		return
	}

	var req service.UpdateTenantQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	quota, err := h.tenantService.UpdateTenantQuota(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新租户配额失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    quota,
	})
}

// GetTenantEnvironments 获取租户下的环境列表
func (h *TenantHandler) GetTenantEnvironments(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "租户ID不能为空",
		})
		return
	}

	environments, err := h.tenantService.GetTenantEnvironments(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取环境列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    environments,
	})
}

// PredefinedTenants 获取预定义租户列表
func (h *TenantHandler) PredefinedTenants(c *gin.Context) {
	tenants, err := h.tenantService.GetPredefinedTenants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取预定义租户失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    tenants,
	})
}

// InitializePredefinedTenants 初始化预定义租户
func (h *TenantHandler) InitializePredefinedTenants(c *gin.Context) {
	if err := h.tenantService.InitializePredefinedTenants(); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "初始化预定义租户失败: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message": "初始化成功",
	})
}

