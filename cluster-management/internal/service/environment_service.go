package service

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnvironmentService 环境服务
type EnvironmentService struct {
	environmentRepo *repository.EnvironmentRepository
	quotaRepo       *repository.QuotaRepository
	auditService    *AuditService
}

// NewEnvironmentService 创建环境服务
func NewEnvironmentService(
	environmentRepo *repository.EnvironmentRepository,
	quotaRepo *repository.QuotaRepository,
	auditService *AuditService,
) *EnvironmentService {
	return &EnvironmentService{
		environmentRepo: environmentRepo,
		quotaRepo:       quotaRepo,
		auditService:    auditService,
	}
}

// CreateEnvironment 创建环境
func (s *EnvironmentService) CreateEnvironment(req CreateEnvironmentRequest) (*model.Environment, error) {
	var env *model.Environment

	err := s.environmentRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		// 检查命名空间是否已存在，使用SELECT FOR UPDATE锁定行
		var existing model.Environment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("cluster_id = ? AND namespace = ?", req.ClusterID, req.Namespace).First(&existing).Error; err == nil {
			return ErrEnvironmentAlreadyExists(req.Namespace)
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		env = &model.Environment{
			TenantID:    model.ParseUUID(req.TenantID),
			ClusterID:   model.ParseUUID(req.ClusterID),
			Namespace:   req.Namespace,
			DisplayName: req.DisplayName,
			Description: req.Description,
			Labels:      req.Labels,
			Status:      model.EnvironmentStatusActive,
		}

		if err := tx.Create(env).Error; err != nil {
			return err
		}

		// 创建资源配额
		if req.Quota != nil {
			resourceQuota := &model.ResourceQuota{
				EnvironmentID: env.ID,
				HardLimits:    req.Quota,
				Status:        model.QuotaStatusActive,
			}

			if err := tx.Create(resourceQuota).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.auditService != nil {
		s.auditService.LogCreate("environment", env.ID.String(), "system", "", "", env)
	}

	return env, nil
}

// GetEnvironment 获取环境
func (s *EnvironmentService) GetEnvironment(id string) (*model.Environment, error) {
	return s.environmentRepo.GetByID(id)
}

// ListEnvironmentsByTenantID 根据租户ID获取环境列表
func (s *EnvironmentService) ListEnvironmentsByTenantID(tenantID string) ([]*model.Environment, error) {
	return s.environmentRepo.ListByTenantID(tenantID)
}

// ListEnvironmentsByClusterID 根据集群ID获取环境列表
func (s *EnvironmentService) ListEnvironmentsByClusterID(clusterID string) ([]*model.Environment, error) {
	return s.environmentRepo.ListByClusterID(clusterID)
}

// ListEnvironments 获取环境列表（分页）
func (s *EnvironmentService) ListEnvironments(page, pageSize int, tenantID, clusterID string) ([]*model.Environment, int64, error) {
	var envs []*model.Environment
	var total int64

	query := s.environmentRepo.GetDB()

	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	query.Model(&model.Environment{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&envs).Error

	return envs, total, err
}

// UpdateEnvironment 更新环境
func (s *EnvironmentService) UpdateEnvironment(id string, req UpdateEnvironmentRequest) (*model.Environment, error) {
	env, err := s.environmentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	oldValue := *env

	if req.DisplayName != "" {
		env.DisplayName = req.DisplayName
	}

	if req.Description != "" {
		env.Description = req.Description
	}

	if req.Labels != nil {
		env.Labels = req.Labels
	}

	if err := s.environmentRepo.Update(env); err != nil {
		return nil, err
	}

	if s.auditService != nil {
		s.auditService.LogUpdate("environment", env.ID.String(), "system", "", "", oldValue, env)
	}

	return env, nil
}

// DeleteEnvironment 删除环境
func (s *EnvironmentService) DeleteEnvironment(id string) error {
	env, err := s.environmentRepo.GetByID(id)
	if err != nil {
		return err
	}

	oldValue := *env

	if err := s.environmentRepo.Delete(id); err != nil {
		return err
	}

	if s.auditService != nil {
		s.auditService.LogDelete("environment", env.ID.String(), "system", "", "", oldValue)
	}

	return nil
}

// GetEnvironmentQuota 获取环境配额
func (s *EnvironmentService) GetEnvironmentQuota(environmentID string) (*model.ResourceQuota, error) {
	return s.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
}

// UpdateEnvironmentQuota 更新环境配额
func (s *EnvironmentService) UpdateEnvironmentQuota(environmentID string, req UpdateEnvironmentQuotaRequest) (*model.ResourceQuota, error) {
	quota, err := s.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}

	quota.HardLimits = req.HardLimits
	if err := s.quotaRepo.UpdateResourceQuota(quota); err != nil {
		return nil, err
	}

	return quota, nil
}

// SyncResourceQuota 同步资源配额
func (s *EnvironmentService) SyncResourceQuota(environmentID string) (*model.ResourceQuota, error) {
	_, err := s.environmentRepo.GetByID(environmentID)
	if err != nil {
		return nil, err
	}

	quota, err := s.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}

	quota.Status = model.QuotaStatusActive
	if err := s.quotaRepo.UpdateResourceQuota(quota); err != nil {
		return nil, err
	}

	return quota, nil
}

// GetNamespaceInfo 获取命名空间信息
func (s *EnvironmentService) GetNamespaceInfo(environmentID string) (*NamespaceInfo, error) {
	env, err := s.environmentRepo.GetByID(environmentID)
	if err != nil {
		return nil, err
	}

	quota, _ := s.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)

	return &NamespaceInfo{
		EnvironmentID: env.ID.String(),
		Namespace:     env.Namespace,
		ClusterID:     env.ClusterID.String(),
		TenantID:      env.TenantID.String(),
		Quota:         quota,
	}, nil
}

