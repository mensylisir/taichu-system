package service

import (
	"context"
	"fmt"
	"time"

	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// TransactionManager 事务管理器
type TransactionManager struct {
	db *gorm.DB
}

// NewTransactionManager 创建事务管理器
func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// ===== 基础事务操作 =====

// ExecuteInTransaction 在事务中执行操作
func (t *TransactionManager) ExecuteInTransaction(operation func(tx *gorm.DB) error) error {
	return t.db.Transaction(operation)
}

// ExecuteWithContext 在事务中执行带上下文的操作
func (t *TransactionManager) ExecuteWithContext(ctx context.Context, operation func(tx *gorm.DB) error) error {
	return t.db.WithContext(ctx).Transaction(operation)
}

// ===== 租户相关事务 =====

// CreateTenantWithDependencies 创建租户及其依赖
func (t *TransactionManager) CreateTenantWithDependencies(req CreateTenantRequest) (*model.Tenant, error) {
	var createdTenant *model.Tenant

	err := t.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建租户
		tenant := &model.Tenant{
			Name:        req.Name,
			DisplayName: req.DisplayName,
			Type:        model.TenantTypeUserCreated,
			Description: req.Description,
			Labels:      req.Labels,
			Status:      model.TenantStatusActive,
			IsSystem:    false,
		}

		if err := tx.Create(tenant).Error; err != nil {
			return fmt.Errorf("创建租户失败: %w", err)
		}
		createdTenant = tenant

		// 2. 创建租户配额
		if req.Quota != nil {
			tenantQuota := &model.TenantQuota{
				TenantID:   tenant.ID,
				HardLimits: req.Quota,
				Allocated:  make(model.JSONMap),
				Available:  req.Quota,
				Status:     model.QuotaStatusActive,
			}

			if err := tx.Create(tenantQuota).Error; err != nil {
				return fmt.Errorf("创建租户配额失败: %w", err)
			}
		}

		return nil
	})

	return createdTenant, err
}

// UpdateTenantWithDependencies 更新租户及其依赖
func (t *TransactionManager) UpdateTenantWithDependencies(tenantID string, updates map[string]interface{}) error {
	return t.db.Transaction(func(tx *gorm.DB) error {
		// 1. 更新租户基本信息
		if err := tx.Model(&model.Tenant{}).Where("id = ?", tenantID).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新租户失败: %w", err)
		}

		// 2. 处理配额更新
		if quota, ok := updates["quota"]; ok {
			if quotaMap, ok := quota.(model.JSONMap); ok {
				// 查找现有配额
				var tenantQuota model.TenantQuota
				if err := tx.Where("tenant_id = ?", tenantID).First(&tenantQuota).Error; err == nil {
					// 更新现有配额
					tenantQuota.HardLimits = quotaMap
					if err := tx.Save(&tenantQuota).Error; err != nil {
						return fmt.Errorf("更新租户配额失败: %w", err)
					}
				} else {
					// 创建新配额
					newQuota := &model.TenantQuota{
						TenantID:   model.ParseUUID(tenantID),
						HardLimits: quotaMap,
						Allocated:  make(model.JSONMap),
						Available:  quotaMap,
						Status:     model.QuotaStatusActive,
					}
					if err := tx.Create(newQuota).Error; err != nil {
						return fmt.Errorf("创建租户配额失败: %w", err)
					}
				}
			}
		}

		return nil
	})
}

// ===== 环境相关事务 =====

// CreateEnvironmentWithDependencies 创建环境及其依赖
func (t *TransactionManager) CreateEnvironmentWithDependencies(req CreateEnvironmentRequest) (*model.Environment, error) {
	var createdEnv *model.Environment

	err := t.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建环境
		env := &model.Environment{
			TenantID:    model.ParseUUID(req.TenantID),
			ClusterID:   model.ParseUUID(req.ClusterID),
			Namespace:   req.Namespace,
			DisplayName: req.DisplayName,
			Description: req.Description,
			Labels:      req.Labels,
			Status:      model.EnvironmentStatusActive,
		}

		if err := tx.Create(env).Error; err != nil {
			return fmt.Errorf("创建环境失败: %w", err)
		}
		createdEnv = env

		// 2. 创建环境配额
		if req.Quota != nil {
			resourceQuota := &model.ResourceQuota{
				EnvironmentID: env.ID,
				HardLimits:    req.Quota,
				Used:          make(model.JSONMap),
				Status:        model.QuotaStatusActive,
			}

			if err := tx.Create(resourceQuota).Error; err != nil {
				return fmt.Errorf("创建环境配额失败: %w", err)
			}
		}

		return nil
	})

	return createdEnv, err
}

// ===== 应用相关事务 =====

// CreateApplicationWithDependencies 创建应用及其依赖
func (t *TransactionManager) CreateApplicationWithDependencies(req CreateApplicationRequest) (*model.Application, error) {
	var createdApp *model.Application

	err := t.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建应用
		app := &model.Application{
			TenantID:      model.ParseUUID(req.TenantID),
			EnvironmentID: model.ParseUUID(req.EnvironmentID),
			Name:          req.Name,
			DisplayName:   req.DisplayName,
			Description:   req.Description,
			Labels:        req.Labels,
		}

		if err := tx.Create(app).Error; err != nil {
			return fmt.Errorf("创建应用失败: %w", err)
		}
		createdApp = app

		// 2. 创建应用资源规格
		if req.ResourceSpec != nil {
			resourceSpec := &model.ApplicationResourceSpec{
				ApplicationID:  app.ID,
				DefaultRequest: req.ResourceSpec.DefaultRequest,
				DefaultLimit:   req.ResourceSpec.DefaultLimit,
				MaxReplicas:    req.ResourceSpec.MaxReplicas,
			}

			if err := tx.Create(resourceSpec).Error; err != nil {
				return fmt.Errorf("创建应用资源规格失败: %w", err)
			}
		}

		return nil
	})

	return createdApp, err
}

