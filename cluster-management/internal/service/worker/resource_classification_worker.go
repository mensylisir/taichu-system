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
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
	"gorm.io/gorm"
)

// ResourceClassificationWorker 存量资源分类工作器
type ResourceClassificationWorker struct {
	clusterRepo         *repository.ClusterRepository
	encryptionSvc       *service.EncryptionService
	clusterManager      *service.ClusterManager
	tenantRepo          *repository.TenantRepository
	environmentRepo     *repository.EnvironmentRepository
	applicationRepo     *repository.ApplicationRepository
	quotaRepo           *repository.QuotaRepository
	resourceClassifier  *service.ResourceClassifier
	classificationRepo  *repository.ResourceClassificationRepository
	wg                  sync.WaitGroup
	ctx                 context.Context
	cancel              context.CancelFunc
	syncInterval        time.Duration
	maxConcurrency      int
	sem                 chan struct{}
}

// NewResourceClassificationWorker 创建存量资源分类工作器
func NewResourceClassificationWorker(
	clusterRepo *repository.ClusterRepository,
	encryptionSvc *service.EncryptionService,
	clusterManager *service.ClusterManager,
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	classificationRepo *repository.ResourceClassificationRepository,
) *ResourceClassificationWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceClassificationWorker{
		clusterRepo:         clusterRepo,
		encryptionSvc:       encryptionSvc,
		clusterManager:      clusterManager,
		tenantRepo:          tenantRepo,
		environmentRepo:     environmentRepo,
		applicationRepo:     applicationRepo,
		quotaRepo:           quotaRepo,
		classificationRepo:  classificationRepo,
		ctx:                 ctx,
		cancel:              cancel,
		syncInterval:        10 * time.Minute, // 每10分钟检查一次
		maxConcurrency:      5,
		sem:                 make(chan struct{}, 5),
	}
}

// Start 启动工作器
func (w *ResourceClassificationWorker) Start() {
	log.Println("Starting ResourceClassificationWorker...")

	// 执行一次初始分类
	w.performResourceClassification()

	w.wg.Add(1)
	go w.scheduler()
}

// Stop 停止工作器
func (w *ResourceClassificationWorker) Stop() {
	log.Println("Stopping ResourceClassificationWorker...")
	w.cancel()
	w.wg.Wait()
}

// scheduler 调度器
func (w *ResourceClassificationWorker) scheduler() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.performResourceClassification()
		}
	}
}

// performResourceClassification 执行资源分类
func (w *ResourceClassificationWorker) performResourceClassification() {
	log.Println("Starting scheduled resource classification...")

	// 获取所有活跃集群
	clusters, err := w.clusterRepo.FindActiveClusters()
	if err != nil {
		log.Printf("Failed to fetch clusters: %v", err)
		return
	}

	log.Printf("Found %d active clusters for resource classification", len(clusters))

	for _, cluster := range clusters {
		w.wg.Add(1)
		go func(c model.Cluster) {
			defer w.wg.Done()
			w.classifyClusterResources(c)
		}(*cluster)
	}
}

