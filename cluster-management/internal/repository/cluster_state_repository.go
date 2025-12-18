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
	result := r.db.Where("cluster_id = ?", state.ClusterID).First(&model.ClusterState{})
	if result.Error == nil {
		return r.db.Model(&model.ClusterState{}).
			Where("cluster_id = ?", state.ClusterID).
			Updates(map[string]interface{}{
				"status":              state.Status,
				"node_count":          state.NodeCount,
				"total_cpu_cores":     state.TotalCPUCores,
				"total_memory_bytes":  state.TotalMemoryBytes,
				"kubernetes_version":  state.KubernetesVersion,
				"api_server_url":      state.APIServerURL,
				"last_heartbeat_at":   state.LastHeartbeatAt,
				"last_sync_at":        state.LastSyncAt,
				"sync_error":          state.SyncError,
				"sync_success":        state.SyncSuccess,
				"updated_at":          state.UpdatedAt,
			}).Error
	}
	return r.db.Create(state).Error
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
