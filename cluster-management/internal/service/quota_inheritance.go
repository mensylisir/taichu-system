package service

import (
	"fmt"
	"time"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

// QuotaInheritance 配额继承机制
type QuotaInheritance struct {
	tenantRepo     *repository.TenantRepository
	environmentRepo *repository.EnvironmentRepository
	quotaRepo      *repository.QuotaRepository
}

// NewQuotaInheritance 创建配额继承管理器
func NewQuotaInheritance(
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	quotaRepo *repository.QuotaRepository,
) *QuotaInheritance {
	return &QuotaInheritance{
		tenantRepo:     tenantRepo,
		environmentRepo: environmentRepo,
		quotaRepo:      quotaRepo,
	}
}

// ===== 租户到环境的继承 =====

// InheritQuotaFromTenant 环境从租户继承配额
func (q *QuotaInheritance) InheritQuotaFromTenant(environmentID string) (*model.ResourceQuota, error) {
	// 1. 获取环境信息
	env, err := q.environmentRepo.GetByID(environmentID)
	if err != nil {
		return nil, fmt.Errorf("环境不存在: %w", err)
	}

	// 2. 获取租户配额
	tenantQuota, err := q.quotaRepo.GetTenantQuotaByTenantID(env.TenantID.String())
	if err != nil {
		// 租户没有配额，返回nil
		return nil, nil
	}

	// 3. 从租户配额继承（按比例分配）
	siblingCount := q.getSiblingEnvironments(env.TenantID.String())
	inheritedQuota := q.calculateInheritedQuota(tenantQuota.HardLimits, siblingCount)

	// 4. 创建或更新环境配额
	resourceQuota := &model.ResourceQuota{
		EnvironmentID: env.ID,
		HardLimits:    inheritedQuota,
		Used:          make(model.JSONMap),
		Status:        model.QuotaStatusActive,
	}

	// 检查是否已存在
	_, err = q.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err == nil {
		// 更新现有配额
		return resourceQuota, q.quotaRepo.UpdateResourceQuota(resourceQuota)
	}

	// 创建新配额
	return resourceQuota, q.quotaRepo.CreateResourceQuota(resourceQuota)
}

// calculateInheritedQuota 计算继承的配额
func (q *QuotaInheritance) calculateInheritedQuota(tenantQuota model.JSONMap, siblingCount int) model.JSONMap {
	inherited := make(model.JSONMap)

	if siblingCount <= 1 {
		// 如果只有这一个环境，继承全部配额
		return tenantQuota
	}

	// 按比例分配配额
	for resource, limit := range tenantQuota {
		if limitStr, ok := limit.(string); ok {
			// 简化分配策略：平分配额
			// 实际生产环境中可能需要更复杂的策略
			inherited[resource] = q.divideQuota(limitStr, siblingCount)
		}
	}

	return inherited
}

// divideQuota 分配配额
func (q *QuotaInheritance) divideQuota(quotaStr string, divisor int) string {
	// TODO: 实现更复杂的配额分配逻辑
	// 目前简化处理，返回原始值
	return quotaStr
}

// getSiblingEnvironments 获取同级环境数量
func (q *QuotaInheritance) getSiblingEnvironments(tenantID string) int {
	envs, _ := q.environmentRepo.ListByTenantID(tenantID)
	return len(envs)
}

// ===== 应用规格继承 =====

// InheritResourceSpecFromEnvironment 应用从环境继承资源规格
func (q *QuotaInheritance) InheritResourceSpecFromEnvironment(applicationID string) (*model.ApplicationResourceSpec, error) {
	// 1. 获取应用信息
	// TODO: 需要通过applicationRepo获取

	// 2. 获取环境信息
	// TODO: 需要传入environmentID

	// 3. 获取环境配额（用于推断默认资源规格）
	// TODO: 实现逻辑

	return nil, fmt.Errorf("未实现")
}

// ===== 批量继承 =====

// BatchInheritFromTenant 批量为租户下所有环境继承配额
func (q *QuotaInheritance) BatchInheritFromTenant(tenantID string) ([]*model.ResourceQuota, error) {
	// 1. 获取租户下所有环境
	envs, err := q.environmentRepo.ListByTenantID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取环境列表失败: %w", err)
	}

	var inheritedQuotas []*model.ResourceQuota

	// 2. 为每个环境继承配额
	for _, env := range envs {
		quota, err := q.InheritQuotaFromTenant(env.ID.String())
		if err != nil {
			// 记录错误但继续处理其他环境
			continue
		}
		if quota != nil {
			inheritedQuotas = append(inheritedQuotas, quota)
		}
	}

	return inheritedQuotas, nil
}

