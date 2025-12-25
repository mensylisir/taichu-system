package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// ApplicationRepository 应用数据访问层
type ApplicationRepository struct {
	db *gorm.DB
}

// NewApplicationRepository 创建应用仓库
func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// GetDB 返回数据库实例
func (r *ApplicationRepository) GetDB() *gorm.DB {
	return r.db
}

// Create 创建应用
func (r *ApplicationRepository) Create(app *model.Application) error {
	return r.db.Create(app).Error
}

// GetByID 根据ID获取应用
func (r *ApplicationRepository) GetByID(id string) (*model.Application, error) {
	var app model.Application
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	err = r.db.Where("id = ?", idUUID).First(&app).Error
	return &app, err
}

// GetByName 根据名称获取应用
func (r *ApplicationRepository) GetByName(environmentID, name string) (*model.Application, error) {
	var app model.Application
	err := r.db.Where("environment_id = ? AND name = ?", environmentID, name).First(&app).Error
	return &app, err
}

// List 获取应用列表
func (r *ApplicationRepository) List() ([]*model.Application, error) {
	var apps []*model.Application
	err := r.db.Find(&apps).Error
	return apps, err
}

// ListAll 获取所有应用
func (r *ApplicationRepository) ListAll() ([]*model.Application, error) {
	var apps []*model.Application
	err := r.db.Find(&apps).Error
	return apps, err
}

// ListByTenantID 根据租户ID获取应用列表
func (r *ApplicationRepository) ListByTenantID(tenantID string) ([]*model.Application, error) {
	var apps []*model.Application
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}
	err = r.db.Where("tenant_id = ?", tenantUUID).Find(&apps).Error
	return apps, err
}

// ListByEnvironmentID 根据环境ID获取应用列表
func (r *ApplicationRepository) ListByEnvironmentID(environmentID string) ([]*model.Application, error) {
	var apps []*model.Application
	environmentUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, err
	}
	err = r.db.Where("environment_id = ?", environmentUUID).Find(&apps).Error
	return apps, err
}

// Update 更新应用
func (r *ApplicationRepository) Update(app *model.Application) error {
	return r.db.Save(app).Error
}

// Delete 删除应用
func (r *ApplicationRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM application_resource_specs WHERE application_id = ?", id).Error; err != nil {
			return err
		}

		if err := tx.Delete(&model.Application{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}

// DeleteByClusterID 根据集群ID删除应用
func (r *ApplicationRepository) DeleteByClusterID(clusterID string) error {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return err
	}

	// 先查询该集群下的所有环境
	var envs []*model.Environment
	r.db.Model(&model.Environment{}).Where("cluster_id = ?", clusterUUID).Find(&envs)

	var envIDs []uuid.UUID
	for _, env := range envs {
		envIDs = append(envIDs, env.ID)
	}

	if len(envIDs) > 0 {
		return r.db.Where("environment_id IN ?", envIDs).Delete(&model.Application{}).Error
	}
	return nil
}

// FindOrphanedApplications 查找孤立的应用（没有关联环境）
func (r *ApplicationRepository) FindOrphanedApplications() ([]*model.Application, error) {
	var apps []*model.Application
	err := r.db.Where("environment_id IS NULL OR environment_id = ?", "").Find(&apps).Error
	return apps, err
}
