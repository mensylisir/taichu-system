package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

type AuditListParams struct {
	EventType string
	Action    string
	User      string
	Result    string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
	StartTime time.Time
	EndTime   time.Time
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(event *model.AuditEvent) error {
	return r.db.Create(event).Error
}

func (r *AuditRepository) GetByID(id string) (*model.AuditEvent, error) {
	var event model.AuditEvent
	err := r.db.Where("id = ?", id).First(&event).Error
	return &event, err
}

func (r *AuditRepository) List(page, pageSize int, resource, resourceID, username, eventType string) ([]*model.AuditEvent, int64, error) {
	var events []*model.AuditEvent
	var total int64

	query := r.db.Model(&model.AuditEvent{})

	if resource != "" {
		query = query.Where("resource = ?", resource)
	}

	if resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}

	if username != "" {
		query = query.Where("username = ?", username)
	}

	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&events).Error

	return events, total, err
}

func (r *AuditRepository) GetAuditLogs(clusterID uuid.UUID, params AuditListParams) ([]*model.AuditEvent, int64, error) {
	var events []*model.AuditEvent
	var total int64

	query := r.db.Model(&model.AuditEvent{})

	if clusterID != uuid.Nil {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if params.EventType != "" {
		query = query.Where("event_type = ?", params.EventType)
	}

	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}

	if params.User != "" {
		query = query.Where("username = ?", params.User)
	}

	if params.Result != "" {
		query = query.Where("result = ?", params.Result)
	}

	if !params.StartTime.IsZero() {
		query = query.Where("created_at >= ?", params.StartTime)
	}

	if !params.EndTime.IsZero() {
		query = query.Where("created_at <= ?", params.EndTime)
	}

	query.Count(&total)

	orderClause := "created_at DESC"
	if params.SortBy != "" {
		orderClause = params.SortBy
		if params.SortOrder != "" {
			orderClause += " " + params.SortOrder
		}
	}

	err := query.Offset(params.Offset).Limit(params.Limit).Order(orderClause).Find(&events).Error

	return events, total, err
}

func (r *AuditRepository) GetDB() *gorm.DB {
	return r.db
}