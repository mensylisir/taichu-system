package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	etcdConfig         *EtcdNodeConfig // etcd配置信息
	sshService         *SSHService     // SSH服务
}

type BackupWithDetails struct {
	*model.ClusterBackup
}

// EtcdNodeConfig etcd节点配置
type EtcdNodeConfig struct {
	Nodes []EtcdNodeInfo
}

// EtcdNodeInfo 单个etcd节点信息
type EtcdNodeInfo struct {
	IP        string
	Port      int
	CACert    string
	Cert      string
	Key       string
	DataDir   string
	Endpoints []string
}

// RegisterEtcdConfig 注册etcd配置
func (s *BackupService) RegisterEtcdConfig(config *EtcdNodeConfig) {
	s.etcdConfig = config
}

// GetEtcdConfig 获取etcd配置
func (s *BackupService) GetEtcdConfig() *EtcdNodeConfig {
	return s.etcdConfig
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
		sshService:         NewSSHService(),
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

	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
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
		return s.performEtcdBackupStandalone(backup, clientset)
	case "resources":
		return s.performResourcesBackup(backup, clientset)
	default:
		return s.handleBackupError(backup, fmt.Errorf("unsupported backup type: %s", backup.BackupType))
	}
}

func (s *BackupService) performFullBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset) error {
	backup.StartedAt = func() *time.Time { now := time.Now(); return &now }()

	// 创建备份存储目录
	storage := NewBackupStorage("/backups")
	backupPath, err := storage.CreateBackupDirectory(backup.ClusterID.String(), backup.BackupName)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to create backup directory: %w", err))
	}

	// 1. 备份etcd数据（支持多种etcd部署方式）
	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	// 尝试自动检测并执行etcd备份
	if err := s.performEtcdBackup(backup, clientset, etcdSnapshotPath); err != nil {
		// 如果etcd备份失败，记录警告但不失败整个备份
		fmt.Printf("Warning: etcd backup failed: %v\n", err)
	} else {
		// 验证快照文件是否实际创建
		if _, err := os.Stat(etcdSnapshotPath); err != nil {
			return s.handleBackupError(backup, fmt.Errorf("etcd snapshot file was not created at %s", etcdSnapshotPath))
		}
		fmt.Println("Successfully created etcd snapshot")
	}

	// 2. 备份Kubernetes资源（真实资源导出）
	resourceService := NewResourceBackupService(clientset)
	if err := resourceService.BackupResources(context.Background(), backupPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to backup resources: %w", err))
	}
	fmt.Println("Successfully created Kubernetes resources backup")

	// 3. 压缩备份文件
	compressedPath := backupPath + ".tar.gz"
	if err := storage.CompressDirectory(backupPath, compressedPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to compress backup: %w", err))
	}

	// 4. 计算备份大小
	size, err := storage.CalculateDirectorySize(backupPath)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to calculate backup size: %w", err))
	}

	// 更新备份状态
	backup.Status = "completed"
	backup.StorageSizeBytes = size
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	// 清理临时文件
	os.RemoveAll(backupPath)

	// 更新集群的last_backup_at字段
	cluster, err := s.clusterRepo.GetByID(backup.ClusterID.String())
	if err == nil {
		cluster.LastBackupAt = backup.SnapshotTimestamp
		s.clusterRepo.Update(cluster)
	}

	return nil
}

func (s *BackupService) performEtcdBackupStandalone(backup *model.ClusterBackup, clientset *kubernetes.Clientset) error {
	backup.StartedAt = func() *time.Time { now := time.Now(); return &now }()

	storage := NewBackupStorage("/backups")
	backupPath, err := storage.CreateBackupDirectory(backup.ClusterID.String(), backup.BackupName)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to create backup directory: %w", err))
	}

	etcdService := NewEtcdBackupService(nil)
	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	if err := etcdService.CreateSnapshot(context.Background(), etcdSnapshotPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to backup etcd: %w", err))
	}

	size, err := storage.CalculateDirectorySize(backupPath)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to calculate backup size: %w", err))
	}

	backup.Status = "completed"
	backup.StorageSizeBytes = size
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	return s.backupRepo.Update(backup)
}

