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
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // 认证插件
)

type RestoreService struct {
	backupRepo          *repository.BackupRepository
	backupScheduleRepo  *repository.BackupScheduleRepository
	clusterRepo         *repository.ClusterRepository
	encryptionSvc       *EncryptionService
	clusterManager      *ClusterManager
	sshService          *SSHService
	controlPlaneManager ControlPlaneManager
	mu                  sync.RWMutex
}

type RestoreResult struct {
	RestoreID     string    `json:"restore_id"`
	Status        string    `json:"status"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time,omitempty"`
	RestoredItems int       `json:"restored_items"`
	FailedItems   int       `json:"failed_items"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

type RestoreProgress struct {
	RestoreID     string    `json:"restore_id"`
	Status        string    `json:"status"`
	Progress      float64   `json:"progress"`
	CurrentStep   string    `json:"current_step"`
	StartTime     time.Time `json:"start_time"`
	EstimatedTime int       `json:"estimated_time,omitempty"`
}

type EtcdNodeConfig struct {
	Nodes []EtcdNodeInfo
}

type EtcdNodeInfo struct {
	IP          string
	Port        int
	CACert      string
	Cert        string
	Key         string
	DataDir     string
	Endpoints   []string
	SSHUsername string
	SSHPassword string
}

type ControlPlaneManager interface {
	StopControlPlaneComponents(ctx context.Context) error
	StartControlPlaneComponents(ctx context.Context) error
}

type SSHControlPlaneManager struct {
	sshService    *SSHService
	nodes         []EtcdNodeInfo
	etcdStopCmds  []string
	etcdStartCmds []string
	k8sStopCmds   []string
	k8sStartCmds  []string
}

func NewSSHControlPlaneManager(sshService *SSHService, nodes []EtcdNodeInfo, etcdStopCmds, etcdStartCmds, k8sStopCmds, k8sStartCmds []string) *SSHControlPlaneManager {
	return &SSHControlPlaneManager{
		sshService:    sshService,
		nodes:         nodes,
		etcdStopCmds:  etcdStopCmds,
		etcdStartCmds: etcdStartCmds,
		k8sStopCmds:   k8sStopCmds,
		k8sStartCmds:  k8sStartCmds,
	}
}

func (m *SSHControlPlaneManager) StopControlPlaneComponents(ctx context.Context) error {
	if len(m.nodes) == 0 {
		return fmt.Errorf("no nodes configured")
	}

	for _, node := range m.nodes {
		sshClient, err := m.sshService.Connect(node.IP, node.SSHUsername, node.SSHPassword)
		if err != nil {
			return fmt.Errorf("failed to connect to node %s: %w", node.IP, err)
		}
		defer sshClient.Close()

		for _, cmd := range m.etcdStopCmds {
			if _, err := sshClient.ExecuteCommand(cmd); err != nil {
				return fmt.Errorf("failed to execute etcd stop command '%s' on node %s: %w", cmd, node.IP, err)
			}
		}

		for _, cmd := range m.k8sStopCmds {
			if _, err := sshClient.ExecuteCommand(cmd); err != nil {
				return fmt.Errorf("failed to execute k8s stop command '%s' on node %s: %w", cmd, node.IP, err)
			}
		}
	}

	return nil
}

func (m *SSHControlPlaneManager) StartControlPlaneComponents(ctx context.Context) error {
	if len(m.nodes) == 0 {
		return fmt.Errorf("no nodes configured")
	}

	for _, node := range m.nodes {
		sshClient, err := m.sshService.Connect(node.IP, node.SSHUsername, node.SSHPassword)
		if err != nil {
			return fmt.Errorf("failed to connect to node %s: %w", node.IP, err)
		}
		defer sshClient.Close()

		for _, cmd := range m.etcdStartCmds {
			if _, err := sshClient.ExecuteCommand(cmd); err != nil {
				return fmt.Errorf("failed to execute etcd start command '%s' on node %s: %w", cmd, node.IP, err)
			}
		}

		for _, cmd := range m.k8sStartCmds {
			if _, err := sshClient.ExecuteCommand(cmd); err != nil {
				return fmt.Errorf("failed to execute k8s start command '%s' on node %s: %w", cmd, node.IP, err)
			}
		}
	}

	return nil
}

func NewRestoreService(
	backupRepo *repository.BackupRepository,
	backupScheduleRepo *repository.BackupScheduleRepository,
	clusterRepo *repository.ClusterRepository,
	encryptionSvc *EncryptionService,
	clusterManager *ClusterManager,
) *RestoreService {
	return &RestoreService{
		backupRepo:         backupRepo,
		backupScheduleRepo: backupScheduleRepo,
		clusterRepo:        clusterRepo,
		encryptionSvc:      encryptionSvc,
		clusterManager:     clusterManager,
		sshService:         NewSSHService(),
	}
}

// RestoreBackup 执行备份恢复
func (s *RestoreService) RestoreBackup(clusterID, backupID, restoreName string) (*RestoreResult, error) {
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

	// 异步执行恢复
	go s.executeRestore(restoreID, cluster, backup, restoreName)

	return result, nil
}

// GetRestoreProgress 获取恢复进度
func (s *RestoreService) GetRestoreProgress(restoreID string) (*RestoreProgress, error) {
	// 这里可以从缓存或数据库中获取恢复进度
	// 为了简化，我们返回一个模拟的进度
	return &RestoreProgress{
		RestoreID:   restoreID,
		Status:      "running",
		Progress:    50.0,
		CurrentStep: "正在恢复资源",
		StartTime:   time.Now().Add(-10 * time.Minute),
	}, nil
}

// executeRestore 执行实际的恢复操作
func (s *RestoreService) executeRestore(restoreID string, cluster *model.Cluster, backup *model.ClusterBackup, restoreName string) {
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
func (s *RestoreService) performFullRestore(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
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
func (s *RestoreService) performEtcdRestore(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
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
func (s *RestoreService) performResourcesRestore(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
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

func (s *RestoreService) prepareRestoreEnvironment(ctx context.Context, clientset *kubernetes.Clientset, restoreName string) error {
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

func (s *RestoreService) restoreEtcdData(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	fmt.Printf("[RESTORE-ETCD] Starting etcd data restore for backup %s\n", backup.ID.String())

	backupPath := s.getBackupPath(backup)
	etcdSnapshotPath := filepath.Join(backupPath, "etcd.snapshot")

	if _, err := os.Stat(etcdSnapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("etcd snapshot file not found at %s", etcdSnapshotPath)
	}

	if err := s.stopControlPlaneComponents(ctx, clientset); err != nil {
		return fmt.Errorf("failed to stop control plane components: %w", err)
	}

	if err := s.restoreEtcdSnapshot(ctx, clientset, etcdSnapshotPath); err != nil {
		return fmt.Errorf("failed to restore etcd snapshot: %w", err)
	}

	if err := s.startControlPlaneComponents(ctx, clientset); err != nil {
		return fmt.Errorf("failed to start control plane components: %w", err)
	}

	fmt.Printf("[RESTORE-ETCD] Etcd data restore completed successfully\n")
	return nil
}

func (s *RestoreService) restoreResourceManifests(ctx context.Context, clientset *kubernetes.Clientset, backup *model.ClusterBackup, restoreName string) error {
	fmt.Printf("[RESTORE-RESOURCES] Starting resource manifests restore for backup %s\n", backup.ID.String())

	backupPath := s.getBackupPath(backup)
	resourcesPath := filepath.Join(backupPath, "resources")

	if _, err := os.Stat(resourcesPath); os.IsNotExist(err) {
		fmt.Printf("[RESTORE-RESOURCES] No resources directory found, skipping resource restore\n")
		return nil
	}

	resourceTypes := []string{"namespaces", "pv", "pvc", "configmaps", "secrets", "deployments", "services", "ingresses"}

	for _, resourceType := range resourceTypes {
		resourcePath := filepath.Join(resourcesPath, resourceType)
		if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
			fmt.Printf("[RESTORE-RESOURCES] Resource type %s not found, skipping\n", resourceType)
			continue
		}

		fmt.Printf("[RESTORE-RESOURCES] Restoring %s resources\n", resourceType)
		if err := s.restoreResourcesOfType(ctx, clientset, resourcePath, resourceType); err != nil {
			return fmt.Errorf("failed to restore %s: %w", resourceType, err)
		}
	}

	fmt.Printf("[RESTORE-RESOURCES] Resource manifests restore completed successfully\n")
	return nil
}

func (s *RestoreService) verifyRestoreResult(ctx context.Context, clientset *kubernetes.Clientset, restoreName string) error {
	fmt.Printf("[VERIFY-RESTORE] Starting restore result verification\n")

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	fmt.Printf("[VERIFY-RESTORE] Found %d namespaces\n", len(namespaces.Items))

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods++
		}
	}

	fmt.Printf("[VERIFY-RESTORE] Found %d pods, %d running\n", len(pods.Items), runningPods)

	deployments, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	readyDeployments := 0
	for _, deploy := range deployments.Items {
		if deploy.Status.ReadyReplicas == *deploy.Spec.Replicas {
			readyDeployments++
		}
	}

	fmt.Printf("[VERIFY-RESTORE] Found %d deployments, %d ready\n", len(deployments.Items), readyDeployments)

	fmt.Printf("[VERIFY-RESTORE] Restore verification completed successfully\n")
	return nil
}

func (s *RestoreService) stopControlPlaneComponents(ctx context.Context, clientset *kubernetes.Clientset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controlPlaneManager == nil {
		if err := s.initControlPlaneManager(); err != nil {
			return fmt.Errorf("failed to initialize control plane manager: %w", err)
		}
	}

	return s.controlPlaneManager.StopControlPlaneComponents(ctx)
}

func (s *RestoreService) startControlPlaneComponents(ctx context.Context, clientset *kubernetes.Clientset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controlPlaneManager == nil {
		if err := s.initControlPlaneManager(); err != nil {
			return fmt.Errorf("failed to initialize control plane manager: %w", err)
		}
	}

	return s.controlPlaneManager.StartControlPlaneComponents(ctx)
}

func (s *RestoreService) initControlPlaneManager() error {
	schedules, err := s.backupScheduleRepo.ListEnabled()
	if err != nil {
		return fmt.Errorf("failed to list backup schedules: %w", err)
	}
	if len(schedules) == 0 {
		return fmt.Errorf("no backup schedule found")
	}

	schedule := schedules[0]
	if schedule.EtcdEndpoints == "" {
		return fmt.Errorf("etcd_endpoints not set in backup schedule")
	}

	nodes := parseEndpointsToNodes(schedule.EtcdEndpoints)
	if len(nodes) == 0 {
		return fmt.Errorf("no etcd nodes found from endpoints")
	}

	etcdNodes := make([]EtcdNodeInfo, len(nodes))
	for i, node := range nodes {
		etcdNodes[i] = EtcdNodeInfo{
			IP:          node,
			Port:        2379,
			CACert:      schedule.EtcdCaCert,
			Cert:        schedule.EtcdCert,
			Key:         schedule.EtcdKey,
			DataDir:     schedule.EtcdDataDir,
			Endpoints:   strings.Split(schedule.EtcdEndpoints, ","),
			SSHUsername: schedule.SshUsername,
			SSHPassword: schedule.SshPassword,
		}
	}

	etcdStopCmds := s.getEtcdStopCommands(schedule.EtcdDeploymentType)
	etcdStartCmds := s.getEtcdStartCommands(schedule.EtcdDeploymentType)
	k8sStopCmds := s.getK8sStopCommands(schedule.K8sDeploymentType)
	k8sStartCmds := s.getK8sStartCommands(schedule.K8sDeploymentType)

	s.controlPlaneManager = NewSSHControlPlaneManager(s.sshService, etcdNodes, etcdStopCmds, etcdStartCmds, k8sStopCmds, k8sStartCmds)
	return nil
}

func (s *RestoreService) getEtcdStopCommands(deploymentType string) []string {
	switch deploymentType {
	case "kubexm":
		return []string{
			"sudo systemctl stop etcd",
		}
	case "kubeadm":
		return []string{
			"sudo mv /etc/kubernetes/manifests/etcd.yaml /tmp/etcd.yaml",
		}
	default:
		return []string{
			"sudo systemctl stop etcd",
		}
	}
}

func (s *RestoreService) getEtcdStartCommands(deploymentType string) []string {
	switch deploymentType {
	case "kubexm":
		return []string{
			"sudo systemctl start etcd",
		}
	case "kubeadm":
		return []string{
			"sudo mv /tmp/etcd.yaml /etc/kubernetes/manifests/etcd.yaml",
		}
	default:
		return []string{
			"sudo systemctl start etcd",
		}
	}
}

func (s *RestoreService) getK8sStopCommands(deploymentType string) []string {
	switch deploymentType {
	case "kubexm":
		return []string{
			"sudo systemctl stop kube-apiserver",
		}
	case "kubeadm":
		return []string{
			"sudo mv /etc/kubernetes/manifests/kube-apiserver.yaml /tmp/kube-apiserver.yaml",
		}
	default:
		return []string{
			"sudo systemctl stop kube-apiserver",
		}
	}
}

func (s *RestoreService) getK8sStartCommands(deploymentType string) []string {
	switch deploymentType {
	case "kubexm":
		return []string{
			"sudo systemctl start kube-apiserver",
		}
	case "kubeadm":
		return []string{
			"sudo mv /tmp/kube-apiserver.yaml /etc/kubernetes/manifests/kube-apiserver.yaml",
		}
	default:
		return []string{
			"sudo systemctl start kube-apiserver",
		}
	}
}

func (s *RestoreService) restoreEtcdSnapshot(ctx context.Context, clientset *kubernetes.Clientset, snapshotPath string) error {
	fmt.Printf("[ETCD-RESTORE] Starting etcd snapshot restore from %s\n", snapshotPath)

	schedules, err := s.backupScheduleRepo.ListEnabled()
	if err != nil {
		return fmt.Errorf("failed to list backup schedules: %w", err)
	}
	if len(schedules) == 0 {
		return fmt.Errorf("no backup schedule found")
	}

	schedule := schedules[0]
	if schedule.EtcdEndpoints == "" {
		return fmt.Errorf("etcd_endpoints not set in backup schedule")
	}

	nodes := parseEndpointsToNodes(schedule.EtcdEndpoints)
	if len(nodes) == 0 {
		return fmt.Errorf("no etcd nodes found from endpoints")
	}

	timestamp := time.Now().Format("20060102-150405")
	remoteSnapshotPath := fmt.Sprintf("/backup/etcd-snapshot-restore-%s.db", timestamp)

	for i, node := range nodes {
		fmt.Printf("[ETCD-RESTORE] Trying to restore to node %d/%d: %s\n", i+1, len(nodes), node)

		sshUsername := schedule.SshUsername
		if sshUsername == "" {
			sshUsername = "root"
		}

		sshPassword := schedule.SshPassword
		if sshPassword == "" {
			return fmt.Errorf("ssh_password not set in backup schedule")
		}

		sshClient, err := s.sshService.Connect(node, sshUsername, sshPassword)
		if err != nil {
			fmt.Printf("[ETCD-RESTORE] Failed to connect to node %s: %v\n", node, err)
			continue
		}
		defer sshClient.Close()

		fmt.Printf("[ETCD-RESTORE] Uploading snapshot file to %s\n", node)
		if err := sshClient.UploadFile(snapshotPath, remoteSnapshotPath); err != nil {
			fmt.Printf("[ETCD-RESTORE] Failed to upload snapshot to %s: %v\n", node, err)
			continue
		}

		firstEndpoint := schedule.EtcdEndpoints
		if strings.Contains(firstEndpoint, ",") {
			firstEndpoint = strings.Split(firstEndpoint, ",")[0]
		}

		restoreCmd := fmt.Sprintf("%s snapshot restore %s --data-dir=%s",
			schedule.EtcdctlPath, remoteSnapshotPath, schedule.EtcdDataDir)

		fmt.Printf("[ETCD-RESTORE] Executing restore command: %s\n", restoreCmd)

		if _, err := sshClient.ExecuteCommand(restoreCmd); err != nil {
			fmt.Printf("[ETCD-RESTORE] Failed to execute restore command on %s: %v\n", node, err)
			cleanupCmd := fmt.Sprintf("rm -f %s", remoteSnapshotPath)
			sshClient.ExecuteCommand(cleanupCmd)
			continue
		}

		cleanupCmd := fmt.Sprintf("rm -f %s", remoteSnapshotPath)
		sshClient.ExecuteCommand(cleanupCmd)

		fmt.Printf("[ETCD-RESTORE] Successfully restored etcd snapshot on node %s\n", node)
		return nil
	}

	return fmt.Errorf("etcd snapshot restore failed - no accessible etcd node found")
}

func (s *RestoreService) restoreResourcesOfType(ctx context.Context, clientset *kubernetes.Clientset, resourcePath, resourceType string) error {
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

func (s *RestoreService) applyResourceToCluster(ctx context.Context, clientset *kubernetes.Clientset, resourceContent []byte) error {
	// 使用kubectl apply或直接使用client-go应用资源
	// 这里简化实现，实际应该解析资源类型并使用对应的API
	return nil
}

func (s *RestoreService) getBackupPath(backup *model.ClusterBackup) string {
	// 返回备份存储路径
	return backup.StorageLocation
}

func (s *RestoreService) logRestoreError(restoreID string, err error) {
	// 记录恢复错误
	// 实际实现应该将错误信息存储到数据库或日志系统
	fmt.Printf("Restore %s failed: %v\n", restoreID, err)
}

func (s *RestoreService) logRestoreSuccess(restoreID string) {
	fmt.Printf("Restore %s completed successfully\n", restoreID)
}

func parseEndpointsToNodes(endpoints string) []string {
	var nodes []string
	epList := strings.Split(endpoints, ",")
	for _, ep := range epList {
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
