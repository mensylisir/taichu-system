package worker

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

// ClusterInformer 管理 Kubernetes 集群的 Informer
type ClusterInformer struct {
	clusterID           uuid.UUID
	kubeconfig          string
	clientset           *kubernetes.Clientset
	nodeInformer        cache.SharedIndexInformer
	pvcInformer         cache.SharedIndexInformer
	eventInformer       cache.SharedIndexInformer
	stopCh              chan struct{}
	nodeRepo            *repository.NodeRepository
	eventRepo           *repository.EventRepository
	clusterResourceRepo *repository.ClusterResourceRepository
	cache               ResourceCache // 添加缓存层
}

// NewClusterInformer 创建一个新的集群 Informer
func NewClusterInformer(
	clusterID uuid.UUID,
	kubeconfig string,
	nodeRepo *repository.NodeRepository,
	eventRepo *repository.EventRepository,
	clusterResourceRepo *repository.ClusterResourceRepository,
	cache ResourceCache, // 添加缓存参数
) (*ClusterInformer, error) {
	// 使用 kubeconfig 创建客户端配置
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	informer := &ClusterInformer{
		clusterID:           clusterID,
		kubeconfig:          kubeconfig,
		clientset:           clientset,
		stopCh:              make(chan struct{}),
		nodeRepo:            nodeRepo,
		eventRepo:           eventRepo,
		clusterResourceRepo: clusterResourceRepo,
		cache:               cache,
	}

	// 初始化 Informer
	informer.initInformers()

	return informer, nil
}

// initInformers 初始化所有需要的 Informer
func (ci *ClusterInformer) initInformers() {
	// Node Informer
	ci.nodeInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return ci.clientset.CoreV1().Nodes().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return ci.clientset.CoreV1().Nodes().Watch(context.TODO(), options)
			},
		},
		&corev1.Node{},
		time.Minute*10, // 重新同步周期
		cache.Indexers{},
	)

	// PVC Informer
	ci.pvcInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return ci.clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return ci.clientset.CoreV1().PersistentVolumeClaims("").Watch(context.TODO(), options)
			},
		},
		&corev1.PersistentVolumeClaim{},
		time.Minute*10, // 重新同步周期
		cache.Indexers{},
	)

	// Event Informer
	ci.eventInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return ci.clientset.CoreV1().Events("").List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return ci.clientset.CoreV1().Events("").Watch(context.TODO(), options)
			},
		},
		&corev1.Event{},
		time.Minute*10, // 重新同步周期
		cache.Indexers{},
	)

	// 设置事件处理程序
	ci.setupEventHandlers()
}

// setupEventHandlers 设置 Informer 的事件处理程序
func (ci *ClusterInformer) setupEventHandlers() {
	// Node 事件处理
	ci.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			ci.handleNodeAdd(node)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			node := newObj.(*corev1.Node)
			ci.handleNodeUpdate(node)
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			ci.handleNodeDelete(node)
		},
	})

	// PVC 事件处理
	ci.pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pvc := obj.(*corev1.PersistentVolumeClaim)
			ci.handlePVCAdd(pvc)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pvc := newObj.(*corev1.PersistentVolumeClaim)
			ci.handlePVCUpdate(pvc)
		},
		DeleteFunc: func(obj interface{}) {
			pvc := obj.(*corev1.PersistentVolumeClaim)
			ci.handlePVCDelete(pvc)
		},
	})

	// Event 事件处理
	ci.eventInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event := obj.(*corev1.Event)
			ci.handleEventAdd(event)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			event := newObj.(*corev1.Event)
			ci.handleEventUpdate(event)
		},
	})
}

