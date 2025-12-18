package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type ImportRecordRepository struct {
	db *gorm.DB
}

func NewImportRecordRepository(db *gorm.DB) *ImportRecordRepository {
	return &ImportRecordRepository{db: db}
}

func (r *ImportRecordRepository) Create(importRecord *model.ImportRecord) error {
	return r.db.Create(importRecord).Error
}

func (r *ImportRecordRepository) GetByID(id string) (*model.ImportRecord, error) {
	var importRecord model.ImportRecord
	if err := r.db.First(&importRecord, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &importRecord, nil
}

func (r *ImportRecordRepository) List(importSource, status string) ([]*model.ImportRecord, int64, error) {
	var importRecords []*model.ImportRecord
	var total int64

	query := r.db.Model(&model.ImportRecord{})

	if importSource != "" {
		query = query.Where("import_source = ?", importSource)
	}
	if status != "" {
		query = query.Where("import_status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("imported_at DESC").Find(&importRecords).Error; err != nil {
		return nil, 0, err
	}

	return importRecords, total, nil
}

func (r *ImportRecordRepository) Update(importRecord *model.ImportRecord) error {
	return r.db.Save(importRecord).Error
}

func (r *ImportRecordRepository) Delete(id string) error {
	return r.db.Delete(&model.ImportRecord{}, "id = ?", id).Error
}

func (r *ImportRecordRepository) DeleteByClusterID(clusterID uuid.UUID) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.ImportRecord{}).Error
}
