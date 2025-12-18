package repository

import (
	"time"

	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(event *model.Event) error {
	return r.db.Create(event).Error
}

func (r *EventRepository) BulkCreate(events []model.Event) error {
	if len(events) == 0 {
		return nil
	}
	return r.db.CreateInBatches(events, 100).Error
}

func (r *EventRepository) List(clusterID, severity string, since *time.Time) ([]*model.Event, int64, error) {
	var events []*model.Event
	var total int64

	query := r.db.Model(&model.Event{})

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if since != nil {
		query = query.Where("last_timestamp >= ?", *since)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("last_timestamp DESC").Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *EventRepository) GetByClusterID(clusterID string, limit int) ([]model.Event, int64, error) {
	var events []model.Event
	var total int64

	r.db.Model(&model.Event{}).Where("cluster_id = ?", clusterID).Count(&total)

	err := r.db.Where("cluster_id = ?", clusterID).
		Order("last_timestamp DESC").
		Limit(limit).
		Find(&events).Error

	return events, total, err
}

func (r *EventRepository) GetBySeverity(clusterID string, severity string, limit int) ([]model.Event, int64, error) {
	var events []model.Event
	var total int64

	r.db.Model(&model.Event{}).Where("cluster_id = ? AND severity = ?", clusterID, severity).Count(&total)

	err := r.db.Where("cluster_id = ? AND severity = ?", clusterID, severity).
		Order("last_timestamp DESC").
		Limit(limit).
		Find(&events).Error

	return events, total, err
}

func (r *EventRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.Event{}).Error
}

// CheckExists 检查事件是否已存在
func (r *EventRepository) CheckExists(clusterID string, eventType, message, component string, firstTimestamp, lastTimestamp time.Time) (bool, error) {
	var count int64
	err := r.db.Model(&model.Event{}).
		Where("cluster_id = ? AND event_type = ? AND message = ? AND component = ? AND first_timestamp = ? AND last_timestamp = ?",
			clusterID, eventType, message, component, firstTimestamp, lastTimestamp).
		Count(&count).Error
	return count > 0, err
}
