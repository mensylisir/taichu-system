package service

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

// QuotaService 配额服务
type QuotaService struct {
	kubernetesClient *kubernetes.Clientset
	quotaRepo        *repository.QuotaRepository
	environmentRepo  *repository.EnvironmentRepository
	tenantRepo       *repository.TenantRepository
}

// NewQuotaService 创建配额服务
func NewQuotaService(
	kubeClient *kubernetes.Clientset,
	quotaRepo *repository.QuotaRepository,
	environmentRepo *repository.EnvironmentRepository,
	tenantRepo *repository.TenantRepository,
) *QuotaService {
	return &QuotaService{
		kubernetesClient: kubeClient,
		quotaRepo:        quotaRepo,
		environmentRepo:  environmentRepo,
		tenantRepo:       tenantRepo,
	}
}

// SyncResourceQuota 同步资源配额
func (s *QuotaService) SyncResourceQuota(environmentID string) error {
	ctx := context.Background()

	// 获取环境信息
	env, err := s.environmentRepo.GetByID(environmentID)
	if err != nil {
		return err
	}

	// 从K8s获取ResourceQuota
	quota, err := s.kubernetesClient.CoreV1().ResourceQuotas(env.Namespace).Get(ctx, "", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get resource quota for namespace %s: %w", env.Namespace, err)
	}

	// 转换格式
	hardLimits := make(map[string]interface{})
	for k, v := range quota.Status.Hard {
		hardLimits[string(k)] = v.String()
	}

	used := make(map[string]interface{})
	for k, v := range quota.Status.Used {
		used[string(k)] = v.String()
	}

	// 更新数据库
	resourceQuota, err := s.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err != nil {
		// 创建新记录
		now := time.Now()
		resourceQuota = &model.ResourceQuota{
			EnvironmentID: model.ParseUUID(environmentID),
			HardLimits:    hardLimits,
			Used:          used,
			Status:        "active",
			LastSyncedAt:  &now,
		}
		return s.quotaRepo.CreateResourceQuota(resourceQuota)
	}

	// 更新现有记录
	now := time.Now()
	resourceQuota.HardLimits = hardLimits
	resourceQuota.Used = used
	resourceQuota.Status = constants.StatusActive
	resourceQuota.LastSyncedAt = &now

	return s.quotaRepo.UpdateResourceQuota(resourceQuota)
}

// CalculateTenantQuota 计算租户配额使用情况
func (s *QuotaService) CalculateTenantQuota(tenantID string) error {
	// 获取租户
	tenant, err := s.tenantRepo.GetByID(tenantID)
	if err != nil {
		return err
	}

	// 获取租户下的所有环境
	envs, err := s.environmentRepo.ListByTenantID(tenantID)
	if err != nil {
		return err
	}

	// 汇总所有环境的配额
	totalAllocated := make(map[string]int64)
	for _, env := range envs {
		quota, err := s.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if err != nil {
			continue
		}

		for key, value := range quota.HardLimits {
			if intVal, ok := value.(string); ok {
				// 解析资源值 (简化处理，实际需要更复杂的解析逻辑)
				totalAllocated[key] += parseResourceQuantity(intVal)
			}
		}
	}

	// 转换为JSONMap
	allocatedMap := make(model.JSONMap)
	for k, v := range totalAllocated {
		allocatedMap[k] = v
	}

	// 更新租户配额
	tenantQuota, err := s.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		// 创建新记录
		tenantQuota = &model.TenantQuota{
			TenantID:   tenant.ID,
			HardLimits: make(model.JSONMap),
			Allocated:  allocatedMap,
			Status:     model.QuotaStatusActive,
		}
		return s.quotaRepo.CreateTenantQuota(tenantQuota)
	}

	tenantQuota.Allocated = allocatedMap
	return s.quotaRepo.UpdateTenantQuota(tenantQuota)
}

// GetResourceQuota 获取资源配额
func (s *QuotaService) GetResourceQuota(environmentID string) (*model.ResourceQuota, error) {
	return s.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
}

// GetTenantQuota 获取租户配额
func (s *QuotaService) GetTenantQuota(tenantID string) (*model.TenantQuota, error) {
	return s.quotaRepo.GetTenantQuotaByTenantID(tenantID)
}
