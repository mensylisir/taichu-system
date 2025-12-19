package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type AutoscalingPolicyRepository struct {
	db *gorm.DB
}

func NewAutoscalingPolicyRepository(db *gorm.DB) *AutoscalingPolicyRepository {
	return &AutoscalingPolicyRepository{db: db}
}

func (r *AutoscalingPolicyRepository) Upsert(policy *model.AutoscalingPolicy) error {
	// 检查是否已存在该集群的策略
	var existingPolicy model.AutoscalingPolicy
	err := r.db.Where("cluster_id = ?", policy.ClusterID).First(&existingPolicy).Error
	
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	
	// 如果已存在记录，更新它
	if err == nil {
		// 保留原有ID和创建时间戳
		policy.ID = existingPolicy.ID
		policy.CreatedAt = existingPolicy.CreatedAt
		return r.db.Model(&existingPolicy).Where("id = ?", existingPolicy.ID).Updates(policy).Error
	}
	
	// 如果没有记录，创建新记录
	return r.db.Create(policy).Error
}

func (r *AutoscalingPolicyRepository) GetByClusterID(clusterID string) (*model.AutoscalingPolicy, error) {
	var policy model.AutoscalingPolicy
	err := r.db.Where("cluster_id = ?", clusterID).First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *AutoscalingPolicyRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.AutoscalingPolicy{}).Error
}
