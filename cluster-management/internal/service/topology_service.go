package service

import (
	"fmt"

	"github.com/taichu-system/cluster-management/internal/repository"
)

type TopologyService struct {
	clusterRepo    *repository.ClusterRepository
	stateRepo      *repository.ClusterStateRepository
	environmentRepo *repository.ClusterEnvironmentRepository
}

type ClusterInfo struct {
	ID          string
	Name        string
	Status      string
	NodeCount   int
	Version     string
}

type EnvironmentInfo struct {
	Type              string
	Count             int
	Clusters          []ClusterInfo
	TotalNodeCount    int
	HealthyClusters   int
	UnhealthyClusters int
}

type TopologyWithDetails struct {
	Environments []EnvironmentInfo
	Summary      TopologySummary
}

type TopologySummary struct {
	TotalClusters     int
	TotalEnvironments int
	TotalNodes        int
	HealthyClusters   int
}

func NewTopologyService(
	clusterRepo *repository.ClusterRepository,
	stateRepo *repository.ClusterStateRepository,
	environmentRepo *repository.ClusterEnvironmentRepository,
) *TopologyService {
	return &TopologyService{
		clusterRepo:     clusterRepo,
		stateRepo:       stateRepo,
		environmentRepo: environmentRepo,
	}
}

func (s *TopologyService) GetClusterTopology() (*TopologyWithDetails, error) {
	// 获取所有集群及其状态
	clusters, err := s.clusterRepo.FindAllWithState()
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	// 按环境类型分组
	environmentMap := make(map[string][]ClusterInfo)
	var totalNodes int
	var healthyClusters int

	for _, cluster := range clusters {
		clusterInfo := ClusterInfo{
			ID:          cluster.Cluster.ID.String(),
			Name:        cluster.Cluster.Name,
			Status:      cluster.State.Status,
			NodeCount:   cluster.State.NodeCount,
			Version:     cluster.State.KubernetesVersion,
		}

		// 获取环境类型，默认为production
		envType := cluster.Cluster.EnvironmentType
		if envType == "" {
			envType = "production"
		}

		environmentMap[envType] = append(environmentMap[envType], clusterInfo)

		// 累计统计
		totalNodes += cluster.State.NodeCount
		if cluster.State.Status == "healthy" {
			healthyClusters++
		}
	}

	// 转换为响应格式
	environments := make([]EnvironmentInfo, 0, len(environmentMap))
	envCount := 0

	for envType, clusters := range environmentMap {
		healthyCount := 0
		for _, cluster := range clusters {
			if cluster.Status == "healthy" {
				healthyCount++
			}
		}

		environments = append(environments, EnvironmentInfo{
			Type:              envType,
			Count:             len(clusters),
			Clusters:          clusters,
			TotalNodeCount:    s.calculateTotalNodes(clusters),
			HealthyClusters:   healthyCount,
			UnhealthyClusters: len(clusters) - healthyCount,
		})

		envCount++
	}

	return &TopologyWithDetails{
		Environments: environments,
		Summary: TopologySummary{
			TotalClusters:     len(clusters),
			TotalEnvironments: envCount,
			TotalNodes:        totalNodes,
			HealthyClusters:   healthyClusters,
		},
	}, nil
}

func (s *TopologyService) calculateTotalNodes(clusters []ClusterInfo) int {
	total := 0
	for _, cluster := range clusters {
		total += cluster.NodeCount
	}
	return total
}

func (s *TopologyService) UpdateClusterEnvironment(clusterID, environmentType, topologyType string, backupEnabled bool) error {
	return s.environmentRepo.UpdateClusterEnvironment(clusterID, environmentType, topologyType, backupEnabled)
}
