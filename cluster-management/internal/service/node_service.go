package service

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type NodeService struct {
	clusterManager  *ClusterManager
	encryptionSvc   *EncryptionService
	nodeRepo        *repository.NodeRepository
	clusterRepo     *repository.ClusterRepository
}

type NodeWithDetails struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	CPUCores          int       `json:"cpu_cores"`
	CPUUsedCores      float64   `json:"cpu_used_cores"`
	MemoryBytes       int64     `json:"memory_bytes"`
	MemoryUsedBytes   int64     `json:"memory_used_bytes"`
	PodCount          int       `json:"pod_count"`
	Labels            model.JSONMap `json:"labels"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type NodeSummaryStats struct {
	ControlPlaneCount int `json:"control_plane_count"`
	WorkerCount       int `json:"worker_count"`
	ReadyCount        int `json:"ready_count"`
}

func NewNodeService(
	clusterManager *ClusterManager,
	encryptionSvc *EncryptionService,
	nodeRepo *repository.NodeRepository,
	clusterRepo *repository.ClusterRepository,
) *NodeService {
	return &NodeService{
		clusterManager:  clusterManager,
		encryptionSvc:   encryptionSvc,
		nodeRepo:        nodeRepo,
		clusterRepo:     clusterRepo,
	}
}

func (s *NodeService) ListNodes(clusterID, nodeType, status string, page, limit int) ([]*NodeWithDetails, int64, *NodeSummaryStats, error) {
	nodes, total, err := s.nodeRepo.List(clusterID, nodeType, status, page, limit)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 转换为详细结构
	nodeDetails := make([]*NodeWithDetails, 0, len(nodes))
	for _, node := range nodes {
		nodeDetails = append(nodeDetails, &NodeWithDetails{
			ID:              node.ID,
			Name:            node.Name,
			Type:            node.Type,
			Status:          node.Status,
			CPUCores:        node.CPUCores,
			CPUUsedCores:    float64(node.CPUUsedCores),
			MemoryBytes:     node.MemoryBytes,
			MemoryUsedBytes: node.MemoryUsedBytes,
			PodCount:        node.PodCount,
			Labels:          node.Labels,
			CreatedAt:       node.CreatedAt,
			UpdatedAt:       node.UpdatedAt,
		})
	}

	// 计算统计信息
	summary := &NodeSummaryStats{
		ControlPlaneCount: 0,
		WorkerCount:       0,
		ReadyCount:        0,
	}

	for _, node := range nodes {
		if node.Type == "control-plane" {
			summary.ControlPlaneCount++
		} else if node.Type == "worker" {
			summary.WorkerCount++
		}

		if node.Status == "Ready" {
			summary.ReadyCount++
		}
	}

	return nodeDetails, total, summary, nil
}

func (s *NodeService) GetNode(clusterID, nodeName string) (*model.Node, error) {
	node, err := s.nodeRepo.GetByName(clusterID, nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	return node, nil
}

func (s *NodeService) SyncNodesFromKubernetes(clusterID string) error {
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

	// 从Kubernetes获取节点列表
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// 同步每个节点
	for _, k8sNode := range nodes.Items {
		nodeType := s.determineNodeType(k8sNode.Labels)
		podCount := s.countPodsOnNode(ctx, clientset, k8sNode.Name)

		node := &model.Node{
			ClusterID:       cluster.ID,
			Name:            k8sNode.Name,
			Type:            nodeType,
			Status:          s.getNodeStatus(k8sNode),
			CPUCores:        int(k8sNode.Status.Capacity.Cpu().MilliValue() / 1000), // 显示逻辑CPU数
			CPUUsedCores:    0, // 需要从metrics-server获取
			MemoryBytes:     k8sNode.Status.Capacity.Memory().Value(), // 使用Capacity而不是Allocatable
			MemoryUsedBytes: 0, // 需要从metrics-server获取
			PodCount:        podCount,
			Labels:          convertToJSONMap(k8sNode.Labels),
		}

		if err := s.nodeRepo.UpsertSingle(node); err != nil {
			return fmt.Errorf("failed to upsert node: %w", err)
		}
	}

	return nil
}

func (s *NodeService) determineNodeType(labels map[string]string) string {
	if _, ok := labels["node-role.kubernetes.io/master"]; ok {
		return "control-plane"
	}
	if _, ok := labels["node-role.kubernetes.io/control-plane"]; ok {
		return "control-plane"
	}
	return "worker"
}

func (s *NodeService) countPodsOnNode(ctx context.Context, clientset *kubernetes.Clientset, nodeName string) int {
	log.Printf("[NODE-SYNC] Counting pods for node: %s", nodeName)

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		log.Printf("[NODE-SYNC] Error listing pods for node %s: %v", nodeName, err)
		return 0
	}

	log.Printf("[NODE-SYNC] Found %d pods for node %s", len(pods.Items), nodeName)
	for _, pod := range pods.Items {
		log.Printf("[NODE-SYNC] Pod: %s/%s (status: %s)", pod.Namespace, pod.Name, pod.Status.Phase)
	}

	return len(pods.Items)
}

func (s *NodeService) getNodeStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func convertToJSONMap(labels map[string]string) model.JSONMap {
	result := make(model.JSONMap)
	for k, v := range labels {
		result[k] = v
	}
	return result
}
