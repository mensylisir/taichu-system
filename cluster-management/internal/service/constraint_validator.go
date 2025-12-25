package service

import (
	"fmt"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
)

// ConstraintValidator 三层约束检查器
type ConstraintValidator struct {
	tenantRepo     *repository.TenantRepository
	environmentRepo *repository.EnvironmentRepository
	applicationRepo *repository.ApplicationRepository
	quotaRepo      *repository.QuotaRepository
}

// NewConstraintValidator 创建约束检查器
func NewConstraintValidator(
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
) *ConstraintValidator {
	return &ConstraintValidator{
		tenantRepo:     tenantRepo,
		environmentRepo: environmentRepo,
		applicationRepo: applicationRepo,
		quotaRepo:      quotaRepo,
	}
}

// ===== 租户约束 =====

// ValidateTenantCreate 验证租户创建
func (v *ConstraintValidator) ValidateTenantCreate(req CreateTenantRequest) error {
	// 1. 检查租户名是否已存在
	_, err := v.tenantRepo.GetByName(req.Name)
	if err == nil {
		return fmt.Errorf("租户名已存在: %s", req.Name)
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查租户名失败: %w", err)
	}

	// 2. 验证租户名格式
	if !isValidTenantName(req.Name) {
		return fmt.Errorf("租户名格式无效: %s", req.Name)
	}

	// 3. 检查配额设置
	if req.Quota != nil {
		if err := v.validateQuotaFormat(req.Quota); err != nil {
			return fmt.Errorf("配额格式错误: %w", err)
		}
	}

	return nil
}

// ValidateTenantUpdate 验证租户更新
func (v *ConstraintValidator) ValidateTenantUpdate(tenantID string, updates map[string]interface{}) error {
	// 1. 检查租户是否存在
	tenant, err := v.tenantRepo.GetByID(tenantID)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 2. 预定义租户不允许更新某些字段
	if tenant.IsSystem {
		if name, ok := updates["name"]; ok && name != tenant.Name {
			return fmt.Errorf("不允许修改预定义租户的租户名")
		}
	}

	// 3. 检查配额变更
	if quota, ok := updates["quota"]; ok {
		if quotaMap, ok := quota.(model.JSONMap); ok {
			if err := v.validateQuotaFormat(quotaMap); err != nil {
				return fmt.Errorf("配额格式错误: %w", err)
			}
		}
	}

	return nil
}

// ValidateTenantDelete 验证租户删除
func (v *ConstraintValidator) ValidateTenantDelete(tenantID string) error {
	// 1. 检查租户是否存在
	tenant, err := v.tenantRepo.GetByID(tenantID)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 2. 预定义租户不允许删除
	if tenant.IsSystem {
		return fmt.Errorf("不允许删除预定义租户: %s", tenant.Name)
	}

	// 3. 检查是否有子资源（环境）
	envs, err := v.environmentRepo.ListByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("检查环境失败: %w", err)
	}
	if len(envs) > 0 {
		return fmt.Errorf("租户下还有 %d 个环境，无法删除", len(envs))
	}

	// 4. 检查租户配额
	_, err = v.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err == nil {
		return fmt.Errorf("租户配额仍存在，需要先删除配额")
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查租户配额失败: %w", err)
	}

	return nil
}

// ===== 环境约束 =====

// ValidateEnvironmentCreate 验证环境创建
func (v *ConstraintValidator) ValidateEnvironmentCreate(req CreateEnvironmentRequest) error {
	// 1. 检查租户是否存在
	tenant, err := v.tenantRepo.GetByID(req.TenantID)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 2. 检查租户状态
	if tenant.Status != model.TenantStatusActive {
		return fmt.Errorf("租户状态非活跃: %s", tenant.Status)
	}

	// 3. 检查集群是否存在
	// TODO: 需要通过clusterRepo检查

	// 4. 检查命名空间是否已存在
	existingEnv, err := v.environmentRepo.GetByNamespace(req.ClusterID, req.Namespace)
	if err == nil {
		return fmt.Errorf("环境已存在: %s", req.Namespace)
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查环境失败: %w", err)
	}
	_ = existingEnv // 标记为已使用

	// 5. 验证配额设置
	if req.Quota != nil {
		if err := v.validateQuotaFormat(req.Quota); err != nil {
			return fmt.Errorf("配额格式错误: %w", err)
		}

		// 6. 验证是否超过租户配额
		if err := v.validateEnvironmentQuotaAgainstTenant(req.TenantID, req.Quota); err != nil {
			return err
		}
	}

	return nil
}

// ValidateEnvironmentUpdate 验证环境更新
func (v *ConstraintValidator) ValidateEnvironmentUpdate(environmentID string, updates map[string]interface{}) error {
	// 1. 检查环境是否存在
	env, err := v.environmentRepo.GetByID(environmentID)
	if err != nil {
		return fmt.Errorf("环境不存在: %w", err)
	}

	// 2. 检查租户状态
	tenant, err := v.tenantRepo.GetByID(env.TenantID.String())
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}
	if tenant.Status != model.TenantStatusActive {
		return fmt.Errorf("租户状态非活跃: %s", tenant.Status)
	}

	// 3. 检查配额变更
	if quota, ok := updates["quota"]; ok {
		if quotaMap, ok := quota.(model.JSONMap); ok {
			if err := v.validateQuotaFormat(quotaMap); err != nil {
				return fmt.Errorf("配额格式错误: %w", err)
			}
		}
	}

	return nil
}

