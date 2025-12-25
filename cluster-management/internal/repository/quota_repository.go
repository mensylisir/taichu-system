package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// QuotaRepository 配额数据访问层
type QuotaRepository struct {
	db *gorm.DB
}

// NewQuotaRepository 创建配额仓库
func NewQuotaRepository(db *gorm.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// ===== ResourceQuota 操作 =====

// CreateResourceQuota 创建资源配额
func (r *QuotaRepository) CreateResourceQuota(quota *model.ResourceQuota) error {
	return r.db.Create(quota).Error
}

// GetResourceQuotaByEnvironmentID 根据环境ID获取资源配额
func (r *QuotaRepository) GetResourceQuotaByEnvironmentID(environmentID string) (*model.ResourceQuota, error) {
	var quota model.ResourceQuota
	err := r.db.Where("environment_id = ?", environmentID).First(&quota).Error
	return &quota, err
}

// UpdateResourceQuota 更新资源配额
func (r *QuotaRepository) UpdateResourceQuota(quota *model.ResourceQuota) error {
	return r.db.Save(quota).Error
}

// DeleteResourceQuota 删除资源配额
func (r *QuotaRepository) DeleteResourceQuota(environmentID string) error {
	return r.db.Delete(&model.ResourceQuota{}, "environment_id = ?", environmentID).Error
}

// ===== TenantQuota 操作 =====

// CreateTenantQuota 创建租户配额
func (r *QuotaRepository) CreateTenantQuota(quota *model.TenantQuota) error {
	return r.db.Create(quota).Error
}

// GetTenantQuotaByTenantID 根据租户ID获取租户配额
func (r *QuotaRepository) GetTenantQuotaByTenantID(tenantID string) (*model.TenantQuota, error) {
	var quota model.TenantQuota
	err := r.db.Where("tenant_id = ?", tenantID).First(&quota).Error
	return &quota, err
}

// UpdateTenantQuota 更新租户配额
func (r *QuotaRepository) UpdateTenantQuota(quota *model.TenantQuota) error {
	return r.db.Save(quota).Error
}

// DeleteTenantQuota 删除租户配额
func (r *QuotaRepository) DeleteTenantQuota(tenantID string) error {
	return r.db.Delete(&model.TenantQuota{}, "tenant_id = ?", tenantID).Error
}

// ===== ApplicationResourceSpec 操作 =====

// CreateApplicationResourceSpec 创建应用资源规格
func (r *QuotaRepository) CreateApplicationResourceSpec(spec *model.ApplicationResourceSpec) error {
	return r.db.Create(spec).Error
}

// GetApplicationResourceSpecByApplicationID 根据应用ID获取资源规格
func (r *QuotaRepository) GetApplicationResourceSpecByApplicationID(applicationID string) (*model.ApplicationResourceSpec, error) {
	var spec model.ApplicationResourceSpec
	err := r.db.Where("application_id = ?", applicationID).First(&spec).Error
	return &spec, err
}

// GetApplicationResourceSpecsByApplicationIDs 根据应用ID列表批量获取资源规格
func (r *QuotaRepository) GetApplicationResourceSpecsByApplicationIDs(applicationIDs []string) ([]*model.ApplicationResourceSpec, error) {
	var specs []*model.ApplicationResourceSpec
	err := r.db.Where("application_id IN ?", applicationIDs).Find(&specs).Error
	return specs, err
}

// UpdateApplicationResourceSpec 更新应用资源规格
func (r *QuotaRepository) UpdateApplicationResourceSpec(spec *model.ApplicationResourceSpec) error {
	return r.db.Save(spec).Error
}

// DeleteApplicationResourceSpec 删除应用资源规格
func (r *QuotaRepository) DeleteApplicationResourceSpec(applicationID string) error {
	return r.db.Delete(&model.ApplicationResourceSpec{}, "application_id = ?", applicationID).Error
}