// GetTenantQuota 获取租户配额
func (s *EnvironmentService) GetTenantQuota(tenantID string) (*model.TenantQuota, error) {
	return s.quotaRepo.GetTenantQuotaByTenantID(tenantID)
}

// GetEnvironmentApplications 获取环境下的应用列表
func (s *EnvironmentService) GetEnvironmentApplications(environmentID string) ([]*model.Application, error) {
	var apps []*model.Application
	query := s.environmentRepo.GetDB()
	err := query.Where("environment_id = ?", environmentID).Find(&apps).Error
	return apps, err
}

// CreateEnvironmentRequest 创建环境请求
type CreateEnvironmentRequest struct {
	TenantID    string         `json:"tenant_id" binding:"required,uuid"`
	ClusterID   string         `json:"cluster_id" binding:"required,uuid"`
	Namespace   string         `json:"namespace" binding:"required,min=1,max=63"`
	DisplayName string         `json:"display_name" binding:"required,min=1,max=100"`
	Description string         `json:"description" binding:"max=500"`
	Labels      model.JSONMap  `json:"labels"`
	Quota       model.JSONMap  `json:"quota"`
}

// UpdateEnvironmentRequest 更新环境请求
type UpdateEnvironmentRequest struct {
	DisplayName string       `json:"display_name" binding:"omitempty,min=1,max=100"`
	Description string       `json:"description" binding:"omitempty,max=500"`
	Labels      model.JSONMap `json:"labels"`
}

// UpdateEnvironmentQuotaRequest 更新环境配额请求
type UpdateEnvironmentQuotaRequest struct {
	HardLimits model.JSONMap `json:"hard_limits" binding:"required"`
}

// NamespaceInfo 命名空间信息
type NamespaceInfo struct {
	EnvironmentID string             `json:"environment_id"`
	Namespace     string             `json:"namespace"`
	ClusterID     string             `json:"cluster_id"`
	TenantID      string             `json:"tenant_id"`
	Quota         *model.ResourceQuota `json:"quota"`
}