// ValidateEnvironmentDelete 验证环境删除
func (v *ConstraintValidator) ValidateEnvironmentDelete(environmentID string) error {
	// 1. 检查环境是否存在
	env, err := v.environmentRepo.GetByID(environmentID)
	if err != nil {
		return fmt.Errorf("环境不存在: %w", err)
	}
	_ = env // 标记为已使用

	// 2. 检查是否有子资源（应用）
	apps, err := v.applicationRepo.ListByEnvironmentID(environmentID)
	if err != nil {
		return fmt.Errorf("检查应用失败: %w", err)
	}
	if len(apps) > 0 {
		return fmt.Errorf("环境下还有 %d 个应用，无法删除", len(apps))
	}

	// 3. 检查ResourceQuota
	_, err = v.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err == nil {
		return fmt.Errorf("环境配额仍存在，需要先删除配额")
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查环境配额失败: %w", err)
	}

	return nil
}

// ===== 应用约束 =====

// ValidateApplicationCreate 验证应用创建
func (v *ConstraintValidator) ValidateApplicationCreate(req CreateApplicationRequest) error {
	// 1. 检查租户是否存在
	tenant, err := v.tenantRepo.GetByID(req.TenantID)
	if err != nil {
		return fmt.Errorf("租户不存在: %w", err)
	}

	// 2. 检查租户状态
	if tenant.Status != model.TenantStatusActive {
		return fmt.Errorf("租户状态非活跃: %s", tenant.Status)
	}

	// 3. 检查环境是否存在
	env, err := v.environmentRepo.GetByID(req.EnvironmentID)
	if err != nil {
		return fmt.Errorf("环境不存在: %w", err)
	}

	// 4. 检查环境状态
	if env.Status != model.EnvironmentStatusActive {
		return fmt.Errorf("环境状态非活跃: %s", env.Status)
	}

	// 5. 检查应用名是否已存在
	_, err = v.applicationRepo.GetByName(req.EnvironmentID, req.Name)
	if err == nil {
		return fmt.Errorf("应用名已存在: %s", req.Name)
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查应用名失败: %w", err)
	}

	// 6. 验证资源规格
	if req.ResourceSpec != nil {
		if err := v.validateResourceSpec(req.ResourceSpec); err != nil {
			return fmt.Errorf("资源规格错误: %w", err)
		}
	}

	return nil
}

// ValidateApplicationUpdate 验证应用更新
func (v *ConstraintValidator) ValidateApplicationUpdate(applicationID string, updates map[string]interface{}) error {
	// 1. 检查应用是否存在
	app, err := v.applicationRepo.GetByID(applicationID)
	if err != nil {
		return fmt.Errorf("应用不存在: %w", err)
	}

	// 2. 检查环境状态
	env, err := v.environmentRepo.GetByID(app.EnvironmentID.String())
	if err != nil {
		return fmt.Errorf("环境不存在: %w", err)
	}
	if env.Status != model.EnvironmentStatusActive {
		return fmt.Errorf("环境状态非活跃: %s", env.Status)
	}

	// 3. 检查资源规格变更
	if spec, ok := updates["resource_spec"]; ok {
		if appSpec, ok := spec.(*ApplicationResourceSpec); ok {
			if err := v.validateResourceSpec(appSpec); err != nil {
				return fmt.Errorf("资源规格错误: %w", err)
			}
		}
	}

	return nil
}

// ValidateApplicationDelete 验证应用删除
func (v *ConstraintValidator) ValidateApplicationDelete(applicationID string) error {
	// 1. 检查应用是否存在
	_, err := v.applicationRepo.GetByID(applicationID)
	if err != nil {
		return fmt.Errorf("应用不存在: %w", err)
	}

	// 2. 检查ApplicationResourceSpec
	_, err = v.quotaRepo.GetApplicationResourceSpecByApplicationID(applicationID)
	if err == nil {
		return fmt.Errorf("应用资源规格仍存在，需要先删除规格")
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查应用资源规格失败: %w", err)
	}

	return nil
}

// ValidateApplicationScale 验证应用扩缩容
func (v *ConstraintValidator) ValidateApplicationScale(applicationID string, replicas int) error {
	// 1. 检查应用是否存在
	application, err := v.applicationRepo.GetByID(applicationID)
	if err != nil {
		return fmt.Errorf("应用不存在: %w", err)
	}

	// 2. 获取应用资源规格
	resourceSpec, err := v.quotaRepo.GetApplicationResourceSpecByApplicationID(applicationID)
	if err != nil {
		return fmt.Errorf("获取应用资源规格失败: %w", err)
	}

	// 3. 检查副本数是否超限
	if replicas > resourceSpec.MaxReplicas {
		return fmt.Errorf("副本数 %d 超过最大限制 %d", replicas, resourceSpec.MaxReplicas)
	}

	// 4. 检查环境配额是否允许
	env, err := v.environmentRepo.GetByID(application.EnvironmentID.String())
	if err != nil {
		return fmt.Errorf("环境不存在: %w", err)
	}

	resourceQuota, err := v.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
	if err == nil {
		// 简单检查Pod数量
		if resourceQuota.HardLimits["pods"] != nil {
			maxPods := parseResourceQuantity(resourceQuota.HardLimits["pods"].(string))
			if int64(replicas) > maxPods {
				return fmt.Errorf("副本数 %d 超过环境配额限制 %d", replicas, maxPods)
			}
		}
	}

	return nil
}

