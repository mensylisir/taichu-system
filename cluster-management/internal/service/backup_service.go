package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"k8s.io/client-go/kubernetes"
)

type BackupService struct {
	backupRepo         *repository.BackupRepository
	backupScheduleRepo *repository.BackupScheduleRepository
	clusterRepo        *repository.ClusterRepository
	encryptionSvc      *EncryptionService
	clusterManager     *ClusterManager
	sshService         *SSHService
	alertService       *AlertService
	mu                 sync.RWMutex
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
	alertService *AlertService,
) *BackupService {
	return &BackupService{
		backupRepo:         backupRepo,
		backupScheduleRepo: backupScheduleRepo,
		clusterRepo:        clusterRepo,
		encryptionSvc:      encryptionSvc,
		clusterManager:     clusterManager,
		sshService:         NewSSHService(),
		alertService:       alertService,
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
		Status:            constants.StatusPending,
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

	backup.Status = constants.StatusRunning
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
	backup.Status = constants.StatusCompleted
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

	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	if err := s.performEtcdBackup(backup, clientset, etcdSnapshotPath); err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to backup etcd: %w", err))
	}

	size, err := storage.CalculateDirectorySize(backupPath)
	if err != nil {
		return s.handleBackupError(backup, fmt.Errorf("failed to calculate backup size: %w", err))
	}

	backup.Status = constants.StatusCompleted
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

	backup.Status = constants.StatusCompleted
	backup.StorageSizeBytes = size
	backup.CompletedAt = func() *time.Time { now := time.Now(); return &now }()

	return s.backupRepo.Update(backup)
}

func (s *BackupService) handleBackupError(backup *model.ClusterBackup, err error) error {
	backup.Status = constants.StatusFailed
	backup.ErrorMsg = err.Error()
		
	if err := s.backupRepo.Update(backup); err != nil {
		return err
	}

	if s.alertService != nil {
		s.alertService.AlertBackupFailed(backup.ID.String(), backup.ClusterID.String(), err.Error())
	}

	return nil
}

func (s *BackupService) ListBackups(clusterID, status string, page, limit int) ([]*model.ClusterBackup, int64, error) {
	return s.backupRepo.List(clusterID, status, page, limit)
}

func (s *BackupService) GetBackup(clusterID, backupID string) (*model.ClusterBackup, error) {
	return s.backupRepo.GetByID(backupID)
}

func (s *BackupService) DeleteBackup(clusterID, backupID string) error {
	return s.backupRepo.Delete(backupID)
}

func (s *BackupService) ListBackupSchedules(clusterID string) ([]*model.BackupSchedule, error) {
	return s.backupScheduleRepo.ListByClusterID(clusterID)
}

func (s *BackupService) CreateBackupSchedule(clusterID, scheduleName, cronExpression, backupType string, retentionCount int, enabled bool, etcdEndpoints, etcdCaCert, etcdCert, etcdKey, etcdDataDir, etcdctlPath, sshUsername, sshPassword, etcdDeploymentType, k8sDeploymentType, createdBy string) (*model.BackupSchedule, error) {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster ID: %w", err)
	}

	schedule := &model.BackupSchedule{
		ClusterID:          clusterUUID,
		Name:               scheduleName,
		CronExpr:           cronExpression,
		BackupType:         backupType,
		RetentionDays:      retentionCount,
		Enabled:            enabled,
		CreatedBy:          createdBy,
		ScheduleName:       scheduleName,
		CronExpression:     cronExpression,
		RetentionCount:     retentionCount,
		EtcdEndpoints:      etcdEndpoints,
		EtcdCaCert:         etcdCaCert,
		EtcdCert:           etcdCert,
		EtcdKey:            etcdKey,
		EtcdDataDir:        etcdDataDir,
		EtcdctlPath:        etcdctlPath,
		SshUsername:        sshUsername,
		SshPassword:        sshPassword,
		EtcdDeploymentType: etcdDeploymentType,
		K8sDeploymentType:  k8sDeploymentType,
	}

	if err := s.backupScheduleRepo.Create(schedule); err != nil {
		return nil, fmt.Errorf("failed to create backup schedule: %w", err)
	}

	return schedule, nil
}

