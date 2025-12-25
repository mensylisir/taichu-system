package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type AlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Create(alert *model.Alert) error {
	return r.db.Create(alert).Error
}

func (r *AlertRepository) GetByID(id string) (*model.Alert, error) {
	var alert model.Alert
	err := r.db.Where("id = ?", id).First(&alert).Error
	return &alert, err
}

func (r *AlertRepository) List(page, pageSize int, alertType model.AlertType, severity model.AlertSeverity, status model.AlertStatus, resourceType, resourceID string) ([]*model.Alert, int64, error) {
	var alerts []*model.Alert
	var total int64

	query := r.db.Model(&model.Alert{})

	if alertType != "" {
		query = query.Where("type = ?", alertType)
	}

	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	if resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&alerts).Error

	return alerts, total, err
}

func (r *AlertRepository) Update(alert *model.Alert) error {
	return r.db.Save(alert).Error
}

func (r *AlertRepository) Resolve(id string) error {
	return r.db.Model(&model.Alert{}).Where("id = ?", id).Update("status", model.AlertStatusResolved).Error
}

func (r *AlertRepository) GetActiveAlerts() ([]*model.Alert, error) {
	var alerts []*model.Alert
	err := r.db.Where("status = ?", model.AlertStatusActive).Order("created_at DESC").Find(&alerts).Error
	return alerts, err
}

func (r *AlertRepository) GetDB() *gorm.DB {
	return r.db
}
