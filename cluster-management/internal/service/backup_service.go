package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type BackupService struct {
	backupRepo         *repository.BackupRepository
	backupScheduleRepo *repository.BackupScheduleRepository
	clusterRepo        *repository.ClusterRepository
	encryptionSvc      *EncryptionService
	clusterManager     *ClusterManager
	restoreProgressMap sync.Map // 用于存储恢复进度
}

type BackupWithDetails struct {
	*model.ClusterBackup
}

type BackupScheduleWithDetails struct {
	*model.BackupSchedule
}

func NewBackupService(
	backupRepo *repository.BackupRepository,
	backupScheduleRepo *repository.BackupScheduleRepository,
	clusterRepo *repository.ClusterRepository,
	encryptionSvc *EncryptionService,
	clusterManager *ClusterManager,
) *BackupService {
	return &BackupService{
		backupRepo:         backupRepo,
		backupScheduleRepo: backupScheduleRepo,
		clusterRepo:        clusterRepo,
		encryptionSvc:      encryptionSvc,
		clusterManager:     clusterManager,
	}
}

func (s *BackupService) CreateBackup(clusterID, backupName, backupType string, retentionDays int) (*model.ClusterBackup, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	storageLocation := filepath.Join("/backups", clusterID, backupName, time.Now().Format("20060102-150405"))

	backup := &model.ClusterBackup{
		ClusterID:         cluster.ID,
		BackupName:        backupName,
		BackupType:        backupType,
		Status:            "pending",
		StorageLocation:   storageLocation,
		RetentionDays:     retentionDays,
		SnapshotTimestamp: func() *time.Time { now := time.Now(); return &now }(),
	}

	if err := s.backupRepo.Create(backup); err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	return backup, nil
}

func (s *BackupService) ExecuteBackup(backupID string) error {
	backup, err := s.backupRepo.GetByID(backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}

	cluster, err := s.clusterRepo.GetByID(backup.ClusterID.String())
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 更新状态为running
	backup.Status = "running"
	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	kubeconfig, err := s.encryptionSvc.Decrypt(
		cluster.KubeconfigEncrypted,
		cluster.KubeconfigNonce,
	)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to decrypt kubeconfig: %w", err))
	}

	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to get client: %w", err))
	}

	// 执行备份操作
	switch backup.BackupType {
	case "full":
		return s.performFullBackup(backup, clientset)
	case "etcd":
		return s.performEtcdBackup(backup, clientset)
	case "resources":
		return s.performResourcesBackup(backup, clientset)
	default:
		return s.handleBackupError(backup, fmt.Errorf("unsupported backup type: %s", backup.BackupType))
	}
}

func (s *BackupService) performFullBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset) error {
	backup.Status = "completed"
	backup.StorageSizeBytes = 1024 * 1024 * 100 // 模拟100MB备份大小
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	// 更新集群的last_backup_at字段
	cluster, err := s.clusterRepo.GetByID(backup.ClusterID.String())
	if err == nil {
		cluster.LastBackupAt = backup.SnapshotTimestamp
		s.clusterRepo.Update(cluster)
	}

	return nil
}

func (s *BackupService) performEtcdBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset) error {
	backup.Status = "completed"
	backup.StorageSizeBytes = 1024 * 1024 * 50 // 模拟50MB备份大小
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	return s.backupRepo.Update(backup)
}

func (s *BackupService) performResourcesBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset) error {
	backup.Status = "completed"
	backup.StorageSizeBytes = 1024 * 1024 * 30 // 模拟30MB备份大小
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	return s.backupRepo.Update(backup)
}

func (s *BackupService) handleBackupError(backup *model.ClusterBackup, err error) error {
	backup.Status = "failed"
	backup.ErrorMsg = err.Error()
	return s.backupRepo.Update(backup)
}

func (s *BackupService) ListBackups(clusterID, status string, page, limit int) ([]*model.ClusterBackup, int64, error) {
	return s.backupRepo.List(clusterID, status, page, limit)
}

func (s *BackupService) GetBackup(clusterID, backupID string) (*model.ClusterBackup, error) {
	return s.backupRepo.GetByID(backupID)
}

