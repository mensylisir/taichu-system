package service

import (
	"fmt"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

// TenantQuotaController 租户配额控制器 - 逻辑约束
type TenantQuotaController struct {
	quotaRepo       *repository.QuotaRepository
	environmentRepo *repository.EnvironmentRepository
	tenantRepo      *repository.TenantRepository
}

// NewTenantQuotaController 创建租户配额控制器
func NewTenantQuotaController(
	quotaRepo *repository.QuotaRepository,
	environmentRepo *repository.EnvironmentRepository,
	tenantRepo *repository.TenantRepository,
) *TenantQuotaController {
	return &TenantQuotaController{
		quotaRepo:       quotaRepo,
		environmentRepo: environmentRepo,
		tenantRepo:      tenantRepo,
	}
}

// QuotaValidationResult 配额验证结果
type QuotaValidationResult struct {
	IsValid   bool            `json:"is_valid"`
	Message   string          `json:"message"`
	Available model.JSONMap   `json:"available"`
}

// ValidateEnvironmentQuota 验证环境配额设置是否超过租户限制
func (c *TenantQuotaController) ValidateEnvironmentQuota(tenantID string, requestedQuota model.JSONMap) (*QuotaValidationResult, error) {
	// 获取租户配额
	tenantQuota, err := c.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		// 租户配额不存在，允许创建（无限制）
		return &QuotaValidationResult{
			IsValid: true,
			Message: "租户未设置配额限制",
		}, nil
	}

	// 获取租户下所有环境的已分配配额
	allocatedQuota, err := c.calculateAllocatedQuota(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate allocated quota: %w", err)
	}

	// 检查请求的配额是否超过租户总配额
	result := c.checkQuotaExceeded(tenantQuota.HardLimits, allocatedQuota, requestedQuota)

	return result, nil
}

// checkQuotaExceeded 检查配额是否超过限制
func (c *TenantQuotaController) checkQuotaExceeded(hardLimits, allocated, requested model.JSONMap) *QuotaValidationResult {
	exceededResources := make([]string, 0)

	for resource, limit := range hardLimits {
		if limitStr, ok := limit.(string); ok {
			// 获取已分配的数量
			allocatedValue := c.getResourceValue(allocated, resource)
			// 获取请求的数量
			requestedValue := c.getResourceValue(requested, resource)

			// 计算总使用量 = 已分配 + 新请求
			totalUsage := allocatedValue + requestedValue
			limitValue := c.parseResourceQuantity(limitStr)

			if totalUsage > limitValue {
				exceededResources = append(exceededResources, resource)
			}
		}
	}

	if len(exceededResources) > 0 {
		return &QuotaValidationResult{
			IsValid: false,
			Message: fmt.Sprintf("以下资源超过租户限制: %v", exceededResources),
		}
	}

	// 计算可用配额
	available := c.calculateAvailableQuota(hardLimits, allocated)

	return &QuotaValidationResult{
		IsValid:   true,
		Message:   "配额验证通过",
		Available: available,
	}
}

// calculateAllocatedQuota 计算已分配的配额
func (c *TenantQuotaController) calculateAllocatedQuota(tenantID string) (model.JSONMap, error) {
	// 获取租户下的所有环境
	envs, err := c.environmentRepo.ListByTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	allocated := make(model.JSONMap)

	for _, env := range envs {
		// 获取环境的ResourceQuota
		quota, err := c.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if err != nil {
			continue
		}

		// 累加配额
		for resource, value := range quota.HardLimits {
			currentValue := c.getResourceValue(allocated, resource)
			quotaValue := c.getResourceValue(quota.HardLimits, resource)
			_ = value // 标记为已使用
			allocated[resource] = currentValue + quotaValue
		}
	}

	return allocated, nil
}

// calculateAvailableQuota 计算可用配额
func (c *TenantQuotaController) calculateAvailableQuota(hardLimits, allocated model.JSONMap) model.JSONMap {
	available := make(model.JSONMap)

	for resource, limit := range hardLimits {
		if limitStr, ok := limit.(string); ok {
			limitValue := c.parseResourceQuantity(limitStr)
			allocatedValue := c.getResourceValue(allocated, resource)
			availableValue := limitValue - allocatedValue

			if availableValue < 0 {
				availableValue = 0
			}

			available[resource] = c.formatResourceValue(availableValue)
		}
	}

	return available
}

// getResourceValue 获取资源值
func (c *TenantQuotaController) getResourceValue(quota model.JSONMap, resource string) int64 {
	if value, ok := quota[resource]; ok {
		if intVal, ok := value.(int64); ok {
			return intVal
		}
	}
	return 0
}