// ===== 私有辅助方法 =====

// isValidTenantName 验证租户名格式
func isValidTenantName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	// 租户名只能包含字母、数字、连字符和下划线
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

// validateQuotaFormat 验证配额格式
func (v *ConstraintValidator) validateQuotaFormat(quota model.JSONMap) error {
	validKeys := []string{
		model.CommonQuotaKeys.RequestsCPU,
		model.CommonQuotaKeys.RequestsMemory,
		model.CommonQuotaKeys.LimitsCPU,
		model.CommonQuotaKeys.LimitsMemory,
		model.CommonQuotaKeys.Pods,
		model.CommonQuotaKeys.Services,
		model.CommonQuotaKeys.Secrets,
		model.CommonQuotaKeys.ConfigMaps,
	}

	for key := range quota {
		isValid := false
		for _, validKey := range validKeys {
			if key == validKey {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("无效的配额键: %s", key)
		}
	}

	return nil
}

// validateEnvironmentQuotaAgainstTenant 验证环境配额是否超过租户配额
func (v *ConstraintValidator) validateEnvironmentQuotaAgainstTenant(tenantID string, requestedQuota model.JSONMap) error {
	// 获取租户配额
	tenantQuota, err := v.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		// 租户配额不存在，允许创建（无限制）
		return nil
	}

	// 计算已分配配额
	allocated, err := v.calculateAllocatedQuota(tenantID)
	if err != nil {
		return fmt.Errorf("计算已分配配额失败: %w", err)
	}

	// 检查是否超过
	for resource, requested := range requestedQuota {
		if limit, ok := tenantQuota.HardLimits[resource]; ok {
			allocatedValue := getResourceValue(allocated, resource)
			requestedValue := getResourceValue(requestedQuota, resource)
			_ = requested // 标记为已使用
			totalUsage := allocatedValue + requestedValue
			limitValue := parseResourceQuantity(limit.(string))

			if totalUsage > limitValue {
				return fmt.Errorf("资源 %s 请求量 %d 超过租户配额限制 %d", resource, totalUsage, limitValue)
			}
		}
	}

	return nil
}

// validateResourceSpec 验证应用资源规格
func (v *ConstraintValidator) validateResourceSpec(spec *ApplicationResourceSpec) error {
	// 验证MaxReplicas
	if spec.MaxReplicas < 0 {
		return fmt.Errorf("最大副本数不能为负数")
	}

	// 验证资源格式
	if spec.DefaultRequest != nil {
		if err := v.validateResourceList(spec.DefaultRequest); err != nil {
			return fmt.Errorf("默认请求资源错误: %w", err)
		}
	}

	if spec.DefaultLimit != nil {
		if err := v.validateResourceList(spec.DefaultLimit); err != nil {
			return fmt.Errorf("默认限制资源错误: %w", err)
		}
	}

	return nil
}

// validateResourceList 验证资源列表格式
func (v *ConstraintValidator) validateResourceList(resources model.JSONMap) error {
	validKeys := []string{
		model.CommonQuotaKeys.RequestsCPU,
		model.CommonQuotaKeys.RequestsMemory,
		model.CommonQuotaKeys.LimitsCPU,
		model.CommonQuotaKeys.LimitsMemory,
	}

	for key := range resources {
		isValid := false
		for _, validKey := range validKeys {
			if key == validKey {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("无效的资源键: %s", key)
		}
	}

	return nil
}

// calculateAllocatedQuota 计算已分配配额
func (v *ConstraintValidator) calculateAllocatedQuota(tenantID string) (model.JSONMap, error) {
	envs, err := v.environmentRepo.ListByTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	allocated := make(model.JSONMap)

	for _, env := range envs {
		quota, err := v.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if err != nil {
			continue
		}

		for resource, value := range quota.HardLimits {
			currentValue := getResourceValue(allocated, resource)
			quotaValue := getResourceValue(quota.HardLimits, resource)
			_ = value // 标记为已使用
			allocated[resource] = currentValue + quotaValue
		}
	}

	return allocated, nil
}

// getResourceValue 获取资源值
func getResourceValue(quota model.JSONMap, resource string) int64 {
	if value, ok := quota[resource]; ok {
		if intVal, ok := value.(int64); ok {
			return intVal
		}
	}
	return 0
}

// parseResourceQuantity 解析资源数量
// 支持格式: 100m (milli CPU), 1 (CPU cores), 500Mi (memory), 2Gi (memory), 1000 (count)
func parseResourceQuantity(value string) int64 {
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