func (s *BackupService) UpdateBackupSchedule(scheduleID, cronExpression, backupType string, retentionCount int, enabled bool, etcdEndpoints, etcdCaCert, etcdCert, etcdKey, etcdDataDir, etcdctlPath, sshUsername, sshPassword, etcdDeploymentType, k8sDeploymentType string) error {
	schedule, err := s.backupScheduleRepo.GetByID(scheduleID)
	if err != nil {
		return fmt.Errorf("failed to get backup schedule: %w", err)
	}

	schedule.CronExpr = cronExpression
	schedule.BackupType = backupType
	schedule.RetentionDays = retentionCount
	schedule.Enabled = enabled

	schedule.CronExpression = cronExpression
	schedule.RetentionCount = retentionCount

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
	if etcdDeploymentType != "" {
		schedule.EtcdDeploymentType = etcdDeploymentType
	}
	if k8sDeploymentType != "" {
		schedule.K8sDeploymentType = k8sDeploymentType
	}

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

// performEtcdBackup 智能检测并执行etcd备份（通过SSH在etcd节点上执行）
func (s *BackupService) performEtcdBackup(backup *model.ClusterBackup, clientset *kubernetes.Clientset, snapshotPath string) error {
	fmt.Printf("[ETCD-BACKUP] ===== Starting etcd backup for cluster %s =====\n", backup.ClusterID.String())
	fmt.Printf("[ETCD-BACKUP] ===== Snapshot path: %s =====\n", snapshotPath)

	fmt.Printf("[ETCD-BACKUP] ===== Strategy: SSH backup =====\n")
	err := s.backupEtcdViaSSH(backup.ClusterID.String(), snapshotPath)
	if err == nil {
		fmt.Printf("[ETCD-BACKUP] ===== SSH backup succeeded =====\n")
		return nil
	}

	fmt.Printf("[ETCD-BACKUP] ===== SSH backup failed: %v =====\n", err)
	return fmt.Errorf("etcd backup failed: %w", err)
}

// backupEtcdViaSSH 通过SSH在etcd节点上执行etcd备份
func (s *BackupService) backupEtcdViaSSH(clusterID, snapshotPath string) error {
	fmt.Printf("[ETCD-BACKUP] Starting SSH-based etcd backup for cluster %s\n", clusterID)

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

	if schedule.EtcdCaCert == "" || schedule.EtcdCert == "" || schedule.EtcdKey == "" {
		return fmt.Errorf("etcd config not found in backup schedule. Please set etcd_ca_cert, etcd_cert, etcd_key")
	}

	sshUsername := schedule.SshUsername
	if sshUsername == "" {
		sshUsername = "root"
	}
	fmt.Printf("[ETCD-BACKUP] SSH username: %s\n", sshUsername)

	sshPassword := schedule.SshPassword
	if sshPassword == "" {
		return fmt.Errorf("ssh_password not set in backup schedule")
	}
	fmt.Printf("[ETCD-BACKUP] SSH password is set\n")

	// 从etcd_endpoints解析第一个节点IP
	firstEndpoint := schedule.EtcdEndpoints
	if strings.Contains(firstEndpoint, ",") {
		firstEndpoint = strings.Split(firstEndpoint, ",")[0]
	}
	nodeIP := firstEndpoint
	if strings.HasPrefix(nodeIP, "https://") {
		nodeIP = strings.TrimPrefix(nodeIP, "https://")
	} else if strings.HasPrefix(nodeIP, "http://") {
		nodeIP = strings.TrimPrefix(nodeIP, "http://")
	}
	if strings.Contains(nodeIP, ":") {
		nodeIP = strings.Split(nodeIP, ":")[0]
	}
	fmt.Printf("[ETCD-BACKUP] Using etcd node: %s\n", nodeIP)

	timestamp := time.Now().Format("20060102-150405")
	tmpFile := fmt.Sprintf("/backup/etcd-snapshot-%s.db", timestamp)
	fmt.Printf("[ETCD-BACKUP] Using temp file: %s\n", tmpFile)

	fmt.Printf("[ETCD-BACKUP] Connecting to SSH host: %s\n", nodeIP)
	client, err := s.sshService.Connect(nodeIP, sshUsername, sshPassword)
	if err != nil {
		return fmt.Errorf("failed to create SSH client to %s: %w", nodeIP, err)
	}
	defer client.Close()
	fmt.Printf("[ETCD-BACKUP] SSH connection established\n")

	fmt.Printf("[ETCD-BACKUP] Creating /backup directory on remote node...\n")
	if _, err := client.ExecuteCommand("mkdir -p /backup"); err != nil {
		return fmt.Errorf("failed to create /backup directory on %s: %w", nodeIP, err)
	}

	singleEndpoint := fmt.Sprintf("https://%s:2379", nodeIP)
	cmd := fmt.Sprintf("%s --endpoints=%s --cacert=%s --cert=%s --key=%s snapshot save %s",
		schedule.EtcdctlPath, singleEndpoint, schedule.EtcdCaCert, schedule.EtcdCert, schedule.EtcdKey, tmpFile)
	fmt.Printf("[ETCD-BACKUP] Executing command: %s\n", cmd)

	if _, err := client.ExecuteCommand(cmd); err != nil {
		return fmt.Errorf("failed to execute etcdctl on %s: %w", nodeIP, err)
	}
	fmt.Printf("[ETCD-BACKUP] etcdctl command executed successfully\n")

	fmt.Printf("[ETCD-BACKUP] Downloading snapshot file...\n")
	if err := client.DownloadFile(tmpFile, snapshotPath); err != nil {
		return fmt.Errorf("failed to download snapshot from %s: %w", nodeIP, err)
	}
	fmt.Printf("[ETCD-BACKUP] Snapshot downloaded to: %s\n", snapshotPath)

	cleanupCmd := fmt.Sprintf("rm -f %s", tmpFile)
	client.ExecuteCommand(cleanupCmd)
	fmt.Printf("[ETCD-BACKUP] Temp file cleaned up\n")

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

// backupEtcdFromNode 从指定节点备份etcd
// CreateEtcdBackup 创建etcd备份记录
func (s *BackupService) CreateEtcdBackup(clusterID, backupName, backupType string, retentionDays int) (*model.ClusterBackup, error) {
	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster ID: %w", err)
	}

	backup := &model.ClusterBackup{
		ID:                uuid.New(),
		ClusterID:         clusterUUID,
		Name:              backupName,
		BackupName:        backupName,
		BackupType:        "etcd", // etcd备份固定为"etcd"类型
		Status:            "pending",
		RetentionDays:     retentionDays,
		SnapshotTimestamp: func() *time.Time { now := time.Now(); return &now }(),
		CreatedBy:         "system",
		StorageLocation:   fmt.Sprintf("\\backups\\%s\\%s\\%s", clusterID, backupName, time.Now().Format("20060102-150405")),
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
	backup.Status = constants.StatusRunning
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
	backup.Status = constants.StatusCompleted
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
		ID:                uuid.New(),
		ClusterID:         clusterUUID,
		Name:              backupName,
		BackupName:        backupName,
		BackupType:        "resources", // 资源备份固定为"resources"类型
		Status:            "pending",
		RetentionDays:     retentionDays,
		SnapshotTimestamp: func() *time.Time { now := time.Now(); return &now }(),
		CreatedBy:         "system",
		StorageLocation:   fmt.Sprintf("\\backups\\%s\\%s\\%s", clusterID, backupName, time.Now().Format("20060102-150405")),
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
	backup.Status = constants.StatusRunning
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
	backup.Status = constants.StatusCompleted
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