// classifyClusterResources 分类单个集群的资源
func (w *ResourceClassificationWorker) classifyClusterResources(cluster model.Cluster) {
	w.sem <- struct{}{}
	defer func() { <-w.sem }()

	log.Printf("Classifying resources for cluster: %s", cluster.Name)

	// 创建分类历史记录
	history := &model.ResourceClassificationHistory{
		ClusterID:   cluster.ID,
		TriggerType: model.ClassificationTriggerTypeScheduled,
		Status:      model.ClassificationHistoryStatusRunning,
		Details:     make(model.JSONMap),
	}

	if w.classificationRepo != nil {
		if err := w.classificationRepo.CreateHistory(history); err != nil {
			log.Printf("Failed to create classification history for cluster %s: %v", cluster.Name, err)
		}
	}

	startTime := time.Now()

	// 解密kubeconfig
	kubeconfig, err := w.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		log.Printf("Failed to decrypt kubeconfig for cluster %s: %v", cluster.Name, err)
		w.updateHistoryStatus(history, model.ClassificationHistoryStatusFailed, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(w.ctx, 60*time.Second)
	defer cancel()

	// 获取K8s客户端
	clientset, err := w.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		log.Printf("Failed to get client for cluster %s: %v", cluster.Name, err)
		w.updateHistoryStatus(history, model.ClassificationHistoryStatusFailed, err.Error())
		return
	}

	// 1. 检查并分类未分类的命名空间
	classifiedCount, failedCount := w.classifyUnclassifiedNamespaces(ctx, clientset, cluster.ID, history)

	// 2. 检查并补充缺失的应用
	appCount := w.supplementMissingApplications(ctx, clientset, cluster.ID)

	// 3. 同步ResourceQuota信息
	quotaCount := w.syncResourceQuotas(ctx, clientset, cluster.ID)

	// 更新历史记录统计信息
	duration := int(time.Since(startTime).Seconds())
	history.TotalResources = classifiedCount + failedCount
	history.ClassifiedResources = classifiedCount
	history.FailedResources = failedCount
	history.NewEnvironments = classifiedCount
	history.NewApplications = appCount
	history.NewTenants = w.countNewTenants(cluster.ID.String())
	history.DurationSeconds = duration

	// 更新详细信息
	if history.Details == nil {
		history.Details = make(model.JSONMap)
	}
	history.Details["classified_namespaces"] = classifiedCount
	history.Details["failed_namespaces"] = failedCount
	history.Details["new_applications"] = appCount
	history.Details["synced_quotas"] = quotaCount

	w.updateHistoryStatus(history, model.ClassificationHistoryStatusCompleted, "")

	log.Printf("Resource classification completed for cluster: %s, classified: %d, failed: %d, new apps: %d, duration: %ds",
		cluster.Name, classifiedCount, failedCount, appCount, duration)
}

// classifyUnclassifiedNamespaces 分类未分类的命名空间
func (w *ResourceClassificationWorker) classifyUnclassifiedNamespaces(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID, history *model.ResourceClassificationHistory) (int, int) {
	// 获取所有命名空间
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
		return 0, 0
	}

	classifiedCount := 0
	failedCount := 0

	for _, ns := range namespaces.Items {
		// 检查命名空间是否已分类（通过检查是否在environments表中存在）
		existingEnv, err := w.environmentRepo.GetByNamespace(clusterID.String(), ns.Name)
		if err == nil {
			// 已存在，跳过
			continue
		}

		if err != gorm.ErrRecordNotFound {
			log.Printf("Error checking namespace %s: %v", ns.Name, err)
			failedCount++
			continue
		}

		// 未分类，进行分类
		log.Printf("Found unclassified namespace: %s", ns.Name)
		if w.classifySingleNamespace(ctx, clientset, clusterID, ns, history) {
			classifiedCount++
		} else {
			failedCount++
		}
		_ = existingEnv // 标记已使用
	}

	return classifiedCount, failedCount
}

// classifySingleNamespace 分类单个命名空间
func (w *ResourceClassificationWorker) classifySingleNamespace(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID, namespace corev1.Namespace, history *model.ResourceClassificationHistory) bool {
	// 初始化资源分类器（如果需要）
	if w.resourceClassifier == nil {
		w.resourceClassifier = service.NewResourceClassifier(
			clientset,
			w.tenantRepo,
			w.environmentRepo,
			w.applicationRepo,
			w.quotaRepo,
		)
	}

	// 执行分类
	result, err := w.resourceClassifier.ClassifyNamespace(clusterID.String(), namespace)
	if err != nil {
		log.Printf("Failed to classify namespace %s: %v", namespace.Name, err)

		// 记录分类失败
		if w.classificationRepo != nil {
			classification := &model.ResourceClassification{
				ClusterID:       clusterID,
				ResourceType:    "namespace",
				ResourceName:    namespace.Name,
				Namespace:       namespace.Name,
				Status:          model.ClassificationStatusFailed,
				ErrorMessage:    err.Error(),
			}
			_ = w.classificationRepo.Create(classification)
		}

		return false
	}

	log.Printf("Classified namespace %s -> tenant: %s, created %d applications",
		namespace.Name, result.AssignedTenant.Name, len(result.Applications))

	// 记录分类成功
	if w.classificationRepo != nil {
		classification := &model.ResourceClassification{
			ClusterID:         clusterID,
			TenantID:          &result.AssignedTenant.ID,
			EnvironmentID:     &result.Environment.ID,
			ResourceType:      "namespace",
			ResourceName:      namespace.Name,
			Namespace:         namespace.Name,
			AssignedTenant:    result.AssignedTenant.Name,
			AssignedEnv:       result.Environment.Namespace,
			ClassificationRule: "auto",
			Status:            model.ClassificationStatusClassified,
		}
		_ = w.classificationRepo.Create(classification)
	}

	return true
}

