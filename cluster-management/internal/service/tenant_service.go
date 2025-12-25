package service

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantService 租户服务
type TenantService struct {
	tenantRepo     *repository.TenantRepository
	quotaRepo      *repository.QuotaRepository
	environmentRepo *repository.EnvironmentRepository
	auditService   *AuditService
}

// NewTenantService 创建租户服务
func NewTenantService(
	tenantRepo *repository.TenantRepository,
	quotaRepo *repository.QuotaRepository,
	environmentRepo *repository.EnvironmentRepository,
	auditService *AuditService,
) *TenantService {
	return &TenantService{
		tenantRepo:     tenantRepo,
		quotaRepo:      quotaRepo,
		environmentRepo: environmentRepo,
		auditService:   auditService,
	}
}

// CreateTenant 创建租户
func (s *TenantService) CreateTenant(req CreateTenantRequest) (*model.Tenant, error) {
	var tenant *model.Tenant

	err := s.tenantRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		// 检查租户名是否已存在，使用SELECT FOR UPDATE锁定行
		var existing model.Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", req.Name).First(&existing).Error; err == nil {
			return ErrTenantAlreadyExists(req.Name)
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		// 创建租户
		tenant = &model.Tenant{
			Name:        req.Name,
			DisplayName: req.DisplayName,
			Type:        model.TenantTypeUserCreated,
			Description: req.Description,
			Labels:      req.Labels,
			Status:      model.TenantStatusActive,
			IsSystem:    false,
		}

		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		// 创建租户配额
		if req.Quota != nil {
			tenantQuota := &model.TenantQuota{
				TenantID:   tenant.ID,
				HardLimits: req.Quota,
				Status:     model.QuotaStatusActive,
			}

			if err := tx.Create(tenantQuota).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.auditService != nil {
		s.auditService.LogCreate("tenant", tenant.ID.String(), "system", "", "", tenant)
	}

	return tenant, nil
}

// GetTenant 获取租户
func (s *TenantService) GetTenant(id string) (*model.Tenant, error) {
	return s.tenantRepo.GetByID(id)
}

// ListTenants 获取租户列表
func (s *TenantService) ListTenants() ([]*model.Tenant, error) {
	return s.tenantRepo.List()
}

// ListTenantsPaginated 获取租户列表（分页）
func (s *TenantService) ListTenantsPaginated(page, pageSize int, tenantType string) ([]*model.Tenant, int64, error) {
	var tenants []*model.Tenant
	var total int64

	query := s.tenantRepo.GetDB()

	if tenantType != "" {
		query = query.Where("type = ?", tenantType)
	}

	query.Model(&model.Tenant{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&tenants).Error

	return tenants, total, err
}

// UpdateTenant 更新租户
func (s *TenantService) UpdateTenant(id string, req UpdateTenantRequest) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	oldValue := *tenant

	if req.DisplayName != "" {
		tenant.DisplayName = req.DisplayName
	}

	if req.Description != "" {
		tenant.Description = req.Description
	}

	if req.Labels != nil {
		tenant.Labels = req.Labels
	}

	if err := s.tenantRepo.Update(tenant); err != nil {
		return nil, err
	}

	if s.auditService != nil {
		s.auditService.LogUpdate("tenant", tenant.ID.String(), "system", "", "", oldValue, tenant)
	}

	return tenant, nil
}

// DeleteTenant 删除租户
func (s *TenantService) DeleteTenant(id string) error {
	tenant, err := s.tenantRepo.GetByID(id)
	if err != nil {
		return err
	}

	if tenant.IsSystem {
		return ErrSystemTenantCannotBeDeleted
	}

	oldValue := *tenant

	if err := s.tenantRepo.Delete(id); err != nil {
		return err
	}

	if s.auditService != nil {
		s.auditService.LogDelete("tenant", tenant.ID.String(), "system", "", "", oldValue)
	}

	return nil
}

// GetTenantQuota 获取租户配额
func (s *TenantService) GetTenantQuota(tenantID string) (*model.TenantQuota, error) {
	return s.quotaRepo.GetTenantQuotaByTenantID(tenantID)
}

// UpdateTenantQuota 更新租户配额
func (s *TenantService) UpdateTenantQuota(tenantID string, req UpdateTenantQuotaRequest) (*model.TenantQuota, error) {
	quota, err := s.quotaRepo.GetTenantQuotaByTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	quota.HardLimits = req.HardLimits
	if err := s.quotaRepo.UpdateTenantQuota(quota); err != nil {
		return nil, err
	}

	return quota, nil
}

// InitializeSystemTenants 初始化系统租户
func (s *TenantService) InitializeSystemTenants() error {
	return s.tenantRepo.EnsurePredefinedTenants()
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name        string         `json:"name" binding:"required,min=3,max=50"`
	DisplayName string         `json:"display_name" binding:"required,min=1,max=100"`
	Description string         `json:"description" binding:"max=500"`
	Labels      model.JSONMap  `json:"labels"`
	Quota       model.JSONMap  `json:"quota"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	DisplayName string       `json:"display_name" binding:"omitempty,min=1,max=100"`
	Description string       `json:"description" binding:"omitempty,max=500"`
	Labels      model.JSONMap `json:"labels"`
}

// UpdateTenantQuotaRequest 更新租户配额请求
type UpdateTenantQuotaRequest struct {
	HardLimits model.JSONMap `json:"hard_limits" binding:"required"`
}

// GetTenantEnvironments 获取租户下的环境列表
func (s *TenantService) GetTenantEnvironments(tenantID string) ([]*model.Environment, error) {
	return s.environmentRepo.ListByTenantID(tenantID)
}

// GetPredefinedTenants 获取预定义租户列表
func (s *TenantService) GetPredefinedTenants() ([]*model.Tenant, error) {
	tenants, err := s.tenantRepo.List()
	if err != nil {
		return nil, err
	}

	var predefined []*model.Tenant
	for _, tenant := range tenants {
		if tenant.IsSystem {
			predefined = append(predefined, tenant)
		}
	}

	return predefined, nil
}

// InitializePredefinedTenants 初始化预定义租户
func (s *TenantService) InitializePredefinedTenants() error {
	return s.tenantRepo.EnsurePredefinedTenants()
}
