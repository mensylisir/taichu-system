package service

import (
	"context"
	"fmt"
	"time"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrClusterExists = fmt.Errorf("cluster already exists")
	ErrClusterNotFound = fmt.Errorf("cluster not found")
)

type ClusterService struct {
	clusterRepo         *repository.ClusterRepository
	stateRepo           *repository.ClusterStateRepository
	clusterResourceRepo *repository.ClusterResourceRepository
	encryptionService   *EncryptionService
	clusterManager      *ClusterManager
}

func NewClusterService(
	clusterRepo *repository.ClusterRepository,
	stateRepo *repository.ClusterStateRepository,
	clusterResourceRepo *repository.ClusterResourceRepository,
	encryptionService *EncryptionService,
	clusterManager *ClusterManager,
) *ClusterService {
	return &ClusterService{
		clusterRepo:         clusterRepo,
		stateRepo:           stateRepo,
		clusterResourceRepo: clusterResourceRepo,
		encryptionService:   encryptionService,
		clusterManager:      clusterManager,
	}
}

func (s *ClusterService) ValidateKubeconfig(kubeconfig string) (bool, error) {
	return s.clusterManager.ValidateKubeconfig(kubeconfig)
}

func (s *ClusterService) Create(cluster *model.Cluster) (*model.Cluster, error) {
	exists, err := s.clusterRepo.ExistsByName(cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster existence: %w", err)
	}
	if exists {
		return nil, ErrClusterExists
	}

	cluster.CreatedBy = "api-user"
	cluster.UpdatedBy = "api-user"

	if err := s.clusterRepo.Create(cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	return cluster, nil
}

func (s *ClusterService) GetByID(id string) (*model.Cluster, error) {
	cluster, err := s.clusterRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrClusterNotFound
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	return cluster, nil
}

func (s *ClusterService) GetClusterWithState(id string) (*model.ClusterWithState, error) {
	clusterWithState, err := s.clusterRepo.GetWithState(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrClusterNotFound
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// 从 cluster_state 表获取最新的状态数据
	state, err := s.stateRepo.GetByClusterID(id)
	if err == nil {
		// 更新所有状态字段，包括 NodeCount
		clusterWithState.State.Status = state.Status
		clusterWithState.State.NodeCount = state.NodeCount
		clusterWithState.State.KubernetesVersion = state.KubernetesVersion
		clusterWithState.State.APIServerURL = state.APIServerURL
		clusterWithState.State.LastHeartbeatAt = state.LastHeartbeatAt
		clusterWithState.State.LastSyncAt = state.LastSyncAt
		clusterWithState.State.SyncError = state.SyncError
		clusterWithState.State.SyncSuccess = state.SyncSuccess
	}

	// 从 cluster_resources 表获取最新的资源数据
	// 注意：cluster_state 表不再包含资源字段，资源数据存储在 cluster_resources 表中
	// API 调用方需要从 cluster_resources 表获取资源数据
	resource, _ := s.clusterResourceRepo.GetLatestByClusterID(id)
	_ = resource // 确保变量被使用，避免未使用错误

	return clusterWithState, nil
}

// GetClusterResource 获取集群的资源使用情况
func (s *ClusterService) GetClusterResource(clusterID string) (*model.ClusterResource, error) {
	return s.clusterResourceRepo.GetLatestByClusterID(clusterID)
}

// GetClusterResourceRepo 获取clusterResourceRepo实例
func (s *ClusterService) GetClusterResourceRepo() *repository.ClusterResourceRepository {
	return s.clusterResourceRepo
}

func (s *ClusterService) ListClusters(params ListClustersParams) ([]*model.ClusterWithState, int64, error) {
	return s.clusterRepo.List(params)
}

func (s *ClusterService) Update(cluster *model.Cluster) error {
	cluster.UpdatedBy = "api-user"
	return s.clusterRepo.Update(cluster)
}

func (s *ClusterService) Delete(id string) error {
	return s.clusterRepo.Delete(id)
}

func (s *ClusterService) TriggerSync(clusterID string) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	kubeconfig, err := s.encryptionService.Decrypt(
		cluster.KubeconfigEncrypted,
		cluster.KubeconfigNonce,
	)
	if err != nil {
		return fmt.Errorf("failed to decrypt kubeconfig: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := s.clusterManager.HealthCheck(ctx, clientset)
	if err != nil {
		return fmt.Errorf("failed to perform health check: %w", err)
	}

	state := &model.ClusterState{
		ClusterID:         cluster.ID,
		Status:            result.Status,
		NodeCount:         result.NodeCount,
		KubernetesVersion: result.Version,
		APIServerURL:      result.APIServerURL,
		LastHeartbeatAt:   &result.LastHeartbeatAt,
		LastSyncAt:        result.LastHeartbeatAt,
		SyncSuccess:       result.Error == "",
		SyncError:         result.Error,
	}

	return s.stateRepo.Upsert(state)
}

func (s *ClusterService) GetClusterState(clusterID string) (*model.ClusterState, error) {
	return s.stateRepo.GetByClusterID(clusterID)
}

type ListClustersParams = repository.ListClustersParams
