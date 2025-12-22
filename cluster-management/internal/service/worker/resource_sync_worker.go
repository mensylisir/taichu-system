package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
	"gorm.io/gorm"
)

type ResourceSyncWorker struct {
	clusterRepo           *repository.ClusterRepository
	nodeRepo              *repository.NodeRepository
	eventRepo             *repository.EventRepository
	clusterResourceRepo   *repository.ClusterResourceRepository
	clusterStateRepo      *repository.ClusterStateRepository
	autoscalingPolicyRepo *repository.AutoscalingPolicyRepository
	securityPolicyRepo    *repository.SecurityPolicyRepository
	clusterManager        *service.ClusterManager
	encryptionSvc         *service.EncryptionService
	wg                    sync.WaitGroup
	ctx                   context.Context
	cancel                context.CancelFunc
	syncInterval          time.Duration
	maxConcurrency        int
	sem                   chan struct{}
}

func NewResourceSyncWorker(
	clusterRepo *repository.ClusterRepository,
	nodeRepo *repository.NodeRepository,
	eventRepo *repository.EventRepository,
	clusterResourceRepo *repository.ClusterResourceRepository,
	clusterStateRepo *repository.ClusterStateRepository,
	autoscalingPolicyRepo *repository.AutoscalingPolicyRepository,
	securityPolicyRepo *repository.SecurityPolicyRepository,
	clusterManager *service.ClusterManager,
	encryptionSvc *service.EncryptionService,
) *ResourceSyncWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceSyncWorker{
		clusterRepo:           clusterRepo,
		nodeRepo:              nodeRepo,
		eventRepo:             eventRepo,
		clusterResourceRepo:   clusterResourceRepo,
		clusterStateRepo:      clusterStateRepo,
		autoscalingPolicyRepo: autoscalingPolicyRepo,
		securityPolicyRepo:    securityPolicyRepo,
		clusterManager:        clusterManager,
		encryptionSvc:         encryptionSvc,
		ctx:                   ctx,
		cancel:                cancel,
		syncInterval:          5 * time.Minute,
		maxConcurrency:        10,
		sem:                   make(chan struct{}, 10),
	}
}

func (w *ResourceSyncWorker) Start() {
	log.Println("Starting resource sync worker...")

	// 执行一次初始同步
	w.performResourceSync()

	w.wg.Add(1)
	go w.scheduler()
}

func (w *ResourceSyncWorker) Stop() {
	log.Println("Stopping resource sync worker...")
	w.cancel()
	w.wg.Wait()
	close(w.sem)
}

func (w *ResourceSyncWorker) scheduler() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.performResourceSync()
		}
	}
}

func (w *ResourceSyncWorker) performResourceSync() {
	log.Println("Starting scheduled resource sync...")

	clusters, err := w.clusterRepo.FindActiveClusters()
	if err != nil {
		log.Printf("Failed to fetch clusters: %v", err)
		return
	}

	log.Printf("Found %d active clusters for resource sync", len(clusters))

	for _, cluster := range clusters {
		w.wg.Add(1)
		go func(c model.Cluster) {
			defer w.wg.Done()
			w.syncClusterData(c)
		}(*cluster)
	}
}

func (w *ResourceSyncWorker) syncClusterData(cluster model.Cluster) {
	w.sem <- struct{}{}
	defer func() { <-w.sem }()

	kubeconfig, err := w.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		log.Printf("Failed to decrypt kubeconfig for cluster %s: %v", cluster.Name, err)
		return
	}

	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	clientset, err := w.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		log.Printf("Failed to get client for cluster %s: %v", cluster.Name, err)
		return
	}

	// 同步节点信息
	w.syncNodes(ctx, clientset, cluster.ID)

	// 同步事件信息
	w.syncEvents(ctx, clientset, cluster.ID)

	// 同步集群资源使用情况并获取节点信息
	nodes := w.syncClusterResources(ctx, clientset, cluster.ID)

	// 同步自动扩缩容策略
	w.syncAutoscalingPolicies(ctx, clientset, cluster.ID)

	// 同步安全策略
	if err := w.syncSecurityPolicies(ctx, clientset, cluster.ID.String()); err != nil {
		log.Printf("Failed to sync security policies: %v", err)
	}

	// 更新 cluster_state 中的 APIServerURL
	// 传递已获取的节点信息，避免重复API调用
	w.updateAPIServerURL(ctx, nodes, cluster.ID)

	log.Printf("Resource sync completed for cluster %s", cluster.Name)
}

