package repository

import (
	"context"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type SecurityPolicyRepository struct {
	db *gorm.DB
}

func NewSecurityPolicyRepository(db *gorm.DB) *SecurityPolicyRepository {
	return &SecurityPolicyRepository{db: db}
}

func (r *SecurityPolicyRepository) Upsert(ctx context.Context, policy *model.SecurityPolicy) error {
	// 检查是否已存在该集群的策略
	var existingPolicy model.SecurityPolicy
	err := r.db.WithContext(ctx).Where("cluster_id = ?", policy.ClusterID).First(&existingPolicy).Error
	
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	
	// 如果已存在记录，更新它
	if err == nil {
		// 保留原有ID和创建时间戳
		policy.ID = existingPolicy.ID
		policy.CreatedAt = existingPolicy.CreatedAt
		// 只更新非主键字段
		return r.db.WithContext(ctx).Model(&existingPolicy).Where("id = ?", existingPolicy.ID).Omit("id", "created_at").Updates(policy).Error
	}
	
	// 如果没有记录，创建新记录
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *SecurityPolicyRepository) GetByClusterID(ctx context.Context, clusterID string) (*model.SecurityPolicy, error) {
	var policy model.SecurityPolicy
	err := r.db.WithContext(ctx).Where("cluster_id = ?", clusterID).First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *SecurityPolicyRepository) DeleteByClusterID(ctx context.Context, clusterID string) error {
	return r.db.WithContext(ctx).Where("cluster_id = ?", clusterID).Delete(&model.SecurityPolicy{}).Error
}