func (s *BackupService) performResourcesBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset) error {
	backup.StartedAt = func() *time.Time { now := time.Now(); return &now }()

	storage := NewBackupStorage("/backups")
	backupPath, err := storage.CreateBackupDirectory(backup.ClusterID.String(), backup.BackupName)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to create backup directory: %w", err))
	}

	resourceService := NewResourceBackupService(clientset)
	if err := resourceService.BackupResources(context.Background(), backupPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to backup resources: %w", err))
	}

	size, err := storage.CalculateDirectorySize(backupPath)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to calculate backup size: %w", err))
	}

	backup.Status = "completed"
	backup.StorageSizeBytes = size
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
	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
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
	// 获取备份路径
	backupPath := s.getBackupPath(backup)
	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	// 检查快照文件是否存在
	if _, err := os.Stat(etcdSnapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("etcd snapshot file not found at %s", etcdSnapshotPath)
	}

	// 创建etcd服务实例
	etcdService := NewEtcdBackupService(nil)

	// 停止控制平面组件
	if err := etcdService.StopControlPlaneComponents(ctx); err != nil {
		return fmt.Errorf("failed to stop control plane: %w", err)
	}

	// 恢复etcd快照
	if err := etcdService.RestoreSnapshot(ctx, etcdSnapshotPath, "/var/lib/etcd"); err != nil {
		return fmt.Errorf("failed to restore etcd snapshot: %w", err)
	}

	// 启动控制平面组件
	if err := etcdService.StartControlPlaneComponents(ctx); err != nil {
		return fmt.Errorf("failed to start control plane: %w", err)
	}

	// 等待etcd就绪
	if err := etcdService.WaitForEtcdReady(ctx, 60*time.Second); err != nil {
		return fmt.Errorf("etcd not ready after restore: %w", err)
	}

	return nil
}

func (s *BackupService) restoreResourceManifests(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	// 获取备份路径
	backupPath := s.getBackupPath(backup)
	resourcesPath := filepath.Join(backupPath, "resources")

	// 检查资源路径是否存在
	if _, err := os.Stat(resourcesPath); os.IsNotExist(err) {
		return fmt.Errorf("resources backup not found at %s", resourcesPath)
	}

	// 创建资源备份服务实例
	resourceService := NewResourceBackupService(clientset)

	// 恢复Kubernetes资源
	if err := resourceService.RestoreResources(ctx, backupPath); err != nil {
		return fmt.Errorf("failed to restore resources: %w", err)
	}

	return nil
}

func (s *BackupService) verifyRestoreResult(ctx context.Context, clientset *kubernetes.Clientset, restoreName string) error {
	// 等待系统稳定
	time.Sleep(10 * time.Second)

	// 检查API Server是否可用
	if err := s.checkAPIServerHealth(ctx, clientset); err != nil {
		return fmt.Errorf("API Server health check failed: %w", err)
	}

	// 检查核心命名空间是否存在
	namespaces := []string{"default", "kube-system"}
	for _, ns := range namespaces {
		if _, err := clientset.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{}); err != nil {
			return fmt.Errorf("namespace %s not found after restore: %w", ns, err)
		}
	}

	// 检查核心组件是否运行
	if err := s.checkCoreComponents(ctx, clientset); err != nil {
		return fmt.Errorf("core components check failed: %w", err)
	}

	return nil
}

// checkAPIServerHealth 检查API Server健康状态
func (s *BackupService) checkAPIServerHealth(ctx context.Context, clientset *kubernetes.Clientset) error {
	// 尝试获取节点列表
	_, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

// checkCoreComponents 检查核心组件状态
func (s *BackupService) checkCoreComponents(ctx context.Context, clientset *kubernetes.Clientset) error {
	// 检查kube-system命名空间中的核心组件
	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "component=kube-apiserver",
	})

	if err != nil {
		return fmt.Errorf("failed to check core components: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no API Server pods found in kube-system")
	}

	// 检查至少有一个pod处于Running状态
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return nil
		}
	}

	return fmt.Errorf("no running API Server pods found")
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
	// 检查快照文件是否存在
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file not found at %s", snapshotPath)
	}

	// 创建etcd服务实例
	etcdService := NewEtcdBackupService(nil)

	// 恢复快照到默认数据目录
	if err := etcdService.RestoreSnapshot(ctx, snapshotPath, "/var/lib/etcd"); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

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
	// 使用kubectl apply应用资源
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		stdin.Write(resourceContent)
	}()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply resource: %w, stderr: %s", err, stderr.String())
	}

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
		ClusterID:       clusterUUID,
		Name:            scheduleName,
		CronExpr:        cronExpression,
		BackupType:      backupType,
		RetentionDays:   retentionCount,
		Enabled:         enabled,
		CreatedBy:       "system",
		ScheduleName:    scheduleName,
		CronExpression:  cronExpression,
		RetentionCount:  retentionCount,
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

	// 同时更新数据库中旧的字段名
	schedule.CronExpression = cronExpression
	schedule.RetentionCount = retentionCount

	return s.backupScheduleRepo.Update(schedule)
}