// supplementMissingApplications 补充缺失的应用
func (w *ResourceClassificationWorker) supplementMissingApplications(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) int {
	// 获取所有命名空间
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
		return 0
	}

	appCount := 0

	for _, ns := range namespaces.Items {
		// 获取该命名空间下的所有Deployment
		deployments, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Failed to list deployments for namespace %s: %v", ns.Name, err)
			continue
		}

		for _, deployment := range deployments.Items {
			// 检查该Deployment是否已归类到应用
			// 这里简化处理，实际可以根据标签或名称匹配
			appName := w.extractAppName(deployment.Labels)
			if appName == "" {
				appName = deployment.Name
			}

			// 获取对应的环境
			env, err := w.environmentRepo.GetByNamespace(clusterID.String(), ns.Name)
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					log.Printf("Error getting environment for namespace %s: %v", ns.Name, err)
				}
				continue
			}

			// 检查应用是否已存在
			existingApp, err := w.applicationRepo.GetByName(env.ID.String(), appName)
			if err == nil {
				// 应用已存在，跳过
				continue
			}

			if err != gorm.ErrRecordNotFound {
				log.Printf("Error checking application %s: %v", appName, err)
				continue
			}

			// 创建新应用
			app := &model.Application{
				TenantID:        env.TenantID,
				EnvironmentID:   env.ID,
				Name:            appName,
				DisplayName:     appName,
				Description:     "自动补充的应用",
				Labels:          w.extractCommonLabels(deployment.Labels),
				WorkloadTypes:   []string{"Deployment"},
				DeploymentCount: 1,
			}

			if err := w.applicationRepo.Create(app); err != nil {
				log.Printf("Failed to create application %s: %v", appName, err)
			} else {
				log.Printf("Created missing application: %s in namespace %s", appName, ns.Name)
				appCount++

				// 记录应用分类
				if w.classificationRepo != nil {
					classification := &model.ResourceClassification{
						ClusterID:         clusterID,
						TenantID:          &env.TenantID,
						EnvironmentID:     &env.ID,
						ApplicationID:     &app.ID,
						ResourceType:      "deployment",
						ResourceName:      deployment.Name,
						Namespace:         ns.Name,
						AssignedTenant:    "",
						AssignedEnv:       env.Namespace,
						AssignedApp:       appName,
						ClassificationRule: "auto-supplement",
						Status:            model.ClassificationStatusClassified,
					}
					_ = w.classificationRepo.Create(classification)
				}
			}
			_ = existingApp // 标记已使用
		}
	}

	return appCount
}

// syncResourceQuotas 同步ResourceQuota信息
func (w *ResourceClassificationWorker) syncResourceQuotas(ctx context.Context, clientset *kubernetes.Clientset, clusterID uuid.UUID) int {
	// 获取所有命名空间
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
		return 0
	}

	quotaCount := 0

	for _, ns := range namespaces.Items {
		// 获取环境
		env, err := w.environmentRepo.GetByNamespace(clusterID.String(), ns.Name)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				log.Printf("Error getting environment for namespace %s: %v", ns.Name, err)
			}
			continue
		}

		// 检查是否存在ResourceQuota，使用单独的context和更长的超时时间
		quotaCtx, quotaCancel := context.WithTimeout(context.Background(), 30*time.Second)
		quotas, err := clientset.CoreV1().ResourceQuotas(ns.Name).List(quotaCtx, metav1.ListOptions{})
		quotaCancel()
		
		if err != nil {
			log.Printf("Failed to list quotas for namespace %s: %v", ns.Name, err)
			continue
		}

		if len(quotas.Items) == 0 {
			// 没有ResourceQuota，跳过
			continue
		}

		// 检查数据库中是否已有记录
		existingQuota, err := w.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if err == nil {
			// 已存在，跳过
			continue
		}

		if err != gorm.ErrRecordNotFound {
			log.Printf("Error checking quota for environment %s: %v", env.ID.String(), err)
			continue
		}

		// 创建新的ResourceQuota记录
		quota := quotas.Items[0]
		hardLimits := make(map[string]interface{})
		for k, v := range quota.Status.Hard {
			hardLimits[string(k)] = v.String()
		}

		used := make(map[string]interface{})
		for k, v := range quota.Status.Used {
			used[string(k)] = v.String()
		}

		now := time.Now()
		resourceQuota := &model.ResourceQuota{
			EnvironmentID: env.ID,
			HardLimits:    hardLimits,
			Used:          used,
			Status:        constants.StatusActive,
			LastSyncedAt:  &now,
		}

		if err := w.quotaRepo.CreateResourceQuota(resourceQuota); err != nil {
			log.Printf("Failed to create resource quota for environment %s: %v", env.ID.String(), err)
		} else {
			log.Printf("Synced resource quota for namespace %s", ns.Name)
			quotaCount++
		}
		_ = existingQuota // 标记已使用
	}

	return quotaCount
}

