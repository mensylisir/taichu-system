package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type ClusterResourceRepository struct {
	db *gorm.DB
}

func NewClusterResourceRepository(db *gorm.DB) *ClusterResourceRepository {
	return &ClusterResourceRepository{db: db}
}

func (r *ClusterResourceRepository) Create(resource *model.ClusterResource) error {
	return r.db.Create(resource).Error
}

func (r *ClusterResourceRepository) GetLatestByClusterID(clusterID string) (*model.ClusterResource, error) {
	var resource model.ClusterResource
	err := r.db.Where("cluster_id = ?", clusterID).
		Order("timestamp DESC").
		First(&resource).Error
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *ClusterResourceRepository) GetHistory(clusterID string, limit int) ([]model.ClusterResource, error) {
	var resources []model.ClusterResource
	err := r.db.Where("cluster_id = ?", clusterID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&resources).Error
	return resources, err
}

func (r *ClusterResourceRepository) DeleteOldResources(clusterID string, days int) error {
	return r.db.Where("cluster_id = ? AND timestamp < NOW() - INTERVAL '%d days'", clusterID, days).
		Delete(&model.ClusterResource{}).Error
}

// Upsert 创建或更新集群资源记录
// 对于每个集群，我们只保留最新的资源记录
func (r *ClusterResourceRepository) Upsert(resource *model.ClusterResource) error {
	// 检查是否已有该集群的资源记录
	var existingResource model.ClusterResource
	err := r.db.Where("cluster_id = ?", resource.ClusterID).
		Order("timestamp DESC").
		First(&existingResource).Error
	
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	
	// 如果已有记录，更新它
	if err == nil {
		// 保留原有ID和时间戳
		resource.ID = existingResource.ID
		// 更新时间戳为当前时间
		resource.Timestamp = time.Now()
		return r.db.Model(&existingResource).Where("id = ?", existingResource.ID).Updates(resource).Error
	}
	
	// 如果没有记录，创建新记录
	if resource.ID == uuid.Nil {
		resource.ID = uuid.New()
	}
	return r.db.Create(resource).Error
}