// UpdateInheritanceOnTenantQuotaChange 租户配额变更时更新所有子环境配额
func (q *QuotaInheritance) UpdateInheritanceOnTenantQuotaChange(tenantID string) error {
	// 1. 获取租户下所有环境
	envs, err := q.environmentRepo.ListByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("获取环境列表失败: %w", err)
	}

	// 2. 获取租户配额
	tenantQuota, err := q.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		// 租户配额不存在，清空所有子环境配额
		for _, env := range envs {
			_, getErr := q.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
			if getErr == nil {
				// 删除配额记录
				q.quotaRepo.DeleteResourceQuota(env.ID.String())
			}
		}
		return nil
	}

	// 3. 重新计算并更新所有子环境配额
	for _, env := range envs {
		_, getErr := q.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if getErr == nil {
			// 更新现有配额
			inheritedQuota := q.calculateInheritedQuota(tenantQuota.HardLimits, len(envs))
			updatedQuota := &model.ResourceQuota{
				EnvironmentID: env.ID,
				HardLimits:    inheritedQuota,
				Used:          make(model.JSONMap),
				Status:        model.QuotaStatusActive,
			}
			if updateErr := q.quotaRepo.UpdateResourceQuota(updatedQuota); updateErr != nil {
				return fmt.Errorf("更新环境 %s 配额失败: %w", env.Namespace, updateErr)
			}
		} else {
			// 创建新配额
			inheritedQuota := q.calculateInheritedQuota(tenantQuota.HardLimits, len(envs))
			newQuota := &model.ResourceQuota{
				EnvironmentID: env.ID,
				HardLimits:    inheritedQuota,
				Used:          make(model.JSONMap),
				Status:        model.QuotaStatusActive,
			}
			if createErr := q.quotaRepo.CreateResourceQuota(newQuota); createErr != nil {
				return fmt.Errorf("为环境 %s 创建配额失败: %w", env.Namespace, createErr)
			}
		}
	}

	return nil
}

// ===== 继承策略配置 =====

// QuotaInheritanceStrategy 配额继承策略
type QuotaInheritanceStrategy struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetAvailableStrategies 获取可用的继承策略
func (q *QuotaInheritance) GetAvailableStrategies() []QuotaInheritanceStrategy {
	return []QuotaInheritanceStrategy{
		{
			Name:        "equal_split",
			Description: "等分策略：将租户配额平均分配给所有子环境",
		},
		{
			Name:        "weighted_split",
			Description: "加权分配：根据环境权重分配配额",
		},
		{
			Name:        "manual_override",
			Description: "手动覆盖：每个环境单独设置配额，不继承",
		},
	}
}

// ApplyStrategy 应用继承策略
func (q *QuotaInheritance) ApplyStrategy(tenantID, strategyName string) error {
	switch strategyName {
	case "equal_split":
		_, err := q.BatchInheritFromTenant(tenantID)
		return err
	case "manual_override":
		// 手动覆盖：不自动继承，用户单独设置
		return nil
	default:
		return fmt.Errorf("未知的继承策略: %s", strategyName)
	}
}

// ===== 继承状态查询 =====

// GetInheritanceStatus 获取继承状态
func (q *QuotaInheritance) GetInheritanceStatus(environmentID string) (*InheritanceStatus, error) {
	// 1. 获取环境信息
	env, err := q.environmentRepo.GetByID(environmentID)
	if err != nil {
		return nil, fmt.Errorf("环境不存在: %w", err)
	}

	// 2. 获取环境配额
	envQuota, err := q.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err != nil {
		return &InheritanceStatus{
			EnvironmentID: environmentID,
			HasQuota:      false,
			Inherited:     false,
			Source:        "none",
		}, nil
	}

	// 3. 获取租户配额
	tenantQuota, err := q.quotaRepo.GetTenantQuotaByTenantID(env.TenantID.String())
	if err != nil {
		return &InheritanceStatus{
			EnvironmentID:   environmentID,
			HasQuota:        true,
			Inherited:       false,
			Source:          "manual",
			EnvironmentQuota: envQuota.HardLimits,
		}, nil
	}

	// 4. 判断是否为继承
	isInherited := q.isQuotaInherited(envQuota.HardLimits, tenantQuota.HardLimits)

	return &InheritanceStatus{
		EnvironmentID:   environmentID,
		HasQuota:        true,
		Inherited:       isInherited,
		Source:          "tenant",
		TenantQuota:     tenantQuota.HardLimits,
		EnvironmentQuota: envQuota.HardLimits,
	}, nil
}

// isQuotaInherited 判断配额是否为继承
func (q *QuotaInheritance) isQuotaInherited(envQuota, tenantQuota model.JSONMap) bool {
	// 简化判断：如果环境配额与租户配额相同，则认为是继承的
	// 实际生产环境中可能需要更复杂的逻辑
	if len(envQuota) == 0 || len(tenantQuota) == 0 {
		return false
	}

	// 检查所有租户配额键是否都在环境配额中
	for resource, tenantLimit := range tenantQuota {
		if envLimit, ok := envQuota[resource]; !ok || envLimit != tenantLimit {
			// 如果环境配额不等于租户配额，可能是继承后被手动修改
			return false
		}
	}

	return true
}

// InheritanceStatus 继承状态
type InheritanceStatus struct {
	EnvironmentID      string         `json:"environment_id"`
	HasQuota           bool           `json:"has_quota"`
	Inherited          bool           `json:"inherited"`
	Source             string         `json:"source"` // "none", "manual", "tenant"
	TenantQuota        model.JSONMap  `json:"tenant_quota,omitempty"`
	EnvironmentQuota   model.JSONMap  `json:"environment_quota,omitempty"`
	LastInheritedAt    *time.Time     `json:"last_inherited_at,omitempty"`
}
