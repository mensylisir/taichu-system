package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type AuditService struct {
	auditRepo *repository.AuditRepository
}

func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

func (s *AuditService) LogEvent(event *model.AuditEvent) error {
	return s.auditRepo.Create(event)
}

func (s *AuditService) LogCreate(resource, resourceID, username, ipAddress, userAgent string, newValue interface{}) error {
	var newJSON model.JSONMap
	if newValue != nil {
		if val, ok := newValue.(map[string]interface{}); ok {
			newJSON = val
		} else {
			newJSON = make(model.JSONMap)
		}
	}

	event := &model.AuditEvent{
		EventType:  constants.EventTypeCreate,
		Action:     constants.EventTypeCreate,
		Resource:   resource,
		ResourceID: resourceID,
		Username:   username,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		NewValue:   newJSON,
		Result:     constants.StatusSuccess,
	}
	return s.auditRepo.Create(event)
}

func (s *AuditService) LogUpdate(resource, resourceID, username, ipAddress, userAgent string, oldValue, newValue interface{}) error {
	var oldJSON, newJSON model.JSONMap
	if oldValue != nil {
		if val, ok := oldValue.(map[string]interface{}); ok {
			oldJSON = val
		} else {
			oldJSON = make(model.JSONMap)
		}
	}
	if newValue != nil {
		if val, ok := newValue.(map[string]interface{}); ok {
			newJSON = val
		} else {
			newJSON = make(model.JSONMap)
		}
	}

	event := &model.AuditEvent{
		EventType:  "update",
		Action:     "update",
		Resource:   resource,
		ResourceID: resourceID,
		Username:   username,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		OldValue:   oldJSON,
		NewValue:   newJSON,
		Result:     "success",
	}
	return s.auditRepo.Create(event)
}

func (s *AuditService) LogDelete(resource, resourceID, username, ipAddress, userAgent string, oldValue interface{}) error {
	var oldJSON model.JSONMap
	if oldValue != nil {
		if val, ok := oldValue.(map[string]interface{}); ok {
			oldJSON = val
		} else {
			oldJSON = make(model.JSONMap)
		}
	}

	event := &model.AuditEvent{
		EventType:  "delete",
		Action:     "delete",
		Resource:   resource,
		ResourceID: resourceID,
		Username:   username,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		OldValue:   oldJSON,
		Result:     "success",
	}
	return s.auditRepo.Create(event)
}

func (s *AuditService) LogError(resource, resourceID, username, ipAddress, userAgent, action, errorMsg string) error {
	event := &model.AuditEvent{
		EventType:  constants.EventTypeError,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Username:   username,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Result:     constants.StatusFailed,
		ErrorMsg:   errorMsg,
	}
	return s.auditRepo.Create(event)
}

func (s *AuditService) GetAuditLogs(clusterID uuid.UUID, params repository.AuditListParams) ([]*model.AuditEvent, int64, error) {
	return s.auditRepo.GetAuditLogs(clusterID, params)
}

func (s *AuditService) CreateAuditEvent(clusterID uuid.UUID, eventType, action, resource, resourceID, username, ipAddress, userAgent string, oldValue, newValue, details map[string]interface{}, result string) error {
	event := &model.AuditEvent{
		EventType:  eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Username:   username,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		OldValue:   oldValue,
		NewValue:   newValue,
		Details:    details,
		Result:     result,
	}
	return s.auditRepo.Create(event)
}

func (s *AuditService) CreateAuditEventWithTimestamp(clusterID uuid.UUID, eventType, action, resource, resourceID, username, ipAddress, userAgent string, oldValue, newValue, details map[string]interface{}, result string, timestamp time.Time) error {
	event := &model.AuditEvent{
		EventType:  eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Username:   username,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		OldValue:   oldValue,
		NewValue:   newValue,
		Details:    details,
		Result:     result,
		Timestamp:  timestamp,
	}
	return s.auditRepo.Create(event)
}

func (s *AuditService) LogBackupOperation(clusterID uuid.UUID, eventType, backupID, username string, details map[string]interface{}) error {
	return s.CreateAuditEvent(
		clusterID,
		eventType,
		eventType,
		"backup",
		backupID,
		username,
		"",
		"",
		nil,
		details,
		nil,
		constants.StatusSuccess,
	)
}

func (s *AuditService) LogClusterOperation(clusterID uuid.UUID, eventType, clusterIDStr, username string, details map[string]interface{}) error {
	return s.CreateAuditEvent(
		clusterID,
		eventType,
		eventType,
		"cluster",
		clusterIDStr,
		username,
		"",
		"",
		nil,
		details,
		nil,
		constants.StatusSuccess,
	)
}
