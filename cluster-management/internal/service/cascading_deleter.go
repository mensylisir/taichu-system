package service

import (
	"fmt"

	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
)

// CascadingDeleter 级联删除器
type CascadingDeleter struct {
	db              *gorm.DB
	tenantRepo      *repository.TenantRepository
	environmentRepo *repository.EnvironmentRepository
	applicationRepo *repository.ApplicationRepository
	quotaRepo       *repository.QuotaRepository
	constraintValidator *ConstraintValidator
}

// NewCascadingDeleter 创建级联删除器
func NewCascadingDeleter(
	db *gorm.DB,
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	constraintValidator *ConstraintValidator,
) *CascadingDeleter {
	return &CascadingDeleter{
		db:                db,
		tenantRepo:        tenantRepo,
		environmentRepo:   environmentRepo,
		applicationRepo:   applicationRepo,
		quotaRepo:         quotaRepo,
		constraintValidator: constraintValidator,
	}
}

// ===== 应用级联删除 =====

// DeleteApplicationWithDependencies 删除应用及其所有依赖
func (c *CascadingDeleter) DeleteApplicationWithDependencies(applicationID string) error {
	// 1. 验证删除条件
	if err := c.constraintValidator.ValidateApplicationDelete(applicationID); err != nil {
		return fmt.Errorf("删除应用验证失败: %w", err)
	}

	// 2. 开始事务
	return c.db.Transaction(func(tx *gorm.DB) error {
		// 2.1 删除应用资源规格
		if err := c.quotaRepo.DeleteApplicationResourceSpec(applicationID); err != nil {
			return fmt.Errorf("删除应用资源规格失败: %w", err)
		}

		// 2.2 删除应用
		if err := c.applicationRepo.Delete(applicationID); err != nil {
			return fmt.Errorf("删除应用失败: %w", err)
		}

		return nil
	})
}

// ===== 环境级联删除 =====

// DeleteEnvironmentWithDependencies 删除环境及其所有依赖
func (c *CascadingDeleter) DeleteEnvironmentWithDependencies(environmentID string) error {
	// 1. 验证删除条件
	if err := c.constraintValidator.ValidateEnvironmentDelete(environmentID); err != nil {
		return fmt.Errorf("删除环境验证失败: %w", err)
	}

	// 2. 开始事务
	return c.db.Transaction(func(tx *gorm.DB) error {
		// 2.1 获取环境下的所有应用
		apps, err := c.applicationRepo.ListByEnvironmentID(environmentID)
		if err != nil {
			return fmt.Errorf("获取环境下的应用失败: %w", err)
		}

		// 2.2 递归删除所有应用及其依赖
		for _, app := range apps {
			if err := c.deleteApplicationWithDependencies(tx, app.ID.String()); err != nil {
				return fmt.Errorf("删除应用 %s 失败: %w", app.Name, err)
			}
		}

		// 2.3 删除环境配额
		if err := c.quotaRepo.DeleteResourceQuota(environmentID); err != nil {
			return fmt.Errorf("删除环境配额失败: %w", err)
		}

		// 2.4 删除环境
		if err := c.environmentRepo.Delete(environmentID); err != nil {
			return fmt.Errorf("删除环境失败: %w", err)
		}

		return nil
	})
}

// deleteApplicationWithDependencies 在事务中删除应用
func (c *CascadingDeleter) deleteApplicationWithDependencies(tx *gorm.DB, applicationID string) error {
	// 删除应用资源规格
	if err := c.quotaRepo.DeleteApplicationResourceSpec(applicationID); err != nil {
		return err
	}

	// 删除应用
	return c.applicationRepo.Delete(applicationID)
}

// ===== 租户级联删除 =====

