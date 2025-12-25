package repository

import (
	"time"

	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type ClusterStateRepository struct {
	db *gorm.DB
}

func NewClusterStateRepository(db *gorm.DB) *ClusterStateRepository {
	return &ClusterStateRepository{db: db}
}

// GetDB 返回数据库实例
func (r *ClusterStateRepository) GetDB() *gorm.DB {
	return r.db
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
				"status":               state.Status,
				"node_count":           state.NodeCount,
				"kubernetes_version":   state.KubernetesVersion,
				"api_server_url":       state.APIServerURL,
				"last_heartbeat_at":    state.LastHeartbeatAt,
				"last_sync_at":         state.LastSyncAt,
				"sync_error":           state.SyncError,
				"sync_success":         state.SyncSuccess,
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

// UpdateStorageFields 只更新存储相关字段，不覆盖其他字段
func (r *ClusterStateRepository) UpdateStorageFields(clusterID string, totalStorageBytes int64, usedStorageBytes int64, storageUsagePercent float64) error {
	return r.db.Model(&model.ClusterState{}).
		Where("cluster_id = ?", clusterID).
		Updates(map[string]interface{}{
			"total_storage_bytes":   totalStorageBytes,
			"used_storage_bytes":    usedStorageBytes,
			"storage_usage_percent": storageUsagePercent,
			"updated_at":            time.Now(),
		}).Error
}

// UpdateAPIServerURL 只更新 APIServerURL 字段，不覆盖其他字段
func (r *ClusterStateRepository) UpdateAPIServerURL(clusterID string, apiServerURL string) error {
	return r.db.Model(&model.ClusterState{}).
		Where("cluster_id = ?", clusterID).
		Updates(map[string]interface{}{
			"api_server_url": apiServerURL,
			"updated_at":     time.Now(),
		}).Error
}

// UpdateStatus 只更新状态字段
func (r *ClusterStateRepository) UpdateStatus(clusterID string, status string, syncError string, lastSyncAt time.Time) error {
	updates := map[string]interface{}{
		"status":       status,
		"last_sync_at": lastSyncAt,
		"updated_at":   time.Now(),
	}
	if syncError != "" {
		updates["sync_error"] = syncError
		updates["sync_success"] = false
	} else {
		updates["sync_success"] = true
		updates["sync_error"] = ""
	}

	return r.db.Model(&model.ClusterState{}).
		Where("cluster_id = ?", clusterID).
		Updates(updates).Error
}

// UpdateSyncError 只更新同步错误字段
func (r *ClusterStateRepository) UpdateSyncError(clusterID string, syncError string) error {
	return r.db.Model(&model.ClusterState{}).
		Where("cluster_id = ?", clusterID).
		Updates(map[string]interface{}{
			"sync_error":   syncError,
			"sync_success": false,
			"updated_at":   time.Now(),
		}).Error
}

// UpdateHealthStatus 更新健康检查相关字段，不覆盖存储字段和资源字段
func (r *ClusterStateRepository) UpdateHealthStatus(
	clusterID string,
	status string,
	nodeCount int,
	totalCPUCores int,
	totalMemoryBytes int64,
	kubernetesVersion string,
	apiServerURL string,
	lastHeartbeatAt *time.Time,
) error {
	updates := map[string]interface{}{
		"status":             status,
		"node_count":         nodeCount,
		"kubernetes_version": kubernetesVersion,
		"api_server_url":     apiServerURL,
		"updated_at":         time.Now(),
	}

	if lastHeartbeatAt != nil {
		updates["last_heartbeat_at"] = *lastHeartbeatAt
	}

	return r.db.Model(&model.ClusterState{}).
		Where("cluster_id = ?", clusterID).
		Updates(updates).Error
}

// UpdateLastSyncAt 只更新最后同步时间
func (r *ClusterStateRepository) UpdateLastSyncAt(clusterID string, lastSyncAt time.Time) error {
	return r.db.Model(&model.ClusterState{}).
		Where("cluster_id = ?", clusterID).
		Updates(map[string]interface{}{
			"last_sync_at": lastSyncAt,
			"updated_at":   time.Now(),
		}).Error
}