func (s *BackupService) DeleteBackupSchedule(scheduleID string) error {
	return s.backupScheduleRepo.Delete(scheduleID)
}

// UpdateEtcdConfig 更新备份计划的etcd配置
func (s *BackupService) UpdateEtcdConfig(scheduleID, etcdEndpoints, etcdCaCert, etcdCert, etcdKey, etcdDataDir, etcdctlPath, sshUsername, sshPassword string) error {
	schedule, err := s.backupScheduleRepo.GetByID(scheduleID)
	if err != nil {
		return fmt.Errorf("failed to get backup schedule: %w", err)
	}

	if etcdEndpoints != "" {
		schedule.EtcdEndpoints = etcdEndpoints
	}
	if etcdCaCert != "" {
		schedule.EtcdCaCert = etcdCaCert
	}
	if etcdCert != "" {
		schedule.EtcdCert = etcdCert
	}
	if etcdKey != "" {
		schedule.EtcdKey = etcdKey
	}
	if etcdDataDir != "" {
		schedule.EtcdDataDir = etcdDataDir
	}
	if etcdctlPath != "" {
		schedule.EtcdctlPath = etcdctlPath
	}
	if sshUsername != "" {
		schedule.SshUsername = sshUsername
	}
	if sshPassword != "" {
		schedule.SshPassword = sshPassword
	}

	return s.backupScheduleRepo.Update(schedule)
}

// performEtcdBackup 智能检测并执行etcd备份（支持多种部署方式）
func (s *BackupService) performEtcdBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset, snapshotPath string) error {
	fmt.Printf("[ETCD-BACKUP] ===== Starting etcd backup for cluster %s =====\n", backup.ClusterID.String())
	fmt.Printf("[ETCD-BACKUP] ===== Snapshot path: %s =====\n", snapshotPath)

	// 策略1: 尝试通过SSH在standalone etcd节点上执行
	fmt.Printf("[ETCD-BACKUP] ===== Strategy 1: SSH backup =====\n")
	if err := s.backupEtcdViaSSH(backup.ClusterID.String(), snapshotPath); err == nil {
		fmt.Printf("[ETCD-BACKUP] ===== SSH backup succeeded =====\n")
		return nil
	}

	// 策略2: 尝试本地etcdctl（如果管理节点就是etcd节点）
	fmt.Printf("[ETCD-BACKUP] ===== Strategy 2: Local backup =====\n")
	if err := s.backupEtcdLocal(snapshotPath); err == nil {
		fmt.Printf("[ETCD-BACKUP] ===== Local backup succeeded =====\n")
		return nil
	}

	fmt.Printf("[ETCD-BACKUP] ===== All backup strategies failed =====\n")
	return fmt.Errorf("all etcd backup strategies failed")
}

// backupEtcdViaSSH 通过SSH在standalone etcd节点上执行etcd备份
func (s *BackupService) backupEtcdViaSSH(clusterID, snapshotPath string) error {
	fmt.Printf("[ETCD-BACKUP] Starting SSH-based etcd backup for cluster %s\n", clusterID)

	// 从backup_schedules表获取etcd配置
	schedules, err := s.backupScheduleRepo.ListByClusterID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to query backup schedules: %w", err)
	}
	if len(schedules) == 0 {
		return fmt.Errorf("no backup schedule found for cluster %s. Please create a backup schedule with etcd config", clusterID)
	}

	schedule := schedules[0]
	if schedule.EtcdEndpoints == "" {
		return fmt.Errorf("etcd_endpoints not set in backup schedule")
	}

	fmt.Printf("[ETCD-BACKUP] Using etcd endpoints: %s\n", schedule.EtcdEndpoints)

	// 从etcd_endpoints解析节点IP列表
	nodes := parseEndpointsToNodes(schedule.EtcdEndpoints)

	if len(nodes) == 0 {
		return fmt.Errorf("no etcd nodes found from endpoints")
	}

	fmt.Printf("[ETCD-BACKUP] Found %d etcd nodes: %v\n", len(nodes), nodes)

	// 在每个etcd节点上尝试备份
	for i, node := range nodes {
		fmt.Printf("[ETCD-BACKUP] Trying to backup from node %d/%d: %s\n", i+1, len(nodes), node)
		if err := s.backupEtcdFromNode(clusterID, node, snapshotPath); err == nil {
			fmt.Printf("[ETCD-BACKUP] Successfully backed up from node %s\n", node)
			return nil
		}
		fmt.Printf("[ETCD-BACKUP] Failed to backup from node %s: %v\n", node, err)
	}

	return fmt.Errorf("SSH backup failed - no accessible etcd node found")
}

