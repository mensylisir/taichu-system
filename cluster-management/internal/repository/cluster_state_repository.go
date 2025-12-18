package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type ClusterStateRepository struct {
	db *gorm.DB
}

func NewClusterStateRepository(db *gorm.DB) *ClusterStateRepository {
	return &ClusterStateRepository{db: db}
}

func (r *ClusterStateRepository) Create(state *model.ClusterState) error {
	return r.db.Create(state).Error
}

func (r *ClusterStateRepository) GetByClusterID(clusterID string) (*model.ClusterState, error) {
	var state model.ClusterState
	err := r.db.Where("cluster_id = ?", clusterID).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *ClusterStateRepository) Upsert(state *model.ClusterState) error {
	return r.db.Clauses(gorm.OnConflict{
		Columns:   []gorm.Column{{Name: "cluster_id"}},
		DoUpdates: gorm.AssignmentExpression{
			ColumnNames: []string{"status", "node_count", "total_cpu_cores", "total_memory_bytes",
				"kubernetes_version", "api_server_url", "last_heartbeat_at", "last_sync_at",
				"sync_error", "sync_success", "updated_at"},
			Values: []interface{}{state.Status, state.NodeCount, state.TotalCPUCores,
				state.TotalMemoryBytes, state.KubernetesVersion, state.APIServerURL,
				state.LastHeartbeatAt, state.LastSyncAt, state.SyncError, state.SyncSuccess, state.UpdatedAt},
		},
	}).Create(state).Error
}

func (r *ClusterStateRepository) Update(state *model.ClusterState) error {
	return r.db.Save(state).Error
}

func (r *ClusterStateRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.ClusterState{}).Error
}

func (r *ClusterStateRepository) List() ([]*model.ClusterState, error) {
	var states []*model.ClusterState
	err := r.db.Find(&states).Error
	return states, err
}

func (r *ClusterStateRepository) ListByStatus(status string) ([]*model.ClusterState, error) {
	var states []*model.ClusterState
	err := r.db.Where("status = ?", status).Find(&states).Error
	return states, err
}

func (r *ClusterStateRepository) ListFailedSync() ([]*model.ClusterState, error) {
	var states []*model.ClusterState
	err := r.db.Where("sync_success = ?", false).Find(&states).Error
	return states, err
}

func (r *ClusterStateRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.ClusterState{}).Count(&count).Error
	return count, err
}

func (r *ClusterStateRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.ClusterState{}).Where("status = ?", status).Count(&count).Error
	return count, err
}