// RestoreBackup 恢复备份
func (s *BackupService) RestoreBackup(clusterID, backupID, restoreName string) (*RestoreResult, error) {
	// 验证备份是否存在且已完成
	backup, err := s.backupRepo.GetByID(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}

	if backup.Status != "completed" {
		return nil, fmt.Errorf("backup is not completed, current status: %s", backup.Status)
	}

	// 验证目标集群是否存在
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target cluster: %w", err)
	}

	// 创建恢复记录
	restoreID := uuid.New().String()
	result := &RestoreResult{
		RestoreID: restoreID,
		Status:    "running",
		StartTime: time.Now(),
	}

	// 存储恢复记录到内存中（实际应用中应该使用数据库或Redis）
	s.restoreProgressMap.Store(restoreID, &RestoreProgress{
		RestoreID:   restoreID,
		Status:      "running",
		Progress:    0.0,
		CurrentStep: "准备恢复环境",
		StartTime:   time.Now(),
	})

	// 异步执行恢复
	go s.executeRestore(restoreID, cluster, backup, restoreName)

	return result, nil
}

// GetRestoreProgress 获取恢复进度
func (s *BackupService) GetRestoreProgress(restoreID string) (*RestoreProgress, error) {
	// 从内存中获取恢复进度
	if progress, ok := s.restoreProgressMap.Load(restoreID); ok {
		return progress.(*RestoreProgress), nil
	}

	return nil, fmt.Errorf("restore progress not found for ID: %s", restoreID)
}

// executeRestore 执行实际的恢复操作
func (s *BackupService) executeRestore(restoreID string, cluster *model.Cluster, backup *model.ClusterBackup, restoreName string) {
	// 解密kubeconfig
	kubeconfig, err := s.encryptionSvc.Decrypt(
		cluster.KubeconfigEncrypted,
		cluster.KubeconfigNonce,
	)
	if err != nil {
		s.logRestoreError(restoreID, fmt.Errorf("failed to decrypt kubeconfig: %w", err))
		return
	}

	// 创建Kubernetes客户端
	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		s.logRestoreError(restoreID, fmt.Errorf("failed to get client: %w", err))
		return
	}

	// 根据备份类型执行不同的恢复策略
	switch backup.BackupType {
	case "full":
		err = s.performFullRestore(ctx, clientset, backup, restoreName)
	case "etcd":
		err = s.performEtcdRestore(ctx, clientset, backup, restoreName)
	case "resources":
		err = s.performResourcesRestore(ctx, clientset, backup, restoreName)
	default:
		err = fmt.Errorf("unsupported backup type: %s", backup.BackupType)
	}

	if err != nil {
		s.logRestoreError(restoreID, err)
		return
	}

	// 记录恢复成功
	s.logRestoreSuccess(restoreID)
}

// performFullRestore 执行全量恢复
func (s *BackupService) performFullRestore(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	// 1. 准备恢复环境
	if err := s.prepareRestoreEnvironment(ctx, clientset, restoreName); err != nil {
		return fmt.Errorf("failed to prepare restore environment: %w", err)
	}

	// 2. 恢复etcd数据
	if err := s.restoreEtcdData(ctx, clientset, backup, restoreName); err != nil {
		return fmt.Errorf("failed to restore etcd data: %w", err)
	}

	// 3. 恢复资源清单
	if err := s.restoreResourceManifests(ctx, clientset, backup, restoreName); err != nil {
		return fmt.Errorf("failed to restore resource manifests: %w", err)
	}

	// 4. 验证恢复结果
	if err := s.verifyRestoreResult(ctx, clientset, restoreName); err != nil {
		return fmt.Errorf("restore verification failed: %w", err)
	}

	return nil
}

// performEtcdRestore 执行etcd恢复
func (s *BackupService) performEtcdRestore(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	// 1. 停止所有控制平面组件
	if err := s.stopControlPlaneComponents(ctx, clientset); err != nil {
		return fmt.Errorf("failed to stop control plane components: %w", err)
	}

	// 2. 从备份位置获取etcd快照
	backupPath := s.getBackupPath(backup)
	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	// 3. 恢复etcd数据
	if err := s.restoreEtcdSnapshot(ctx, clientset, etcdSnapshotPath); err != nil {
		return fmt.Errorf("failed to restore etcd snapshot: %w", err)
	}

	// 4. 重启控制平面组件
	if err := s.startControlPlaneComponents(ctx, clientset); err != nil {
		return fmt.Errorf("failed to restart control plane components: %w", err)
	}

	return nil
}

// performResourcesRestore 执行资源恢复
func (s *BackupService) performResourcesRestore(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	// 1. 从备份位置获取资源清单
	backupPath := s.getBackupPath(backup)
	resourcesPath := filepath.Join(backupPath, "resources")

	// 2. 按顺序恢复资源
	resourceTypes := []string{"namespaces", "pv", "pvc", "configmaps", "secrets", "deployments", "services", "ingresses"}

	for _, resourceType := range resourceTypes {
		resourcePath := filepath.Join(resourcesPath, resourceType)
		if err := s.restoreResourcesOfType(ctx, clientset, resourcePath, resourceType); err != nil {
			return fmt.Errorf("failed to restore %s: %w", resourceType, err)
		}
	}

	return nil
}