// parseEndpointsToNodes 从endpoints解析节点IP列表
func parseEndpointsToNodes(endpoints string) []string {
	var nodes []string

	// 按逗号分割endpoints
	epList := strings.Split(endpoints, ",")

	for _, ep := range epList {
		// 解析host: https://172.30.1.12:2379 -> 172.30.1.12
		host := ep
		if strings.HasPrefix(host, "https://") {
			host = strings.TrimPrefix(host, "https://")
		} else if strings.HasPrefix(host, "http://") {
			host = strings.TrimPrefix(host, "http://")
		}
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
		nodes = append(nodes, host)
	}

	return nodes
}

// backupEtcdFromNode 从指定节点备份etcd
func (s *BackupService) backupEtcdFromNode(clusterID, nodeIP, snapshotPath string) error {
	fmt.Printf("[ETCD-BACKUP] Starting etcd backup for cluster %s from node %s\n", clusterID, nodeIP)

	// 使用 /backup/ 目录，与用户环境保持一致
	timestamp := time.Now().Format("20060102-150405")
	tmpFile := fmt.Sprintf("/backup/etcd-snapshot-%s.db", timestamp)
	fmt.Printf("[ETCD-BACKUP] Using temp file: %s\n", tmpFile)

	// 从backup_schedules表获取etcd配置
	schedules, err := s.backupScheduleRepo.ListByClusterID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to query backup schedules: %w", err)
	}
	if len(schedules) == 0 {
		return fmt.Errorf("no backup schedule found for cluster %s. Please create a backup schedule with etcd config", clusterID)
	}

	schedule := schedules[0]
	fmt.Printf("[ETCD-BACKUP] Found backup schedule: %s\n", schedule.Name)

	if schedule.EtcdEndpoints == "" || schedule.EtcdCaCert == "" || schedule.EtcdCert == "" || schedule.EtcdKey == "" {
		return fmt.Errorf("etcd config not found in backup schedule. Please set etcd_endpoints, etcd_ca_cert, etcd_cert, etcd_key")
	}

	fmt.Printf("[ETCD-BACKUP] etcd endpoints: %s\n", schedule.EtcdEndpoints)
	fmt.Printf("[ETCD-BACKUP] etcd ca cert: %s\n", schedule.EtcdCaCert)
	fmt.Printf("[ETCD-BACKUP] etcd cert: %s\n", schedule.EtcdCert)
	fmt.Printf("[ETCD-BACKUP] etcd key: %s\n", schedule.EtcdKey)
	fmt.Printf("[ETCD-BACKUP] etcdctl path: %s\n", schedule.EtcdctlPath)

	// 从etcd_endpoints中解析第一个host
	firstEndpoint := schedule.EtcdEndpoints
	if strings.Contains(firstEndpoint, ",") {
		firstEndpoint = strings.Split(firstEndpoint, ",")[0]
		fmt.Printf("[ETCD-BACKUP] Using first endpoint: %s\n", firstEndpoint)
	}

	// 解析host: https://172.30.1.12:2379 -> 172.30.1.12
	sshHost := firstEndpoint
	if strings.HasPrefix(sshHost, "https://") {
		sshHost = strings.TrimPrefix(sshHost, "https://")
	} else if strings.HasPrefix(sshHost, "http://") {
		sshHost = strings.TrimPrefix(sshHost, "http://")
	}
	if strings.Contains(sshHost, ":") {
		sshHost = strings.Split(sshHost, ":")[0]
	}

	fmt.Printf("[ETCD-BACKUP] Parsed SSH host: %s\n", sshHost)

	sshUsername := schedule.SshUsername
	if sshUsername == "" {
		sshUsername = "root"
	}
	fmt.Printf("[ETCD-BACKUP] SSH username: %s\n", sshUsername)

	sshPassword := schedule.SshPassword
	if sshPassword == "" {
		return fmt.Errorf("ssh_password not set in backup schedule")
	}
	fmt.Printf("[ETCD-BACKUP] SSH password is set: %s\n", "***")

	// 通过SSH服务连接远程节点
	fmt.Printf("[ETCD-BACKUP] Connecting to SSH...\n")
	client, err := s.sshService.Connect(sshHost, sshUsername, sshPassword)
	if err != nil {
		return fmt.Errorf("failed to create SSH client to %s: %w", sshHost, err)
	}
	defer client.Close()
	fmt.Printf("[ETCD-BACKUP] SSH connection established\n")

	// 确保远程节点存在 /backup 目录
	fmt.Printf("[ETCD-BACKUP] Creating /backup directory on remote node...\n")
	if _, err := client.ExecuteCommand("mkdir -p /backup"); err != nil {
		return fmt.Errorf("failed to create /backup directory on %s: %w", sshHost, err)
	}
	fmt.Printf("[ETCD-BACKUP] /backup directory ready\n")

	// 构建etcdctl备份命令（只使用单个端点）
	cmd := fmt.Sprintf("%s --endpoints=%s --cacert=%s --cert=%s --key=%s snapshot save %s",
		schedule.EtcdctlPath, firstEndpoint, schedule.EtcdCaCert, schedule.EtcdCert, schedule.EtcdKey, tmpFile)

	fmt.Printf("[ETCD-BACKUP] Executing command: %s\n", cmd)

	// 在远程节点执行etcdctl命令
	if _, err := client.ExecuteCommand(cmd); err != nil {
		return fmt.Errorf("failed to execute etcdctl on %s: %w", sshHost, err)
	}

	fmt.Printf("[ETCD-BACKUP] etcdctl command executed successfully\n")

	// 通过SFTP下载快照文件到本地
	fmt.Printf("[ETCD-BACKUP] Downloading snapshot file...\n")
	if err := client.DownloadFile(tmpFile, snapshotPath); err != nil {
		return fmt.Errorf("failed to download snapshot from %s: %w", sshHost, err)
	}
	fmt.Printf("[ETCD-BACKUP] Snapshot downloaded to: %s\n", snapshotPath)

	// 清理远程临时文件
	cleanupCmd := fmt.Sprintf("rm -f %s", tmpFile)
	fmt.Printf("[ETCD-BACKUP] Cleaning up temp file...\n")
	client.ExecuteCommand(cleanupCmd)
	fmt.Printf("[ETCD-BACKUP] Temp file cleaned up\n")

	// 验证本地快照文件
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file was not created at %s", snapshotPath)
	}

	fileInfo, err := os.Stat(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to stat snapshot file: %w", err)
	}
	fmt.Printf("[ETCD-BACKUP] Backup completed successfully, file size: %d bytes\n", fileInfo.Size())

	return nil
}

