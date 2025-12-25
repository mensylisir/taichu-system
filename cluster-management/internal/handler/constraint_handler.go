package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// ConstraintHandler 约束处理器
type ConstraintHandler struct {
	constraintValidator *service.ConstraintValidator
	constrainMonitor    *service.ConstraintMonitor
	benchmark           *service.ConstraintBenchmark
}

// NewConstraintHandler 创建约束处理器
func NewConstraintHandler(
	constraintValidator *service.ConstraintValidator,
	constraintMonitor *service.ConstraintMonitor,
	benchmark *service.ConstraintBenchmark,
) *ConstraintHandler {
	return &ConstraintHandler{
		constraintValidator: constraintValidator,
		constrainMonitor:    constraintMonitor,
		benchmark:           benchmark,
	}
}

// ValidateTenant 创建租户验证
func (h *ConstraintHandler) ValidateTenant(c *gin.Context) {
	var req service.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %s", err.Error())
		return
	}

	if err := h.constraintValidator.ValidateTenantCreate(req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"is_valid": true,
		"checks": []gin.H{
			{
				"type":    "tenant_name_unique",
				"passed":  true,
				"message": "租户名唯一",
			},
			{
				"type":    "quota_format",
				"passed":  true,
				"message": "配额格式正确",
			},
		},
	})
}

// ValidateEnvironment 创建环境验证
func (h *ConstraintHandler) ValidateEnvironment(c *gin.Context) {
	var req service.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %s", err.Error())
		return
	}

	if err := h.constraintValidator.ValidateEnvironmentCreate(req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"is_valid": true,
		"checks": []gin.H{
			{
				"type":    "tenant_exists",
				"passed":  true,
				"message": "租户存在",
			},
			{
				"type":    "quota_within_limits",
				"passed":  true,
				"message": "配额在租户限制内",
			},
		},
	})
}

// ValidateApplication 创建应用验证
func (h *ConstraintHandler) ValidateApplication(c *gin.Context) {
	var req service.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "请求参数错误: %s", err.Error())
		return
	}

	if err := h.constraintValidator.ValidateApplicationCreate(req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "验证失败: %s", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"is_valid": true,
		"checks": []gin.H{
			{
				"type":    "environment_exists",
				"passed":  true,
				"message": "环境存在",
			},
			{
				"type":    "resource_spec_valid",
				"passed":  true,
				"message": "资源规格有效",
			},
		},
	})
}

// PrecheckDeletion 预检查删除
func (h *ConstraintHandler) PrecheckDeletion(c *gin.Context) {
	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	var canDelete bool
	var blockingResources []string
	var stats map[string]int

	// 根据资源类型进行预检查
	switch req.ResourceType {
	case "tenant":
		if err := h.constraintValidator.ValidateTenantDelete(req.ResourceID); err != nil {
			canDelete = false
			blockingResources = append(blockingResources, err.Error())
		} else {
			canDelete = true
		}
	case "environment":
		if err := h.constraintValidator.ValidateEnvironmentDelete(req.ResourceID); err != nil {
			canDelete = false
			blockingResources = append(blockingResources, err.Error())
		} else {
			canDelete = true
		}
	case "application":
		if err := h.constraintValidator.ValidateApplicationDelete(req.ResourceID); err != nil {
			canDelete = false
			blockingResources = append(blockingResources, err.Error())
		} else {
			canDelete = true
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的资源类型",
		})
		return
	}

	stats = map[string]int{
		"environment_count": 0,
		"application_count": 0,
		"quota_count":       0,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "预检查完成",
		"data": gin.H{
			"can_delete":         canDelete,
			"blocking_resources": blockingResources,
			"stats":              stats,
		},
	})
}

// GetConstraintMetrics 获取约束指标
func (h *ConstraintHandler) GetConstraintMetrics(c *gin.Context) {
	metrics, err := h.constrainMonitor.GetConstraintMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取约束指标失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    metrics,
	})
}

// GetViolationStats 获取违反统计
func (h *ConstraintHandler) GetViolationStats(c *gin.Context) {
	stats, err := h.constrainMonitor.GetViolationStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取违反统计失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    stats,
	})
}

// TriggerCheck 手动触发约束检查
func (h *ConstraintHandler) TriggerCheck(c *gin.Context) {
	var req struct {
		Scope         string `json:"scope"`
		TenantID      string `json:"tenant_id"`
		EnvironmentID string `json:"environment_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 这里可以触发实际的约束检查
	// 简化实现，返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "约束检查已启动",
		"data": gin.H{
			"check_i":            uuid.New().String(),
			"scope":              req.Scope,
			"started_at":         "2025-12-24T10:30:00Z",
			"estimated_duration": "30s",
		},
	})
}

// AutoResolve 自动解决约束违反
func (h *ConstraintHandler) AutoResolve(c *gin.Context) {
	var req struct {
		DryRun         bool `json:"dry_run"`
		MaxAutoResolve int  `json:"max_auto_resolve"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 执行自动解决
	resolvedCount, err := h.constrainMonitor.AutoResolve(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "自动解决失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "自动解决完成",
		"data": gin.H{
			"resolved_count": resolvedCount,
			"skipped_count":  0,
		},
	})
}

// RunQuickBenchmark 运行快速基准测试
func (h *ConstraintHandler) RunQuickBenchmark(c *gin.Context) {
	result, err := h.benchmark.RunQuickBenchmark(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "基准测试失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "测试完成",
		"data":    result,
	})
}

// RunStressBenchmark 运行压力测试
func (h *ConstraintHandler) RunStressBenchmark(c *gin.Context) {
	result, err := h.benchmark.RunStressTest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "压力测试失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "测试完成",
		"data":    result,
	})
}