// parseResourceQuantity 解析资源数量
// 支持格式: 100m (milli CPU), 1 (CPU cores), 500Mi (memory), 2Gi (memory), 1000 (count)
func (c *TenantQuotaController) parseResourceQuantity(value string) int64 {
	if value == "" {
		return 0
	}

	// 提取数值和单位
	var numStr, unit string
	for i, r := range value {
		if r >= '0' && r <= '9' || r == '.' {
			numStr += string(r)
		} else {
			unit = value[i:]
			break
		}
	}

	// 解析数值
	var num float64
	if numStr != "" {
		fmt.Sscanf(numStr, "%f", &num)
	}

	// 如果没有单位，返回整数部分
	if unit == "" {
		return int64(num)
	}

	// CPU单位处理
	if unit == "m" {
		// 100m = 0.1 CPU core
		return int64(num)
	}

	// 内存单位处理 (转换为字节)
	switch unit {
	case "Ki":
		return int64(num * 1024)
	case "Mi":
		return int64(num * 1024 * 1024)
	case "Gi":
		return int64(num * 1024 * 1024 * 1024)
	case "Ti":
		return int64(num * 1024 * 1024 * 1024 * 1024)
	case "k", "K": // Kubernetes支持k作为Ki的简写
		return int64(num * 1024)
	case "M": // Kubernetes支持M作为Mi的简写
		return int64(num * 1024 * 1024)
	case "G": // Kubernetes支持G作为Gi的简写
		return int64(num * 1024 * 1024 * 1024)
	case "T": // Kubernetes支持T作为Ti的简写
		return int64(num * 1024 * 1024 * 1024 * 1024)
	}

	// 数值部分
	return int64(num)
}

// formatResourceValue 格式化资源值
func (c *TenantQuotaController) formatResourceValue(value int64) string {
	return fmt.Sprintf("%d", value)
}

// UpdateTenantQuota 更新租户配额
func (c *TenantQuotaController) UpdateTenantQuota(tenantID string, hardLimits model.JSONMap) error {
	// 获取或创建租户配额
	tenantQuota, err := c.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		// 创建新记录
		tenantQuota = &model.TenantQuota{
			TenantID:   model.ParseUUID(tenantID),
			HardLimits: hardLimits,
			Status:     model.QuotaStatusActive,
		}
		return c.quotaRepo.CreateTenantQuota(tenantQuota)
	}

	// 更新现有记录
	tenantQuota.HardLimits = hardLimits
	return c.quotaRepo.UpdateTenantQuota(tenantQuota)
}

// GetTenantQuotaStatus 获取租户配额状态
func (c *TenantQuotaController) GetTenantQuotaStatus(tenantID string) (*TenantQuotaStatus, error) {
	tenant, err := c.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	tenantQuota, err := c.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		// 无配额限制
		return &TenantQuotaStatus{
			TenantID:        tenantID,
			TenantName:      tenant.Name,
			HasQuotaLimit:   false,
			HardLimits:      nil,
			Allocated:       nil,
			Available:       nil,
			UsagePercent:    0,
		}, nil
	}

	allocated, err := c.calculateAllocatedQuota(tenantID)
	if err != nil {
		return nil, err
	}

	available := c.calculateAvailableQuota(tenantQuota.HardLimits, allocated)

	// 计算使用率
	usagePercent := c.calculateUsagePercent(tenantQuota.HardLimits, allocated)

	return &TenantQuotaStatus{
		TenantID:      tenantID,
		TenantName:    tenant.Name,
		HasQuotaLimit: true,
		HardLimits:    tenantQuota.HardLimits,
		Allocated:     allocated,
		Available:     available,
		UsagePercent:  usagePercent,
	}, nil
}

// TenantQuotaStatus 租户配额状态
type TenantQuotaStatus struct {
	TenantID      string         `json:"tenant_id"`
	TenantName    string         `json:"tenant_name"`
	HasQuotaLimit bool           `json:"has_quota_limit"`
	HardLimits    model.JSONMap  `json:"hard_limits"`
	Allocated     model.JSONMap  `json:"allocated"`
	Available     model.JSONMap  `json:"available"`
	UsagePercent  float64        `json:"usage_percent"`
}

// calculateUsagePercent 计算使用率
func (c *TenantQuotaController) calculateUsagePercent(hardLimits, allocated model.JSONMap) float64 {
	if len(hardLimits) == 0 {
		return 0
	}

	totalUsage := 0.0
	totalLimit := 0.0

	for resource, limit := range hardLimits {
		if limitStr, ok := limit.(string); ok {
			limitValue := c.parseResourceQuantity(limitStr)
			allocatedValue := c.getResourceValue(allocated, resource)

			if limitValue > 0 {
				usage := float64(allocatedValue) / float64(limitValue) * 100
				totalUsage += usage
				totalLimit += 1.0
			}
		}
	}

	if totalLimit == 0 {
		return 0
	}

	return totalUsage / totalLimit
}
