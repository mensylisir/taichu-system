package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type BackupRepository struct {
	db *gorm.DB
}

func NewBackupRepository(db *gorm.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

func (r *BackupRepository) Create(backup *model.ClusterBackup) error {
	return r.db.Create(backup).Error
}

func (r *BackupRepository) GetByID(id string) (*model.ClusterBackup, error) {
	var backup model.ClusterBackup
	if err := r.db.First(&backup, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &backup, nil
}

func (r *BackupRepository) List(clusterID, status string, page, limit int) ([]*model.ClusterBackup, int64, error) {
	var backups []*model.ClusterBackup
	var total int64

	query := r.db.Model(&model.ClusterBackup{})

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&backups).Error; err != nil {
		return nil, 0, err
	}

	return backups, total, nil
}

func (r *BackupRepository) Update(backup *model.ClusterBackup) error {
	return r.db.Save(backup).Error
}

func (r *BackupRepository) Delete(id string) error {
	return r.db.Delete(&model.ClusterBackup{}, "id = ?", id).Error
}

func (r *BackupRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.ClusterBackup{}).Error
}