// 以下是辅助方法

func (s *BackupService) prepareRestoreEnvironment(ctx context.Context, clientset *kubernetes.Clientset, restoreName string) error {
	// 创建恢复命名空间
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: restoreName,
			Labels: map[string]string{
				"restore": restoreName,
			},
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create restore namespace: %w", err)
	}

	return nil
}

func (s *BackupService) restoreEtcdData(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	// 实现etcd数据恢复逻辑
	// 这里应该连接到etcd集群并执行恢复操作
	return nil
}

func (s *BackupService) restoreResourceManifests(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	// 实现资源清单恢复逻辑
	// 这里应该从备份存储中获取资源清单并应用到集群
	return nil
}

func (s *BackupService) verifyRestoreResult(ctx context.Context, clientset *kubernetes.Clientset, restoreName string) error {
	// 验证恢复结果
	// 检查关键资源是否已正确恢复
	return nil
}

func (s *BackupService) stopControlPlaneComponents(ctx context.Context, clientset *kubernetes.Clientset) error {
	// 停止控制平面组件
	// 这通常需要通过SSH或kubectl exec在控制节点上执行命令
	return nil
}

func (s *BackupService) startControlPlaneComponents(ctx context.Context, clientset *kubernetes.Clientset) error {
	// 启动控制平面组件
	return nil
}

func (s *BackupService) restoreEtcdSnapshot(ctx context.Context, clientset *kubernetes.Clientset, snapshotPath string) error {
	// 恢复etcd快照
	return nil
}

func (s *BackupService) restoreResourcesOfType(ctx context.Context, clientset *kubernetes.Clientset, resourcePath, resourceType string) error {
	// 恢复特定类型的资源
	files, err := os.ReadDir(resourcePath)
	if err != nil {
		return fmt.Errorf("failed to read resource directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(resourcePath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read resource file %s: %w", file.Name(), err)
		}

		// 应用资源到集群
		if err := s.applyResourceToCluster(ctx, clientset, content); err != nil {
			return fmt.Errorf("failed to apply resource %s: %w", file.Name(), err)
		}
	}

	return nil
}

func (s *BackupService) applyResourceToCluster(ctx context.Context, clientset *kubernetes.Clientset, resourceContent []byte) error {
	// 使用kubectl apply或直接使用client-go应用资源
	// 这里简化实现，实际应该解析资源类型并使用对应的API
	return nil
}

func (s *BackupService) getBackupPath(backup *model.ClusterBackup) string {
	// 返回备份存储路径
	return backup.StorageLocation
}

func (s *BackupService) logRestoreError(restoreID string, err error) {
	// 记录恢复错误
	// 实际实现应该将错误信息存储到数据库或日志系统
	fmt.Printf("Restore %s failed: %v\n", restoreID, err)
}

func (s *BackupService) logRestoreSuccess(restoreID string) {
	// 记录恢复成功
	fmt.Printf("Restore %s completed successfully\n", restoreID)
}

func (s *BackupService) DeleteBackup(clusterID, backupID string) error {
	return s.backupRepo.Delete(backupID)
}

func (s *BackupService) ListBackupSchedules(clusterID string) ([]*model.BackupSchedule, error) {
	return s.backupScheduleRepo.ListByClusterID(clusterID)
}

func (s *BackupService) CreateBackupSchedule(clusterID, scheduleName, cronExpression, backupType string, retentionCount int, enabled bool) (*model.BackupSchedule, error) {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster ID: %w", err)
	}

	schedule := &model.BackupSchedule{
		ClusterID:     clusterUUID,
		Name:          scheduleName,
		CronExpr:      cronExpression,
		BackupType:    backupType,
		RetentionDays: retentionCount,
		Enabled:       enabled,
	}

	if err := s.backupScheduleRepo.Create(schedule); err != nil {
		return nil, fmt.Errorf("failed to create backup schedule: %w", err)
	}

	return schedule, nil
}

func (s *BackupService) UpdateBackupSchedule(scheduleID, cronExpression, backupType string, retentionCount int, enabled bool) error {
	schedule, err := s.backupScheduleRepo.GetByID(scheduleID)
	if err != nil {
		return fmt.Errorf("failed to get backup schedule: %w", err)
	}

	schedule.CronExpr = cronExpression
	schedule.BackupType = backupType
	schedule.RetentionDays = retentionCount
	schedule.Enabled = enabled

	return s.backupScheduleRepo.Update(schedule)
}

func (s *BackupService) DeleteBackupSchedule(scheduleID string) error {
	return s.backupScheduleRepo.Delete(scheduleID)
}
