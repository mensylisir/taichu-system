package service

import (

	"github.com/google/uuid"
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

func (s *AuditService) CreateAuditEvent(clusterID uuid.UUID, eventType, action, resource, resourceID, user, ipAddress, userAgent string, oldValue, newValue map[string]interface{}, details map[string]interface{}, result string) error {
	auditEvent := &model.AuditEvent{
		ClusterID:  clusterID,
		EventType:  eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		User:       user,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		OldValue:   oldValue,
		NewValue:   newValue,
		Details:    details,
		Result:     result,
	}

	if result == "failed" && details["error"] != nil {
		auditEvent.ErrorMsg = details["error"].(string)
	}

	return s.auditRepo.Create(auditEvent)
}

func (s *AuditService) LogClusterOperation(clusterID uuid.UUID, operation, resource string, user string, oldValue, newValue map[string]interface{}) error {
	return s.CreateAuditEvent(
		clusterID,
		"cluster",
		operation,
		resource,
		"",
		user,
		"",
		"",
		oldValue,
		newValue,
		map[string]interface{}{
			"operation": operation,
		},
		"success",
	)
}

func (s *AuditService) LogNodeOperation(clusterID uuid.UUID, operation, nodeName string, user string, oldValue, newValue map[string]interface{}) error {
	return s.CreateAuditEvent(
		clusterID,
		"node",
		operation,
		"node",
		nodeName,
		user,
		"",
		"",
		oldValue,
		newValue,
		map[string]interface{}{
			"node_name": nodeName,
		},
		"success",
	)
}

func (s *AuditService) LogSecurityPolicyOperation(clusterID uuid.UUID, operation string, user string, oldValue, newValue map[string]interface{}) error {
	return s.CreateAuditEvent(
		clusterID,
		"security",
		operation,
		"security-policy",
		"",
		user,
		"",
		"",
		oldValue,
		newValue,
		map[string]interface{}{
			"operation": operation,
		},
		"success",
	)
}

func (s *AuditService) LogBackupOperation(clusterID uuid.UUID, operation, backupID string, user string, details map[string]interface{}) error {
	return s.CreateAuditEvent(
		clusterID,
		"backup",
		operation,
		"backup",
		backupID,
		user,
		"",
		"",
		nil,
		nil,
		details,
		"success",
	)
}

func (s *AuditService) GetAuditLogs(clusterID uuid.UUID, params repository.AuditListParams) ([]*model.AuditEvent, int64, error) {
	return s.auditRepo.ListByCluster(clusterID, params)
}