func (w *ResourceSyncWorker) syncNodes(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list nodes: %v", err)
		return
	}

	// 先删除不存在的节点
	existingNodes, err := w.nodeRepo.GetByClusterID(clusterID.String())
	if err != nil {
		log.Printf("Failed to get existing nodes: %v", err)
		return
	}

	existingNodeMap := make(map[string]bool)
	for _, node := range existingNodes {
		existingNodeMap[node.Name] = true
	}

	for _, k8sNode := range nodes.Items {
		nodeName := k8sNode.Name
		if existingNodeMap[nodeName] {
			existingNodeMap[nodeName] = false // 标记为仍然存在
		}

		// 先检查节点是否已存在
		existingNode, err := w.nodeRepo.GetByName(clusterID.String(), nodeName)
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Failed to check existing node %s: %v", nodeName, err)
			continue
		}

		// 转换节点信息
		node := &model.Node{
			ClusterID: clusterID,
			Name:      nodeName,
			Type:      w.getNodeType(k8sNode),
			Status:    w.getNodeStatus(k8sNode),
			Labels:    w.getNodeLabels(k8sNode),
		}

		// 如果节点已存在，使用原有ID
		if err == nil && existingNode != nil {
			node.ID = existingNode.ID
		}

		// 设置资源信息
		if allocatable := k8sNode.Status.Allocatable; allocatable != nil {
			node.CPUCores = int(allocatable.Cpu().MilliValue() / 1000)
			node.MemoryBytes = allocatable.Memory().Value()
		}

		// 使用UpsertSingle方法创建或更新节点
		if err := w.nodeRepo.UpsertSingle(node); err != nil {
			log.Printf("Failed to upsert node %s: %v", nodeName, err)
		}
	}

	// 删除不存在的节点
	for nodeName, exists := range existingNodeMap {
		if exists {
			node, err := w.nodeRepo.GetByName(clusterID.String(), nodeName)
			if err != nil {
				log.Printf("Failed to get node %s for deletion: %v", nodeName, err)
				continue
			}
			if err := w.nodeRepo.Delete(node.ID); err != nil {
				log.Printf("Failed to delete node %s: %v", nodeName, err)
			}
		}
	}

	log.Printf("Synced %d nodes for cluster %s", len(nodes.Items), clusterID)
}