// DeleteTenantWithDependencies 删除租户及其所有依赖
func (c *CascadingDeleter) DeleteTenantWithDependencies(tenantID string) error {
	// 1. 验证删除条件
	if err := c.constraintValidator.ValidateTenantDelete(tenantID); err != nil {
		return fmt.Errorf("删除租户验证失败: %w", err)
	}

	// 2. 开始事务
	return c.db.Transaction(func(tx *gorm.DB) error {
		// 2.1 获取租户下的所有环境
		envs, err := c.environmentRepo.ListByTenantID(tenantID)
		if err != nil {
			return fmt.Errorf("获取租户下的环境失败: %w", err)
		}

		// 2.2 递归删除所有环境及其依赖
		for _, env := range envs {
			if err := c.deleteEnvironmentWithDependencies(tx, env.ID.String()); err != nil {
				return fmt.Errorf("删除环境 %s 失败: %w", env.Namespace, err)
			}
		}

		// 2.3 删除租户配额
		if err := c.quotaRepo.DeleteTenantQuota(tenantID); err != nil {
			return fmt.Errorf("删除租户配额失败: %w", err)
		}

		// 2.4 删除租户
		if err := c.tenantRepo.Delete(tenantID); err != nil {
			return fmt.Errorf("删除租户失败: %w", err)
		}

		return nil
	})
}

// deleteEnvironmentWithDependencies 在事务中删除环境
func (c *CascadingDeleter) deleteEnvironmentWithDependencies(tx *gorm.DB, environmentID string) error {
	// 获取环境下的所有应用
	apps, err := c.applicationRepo.ListByEnvironmentID(environmentID)
	if err != nil {
		return err
	}

	// 删除所有应用及其依赖
	for _, app := range apps {
		if err := c.deleteApplicationWithDependencies(tx, app.ID.String()); err != nil {
			return err
		}
	}

	// 删除环境配额
	if err := c.quotaRepo.DeleteResourceQuota(environmentID); err != nil {
		return err
	}

	// 删除环境
	return c.environmentRepo.Delete(environmentID)
}

// ===== 批量删除 =====

// BatchDeleteApplications 批量删除应用
func (c *CascadingDeleter) BatchDeleteApplications(applicationIDs []string) error {
	if len(applicationIDs) == 0 {
		return nil
	}

	return c.db.Transaction(func(tx *gorm.DB) error {
		for _, appID := range applicationIDs {
			// 验证每个应用是否可以删除
			if err := c.constraintValidator.ValidateApplicationDelete(appID); err != nil {
				return fmt.Errorf("应用 %s 不允许删除: %w", appID, err)
			}
		}

		// 执行删除
		for _, appID := range applicationIDs {
			if err := c.deleteApplicationWithDependencies(tx, appID); err != nil {
				return fmt.Errorf("删除应用 %s 失败: %w", appID, err)
			}
		}

		return nil
	})
}

// BatchDeleteEnvironments 批量删除环境
func (c *CascadingDeleter) BatchDeleteEnvironments(environmentIDs []string) error {
	if len(environmentIDs) == 0 {
		return nil
	}

	return c.db.Transaction(func(tx *gorm.DB) error {
		for _, envID := range environmentIDs {
			// 验证每个环境是否可以删除
			if err := c.constraintValidator.ValidateEnvironmentDelete(envID); err != nil {
				return fmt.Errorf("环境 %s 不允许删除: %w", envID, err)
			}
		}

		// 执行删除
		for _, envID := range environmentIDs {
			if err := c.deleteEnvironmentWithDependencies(tx, envID); err != nil {
				return fmt.Errorf("删除环境 %s 失败: %w", envID, err)
			}
		}

		return nil
	})
}

// ===== 软删除支持 =====

// SoftDeleteApplication 软删除应用
func (c *CascadingDeleter) SoftDeleteApplication(applicationID string) error {
	// 1. 验证删除条件（软删除通常更宽松）
	// 软删除允许删除有依赖的应用，只是标记为已删除

	// 2. 开始事务
	return c.db.Transaction(func(tx *gorm.DB) error {
		// 2.1 标记应用为已删除
		// TODO: 需要在Application模型中添加软删除支持

		// 2.2 软删除应用资源规格
		// TODO: 需要在ApplicationResourceSpec模型中添加软删除支持

		return nil
	})
}

