package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
)

// InformerResourceSyncWorker 基于 Informer 模式的资源同步工作器
type InformerResourceSyncWorker struct {
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

	// Informer 管理
	clusterInformers map[uuid.UUID]*ClusterInformer
	informerMutex    sync.RWMutex

	// 缓存层
	cache ResourceCache
}

// NewInformerResourceSyncWorker 创建一个新的基于 Informer 模式的资源同步工作器
func NewInformerResourceSyncWorker(
	clusterRepo *repository.ClusterRepository,
	nodeRepo *repository.NodeRepository,
	eventRepo *repository.EventRepository,
	clusterResourceRepo *repository.ClusterResourceRepository,
	clusterStateRepo *repository.ClusterStateRepository,
	autoscalingPolicyRepo *repository.AutoscalingPolicyRepository,
	securityPolicyRepo *repository.SecurityPolicyRepository,
	clusterManager *service.ClusterManager,
	encryptionSvc *service.EncryptionService,
) *InformerResourceSyncWorker {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建缓存，TTL 为 10 分钟
	cache := NewMemoryResourceCache(10 * time.Minute)

	return &InformerResourceSyncWorker{
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
		syncInterval:          5 * time.Minute, // 修改为5分钟同步
		maxConcurrency:        3,               // 降低并发数
		sem:                   make(chan struct{}, 3),
		clusterInformers:      make(map[uuid.UUID]*ClusterInformer),
		cache:                 cache,
	}
}

// Start 启动 Informer 资源同步工作器
func (w *InformerResourceSyncWorker) Start() {
	log.Println("Starting informer-based resource sync worker...")

	// 初始化所有集群的 Informer
	w.initializeClusterInformers()

	// 定期检查集群变化，添加或删除 Informer
	w.wg.Add(1)
	go w.clusterChangeMonitor()

	// 立即执行一次策略同步
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.syncPoliciesForAllClusters()
	}()

	// 定期执行非 Informer 支持的同步任务（如自动扩缩容策略和安全策略）
	w.wg.Add(1)
	go w.policySyncScheduler()
}

// Stop 停止 Informer 资源同步工作器
func (w *InformerResourceSyncWorker) Stop() {
	log.Println("Stopping informer-based resource sync worker...")
	w.cancel()

	// 停止所有 Informer
	w.informerMutex.Lock()
	for _, informer := range w.clusterInformers {
		informer.Stop()
	}
	w.clusterInformers = make(map[uuid.UUID]*ClusterInformer)
	w.informerMutex.Unlock()

	w.wg.Wait()
}

// initializeClusterInformers 初始化所有现有集群的 Informer
func (w *InformerResourceSyncWorker) initializeClusterInformers() {
	clusters, err := w.clusterRepo.FindActiveClusters()
	if err != nil {
		log.Printf("Failed to fetch clusters: %v", err)
		return
	}

	log.Printf("Initializing informers for %d active clusters", len(clusters))

	for _, cluster := range clusters {
		w.addClusterInformer(*cluster)
	}
}

// clusterChangeMonitor 监控集群变化，添加或删除 Informer
func (w *InformerResourceSyncWorker) clusterChangeMonitor() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.syncClusterInformers()
		}
	}
}

// policySyncScheduler 定期执行策略同步任务
func (w *InformerResourceSyncWorker) policySyncScheduler() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.syncPoliciesForAllClusters()
		}
	}
}

// syncClusterInformers 同步集群 Informer，添加新集群或删除已不存在的集群
func (w *InformerResourceSyncWorker) syncClusterInformers() {
	log.Println("Syncing cluster informers...")

	clusters, err := w.clusterRepo.FindActiveClusters()
	if err != nil {
		log.Printf("Failed to fetch clusters: %v", err)
		return
	}

	// 创建当前活跃集群的映射
	activeClusters := make(map[uuid.UUID]bool)
	for _, cluster := range clusters {
		activeClusters[cluster.ID] = true
	}

	// 删除不再活跃的集群的 Informer
	w.informerMutex.Lock()
	for clusterID, informer := range w.clusterInformers {
		if !activeClusters[clusterID] {
			log.Printf("Stopping informer for inactive cluster %s", clusterID.String())
			informer.Stop()
			delete(w.clusterInformers, clusterID)
		}
	}
	w.informerMutex.Unlock()

	// 为新集群添加 Informer
	for _, cluster := range clusters {
		w.addClusterInformer(*cluster)
	}
}