// Start 启动 Informer
func (ci *ClusterInformer) Start() {
	log.Printf("Starting informers for cluster %s", ci.clusterID.String())
	go ci.nodeInformer.Run(ci.stopCh)
	go ci.pvcInformer.Run(ci.stopCh)
	go ci.eventInformer.Run(ci.stopCh)

	// 等待缓存同步
	if !cache.WaitForCacheSync(ci.stopCh, ci.nodeInformer.HasSynced, ci.pvcInformer.HasSynced, ci.eventInformer.HasSynced) {
		log.Printf("Failed to sync caches for cluster %s", ci.clusterID.String())
		return
	}

	log.Printf("Successfully synced caches for cluster %s", ci.clusterID.String())

	// 缓存同步后，立即更新一次集群资源统计
	ci.updateClusterResourceStats()

	// 启动定时更新集群资源统计的goroutine，每5分钟更新一次
	go ci.startPeriodicResourceStatsUpdate()
}

// Stop 停止 Informer
func (ci *ClusterInformer) Stop() {
	log.Printf("Stopping informers for cluster %s", ci.clusterID.String())
	close(ci.stopCh)
}

// handleNodeAdd 处理节点添加事件
func (ci *ClusterInformer) handleNodeAdd(node *corev1.Node) {
	// 减少日志输出，避免冲掉正常应用日志
	// log.Printf("Node added: %s in cluster %s", node.Name, ci.clusterID.String())
	// 转换为内部模型并保存到数据库
	ci.syncNodeToDB(node)
	// 移除频繁的集群资源统计更新，改为定时更新
}

// handleNodeUpdate 处理节点更新事件
func (ci *ClusterInformer) handleNodeUpdate(node *corev1.Node) {
	// 减少日志输出，避免冲掉正常应用日志
	// log.Printf("Node updated: %s in cluster %s", node.Name, ci.clusterID.String())
	// 转换为内部模型并更新到数据库
	ci.syncNodeToDB(node)
	// 移除频繁的集群资源统计更新，改为定时更新
}

// handleNodeDelete 处理节点删除事件
func (ci *ClusterInformer) handleNodeDelete(node *corev1.Node) {
	// 减少日志输出，避免冲掉正常应用日志
	// log.Printf("Node deleted: %s in cluster %s", node.Name, ci.clusterID.String())
	// 从数据库中删除节点
	ci.deleteNodeFromDB(node)
	// 移除频繁的集群资源统计更新，改为定时更新
}

// handlePVCAdd 处理 PVC 添加事件
func (ci *ClusterInformer) handlePVCAdd(pvc *corev1.PersistentVolumeClaim) {
	// 减少日志输出，避免冲掉正常应用日志
	// log.Printf("PVC added: %s/%s in cluster %s", pvc.Namespace, pvc.Name, ci.clusterID.String())
	// 移除频繁的集群资源统计更新，改为定时更新
}

// handlePVCUpdate 处理 PVC 更新事件
func (ci *ClusterInformer) handlePVCUpdate(pvc *corev1.PersistentVolumeClaim) {
	// 减少日志输出，避免冲掉正常应用日志
	// log.Printf("PVC updated: %s/%s in cluster %s", pvc.Namespace, pvc.Name, ci.clusterID.String())
	// 移除频繁的集群资源统计更新，改为定时更新
}

// handlePVCDelete 处理 PVC 删除事件
func (ci *ClusterInformer) handlePVCDelete(pvc *corev1.PersistentVolumeClaim) {
	log.Printf("PVC deleted: %s/%s in cluster %s", pvc.Namespace, pvc.Name, ci.clusterID.String())
	// 移除频繁的集群资源统计更新，改为定时更新
}

// handleEventAdd 处理事件添加
func (ci *ClusterInformer) handleEventAdd(event *corev1.Event) {
	// Informer模式下需要处理事件存储，因为传统模式的5分钟同步不会运行
	// 只处理重要事件，避免数据库急剧增长
	if ci.isImportantEvent(event) {
		ci.syncEventToDB(event)
	}
}

// handleEventUpdate 处理事件更新
func (ci *ClusterInformer) handleEventUpdate(event *corev1.Event) {
	// Informer模式下需要处理事件存储，因为传统模式的5分钟同步不会运行
	// 只处理重要事件，避免数据库急剧增长
	if ci.isImportantEvent(event) {
		ci.syncEventToDB(event)
	}
}

