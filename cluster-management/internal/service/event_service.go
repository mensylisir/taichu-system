package service

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type EventService struct {
	clusterManager  *ClusterManager
	encryptionSvc   *EncryptionService
	eventRepo       *repository.EventRepository
	clusterRepo     *repository.ClusterRepository
}

type EventSummaryStats struct {
	Critical int `json:"critical"`
	Error    int `json:"error"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

func NewEventService(
	clusterManager *ClusterManager,
	encryptionSvc *EncryptionService,
	eventRepo *repository.EventRepository,
	clusterRepo *repository.ClusterRepository,
) *EventService {
	return &EventService{
		clusterManager:  clusterManager,
		encryptionSvc:   encryptionSvc,
		eventRepo:       eventRepo,
		clusterRepo:     clusterRepo,
	}
}

func (s *EventService) ListEvents(clusterID, severity string, since *time.Time) ([]*model.Event, int64, *EventSummaryStats, error) {
	events, total, err := s.eventRepo.List(clusterID, severity, since)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to list events: %w", err)
	}

	// 计算统计信息
	summary := &EventSummaryStats{
		Critical: 0,
		Error:    0,
		Warning:  0,
		Info:     0,
	}

	for _, event := range events {
		switch event.Severity {
		case "critical":
			summary.Critical++
		case "error":
			summary.Error++
		case "warning":
			summary.Warning++
		case "info":
			summary.Info++
		}
	}

	return events, total, summary, nil
}

func (s *EventService) SyncEventsFromKubernetes(clusterID string) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt kubeconfig: %w", err)
	}

	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	// 获取集群事件
	events, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	// 同步每个事件
	for _, k8sEvent := range events.Items {
		event := &model.Event{
			ClusterID:      cluster.ID,
			EventType:      k8sEvent.Type,
			Message:        k8sEvent.Message,
			Severity:       s.determineSeverity(k8sEvent.Type, k8sEvent.Reason),
			Component:      k8sEvent.Source.Component,
			FirstTimestamp: k8sEvent.FirstTimestamp.Time,
			LastTimestamp:  k8sEvent.LastTimestamp.Time,
			Count:          int(k8sEvent.Count),
		}

		if err := s.eventRepo.Create(event); err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}
	}

	return nil
}

func (s *EventService) determineSeverity(eventType, reason string) string {
	if eventType == "Warning" {
		return "warning"
	}

	switch reason {
	case "Failed", "FailedScheduling", "FailedMount":
		return "error"
	case "Killing", "Preempting":
		return "warning"
	case "Created", "Started", "Scheduled":
		return "info"
	default:
		return "info"
	}
}
