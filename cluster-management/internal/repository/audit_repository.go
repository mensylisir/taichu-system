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

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(auditEvent *model.AuditEvent) error {
	return r.db.Create(auditEvent).Error
}

func (r *AuditRepository) GetByID(id string) (*model.AuditEvent, error) {
	var auditEvent model.AuditEvent
	if err := r.db.First(&auditEvent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &auditEvent, nil
}

func (r *AuditRepository) ListByCluster(clusterID uuid.UUID, params AuditListParams) ([]*model.AuditEvent, int64, error) {
	var events []*model.AuditEvent
	var total int64

	query := r.db.Model(&model.AuditEvent{})

	// 如果 clusterID 不是 nil，则按集群过滤；否则查询所有记录
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

	if !params.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", params.StartTime)
	}

	if !params.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", params.EndTime)
	}

	if params.Result != "" {
		query = query.Where("result = ?", params.Result)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	if params.SortBy != "" {
		if params.SortOrder == "asc" {
			query = query.Order(params.SortBy + " ASC")
		} else {
			query = query.Order(params.SortBy + " DESC")
		}
	} else {
		query = query.Order("timestamp DESC")
	}

	if err := query.Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *AuditRepository) DeleteByClusterID(clusterID uuid.UUID) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.AuditEvent{}).Error
}

func (r *AuditRepository) DeleteOlderThan(age time.Duration) error {
	cutoff := time.Now().Add(-age)
	return r.db.Where("timestamp < ?", cutoff).Delete(&model.AuditEvent{}).Error
}

type AuditListParams struct {
	EventType string
	Action    string
	User      string
	StartTime time.Time
	EndTime   time.Time
	Result    string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}
