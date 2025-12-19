package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type TopologyHandler struct {
	topologyService *service.TopologyService
}

type ClusterNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	NodeCount   int    `json:"node_count"`
	Version     string `json:"version"`
}

type EnvironmentTopology struct {
	Type             string        `json:"type"`
	Count            int           `json:"count"`
	Clusters         []ClusterNode `json:"clusters"`
	TotalNodeCount   int           `json:"total_node_count"`
	HealthyClusters  int           `json:"healthy_clusters"`
	UnhealthyClusters int          `json:"unhealthy_clusters"`
}

type TopologyResponse struct {
	Environments []EnvironmentTopology `json:"environments"`
	Summary      TopologySummary       `json:"summary"`
}

type TopologySummary struct {
	TotalClusters     int `json:"total_clusters"`
	TotalEnvironments int `json:"total_environments"`
	TotalNodes        int `json:"total_nodes"`
	HealthyClusters   int `json:"healthy_clusters"`
}

func NewTopologyHandler(topologyService *service.TopologyService) *TopologyHandler {
	return &TopologyHandler{
		topologyService: topologyService,
	}
}

func (h *TopologyHandler) GetTopology(c *gin.Context) {
	topology, err := h.topologyService.GetClusterTopology()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to get topology: %v", err)
		return
	}

	response := TopologyResponse{
		Environments: make([]EnvironmentTopology, 0, len(topology.Environments)),
		Summary: TopologySummary{
			TotalClusters:     topology.Summary.TotalClusters,
			TotalEnvironments: topology.Summary.TotalEnvironments,
			TotalNodes:        topology.Summary.TotalNodes,
			HealthyClusters:   topology.Summary.HealthyClusters,
		},
	}

	for _, env := range topology.Environments {
		clusters := make([]ClusterNode, 0, len(env.Clusters))
		for _, cluster := range env.Clusters {
			clusters = append(clusters, ClusterNode{
				ID:          cluster.ID,
				Name:        cluster.Name,
				Status:      cluster.Status,
				NodeCount:   cluster.NodeCount,
				Version:     cluster.Version,
			})
		}

		response.Environments = append(response.Environments, EnvironmentTopology{
			Type:             env.Type,
			Count:            env.Count,
			Clusters:         clusters,
			TotalNodeCount:   env.TotalNodeCount,
			HealthyClusters:  env.HealthyClusters,
			UnhealthyClusters: env.UnhealthyClusters,
		})
	}

	utils.Success(c, http.StatusOK, response)
}