// extractAppName 提取应用名称
func (w *ResourceClassificationWorker) extractAppName(labels map[string]string) string {
	// 优先级顺序
	if name, ok := labels[model.AppLabelKubernetesName]; ok {
		return name
	}
	if app, ok := labels[model.AppLabelStandard]; ok {
		return app
	}
	if k8sApp, ok := labels[model.AppLabelK8sApp]; ok {
		return k8sApp
	}
	return ""
}

// extractCommonLabels 提取公共标签
func (w *ResourceClassificationWorker) extractCommonLabels(labels map[string]string) model.JSONMap {
	commonLabels := make(map[string]interface{})
	importantKeys := []string{
		model.AppLabelKubernetesName,
		model.AppLabelStandard,
		model.AppLabelK8sApp,
		"version",
		"component",
		"tier",
	}

	for _, key := range importantKeys {
		if val, ok := labels[key]; ok {
			commonLabels[key] = val
		}
	}

	return commonLabels
}

// TriggerClassification 手动触发分类（用于API调用）
func (w *ResourceClassificationWorker) TriggerClassification(clusterID string) error {
	// 获取集群
	cluster, err := w.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.classifyClusterResources(*cluster)
	}()

	return nil
}

// updateHistoryStatus 更新历史记录状态
func (w *ResourceClassificationWorker) updateHistoryStatus(history *model.ResourceClassificationHistory, status string, errorMessage string) {
	if history == nil {
		return
	}

	history.Status = status
	history.ErrorMessage = errorMessage

	if status == model.ClassificationHistoryStatusCompleted {
		now := time.Now()
		history.CompletedAt = &now
	}

	if w.classificationRepo != nil {
		_ = w.classificationRepo.UpdateHistory(history)
	}
}

// countNewTenants 统计新租户数
func (w *ResourceClassificationWorker) countNewTenants(clusterID string) int {
	environments, err := w.environmentRepo.ListByClusterID(clusterID)
	if err != nil {
		return 0
	}

	tenantSet := make(map[string]bool)
	for _, env := range environments {
		tenantSet[env.TenantID.String()] = true
	}

	return len(tenantSet)
}

// GetClassificationHistory 获取分类历史
func (w *ResourceClassificationWorker) GetClassificationHistory(clusterID string, limit int) ([]*model.ResourceClassificationHistory, error) {
	if w.classificationRepo == nil {
		return nil, fmt.Errorf("classification repository not initialized")
	}
	return w.classificationRepo.ListHistoryByCluster(clusterID, limit)
}

// GetLatestClassificationStatus 获取最新分类状态
func (w *ResourceClassificationWorker) GetLatestClassificationStatus(clusterID string) (*model.ResourceClassificationHistory, error) {
	if w.classificationRepo == nil {
		return nil, fmt.Errorf("classification repository not initialized")
	}
	return w.classificationRepo.GetLatestHistory(clusterID)
}

// GetClassificationStats 获取分类统计信息
func (w *ResourceClassificationWorker) GetClassificationStats(clusterID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	environments, err := w.environmentRepo.ListByClusterID(clusterID)
	if err != nil {
		return nil, err
	}

	applications, err := w.applicationRepo.List()
	if err != nil {
		return nil, err
	}

	tenantSet := make(map[string]bool)
	for _, env := range environments {
		tenantSet[env.TenantID.String()] = true
	}

	appCount := 0
	for _, app := range applications {
		for _, env := range environments {
			if app.EnvironmentID == env.ID {
				appCount++
				break
			}
		}
	}

	stats["total_tenants"] = len(tenantSet)
	stats["total_environments"] = len(environments)
	stats["total_applications"] = appCount

	return stats, nil
}
