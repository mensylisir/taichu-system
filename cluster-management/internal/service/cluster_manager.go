package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterManager struct {
	clients map[string]*ClusterClient
	mutex   sync.RWMutex
	timeout time.Duration
	maxClients int
}

type ClusterClient struct {
	Clientset *kubernetes.Clientset
	LastUsed  time.Time
}

type HealthCheckResult struct {
	Status            string    `json:"status"`
	Version           string    `json:"version"`
	NodeCount         int       `json:"node_count"`
	TotalCPUCores     int       `json:"total_cpu_cores"`
	TotalMemoryBytes  int64     `json:"total_memory_bytes"`
	APIServerURL      string    `json:"api_server_url"`
	LastHeartbeatAt   time.Time `json:"last_heartbeat_at"`
	Error             string    `json:"error,omitempty"`
}

func NewClusterManager(timeout time.Duration, maxClients int) *ClusterManager {
	return &ClusterManager{
		clients:   make(map[string]*ClusterClient),
		timeout:   timeout,
		maxClients: maxClients,
	}
}

func (cm *ClusterManager) GetClient(ctx context.Context, kubeconfig string) (*kubernetes.Clientset, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cacheKey := cm.generateKubeconfigHash(kubeconfig)

	if client, exists := cm.clients[cacheKey]; exists {
		client.LastUsed = time.Now()
		return client.Clientset, nil
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config: %w", err)
	}

	config.Timeout = cm.timeout

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	cm.clients[cacheKey] = &ClusterClient{
		Clientset: clientset,
		LastUsed:  time.Now(),
	}

	cm.cleanupOldClients()

	return clientset, nil
}

func (cm *ClusterManager) cleanupOldClients() {
	if len(cm.clients) <= cm.maxClients {
		return
	}

	var oldestKey string
	var oldestTime = time.Now()

	for key, client := range cm.clients {
		if client.LastUsed.Before(oldestTime) {
			oldestTime = client.LastUsed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(cm.clients, oldestKey)
	}
}

func (cm *ClusterManager) RemoveClient(kubeconfig string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cacheKey := cm.generateKubeconfigHash(kubeconfig)
	delete(cm.clients, cacheKey)
}

func (cm *ClusterManager) HealthCheck(ctx context.Context, clientset *kubernetes.Clientset) (*HealthCheckResult, error) {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return &HealthCheckResult{
			Status: "disconnected",
			Error:  err.Error(),
		}, nil
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return &HealthCheckResult{
			Status:  "unhealthy",
			Version: version.String(),
			Error:   err.Error(),
		}, nil
	}

	var totalCPU int64
	var totalMemory int64
	readyNodes := 0

	for _, node := range nodes.Items {
		if cm.isNodeReady(node) {
			readyNodes++
		}
		totalCPU += node.Status.Allocatable.Cpu().MilliValue()
		totalMemory += node.Status.Allocatable.Memory().Value()
	}

	return &HealthCheckResult{
		Status:            "healthy",
		Version:           version.String(),
		NodeCount:         readyNodes,
		TotalCPUCores:     int(totalCPU / 1000),
		TotalMemoryBytes:  totalMemory,
		LastHeartbeatAt:   time.Now(),
	}, nil
}

func (cm *ClusterManager) isNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (cm *ClusterManager) generateKubeconfigHash(kubeconfig string) string {
	hash := sha256.Sum256([]byte(kubeconfig))
	return hex.EncodeToString(hash[:])
}

func (cm *ClusterManager) GetAPIServerURL(kubeconfig string) (string, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return "", err
	}
	return config.Host, nil
}

func (cm *ClusterManager) ValidateKubeconfig(kubeconfig string) (bool, error) {
	if kubeconfig == "" {
		return false, fmt.Errorf("kubeconfig is empty")
	}

	_, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return false, fmt.Errorf("invalid kubeconfig: %w", err)
	}

	return true, nil
}