// ===== 删除统计 =====

// GetDeletionStatistics 获取删除统计信息
func (c *CascadingDeleter) GetDeletionStatistics(resourceType, resourceID string) (*DeletionStatistics, error) {
	var stats DeletionStatistics

	switch resourceType {
	case "tenant":
		envs, err := c.environmentRepo.ListByTenantID(resourceID)
		if err != nil {
			return nil, fmt.Errorf("获取环境列表失败: %w", err)
		}

		stats.TenantID = resourceID
		stats.EnvironmentCount = len(envs)

		for _, env := range envs {
			apps, err := c.applicationRepo.ListByEnvironmentID(env.ID.String())
			if err != nil {
				continue
			}
			stats.ApplicationCount += len(apps)

			// 检查配额
			_, err = c.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
			if err == nil {
				stats.QuotaCount++
			}
		}

		// 检查租户配额
		_, err = c.quotaRepo.GetTenantQuotaByTenantID(resourceID)
		if err == nil {
			stats.TenantQuotaCount = 1
		}

	case "environment":
		apps, err := c.applicationRepo.ListByEnvironmentID(resourceID)
		if err != nil {
			return nil, fmt.Errorf("获取应用列表失败: %w", err)
		}

		stats.EnvironmentID = resourceID
		stats.ApplicationCount = len(apps)

		// 检查配额
		_, err = c.quotaRepo.GetResourceQuotaByEnvironmentID(resourceID)
		if err == nil {
			stats.QuotaCount = 1
		}

	case "application":
		stats.ApplicationID = resourceID

		// 检查资源规格
		_, err := c.quotaRepo.GetApplicationResourceSpecByApplicationID(resourceID)
		if err == nil {
			stats.ResourceSpecCount = 1
		}
	}

	return &stats, nil
}

// DeletionStatistics 删除统计信息
type DeletionStatistics struct {
	TenantID           string `json:"tenant_id,omitempty"`
	EnvironmentID      string `json:"environment_id,omitempty"`
	ApplicationID      string `json:"application_id,omitempty"`
	EnvironmentCount   int    `json:"environment_count"`
	ApplicationCount   int    `json:"application_count"`
	QuotaCount         int    `json:"quota_count"`
	TenantQuotaCount   int    `json:"tenant_quota_count"`
	ResourceSpecCount  int    `json:"resource_spec_count"`
}

// ===== 预检查 =====

// PreCheckDeletion 预检查删除操作
func (c *CascadingDeleter) PreCheckDeletion(resourceType, resourceID string) (*PreCheckResult, error) {
	stats, err := c.GetDeletionStatistics(resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("获取删除统计失败: %w", err)
	}

	result := &PreCheckResult{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		CanDelete:    true,
		Stats:        stats,
	}

	// 检查是否有依赖
	if stats.ApplicationCount > 0 {
		result.CanDelete = false
		result.BlockingResources = append(result.BlockingResources,
			fmt.Sprintf("%d 个应用", stats.ApplicationCount))
	}

	if stats.QuotaCount > 0 {
		result.CanDelete = false
		result.BlockingResources = append(result.BlockingResources,
			fmt.Sprintf("%d 个配额", stats.QuotaCount))
	}

	if stats.EnvironmentCount > 0 {
		result.CanDelete = false
		result.BlockingResources = append(result.BlockingResources,
			fmt.Sprintf("%d 个环境", stats.EnvironmentCount))
	}

	return result, nil
}

// PreCheckResult 预检查结果
type PreCheckResult struct {
	ResourceType      string             `json:"resource_type"`
	ResourceID        string             `json:"resource_id"`
	CanDelete         bool               `json:"can_delete"`
	BlockingResources []string           `json:"blocking_resources,omitempty"`
	Stats             *DeletionStatistics `json:"stats,omitempty"`
}
