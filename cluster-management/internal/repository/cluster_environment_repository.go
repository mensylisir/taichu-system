package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClusterEnvironmentRepository struct {
	db *gorm.DB
}

func NewClusterEnvironmentRepository(db *gorm.DB) *ClusterEnvironmentRepository {
	return &ClusterEnvironmentRepository{db: db}
}

func (r *ClusterEnvironmentRepository) Create(env *model.ClusterEnvironment) error {
	return r.db.Create(env).Error
}

func (r *ClusterEnvironmentRepository) GetByClusterID(clusterID string) (*model.ClusterEnvironment, error) {
	var env model.ClusterEnvironment
	if err := r.db.Where("cluster_id = ?", clusterID).First(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

func (r *ClusterEnvironmentRepository) ListByType(environmentType string) ([]*model.ClusterEnvironment, error) {
	var envs []*model.ClusterEnvironment
	if err := r.db.Where("environment_type = ?", environmentType).Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

func (r *ClusterEnvironmentRepository) UpdateClusterEnvironment(clusterID, environmentType, topologyType string, backupEnabled bool) error {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return err
	}

	// 使用Upsert操作
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cluster_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"environment_type", "topology_type", "backup_enabled"}),
	}).Create(&model.ClusterEnvironment{
		ClusterID:      clusterUUID,
		EnvironmentType: environmentType,
		TopologyType:    topologyType,
		BackupEnabled:   backupEnabled,
	}).Error
}

func (r *ClusterEnvironmentRepository) Delete(clusterID string) error {
	return r.db.Delete(&model.ClusterEnvironment{}, "cluster_id = ?", clusterID).Error
}

func (r *ClusterEnvironmentRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.ClusterEnvironment{}).Error
}