// syncNodeToDB 同步节点信息到数据库
func (ci *ClusterInformer) syncNodeToDB(node *corev1.Node) {
	// 检查节点是否已存在
	existingNode, err := ci.nodeRepo.GetByName(ci.clusterID.String(), node.Name)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Printf("Failed to check existing node %s: %v", node.Name, err)
		return
	}

	// 转换节点信息
	internalNode := &model.Node{
		ClusterID: ci.clusterID,
		Name:      node.Name,
		Type:      ci.getNodeType(node),
		Status:    ci.getNodeStatus(node),
		Labels:    ci.getNodeLabels(node),
	}

	// 如果节点已存在，使用原有ID
	if err == nil && existingNode != nil {
		internalNode.ID = existingNode.ID
	}

	// 设置资源信息
	if allocatable := node.Status.Allocatable; allocatable != nil {
		internalNode.CPUCores = int(allocatable.Cpu().MilliValue() / 1000)
		internalNode.MemoryBytes = allocatable.Memory().Value()
	}

	// 使用UpsertSingle方法创建或更新节点
	if err := ci.nodeRepo.UpsertSingle(internalNode); err != nil {
		log.Printf("Failed to upsert node %s: %v", node.Name, err)
	}
}

// deleteNodeFromDB 从数据库中删除节点
func (ci *ClusterInformer) deleteNodeFromDB(node *corev1.Node) {
	// 获取节点信息
	existingNode, err := ci.nodeRepo.GetByName(ci.clusterID.String(), node.Name)
	if err != nil {
		log.Printf("Failed to get node %s for deletion: %v", node.Name, err)
		return
	}

	// 从数据库删除节点
	if err := ci.nodeRepo.Delete(existingNode.ID); err != nil {
		log.Printf("Failed to delete node %s: %v", node.Name, err)
	}
}

// getNodeStatus 获取节点状态
func (ci *ClusterInformer) getNodeStatus(node *corev1.Node) string {
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

// getNodeType 获取节点类型
func (ci *ClusterInformer) getNodeType(node *corev1.Node) string {
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

// getNodeLabels 获取节点标签
func (ci *ClusterInformer) getNodeLabels(node *corev1.Node) model.JSONMap {
	labels := make(model.JSONMap)
	for k, v := range node.Labels {
		labels[k] = v
	}
	return labels
}

// updateClusterResourceStats 更新集群资源统计
func (ci *ClusterInformer) updateClusterResourceStats() {
	// 从 Informer 缓存中获取所有 PVC
	pvcs := ci.pvcInformer.GetStore().List()

	var totalUsedStorageBytes int64
	for _, item := range pvcs {
		pvc := item.(*corev1.PersistentVolumeClaim)
		// 如果PVC已绑定到PV，获取已使用空间
		if pvc.Status.Phase == corev1.ClaimBound && pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
				totalUsedStorageBytes += storage.Value()
			}
		}
	}

	// 从 Informer 缓存中获取所有节点
	nodes := ci.nodeInformer.GetStore().List()
	var totalCPUCores int
	var totalMemoryBytes int64
	var totalStorageBytes int64

	for _, item := range nodes {
		node := item.(*corev1.Node)
		// 计算节点资源
		for resourceName, alloc := range node.Status.Allocatable {
			if resourceName == corev1.ResourceCPU {
				totalCPUCores += int(alloc.Value())
			} else if resourceName == corev1.ResourceMemory {
				totalMemoryBytes += alloc.Value()
			} else if resourceName == corev1.ResourceEphemeralStorage {
				totalStorageBytes += alloc.Value()
			}
		}
	}

	// 获取Pod信息以计算资源使用情况
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pods, err := ci.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list pods for resource calculation: %v", err)
		return
	}

	var usedCPUCores float64
	var usedMemoryBytes int64
	podCount := 0

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podCount++
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
		storageUsagePercent = (float64(totalUsedStorageBytes) / float64(totalStorageBytes)) * 100
	}

	// 创建集群资源统计对象
	clusterResource := &model.ClusterResource{
		ClusterID:           ci.clusterID,
		Timestamp:           time.Now(),
		TotalCPUCores:       totalCPUCores,
		UsedCPUCores:        usedCPUCores,
		CPUUsagePercent:     cpuUsagePercent,
		TotalMemoryBytes:    totalMemoryBytes,
		UsedMemoryBytes:     usedMemoryBytes,
		MemoryUsagePercent:  memoryUsagePercent,
		TotalStorageBytes:   totalStorageBytes,
		UsedStorageBytes:    totalUsedStorageBytes,
		StorageUsagePercent: storageUsagePercent,
	}

	// 保存到数据库
	if err := ci.clusterResourceRepo.Upsert(clusterResource); err != nil {
		log.Printf("Failed to update cluster resource stats: %v", err)
	} else {
		// 减少日志输出频率，只在调试模式下输出
		// log.Printf("Updated cluster resource stats for cluster %s", ci.clusterID.String())
	}
}

