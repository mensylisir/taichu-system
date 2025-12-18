package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type BackupScheduleRepository struct {
	db *gorm.DB
}

func NewBackupScheduleRepository(db *gorm.DB) *BackupScheduleRepository {
	return &BackupScheduleRepository{db: db}
}

func (r *BackupScheduleRepository) Create(schedule *model.BackupSchedule) error {
	return r.db.Create(schedule).Error
}

func (r *BackupScheduleRepository) GetByID(id string) (*model.BackupSchedule, error) {
	var schedule model.BackupSchedule
	if err := r.db.First(&schedule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *BackupScheduleRepository) ListByClusterID(clusterID string) ([]*model.BackupSchedule, error) {
	var schedules []*model.BackupSchedule
	if err := r.db.Where("cluster_id = ?", clusterID).Order("created_at DESC").Find(&schedules).Error; err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *BackupScheduleRepository) Update(schedule *model.BackupSchedule) error {
	return r.db.Save(schedule).Error
}

func (r *BackupScheduleRepository) Delete(id string) error {
	return r.db.Delete(&model.BackupSchedule{}, "id = ?", id).Error
}

func (r *BackupScheduleRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.BackupSchedule{}).Error
}
