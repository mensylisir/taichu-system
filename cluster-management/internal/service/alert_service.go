package service

import (
	"fmt"
	"log"
	"time"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type AlertService struct {
	alertRepo *repository.AlertRepository
}

func NewAlertService(alertRepo *repository.AlertRepository) *AlertService {
	return &AlertService{
		alertRepo: alertRepo,
	}
}

func (s *AlertService) CreateAlert(alertType model.AlertType, severity model.AlertSeverity, title, description, resourceType, resourceID string, metadata model.JSONMap) (*model.Alert, error) {
	alert := &model.Alert{
		Type:        alertType,
		Severity:    severity,
		Status:      model.AlertStatusActive,
		Title:       title,
		Description: description,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:    metadata,
	}

	if err := s.alertRepo.Create(alert); err != nil {
		return nil, err
	}

	s.sendNotification(alert)

	return alert, nil
}

func (s *AlertService) ResolveAlert(id string) error {
	now := time.Now()
	return s.alertRepo.GetDB().Model(&model.Alert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      model.AlertStatusResolved,
		"resolved_at": now,
	}).Error
}

func (s *AlertService) ListAlerts(page, pageSize int, alertType model.AlertType, severity model.AlertSeverity, status model.AlertStatus, resourceType, resourceID string) ([]*model.Alert, int64, error) {
	return s.alertRepo.List(page, pageSize, alertType, severity, status, resourceType, resourceID)
}

func (s *AlertService) GetActiveAlerts() ([]*model.Alert, error) {
	return s.alertRepo.GetActiveAlerts()
}

func (s *AlertService) sendNotification(alert *model.Alert) {
	log.Printf("[ALERT] %s - %s: %s", alert.Severity, alert.Title, alert.Description)
	
	if alert.Severity == model.AlertSeverityCritical || alert.Severity == model.AlertSeverityHigh {
		log.Printf("[ALERT] High severity alert detected! Resource: %s/%s", alert.ResourceType, alert.ResourceID)
	}
}

func (s *AlertService) AlertBackupFailed(backupID, clusterID, errorMsg string) error {
	metadata := model.JSONMap{
		"backup_id":  backupID,
		"cluster_id": clusterID,
		"error":      errorMsg,
	}

	_, err := s.CreateAlert(
		model.AlertTypeBackupFailed,
		model.AlertSeverityHigh,
		fmt.Sprintf("Backup failed: %s", backupID),
		fmt.Sprintf("Backup %s for cluster %s failed: %s", backupID, clusterID, errorMsg),
		"backup",
		backupID,
		metadata,
	)

	return err
}

func (s *AlertService) AlertScheduleFailed(scheduleID, clusterID, errorMsg string) error {
	metadata := model.JSONMap{
		"schedule_id": scheduleID,
		"cluster_id":  clusterID,
		"error":       errorMsg,
	}

	_, err := s.CreateAlert(
		model.AlertTypeScheduleFailed,
		model.AlertSeverityHigh,
		fmt.Sprintf("Schedule failed: %s", scheduleID),
		fmt.Sprintf("Backup schedule %s for cluster %s failed: %s", scheduleID, clusterID, errorMsg),
		"schedule",
		scheduleID,
		metadata,
	)

	return err
}

func (s *AlertService) AlertSystemError(component, errorMsg string) error {
	metadata := model.JSONMap{
		"component": component,
		"error":     errorMsg,
	}

	_, err := s.CreateAlert(
		model.AlertTypeSystemError,
		model.AlertSeverityCritical,
		fmt.Sprintf("System error: %s", component),
		fmt.Sprintf("Component %s encountered an error: %s", component, errorMsg),
		"system",
		component,
		metadata,
	)

	return err
}