// isImportantEvent 判断事件是否重要，只同步重要事件以避免数据库急剧增长
func (ci *ClusterInformer) isImportantEvent(event *corev1.Event) bool {
	// 只同步警告级别的事件
	if event.Type == "Warning" {
		return true
	}

	// 只同步最近30分钟内的事件，避免存储过多历史事件
	if event.LastTimestamp.Time.Before(time.Now().Add(-30 * time.Minute)) {
		return false
	}

	// 同步与节点、Pod、PVC相关的重要正常事件
	importantComponents := map[string]bool{
		"kubelet":                 true,
		"kube-controller-manager": true,
		"kube-scheduler":          true,
		"container-runtime":       true,
		"network-plugin":          true,
		"volume-scheduler":        true,
		"attach-detach":           true,
	}

	if importantComponents[event.Source.Component] {
		return true
	}

	// 过滤掉一些频繁但不重要的事件
	ignoredReasons := map[string]bool{
		"Started":   true,
		"Created":   true,
		"Pulled":    true,
		"Scheduled": true,
	}

	if ignoredReasons[event.Reason] {
		return false
	}

	return true
}

// syncEventToDB 将事件同步到数据库
func (ci *ClusterInformer) syncEventToDB(event *corev1.Event) {
	// 检查事件是否已存在
	exists, err := ci.eventRepo.CheckExists(
		ci.clusterID.String(),
		event.Type,
		event.Message,
		event.Source.Component,
		event.FirstTimestamp.Time,
		event.LastTimestamp.Time,
	)
	if err != nil {
		log.Printf("Failed to check event existence: %v", err)
		return
	}
	if exists {
		return // 跳过已存在的事件
	}

	// 转换事件信息
	dbEvent := &model.Event{
		ID:             uuid.New(), // 生成新的UUID
		ClusterID:      ci.clusterID,
		EventType:      event.Type,
		Message:        event.Message,
		Severity:       ci.getEventSeverity(event.Type),
		Component:      event.Source.Component,
		FirstTimestamp: event.FirstTimestamp.Time,
		LastTimestamp:  event.LastTimestamp.Time,
		Count:          int(event.Count),
	}

	// 如果事件与特定节点相关，设置节点ID
	if event.InvolvedObject.Kind == "Node" {
		node, err := ci.nodeRepo.GetByName(ci.clusterID.String(), event.InvolvedObject.Name)
		if err == nil && node != nil {
			dbEvent.NodeID = &node.ID
		}
	}

	// 创建事件
	if err := ci.eventRepo.Create(dbEvent); err != nil {
		log.Printf("Failed to create event: %v", err)
	}
}

// getEventSeverity 获取事件严重性级别
func (ci *ClusterInformer) getEventSeverity(eventType string) string {
	switch eventType {
	case "Warning":
		return "warning"
	case "Normal":
		return "info"
	default:
		return "unknown"
	}
}

// startPeriodicResourceStatsUpdate 启动定时更新集群资源统计的goroutine
func (ci *ClusterInformer) startPeriodicResourceStatsUpdate() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟更新一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ci.updateClusterResourceStats()
		case <-ci.stopCh:
			log.Printf("Stopping periodic resource stats update for cluster %s", ci.clusterID.String())
			return
		}
	}
}
