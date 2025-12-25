package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// ResourceClassificationRepository 资源分类仓库
type ResourceClassificationRepository struct {
	db *gorm.DB
}

// NewResourceClassificationRepository 创建资源分类仓库
func NewResourceClassificationRepository(db *gorm.DB) *ResourceClassificationRepository {
	return &ResourceClassificationRepository{db: db}
}

// Create 创建分类记录
func (r *ResourceClassificationRepository) Create(classification *model.ResourceClassification) error {
	return r.db.Create(classification).Error
}

// GetByID 根据ID获取分类记录
func (r *ResourceClassificationRepository) GetByID(id uuid.UUID) (*model.ResourceClassification, error) {
	var classification model.ResourceClassification
	err := r.db.Where("id = ?", id).First(&classification).Error
	return &classification, err
}

// GetByResource 根据资源获取分类记录
func (r *ResourceClassificationRepository) GetByResource(clusterID, resourceType, resourceName string) (*model.ResourceClassification, error) {
	var classification model.ResourceClassification
	err := r.db.Where("cluster_id = ? AND resource_type = ? AND resource_name = ?", clusterID, resourceType, resourceName).First(&classification).Error
	return &classification, err
}

// ListByCluster 根据集群ID获取分类记录列表
func (r *ResourceClassificationRepository) ListByCluster(clusterID string) ([]*model.ResourceClassification, error) {
	var classifications []*model.ResourceClassification
	err := r.db.Where("cluster_id = ?", clusterID).Find(&classifications).Error
	return classifications, err
}

// ListByStatus 根据状态获取分类记录列表
func (r *ResourceClassificationRepository) ListByStatus(status string) ([]*model.ResourceClassification, error) {
	var classifications []*model.ResourceClassification
	err := r.db.Where("status = ?", status).Find(&classifications).Error
	return classifications, err
}

// Update 更新分类记录
func (r *ResourceClassificationRepository) Update(classification *model.ResourceClassification) error {
	return r.db.Save(classification).Error
}

// Delete 删除分类记录
func (r *ResourceClassificationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.ResourceClassification{}, id).Error
}

// CreateHistory 创建分类历史记录
func (r *ResourceClassificationRepository) CreateHistory(history *model.ResourceClassificationHistory) error {
	return r.db.Create(history).Error
}

// GetLatestHistory 获取最新的分类历史
func (r *ResourceClassificationRepository) GetLatestHistory(clusterID string) (*model.ResourceClassificationHistory, error) {
	var history model.ResourceClassificationHistory
	err := r.db.Where("cluster_id = ?", clusterID).Order("created_at DESC").First(&history).Error
	return &history, err
}

// ListHistoryByCluster 根据集群ID获取分类历史列表
func (r *ResourceClassificationRepository) ListHistoryByCluster(clusterID string, limit int) ([]*model.ResourceClassificationHistory, error) {
	var histories []*model.ResourceClassificationHistory
	query := r.db.Where("cluster_id = ?", clusterID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&histories).Error
	return histories, err
}

// UpdateHistory 更新分类历史记录
func (r *ResourceClassificationRepository) UpdateHistory(history *model.ResourceClassificationHistory) error {
	return r.db.Save(history).Error
}

// GetDB 获取数据库连接
func (r *ResourceClassificationRepository) GetDB() *gorm.DB {
	return r.db
}
