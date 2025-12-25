package service

import (
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ApplicationService 应用服务
type ApplicationService struct {
	applicationRepo *repository.ApplicationRepository
	quotaRepo       *repository.QuotaRepository
	auditService    *AuditService
}

// NewApplicationService 创建应用服务
func NewApplicationService(
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	auditService *AuditService,
) *ApplicationService {
	return &ApplicationService{
		applicationRepo: applicationRepo,
		quotaRepo:       quotaRepo,
		auditService:    auditService,
	}
}

// CreateApplication 创建应用
func (s *ApplicationService) CreateApplication(req CreateApplicationRequest) (*model.Application, error) {
	var app *model.Application

	err := s.applicationRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		// 检查应用名是否已存在，使用SELECT FOR UPDATE锁定行
		var existing model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment_id = ? AND name = ?", req.EnvironmentID, req.Name).First(&existing).Error; err == nil {
			return ErrApplicationAlreadyExists(req.Name)
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		app = &model.Application{
			TenantID:      model.ParseUUID(req.TenantID),
			EnvironmentID: model.ParseUUID(req.EnvironmentID),
			Name:          req.Name,
			DisplayName:   req.DisplayName,
			Description:   req.Description,
			Labels:        req.Labels,
		}

		if err := tx.Create(app).Error; err != nil {
			return err
		}

		// 创建资源规格
		if req.ResourceSpec != nil {
			spec := &model.ApplicationResourceSpec{
				ApplicationID:  app.ID,
				DefaultRequest: req.ResourceSpec.DefaultRequest,
				DefaultLimit:   req.ResourceSpec.DefaultLimit,
				MaxReplicas:    req.ResourceSpec.MaxReplicas,
			}

			if err := tx.Create(spec).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.auditService != nil {
		s.auditService.LogCreate("application", app.ID.String(), "system", "", "", app)
	}

	return app, nil
}

// GetApplication 获取应用
func (s *ApplicationService) GetApplication(id string) (*model.Application, error) {
	return s.applicationRepo.GetByID(id)
}

// ListApplications 获取应用列表（分页）
func (s *ApplicationService) ListApplications(page, pageSize int, tenantID, environmentID string) ([]*model.Application, int64, error) {
	var apps []*model.Application
	var total int64

	query := s.applicationRepo.GetDB()

	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if environmentID != "" {
		query = query.Where("environment_id = ?", environmentID)
	}

	query.Model(&model.Application{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&apps).Error

	return apps, total, err
}

// UpdateApplication 更新应用
func (s *ApplicationService) UpdateApplication(id string, req UpdateApplicationRequest) (*model.Application, error) {
	app, err := s.applicationRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	oldValue := *app

	if req.DisplayName != "" {
		app.DisplayName = req.DisplayName
	}

	if req.Description != "" {
		app.Description = req.Description
	}

	if req.Labels != nil {
		app.Labels = req.Labels
	}

	if err := s.applicationRepo.Update(app); err != nil {
		return nil, err
	}

	if s.auditService != nil {
		s.auditService.LogUpdate("application", app.ID.String(), "system", "", "", oldValue, app)
	}

	return app, nil
}

// DeleteApplication 删除应用
func (s *ApplicationService) DeleteApplication(id string) error {
	app, err := s.applicationRepo.GetByID(id)
	if err != nil {
		return err
	}

	oldValue := *app

	if err := s.applicationRepo.Delete(id); err != nil {
		return err
	}

	if s.auditService != nil {
		s.auditService.LogDelete("application", app.ID.String(), "system", "", "", oldValue)
	}

	return nil
}

// ListApplicationsByTenantID 根据租户ID获取应用列表
func (s *ApplicationService) ListApplicationsByTenantID(tenantID string) ([]*model.Application, error) {
	return s.applicationRepo.ListByTenantID(tenantID)
}

// ListApplicationsByEnvironmentID 根据环境ID获取应用列表
func (s *ApplicationService) ListApplicationsByEnvironmentID(environmentID string) ([]*model.Application, error) {
	return s.applicationRepo.ListByEnvironmentID(environmentID)
}

// GetApplicationResourceSpec 获取应用资源规格
func (s *ApplicationService) GetApplicationResourceSpec(applicationID string) (*model.ApplicationResourceSpec, error) {
	return s.quotaRepo.GetApplicationResourceSpecByApplicationID(applicationID)
}

// UpdateApplicationResourceSpec 更新应用资源规格
func (s *ApplicationService) UpdateApplicationResourceSpec(applicationID string, req UpdateApplicationResourceSpecRequest) (*model.ApplicationResourceSpec, error) {
	spec, err := s.quotaRepo.GetApplicationResourceSpecByApplicationID(applicationID)
	if err != nil {
		return nil, err
	}

	if req.DefaultRequest != nil {
		spec.DefaultRequest = req.DefaultRequest
	}

	if req.DefaultLimit != nil {
		spec.DefaultLimit = req.DefaultLimit
	}

	if req.MaxReplicas > 0 {
		spec.MaxReplicas = req.MaxReplicas
	}

	if err := s.quotaRepo.UpdateApplicationResourceSpec(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// ScaleApplication 扩缩容应用
func (s *ApplicationService) ScaleApplication(applicationID string, replicas int) error {
	spec, err := s.quotaRepo.GetApplicationResourceSpecByApplicationID(applicationID)
	if err != nil {
		return err
	}

	spec.CurrentReplicas = replicas
	return s.quotaRepo.UpdateApplicationResourceSpec(spec)
}

// DiscoverApplications 发现应用
func (s *ApplicationService) DiscoverApplications(req DiscoverApplicationsRequest) ([]*model.Application, error) {
	var apps []*model.Application
	query := s.applicationRepo.GetDB()

	if req.EnvironmentID != "" {
		query = query.Where("environment_id = ?", req.EnvironmentID)
	}

	if req.ClusterID != "" {
		query = query.Joins("JOIN environments ON applications.environment_id = environments.id").
			Where("environments.cluster_id = ?", req.ClusterID)
	}

	if req.Labels != nil {
		for key, value := range req.Labels {
			query = query.Where("labels->? = ?", key, value)
		}
	}

	err := query.Find(&apps).Error
	return apps, err
}

// AggregateApplications 聚合应用
func (s *ApplicationService) AggregateApplications(req AggregateApplicationsRequest) (*AggregateApplicationsResult, error) {
	var apps []*model.Application
	query := s.applicationRepo.GetDB()

	if req.TenantID != "" {
		query = query.Where("tenant_id = ?", req.TenantID)
	}

	if req.EnvironmentID != "" {
		query = query.Where("environment_id = ?", req.EnvironmentID)
	}

	err := query.Find(&apps).Error
	if err != nil {
		return nil, err
	}

	result := &AggregateApplicationsResult{
		TotalApplications: len(apps),
		TotalReplicas:     0,
		Applications:      apps,
	}

	if len(apps) > 0 {
		applicationIDs := make([]string, 0, len(apps))
		for _, app := range apps {
			applicationIDs = append(applicationIDs, app.ID.String())
		}

		specs, specErr := s.quotaRepo.GetApplicationResourceSpecsByApplicationIDs(applicationIDs)
		if specErr == nil {
			for _, spec := range specs {
				result.TotalReplicas += spec.CurrentReplicas
			}
		}
	}

	return result, nil
}

// GetApplicationMetrics 获取应用指标
func (s *ApplicationService) GetApplicationMetrics(applicationID string) (*ApplicationMetrics, error) {
	app, err := s.applicationRepo.GetByID(applicationID)
	if err != nil {
		return nil, err
	}

	spec, specErr := s.quotaRepo.GetApplicationResourceSpecByApplicationID(applicationID)
	replicas := 0
	if specErr == nil {
		replicas = spec.CurrentReplicas
	}

	return &ApplicationMetrics{
		ApplicationID:   applicationID,
		ApplicationName: app.Name,
		Replicas:       replicas,
		Status:         constants.StatusActive,
	}, nil
}

// CreateApplicationRequest 创建应用请求
type CreateApplicationRequest struct {
	TenantID      string                   `json:"tenant_id" binding:"required,uuid"`
	EnvironmentID string                   `json:"environment_id" binding:"required,uuid"`
	Name          string                   `json:"name" binding:"required,min=1,max=63"`
	DisplayName   string                   `json:"display_name" binding:"required,min=1,max=100"`
	Description   string                   `json:"description" binding:"max=500"`
	Labels        model.JSONMap            `json:"labels"`
	ResourceSpec  *ApplicationResourceSpec `json:"resource_spec"`
}

// UpdateApplicationRequest 更新应用请求
type UpdateApplicationRequest struct {
	DisplayName string       `json:"display_name" binding:"omitempty,min=1,max=100"`
	Description string       `json:"description" binding:"omitempty,max=500"`
	Labels      model.JSONMap `json:"labels"`
}

// UpdateApplicationResourceSpecRequest 更新应用资源规格请求
type UpdateApplicationResourceSpecRequest struct {
	DefaultRequest model.JSONMap `json:"default_request" binding:"required"`
	DefaultLimit   model.JSONMap `json:"default_limit" binding:"required"`
	MaxReplicas    int           `json:"max_replicas" binding:"omitempty,min=1,max=1000"`
}

// DiscoverApplicationsRequest 发现应用请求
type DiscoverApplicationsRequest struct {
	EnvironmentID string       `json:"environment_id" binding:"omitempty,uuid"`
	ClusterID     string       `json:"cluster_id" binding:"omitempty,uuid"`
	Labels        model.JSONMap `json:"labels"`
}

// AggregateApplicationsRequest 聚合应用请求
type AggregateApplicationsRequest struct {
	TenantID      string `json:"tenant_id" binding:"omitempty,uuid"`
	EnvironmentID string `json:"environment_id" binding:"omitempty,uuid"`
}

// AggregateApplicationsResult 聚合应用结果
type AggregateApplicationsResult struct {
	TotalApplications int                   `json:"total_applications"`
	TotalReplicas     int                   `json:"total_replicas"`
	Applications      []*model.Application  `json:"applications"`
}

// ApplicationMetrics 应用指标
type ApplicationMetrics struct {
	ApplicationID   string `json:"application_id"`
	ApplicationName string `json:"application_name"`
	Replicas       int    `json:"replicas"`
	Status         string `json:"status"`
}

// ApplicationResourceSpec 应用资源规格
type ApplicationResourceSpec struct {
	DefaultRequest model.JSONMap `json:"default_request"`
	DefaultLimit   model.JSONMap `json:"default_limit"`
	MaxReplicas    int           `json:"max_replicas"`
}