// backupEtcdLocal 在本地执行etcd备份
func (s *BackupService) backupEtcdLocal(snapshotPath string) error {
	// 检查本地是否有etcdctl
	cmd := exec.CommandContext(context.Background(), "which", "etcdctl")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("etcdctl not found locally")
	}

	// 确保输出目录存在
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 尝试本地快照
	etcdCmd := fmt.Sprintf("etcdctl --endpoints=https://127.0.0.1:2379 snapshot save %s", snapshotPath)
	cmd = exec.CommandContext(context.Background(), "sh", "-c", etcdCmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("local etcd backup failed: %w", err)
	}

	// 验证快照文件是否创建
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file was not created at %s", snapshotPath)
	}

	return nil
}

// CreateEtcdBackup 创建etcd备份记录
func (s *BackupService) CreateEtcdBackup(clusterID, backupName, backupType string, retentionDays int) (*model.ClusterBackup, error) {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster ID: %w", err)
	}

	backup := &model.ClusterBackup{
		ID:              uuid.New(),
		ClusterID:       clusterUUID,
		Name:            backupName,
		BackupName:      backupName,
		BackupType:      "etcd", // etcd备份固定为"etcd"类型
		Status:          "pending",
		RetentionDays:   retentionDays,
		SnapshotTimestamp: func() *time.Time { now := time.Now(); return &now }(),
		CreatedBy:       "system",
		StorageLocation: fmt.Sprintf("\\backups\\%s\\%s\\%s", clusterID, backupName, time.Now().Format("20060102-150405")),
	}

	if err := s.backupRepo.Create(backup); err != nil {
		return nil, fmt.Errorf("failed to create etcd backup: %w", err)
	}

	return backup, nil
}