func (w *ResourceSyncWorker) syncEvents(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) {
	// 获取最近一小时的事件
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	events, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		Limit: 100,
	})
	if err != nil {
		log.Printf("Failed to list events: %v", err)
		return
	}

	// 过滤最近一小时的事件
	var recentEvents []corev1.Event
	for _, event := range events.Items {
		if event.LastTimestamp.Time.After(oneHourAgo) {
			recentEvents = append(recentEvents, event)
		}
	}

	for _, k8sEvent := range recentEvents {
		// 检查事件是否已存在
		exists, err := w.eventRepo.CheckExists(
			clusterID.String(),
			k8sEvent.Type,
			k8sEvent.Message,
			k8sEvent.Source.Component,
			k8sEvent.FirstTimestamp.Time,
			k8sEvent.LastTimestamp.Time,
		)
		if err != nil {
			log.Printf("Failed to check event existence: %v", err)
			continue
		}
		if exists {
			continue // 跳过已存在的事件
		}

		// 转换事件信息
		event := &model.Event{
			ID:             uuid.New(), // 生成新的UUID
			ClusterID:      clusterID,
			EventType:      k8sEvent.Type,
			Message:        k8sEvent.Message,
			Severity:       w.getEventSeverity(k8sEvent.Type),
			Component:      k8sEvent.Source.Component,
			FirstTimestamp: k8sEvent.FirstTimestamp.Time,
			LastTimestamp:  k8sEvent.LastTimestamp.Time,
			Count:          int(k8sEvent.Count),
		}

		// 如果事件与特定节点相关，设置节点ID
		if k8sEvent.InvolvedObject.Kind == "Node" {
			node, err := w.nodeRepo.GetByName(clusterID.String(), k8sEvent.InvolvedObject.Name)
			if err == nil && node != nil {
				event.NodeID = &node.ID
			}
		}

		// 创建事件
		if err := w.eventRepo.Create(event); err != nil {
			log.Printf("Failed to create event: %v", err)
		}
	}

	log.Printf("Synced %d events for cluster %s", len(recentEvents), clusterID)
}

func (w *ResourceSyncWorker) syncClusterResources(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) []corev1.Node {
	// 获取节点信息以计算资源使用情况
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list nodes for resource sync: %v", err)
		return nil
	}

	var totalCPUCores int
	var totalMemoryBytes int64
	var totalStorageBytes int64

	for _, node := range nodes.Items {
		if allocatable := node.Status.Allocatable; allocatable != nil {
			totalCPUCores += int(allocatable.Cpu().MilliValue() / 1000)
			totalMemoryBytes += allocatable.Memory().Value()
			if capacity := node.Status.Capacity; capacity != nil {
				if storage, ok := capacity["ephemeral-storage"]; ok {
					totalStorageBytes += storage.Value()
				}
			}
		}
	}

	// 获取Pod信息以计算资源使用情况
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list pods for resource sync: %v", err)
		return nil
	}

	var usedCPUCores float64
	var usedMemoryBytes int64
	var usedStorageBytes int64

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			for _, container := range pod.Spec.Containers {
				if requests := container.Resources.Requests; requests != nil {
					usedCPUCores += float64(requests.Cpu().MilliValue()) / 1000.0
					usedMemoryBytes += requests.Memory().Value()
				}
			}
		}
	}

	// 计算使用百分比
	cpuUsagePercent := 0.0
	memoryUsagePercent := 0.0
	storageUsagePercent := 0.0

	if totalCPUCores > 0 {
		cpuUsagePercent = (usedCPUCores / float64(totalCPUCores)) * 100
	}
	if totalMemoryBytes > 0 {
		memoryUsagePercent = (float64(usedMemoryBytes) / float64(totalMemoryBytes)) * 100
	}
	if totalStorageBytes > 0 {
		// 获取已使用的存储信息
		usedStorageBytes = calculateUsedStorageBytes(clientset, ctx)
		storageUsagePercent = (float64(usedStorageBytes) / float64(totalStorageBytes)) * 100
	} else {
		usedStorageBytes = 0
	}

	// 创建集群资源记录
	clusterResource := &model.ClusterResource{
		ClusterID:           clusterID,
		Timestamp:           time.Now(),
		TotalCPUCores:       totalCPUCores,
		UsedCPUCores:        usedCPUCores,
		CPUUsagePercent:     cpuUsagePercent,
		TotalMemoryBytes:    totalMemoryBytes,
		UsedMemoryBytes:     usedMemoryBytes,
		MemoryUsagePercent:  memoryUsagePercent,
		TotalStorageBytes:   totalStorageBytes,
		UsedStorageBytes:    usedStorageBytes,
		StorageUsagePercent: storageUsagePercent,
	}

	// 使用Upsert方法创建或更新集群资源记录
	if err := w.clusterResourceRepo.Upsert(clusterResource); err != nil {
		log.Printf("Failed to upsert cluster resource: %v", err)
	}

	// 只更新 last_sync_at 字段，表示资源同步已完成
	// 资源数据存储在 cluster_resources 表中，API会从该表获取最新数据
	err = w.clusterStateRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.ClusterState{}).Where("cluster_id = ?", clusterID.String()).Updates(map[string]interface{}{
			"last_sync_at": time.Now(),
			"updated_at":   time.Now(),
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Failed to update sync timestamp for cluster %s: %v", clusterID, err)
	} else {
		log.Printf("Resource sync completed for cluster %s", clusterID)
	}

	log.Printf("Synced cluster resources for cluster %s", clusterID)

	// 返回节点列表，供其他函数使用
	return nodes.Items
}

