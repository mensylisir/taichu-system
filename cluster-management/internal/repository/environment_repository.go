package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// EnvironmentRepository 环境数据访问层
type EnvironmentRepository struct {
	db *gorm.DB
}

// NewEnvironmentRepository 创建环境仓库
func NewEnvironmentRepository(db *gorm.DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

// Create 创建环境
func (r *EnvironmentRepository) Create(env *model.Environment) error {
	return r.db.Create(env).Error
}

// GetByID 根据ID获取环境
func (r *EnvironmentRepository) GetByID(id string) (*model.Environment, error) {
	var env model.Environment
	err := r.db.Where("id = ?", id).First(&env).Error
	return &env, err
}

// GetByNamespace 根据Namespace获取环境
func (r *EnvironmentRepository) GetByNamespace(clusterID, namespace string) (*model.Environment, error) {
	var env model.Environment
	err := r.db.Where("cluster_id = ? AND namespace = ?", clusterID, namespace).First(&env).Error
	return &env, err
}

// ListByTenantID 根据租户ID获取环境列表
func (r *EnvironmentRepository) ListByTenantID(tenantID string) ([]*model.Environment, error) {
	var envs []*model.Environment
	err := r.db.Where("tenant_id = ?", tenantID).Find(&envs).Error
	return envs, err
}

// ListByClusterID 根据集群ID获取环境列表
func (r *EnvironmentRepository) ListByClusterID(clusterID string) ([]*model.Environment, error) {
	var envs []*model.Environment
	err := r.db.Where("cluster_id = ?", clusterID).Find(&envs).Error
	return envs, err
}

// Update 更新环境
func (r *EnvironmentRepository) Update(env *model.Environment) error {
	return r.db.Save(env).Error
}

// Delete 删除环境
func (r *EnvironmentRepository) Delete(id string) error {
	return r.db.Delete(&model.Environment{}, "id = ?", id).Error
}

// DeleteByClusterID 根据集群ID删除环境
func (r *EnvironmentRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.Environment{}).Error
}

// ListAll 获取所有环境（分页）
func (r *EnvironmentRepository) ListAll(page, pageSize int) ([]*model.Environment, int64, error) {
	var envs []*model.Environment
	var total int64

	query := r.db.Model(&model.Environment{})

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&envs).Error

	return envs, total, err
}

// FindOrphanedEnvironments 查找孤立的环境（没有关联租户）
func (r *EnvironmentRepository) FindOrphanedEnvironments() ([]*model.Environment, error) {
	var envs []*model.Environment
	err := r.db.Where("tenant_id IS NULL OR tenant_id = ?", "").Find(&envs).Error
	return envs, err
}

// GetDB 获取数据库连接
func (r *EnvironmentRepository) GetDB() *gorm.DB {
	return r.db
}