// ExecuteEtcdBackup 执行etcd备份
func (s *BackupService) ExecuteEtcdBackup(backupID string) error {
	backup, err := s.backupRepo.GetByID(backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}

	// 更新状态为running
	backup.Status = "running"
	backup.StartedAt = func() *time.Time { now := time.Now(); return &now }()
	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	// 获取集群配置
	cluster, err := s.clusterRepo.GetByID(backup.ClusterID.String())
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to get cluster: %w", err))
	}

	// 解密kubeconfig
	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to decrypt kubeconfig: %w", err))
	}

	// 创建Kubernetes客户端
	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to get client: %w", err))
	}

	// 创建备份目录
	storage := NewBackupStorage("/backups")
	backupPath, err := storage.CreateBackupDirectory(backup.ClusterID.String(), backup.BackupName)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to create backup directory: %w", err))
	}

	// 执行etcd备份
	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	if err := s.performEtcdBackup(backup, clientset, etcdSnapshotPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("etcd backup failed: %w", err))
	}

	// 验证快照文件
	if _, err := os.Stat(etcdSnapshotPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("etcd snapshot file was not created at %s", etcdSnapshotPath))
	}

	// 计算备份大小
	size, err := storage.CalculateDirectorySize(backupPath)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to calculate backup size: %w", err))
	}

	// 更新备份成功状态
	backup.Status = "completed"
	backup.SizeBytes = size
	backup.StorageSizeBytes = size
	backup.Location = backupPath
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	fmt.Printf("Etcd backup completed successfully: %s\n", backup.ID.String())
	return nil
}

// CreateResourceBackup 创建资源备份记录
func (s *BackupService) CreateResourceBackup(clusterID, backupName, backupType string, retentionDays int) (*model.ClusterBackup, error) {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster ID: %w", err)
	}

	backup := &model.ClusterBackup{
		ID:              uuid.New(),
		ClusterID:       clusterUUID,
		Name:            backupName,
		BackupName:      backupName,
		BackupType:      "resources", // 资源备份固定为"resources"类型
		Status:          "pending",
		RetentionDays:   retentionDays,
		SnapshotTimestamp: func() *time.Time { now := time.Now(); return &now }(),
		CreatedBy:       "system",
		StorageLocation: fmt.Sprintf("\\backups\\%s\\%s\\%s", clusterID, backupName, time.Now().Format("20060102-150405")),
	}

	if err := s.backupRepo.Create(backup); err != nil {
		return nil, fmt.Errorf("failed to create resource backup: %w", err)
	}

	return backup, nil
}

// ExecuteResourceBackup 执行资源备份
func (s *BackupService) ExecuteResourceBackup(backupID string) error {
	backup, err := s.backupRepo.GetByID(backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}

	// 更新状态为running
	backup.Status = "running"
	backup.StartedAt = func() *time.Time { now := time.Now(); return &now }()
	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	// 获取集群配置
	cluster, err := s.clusterRepo.GetByID(backup.ClusterID.String())
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to get cluster: %w", err))
	}

	// 解密kubeconfig
	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to decrypt kubeconfig: %w", err))
	}

	// 创建Kubernetes客户端
	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to get client: %w", err))
	}

	// 创建备份目录
	storage := NewBackupStorage("/backups")
	backupPath, err := storage.CreateBackupDirectory(backup.ClusterID.String(), backup.BackupName)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to create backup directory: %w", err))
	}

	// 执行资源备份（通过clientset API）
	if err := s.backupResourcesViaClientset(ctx, clientset, backupPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to backup resources: %w", err))
	}

	// 计算备份大小
	size, err := storage.CalculateDirectorySize(backupPath)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to calculate backup size: %w", err))
	}

	// 更新备份成功状态
	backup.Status = "completed"
	backup.SizeBytes = size
	backup.StorageSizeBytes = size
	backup.Location = backupPath
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	if err := s.backupRepo.Update(backup); err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	fmt.Printf("Resource backup completed successfully: %s\n", backup.ID.String())
	return nil
}

// backupResourcesViaClientset 通过clientset API备份资源
func (s *BackupService) backupResourcesViaClientset(ctx context.Context, clientset *kubernetes.Clientset, backupPath string) error {
	// 使用ResourceBackupService进行资源备份
	resourceBackupService := NewResourceBackupService(clientset)

	fmt.Println("[RESOURCE-BACKUP] Starting resource backup via clientset")

	// 调用备份方法
	if err := resourceBackupService.BackupResources(ctx, backupPath); err != nil {
		return fmt.Errorf("failed to backup resources: %w", err)
	}

	fmt.Println("[RESOURCE-BACKUP] Resource backup completed successfully")
	return nil
}