func (w *ResourceSyncWorker) getNodeStatus(node corev1.Node) string {
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

func (w *ResourceSyncWorker) getNodeType(node corev1.Node) string {
	// 检查是否有master角色标签
	if _, hasMasterLabel := node.Labels["node-role.kubernetes.io/master"]; hasMasterLabel {
		return "control-plane"
	}

	// 检查是否有control-plane角色标签
	if _, hasControlPlaneLabel := node.Labels["node-role.kubernetes.io/control-plane"]; hasControlPlaneLabel {
		return "control-plane"
	}

	// 默认返回worker
	return "worker"
}

func (w *ResourceSyncWorker) getNodeLabels(node corev1.Node) model.JSONMap {
	labels := make(model.JSONMap)
	for k, v := range node.Labels {
		labels[k] = v
	}
	return labels
}

func (w *ResourceSyncWorker) getEventSeverity(eventType string) string {
	switch eventType {
	case "Warning":
		return "warning"
	case "Normal":
		return "info"
	default:
		return "info"
	}
}

func (w *ResourceSyncWorker) TriggerSync(clusterID uuid.UUID) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		cluster, err := w.clusterRepo.GetByID(clusterID.String())
		if err != nil {
			log.Printf("Failed to get cluster for resource sync: %v", err)
			return
		}
		w.syncClusterData(*cluster)
	}()
}

// syncAutoscalingPolicies 同步自动扩缩容策略
func (w *ResourceSyncWorker) syncAutoscalingPolicies(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) {
	log.Printf("Syncing autoscaling policies for cluster %s", clusterID)

	// 获取HPA策略
	hpaList, err := clientset.AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list HPAs: %v", err)
		return
	}

	// 检查Cluster Autoscaler是否启用
	clusterAutoscalerEnabled := false
	deployments, err := clientset.AppsV1().Deployments("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "app=cluster-autoscaler",
	})
	if err == nil && len(deployments.Items) > 0 {
		clusterAutoscalerEnabled = true
	}

	// 创建或更新自动扩缩容策略
	policy := &model.AutoscalingPolicy{
		ID:                       uuid.New(),
		ClusterID:                clusterID,
		Enabled:                  len(hpaList.Items) > 0 || clusterAutoscalerEnabled,
		HPACount:                 len(hpaList.Items),
		ClusterAutoscalerEnabled: clusterAutoscalerEnabled,
	}

	// 设置默认值
	policy.MinNodes = 1
	policy.MaxNodes = 10
	policy.ScaleUpThreshold = 70
	policy.ScaleDownThreshold = 30

	// 如果存在HPA，从第一个HPA获取阈值作为默认值
	if len(hpaList.Items) > 0 {
		hpa := hpaList.Items[0]
		if len(hpa.Spec.Metrics) > 0 && hpa.Spec.Metrics[0].Resource != nil {
			if hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != nil {
				policy.ScaleUpThreshold = int(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
			}
		}
	}

	// 使用Upsert方法创建或更新策略
	if err := w.autoscalingPolicyRepo.Upsert(policy); err != nil {
		log.Printf("Failed to upsert autoscaling policy: %v", err)
	}

	log.Printf("Synced autoscaling policies for cluster %s", clusterID)
}

