package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type ExpansionService struct {
	expansionRepo     *repository.ExpansionRepository
	clusterRepo       *repository.ClusterRepository
	stateRepo         *repository.ClusterStateRepository
	clusterResourceRepo *repository.ClusterResourceRepository
}

func NewExpansionService(
	expansionRepo *repository.ExpansionRepository,
	clusterRepo *repository.ClusterRepository,
	stateRepo *repository.ClusterStateRepository,
	clusterResourceRepo *repository.ClusterResourceRepository,
) *ExpansionService {
	return &ExpansionService{
		expansionRepo:       expansionRepo,
		clusterRepo:         clusterRepo,
		stateRepo:           stateRepo,
		clusterResourceRepo: clusterResourceRepo,
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

	// 从 cluster_resources 表获取当前资源使用情况
	resource, err := s.clusterResourceRepo.GetLatestByClusterID(clusterID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster resource: %w", err)
	}

	expansion := &model.ClusterExpansion{
		ClusterID:    clusterID,
		OldNodeCount: state.NodeCount,
		NewNodeCount: newNodeCount,
		OldCPUCores:  resource.TotalCPUCores,
		NewCPUCores:  newCPUCores,
		OldMemoryGB:  int(resource.TotalMemoryBytes / 1024 / 1024 / 1024),
		NewMemoryGB:  newMemoryGB,
		OldStorageGB: int(resource.TotalStorageBytes / 1024 / 1024 / 1024),
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

	// 更新 cluster_state 表中的 node_count 字段
	state.NodeCount = expansion.NewNodeCount
	state.UpdatedAt = time.Now()

	if err := s.stateRepo.GetDB().Model(&model.ClusterState{}).Where("cluster_id = ?", expansion.ClusterID.String()).Updates(map[string]interface{}{
		"node_count": state.NodeCount,
		"updated_at": state.UpdatedAt,
	}).Error; err != nil {
		return s.failExpansion(expansion, fmt.Errorf("failed to update cluster state: %w", err))
	}

	// 更新 cluster_resources 表中的资源字段
	resource, err := s.clusterResourceRepo.GetLatestByClusterID(expansion.ClusterID.String())
	if err != nil {
		return s.failExpansion(expansion, fmt.Errorf("failed to get cluster resource: %w", err))
	}

	resource.TotalCPUCores = expansion.NewCPUCores
	resource.TotalMemoryBytes = int64(expansion.NewMemoryGB) * 1024 * 1024 * 1024
	resource.TotalStorageBytes = int64(expansion.NewStorageGB) * 1024 * 1024 * 1024
	resource.Timestamp = time.Now()

	if err := s.clusterResourceRepo.Upsert(resource); err != nil {
		return s.failExpansion(expansion, fmt.Errorf("failed to update cluster resource: %w", err))
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