// ===== 批量操作事务 =====

// BatchCreateApplications 批量创建应用
func (t *TransactionManager) BatchCreateApplications(requests []CreateApplicationRequest) ([]*model.Application, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	var createdApps []*model.Application

	err := t.db.Transaction(func(tx *gorm.DB) error {
		for i, req := range requests {
			// 1. 创建应用
			app := &model.Application{
				TenantID:      model.ParseUUID(req.TenantID),
				EnvironmentID: model.ParseUUID(req.EnvironmentID),
				Name:          req.Name,
				DisplayName:   req.DisplayName,
				Description:   req.Description,
				Labels:        req.Labels,
			}

			if err := tx.Create(app).Error; err != nil {
				return fmt.Errorf("创建第 %d 个应用失败: %w", i+1, err)
			}
			createdApps = append(createdApps, app)

			// 2. 创建应用资源规格
			if req.ResourceSpec != nil {
				resourceSpec := &model.ApplicationResourceSpec{
					ApplicationID:  app.ID,
					DefaultRequest: req.ResourceSpec.DefaultRequest,
					DefaultLimit:   req.ResourceSpec.DefaultLimit,
					MaxReplicas:    req.ResourceSpec.MaxReplicas,
				}

				if err := tx.Create(resourceSpec).Error; err != nil {
					return fmt.Errorf("创建第 %d 个应用的资源规格失败: %w", i+1, err)
				}
			}
		}

		return nil
	})

	return createdApps, err
}

// ===== 资源导入事务 =====

// ImportClusterWithClassification 导入集群并进行资源分类
func (t *TransactionManager) ImportClusterWithClassification(
	cluster *model.Cluster,
	classificationResults *ClassificationResult,
) error {
	return t.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建集群记录
		if err := tx.Create(cluster).Error; err != nil {
			return fmt.Errorf("创建集群记录失败: %w", err)
		}

		// 2. 确保预定义租户存在
		// 这里需要通过TenantRepository操作，但可能需要传入
		// 简化处理：假设租户已存在

		// 3. 创建环境记录（如果有分类结果）
		if classificationResults != nil {
			// TODO: 根据分类结果创建环境和应用记录
			// 这里需要传入更多参数，如分类详情
		}

		return nil
	})
}

// ===== 同步操作事务 =====

// SyncResourceQuotaWithDependencies 同步资源配额及其依赖
func (t *TransactionManager) SyncResourceQuotaWithDependencies(environmentID string, quotaData *model.ResourceQuota) error {
	return t.db.Transaction(func(tx *gorm.DB) error {
		// 1. 检查配额是否已存在
		var existingQuota model.ResourceQuota
		err := tx.Where("environment_id = ?", environmentID).First(&existingQuota).Error

		if err == nil {
			// 更新现有配额
			quotaData.ID = existingQuota.ID
			if err := tx.Save(quotaData).Error; err != nil {
				return fmt.Errorf("更新资源配额失败: %w", err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新配额
			if err := tx.Create(quotaData).Error; err != nil {
				return fmt.Errorf("创建资源配额失败: %w", err)
			}
		} else {
			return fmt.Errorf("查询资源配额失败: %w", err)
		}

		// 2. 更新租户配额统计
		// TODO: 需要获取环境信息以更新租户配额

		return nil
	})
}

// ===== 上下文支持 =====

// ExecuteWithTimeout 在超时控制下执行事务
func (t *TransactionManager) ExecuteWithTimeout(timeoutSeconds int, operation func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	return t.db.WithContext(ctx).Transaction(operation)
}

// ===== 错误处理 =====

// TransactionError 事务错误
type TransactionError struct {
	Op       string `json:"operation"`
	Message  string `json:"message"`
	Original error  `json:"original_error"`
}

func (e *TransactionError) Error() string {
	return fmt.Sprintf("事务操作 %s 失败: %s (原始错误: %v)", e.Op, e.Message, e.Original)
}

// WrapTransactionError 包装事务错误
func (t *TransactionManager) WrapTransactionError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &TransactionError{
		Op:       op,
		Message:  err.Error(),
		Original: err,
	}
}

// ===== 重试机制 =====

// ExecuteWithRetry 带重试的事务执行
func (t *TransactionManager) ExecuteWithRetry(maxRetries int, operation func(tx *gorm.DB) error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := t.db.Transaction(operation)
		if err == nil {
			return nil
		}

		lastErr = err

		// 判断是否应该重试
		if !isRetryableError(err) {
			break
		}

		// 等待后重试
		// TODO: 实现指数退避策略
	}

	return lastErr
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	// TODO: 实现错误类型判断
	// 例如：数据库锁等待、网络超时等错误可以重试
	// 而业务逻辑错误（如验证失败）不应该重试
	return false
}