// syncSecurityPolicies 同步安全策略
func (w *ResourceSyncWorker) syncSecurityPolicies(ctx context.Context, clientset *kubernetes.Clientset, clusterID string) error {
	// 创建安全策略检测器
	detector := service.NewSecurityPolicyDetector(clientset)

	// 检测RBAC状态
	rbacStatus, err := detector.DetectRBAC(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect RBAC status: %w", err)
	}

	// 检测网络策略状态
	networkPolicyStatus, err := detector.DetectNetworkPolicy(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect network policy status: %w", err)
	}

	// 检测Pod安全策略
	podSecurityStatus, err := detector.DetectPodSecurity(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect pod security status: %w", err)
	}

	// 检测审计日志状态
	auditLoggingStatus, err := detector.DetectAuditLogging(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect audit logging status: %w", err)
	}

	// 创建安全策略模型
	securityPolicy := &model.SecurityPolicy{
		ID:                     uuid.New(), // 生成新的 UUID
		ClusterID:              uuid.MustParse(clusterID),
		PodSecurityStandard:    podSecurityStatus.Standard,
		NetworkPoliciesEnabled: networkPolicyStatus.Enabled,
		RBACEnabled:            rbacStatus.Enabled,
		AuditLoggingEnabled:    auditLoggingStatus.Enabled,
		// 添加新字段
		RBACDetails:              rbacStatus.Details,
		NetworkPolicyDetails:     networkPolicyStatus.Details,
		PodSecurityDetails:       podSecurityStatus.Details,
		AuditLoggingDetails:      auditLoggingStatus.Details,
		NetworkPoliciesCount:     networkPolicyStatus.PoliciesCount,
		RBACRolesCount:           rbacStatus.RolesCount,
		CNIPlugin:                networkPolicyStatus.CNIPlugin,
		PodSecurityAdmissionMode: podSecurityStatus.AdmissionMode,
	}

	// 保存到数据库
	if err := w.securityPolicyRepo.Upsert(ctx, securityPolicy); err != nil {
		return fmt.Errorf("failed to save security policy: %w", err)
	}

	return nil
}

// calculateUsedStorageBytes 计算集群中已使用的存储字节数
func calculateUsedStorageBytes(clientset *kubernetes.Clientset, ctx context.Context) int64 {
	var totalUsedStorageBytes int64

	// 使用更高效的方法获取所有PVC，避免遍历每个命名空间
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list PVCs: %v", err)
		return 0
	}

	for _, pvc := range pvcs.Items {
		// 如果PVC已绑定到PV，获取已使用空间
		if pvc.Status.Phase == corev1.ClaimBound && pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
				totalUsedStorageBytes += storage.Value()
			}
		}
	}

	return totalUsedStorageBytes
}

// updateAPIServerURL 更新 cluster_state 中的 APIServerURL
func (w *ResourceSyncWorker) updateAPIServerURL(ctx context.Context, nodes []corev1.Node, clusterID uuid.UUID) {
	var apiServerURL string

	// 使用传入的节点信息推断，避免额外的API调用
	if len(nodes) > 0 {
		// 使用第一个节点的 IP 作为 APIServer URL
		for _, address := range nodes[0].Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				apiServerURL = "https://" + address.Address + ":6443"
				break
			}
		}
	}

	// 使用数据库事务更新 cluster_state 表中的 APIServerURL
	// 确保API服务器URL更新是原子的，不与其他Worker的更新冲突
	err := w.clusterStateRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.ClusterState{}).Where("cluster_id = ?", clusterID.String()).Updates(map[string]interface{}{
			"api_server_url": apiServerURL,
			"updated_at":     time.Now(),
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Failed to update APIServerURL for cluster %s: %v", clusterID, err)
	} else {
		log.Printf("Updated APIServerURL for cluster %s: %s", clusterID, apiServerURL)
	}
}