// addClusterInformer 为集群添加 Informer
func (w *InformerResourceSyncWorker) addClusterInformer(cluster model.Cluster) {
	w.informerMutex.Lock()
	defer w.informerMutex.Unlock()

	// 检查是否已存在
	if _, exists := w.clusterInformers[cluster.ID]; exists {
		return
	}

	// 解密 kubeconfig
	kubeconfig, err := w.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		log.Printf("Failed to decrypt kubeconfig for cluster %s: %v", cluster.Name, err)
		return
	}

	// 创建 Informer
	informer, err := NewClusterInformer(
		w.ctx,
		cluster.ID,
		kubeconfig,
		w.nodeRepo,
		w.eventRepo,
		w.clusterResourceRepo,
		w.cache,
	)
	if err != nil {
		log.Printf("Failed to create informer for cluster %s: %v", cluster.Name, err)
		return
	}

	// 保存并启动 Informer
	w.clusterInformers[cluster.ID] = informer
	go informer.Start()

	log.Printf("Started informer for cluster %s", cluster.Name)
}

// syncPoliciesForAllClusters 为所有集群同步策略
func (w *InformerResourceSyncWorker) syncPoliciesForAllClusters() {
	log.Println("Starting policy sync for all clusters...")

	clusters, err := w.clusterRepo.FindActiveClusters()
	if err != nil {
		log.Printf("Failed to fetch clusters: %v", err)
		return
	}

	for _, cluster := range clusters {
		w.wg.Add(1)
		go func(c model.Cluster) {
			defer w.wg.Done()
			w.syncClusterPolicies(c)
		}(*cluster)
	}
}

// syncClusterPolicies 同步单个集群的策略
func (w *InformerResourceSyncWorker) syncClusterPolicies(cluster model.Cluster) {
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

	// 同步自动扩缩容策略
	w.syncAutoscalingPolicies(ctx, clientset, cluster.ID)

	// 同步安全策略
	if err := w.syncSecurityPolicies(ctx, clientset, cluster.ID.String()); err != nil {
		log.Printf("Failed to sync security policies: %v", err)
	}

	// 更新 APIServer URL
	w.updateAPIServerURL(ctx, clientset, cluster.ID)

	log.Printf("Policy sync completed for cluster %s", cluster.Name)
}

// syncAutoscalingPolicies 同步自动扩缩容策略（从原始资源同步工作器复制）
func (w *InformerResourceSyncWorker) syncAutoscalingPolicies(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) {
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
		ID:                       uuid.New(), // 生成新的 UUID
		ClusterID:                clusterID,
		HPACount:                 len(hpaList.Items),
		ClusterAutoscalerEnabled: clusterAutoscalerEnabled,
	}

	// 保存到数据库
	if err := w.autoscalingPolicyRepo.Upsert(policy); err != nil {
		log.Printf("Failed to save autoscaling policy: %v", err)
		return
	}

	log.Printf("Synced autoscaling policies for cluster %s", clusterID)
}

// syncSecurityPolicies 同步安全策略（从原始资源同步工作器复制）
func (w *InformerResourceSyncWorker) syncSecurityPolicies(ctx context.Context, clientset *kubernetes.Clientset, clusterID string) error {
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
		RBACEnabled:            rbacStatus.Enabled,
		RBACDetails:            rbacStatus.Details,
		NetworkPoliciesEnabled: networkPolicyStatus.Enabled,
		NetworkPolicyDetails:   networkPolicyStatus.Details,
		// PodSecurityStandard 替代 PodSecurityEnabled
		PodSecurityStandard: podSecurityStatus.Standard,
		PodSecurityDetails:  podSecurityStatus.Details,
		AuditLoggingEnabled: auditLoggingStatus.Enabled,
		AuditLoggingDetails: auditLoggingStatus.Details,
	}

	// 保存到数据库
	if err := w.securityPolicyRepo.Upsert(context.Background(), securityPolicy); err != nil {
		return fmt.Errorf("failed to save security policy: %w", err)
	}

	log.Printf("Synced security policies for cluster %s", clusterID)
	return nil
}

// updateAPIServerURL 更新 cluster_state 中的 APIServerURL
func (w *InformerResourceSyncWorker) updateAPIServerURL(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) {
	// 获取 APIServer URL（通过配置获取）
	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, "kube-apiserver-original", metav1.GetOptions{})
	var apiServerURL string
	if err == nil {
		if url, ok := config.Data["api-server-url"]; ok {
			apiServerURL = url
		}
	}

	// 如果没有找到，尝试从节点信息推断
	if apiServerURL == "" {
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err == nil && len(nodes.Items) > 0 {
			// 使用第一个节点的 IP 作为 APIServer URL
			apiServerURL = "https://" + nodes.Items[0].Status.Addresses[0].Address + ":6443"
		}
	}

	// 使用数据库事务更新 cluster_state 表中的 APIServerURL
	// 确保API服务器URL更新是原子的，不与其他Worker的更新冲突
	err = w.clusterStateRepo.GetDB().Transaction(func(tx *gorm.DB) error {
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
