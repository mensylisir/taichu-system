package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type ExpansionService struct {
	expansionRepo *repository.ExpansionRepository
	clusterRepo   *repository.ClusterRepository
	stateRepo     *repository.ClusterStateRepository
}

func NewExpansionService(
	expansionRepo *repository.ExpansionRepository,
	clusterRepo *repository.ClusterRepository,
	stateRepo *repository.ClusterStateRepository,
) *ExpansionService {
	return &ExpansionService{
		expansionRepo: expansionRepo,
		clusterRepo:   clusterRepo,
		stateRepo:     stateRepo,
	}
}

func (s *ExpansionService) RequestExpansion(clusterID uuid.UUID, newNodeCount, newCPUCores, newMemoryGB, newStorageGB int, reason string, requestedBy string) (*model.ClusterExpansion, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	state, err := s.stateRepo.GetByClusterID(clusterID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster state: %w", err)
	}

	expansion := &model.ClusterExpansion{
		ClusterID:    clusterID,
		OldNodeCount: state.NodeCount,
		NewNodeCount: newNodeCount,
		OldCPUCores:  state.TotalCPUCores,
		NewCPUCores:  newCPUCores,
		OldMemoryGB:  int(state.TotalMemoryBytes / 1024 / 1024 / 1024),
		NewMemoryGB:  newMemoryGB,
		OldStorageGB: int(state.TotalStorageBytes / 1024 / 1024 / 1024),
		NewStorageGB: newStorageGB,
		Status:       "pending",
		Reason:       reason,
		Details: map[string]interface{}{
			"cluster_name": cluster.Name,
		},
		RequestedBy: requestedBy,
	}

	if err := s.expansionRepo.Create(expansion); err != nil {
		return nil, fmt.Errorf("failed to create expansion record: %w", err)
	}

	return expansion, nil
}

func (s *ExpansionService) ExecuteExpansion(expansionID string) error {
	expansion, err := s.expansionRepo.GetByID(expansionID)
	if err != nil {
		return fmt.Errorf("failed to get expansion: %w", err)
	}

	expansion.Status = "in_progress"
	expansion.ExecutedBy = "system"
	now := time.Now()
	expansion.StartedAt = &now

	if err := s.expansionRepo.Update(expansion); err != nil {
		return fmt.Errorf("failed to update expansion status: %w", err)
	}

	state, err := s.stateRepo.GetByClusterID(expansion.ClusterID.String())
	if err != nil {
		return s.failExpansion(expansion, fmt.Errorf("failed to get cluster state: %w", err))
	}

	state.NodeCount = expansion.NewNodeCount
	state.TotalCPUCores = expansion.NewCPUCores
	state.TotalMemoryBytes = int64(expansion.NewMemoryGB) * 1024 * 1024 * 1024
	state.TotalStorageBytes = int64(expansion.NewStorageGB) * 1024 * 1024 * 1024

	if err := s.stateRepo.Update(state); err != nil {
		return s.failExpansion(expansion, fmt.Errorf("failed to update cluster state: %w", err))
	}

	expansion.Status = "completed"
	expansion.CompletedAt = &now

	return s.expansionRepo.Update(expansion)
}

func (s *ExpansionService) failExpansion(expansion *model.ClusterExpansion, err error) error {
	expansion.Status = "failed"
	expansion.ErrorMsg = err.Error()
	now := time.Now()
	expansion.CompletedAt = &now
	return s.expansionRepo.Update(expansion)
}

func (s *ExpansionService) GetExpansionHistory(clusterID uuid.UUID) ([]*model.ClusterExpansion, error) {
	return s.expansionRepo.GetByClusterID(clusterID)
}

func (s *ExpansionService) GetExpansion(expansionID string) (*model.ClusterExpansion, error) {
	return s.expansionRepo.GetByID(expansionID)
}
