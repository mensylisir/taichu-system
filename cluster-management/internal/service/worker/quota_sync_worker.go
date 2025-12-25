package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
)

// QuotaSyncWorker 配额同步工作器
type QuotaSyncWorker struct {
	kubernetesClient  *kubernetes.Clientset
	quotaRepo         *repository.QuotaRepository
	environmentRepo   *repository.EnvironmentRepository
	quotaService      *service.QuotaService
	syncInterval      time.Duration
	quit              chan bool
}

// NewQuotaSyncWorker 创建配额同步工作器
func NewQuotaSyncWorker(
	kubeClient *kubernetes.Clientset,
	quotaRepo *repository.QuotaRepository,
	environmentRepo *repository.EnvironmentRepository,
	quotaService *service.QuotaService,
	syncInterval time.Duration,
) *QuotaSyncWorker {
	return &QuotaSyncWorker{
		kubernetesClient: kubeClient,
		quotaRepo:        quotaRepo,
		environmentRepo:  environmentRepo,
		quotaService:     quotaService,
		syncInterval:     syncInterval,
		quit:             make(chan bool),
	}
}

// Start 启动工作器
func (w *QuotaSyncWorker) Start() {
	log.Printf("Starting QuotaSyncWorker with interval: %v", w.syncInterval)

	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.syncAllResourceQuotas()
		case <-w.quit:
			log.Println("QuotaSyncWorker stopped")
			return
		}
	}
}

// Stop 停止工作器
func (w *QuotaSyncWorker) Stop() {
	close(w.quit)
}

// syncAllResourceQuotas 同步所有资源配额
func (w *QuotaSyncWorker) syncAllResourceQuotas() {
	log.Println("Starting quota synchronization...")

	// 获取所有环境
	envs, _, err := w.environmentRepo.ListAll(0, 0)
	if err != nil {
		log.Printf("Failed to list environments: %v", err)
		return
	}

	successCount := 0
	failCount := 0

	for _, env := range envs {
		if err := w.syncResourceQuota(env.ID.String()); err != nil {
			log.Printf("Failed to sync quota for environment %s: %v", env.Namespace, err)
			failCount++
		} else {
			successCount++
		}
	}

	log.Printf("Quota synchronization completed: %d success, %d failed", successCount, failCount)
}

// syncResourceQuota 同步单个资源配额
func (w *QuotaSyncWorker) syncResourceQuota(environmentID string) error {
	// 获取环境信息
	env, err := w.environmentRepo.GetByID(environmentID)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	ctx := context.Background()

	// 获取K8s中的ResourceQuota
	quotaList, err := w.kubernetesClient.CoreV1().ResourceQuotas(env.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list resource quotas: %w", err)
	}

	if len(quotaList.Items) == 0 {
		log.Printf("No resource quota found for namespace %s", env.Namespace)
		return nil
	}

	// 取第一个ResourceQuota（假设每个命名空间只有一个）
	quota := quotaList.Items[0]

	// 转换为数据库格式
	hardLimits := make(map[string]interface{})
	for k, v := range quota.Status.Hard {
		hardLimits[string(k)] = v.String()
	}

	used := make(map[string]interface{})
	for k, v := range quota.Status.Used {
		used[string(k)] = v.String()
	}

	// 更新数据库
	resourceQuota, err := w.quotaRepo.GetResourceQuotaByEnvironmentID(environmentID)
	if err != nil {
		// 创建新记录
		now := time.Now()
		resourceQuota = &model.ResourceQuota{
			EnvironmentID: env.ID,
			HardLimits:    hardLimits,
			Used:          used,
			Status:        constants.StatusActive,
			LastSyncedAt:  &now,
		}
		return w.quotaRepo.CreateResourceQuota(resourceQuota)
	}

	// 更新现有记录
	now := time.Now()
	resourceQuota.HardLimits = hardLimits
	resourceQuota.Used = used
	resourceQuota.Status = constants.StatusActive
	resourceQuota.LastSyncedAt = &now

	return w.quotaRepo.UpdateResourceQuota(resourceQuota)
}
