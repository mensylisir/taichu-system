package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"
)

// EtcdBackupConfig etcd备份配置
type EtcdBackupConfig struct {
	Endpoint        string
	CAFile          string
	CertFile        string
	KeyFile         string
	Timeout         time.Duration
	DataDir         string
}

// EtcdBackupService etcd备份服务
type EtcdBackupService struct {
	config *EtcdBackupConfig
}

func NewEtcdBackupService(config *EtcdBackupConfig) *EtcdBackupService {
	return &EtcdBackupService{config: config}
}

// CreateSnapshot 创建etcd快照
func (s *EtcdBackupService) CreateSnapshot(ctx context.Context, outputPath string) error {
	// 检查配置是否存在
	if s.config == nil {
		return fmt.Errorf("etcd backup not configured - please configure etcd endpoint, certificates, and timeout")
	}

	// 确保输出目录存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	args := []string{
		"snapshot", "save",
		"--endpoints", s.config.Endpoint,
		"--cacert", s.config.CAFile,
		"--cert", s.config.CertFile,
		"--key", s.config.KeyFile,
		"--timeout", s.config.Timeout.String(),
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "etcdctl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create etcd snapshot: %w, stderr: %s", err, stderr.String())
	}

	// 验证快照文件是否创建成功
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file was not created")
	}

	return nil
}

// RestoreSnapshot 恢复etcd快照
func (s *EtcdBackupService) RestoreSnapshot(ctx context.Context, snapshotPath, dataDir string) error {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	args := []string{
		"snapshot", "restore", snapshotPath,
		"--data-dir", dataDir,
		"--name", "restored",
	}

	cmd := exec.CommandContext(ctx, "etcdctl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore etcd snapshot: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// VerifySnapshot 验证快照完整性
func (s *EtcdBackupService) VerifySnapshot(ctx context.Context, snapshotPath string) error {
	args := []string{
		"snapshot", "status", snapshotPath,
		"--write-out=json",
	}

	cmd := exec.CommandContext(ctx, "etcdctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to verify snapshot: %w", err)
	}

	// 解析JSON输出
	var status struct {
		Hash      uint32 `json:"hash"`
		Revision  int64  `json:"revision"`
		TotalKey  int    `json:"totalKey"`
		TotalSize int64  `json:"totalSize"`
	}

	if err := json.Unmarshal(output, &status); err != nil {
		return fmt.Errorf("failed to parse snapshot status: %w", err)
	}

	// 验证快照状态
	if status.TotalSize == 0 {
		return fmt.Errorf("snapshot is empty")
	}

	if status.TotalKey < 0 {
		return fmt.Errorf("invalid key count in snapshot")
	}

	return nil
}

// GetEtcdConfigFromKubeconfig 从kubeconfig获取etcd配置
func (s *EtcdBackupService) GetEtcdConfigFromKubeconfig(ctx context.Context, kubeconfig string) (*EtcdBackupConfig, error) {
	// 解析kubeconfig
	config, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// 获取当前上下文
	contextName := config.CurrentContext
	if contextName == "" {
		return nil, fmt.Errorf("no current context in kubeconfig")
	}

	contextConfig, ok := config.Contexts[contextName]
	if !ok {
		return nil, fmt.Errorf("context %s not found", contextName)
	}

	clusterConfig, ok := config.Clusters[contextConfig.Cluster]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", contextConfig.Cluster)
	}

	// 构建etcd配置
	// 注意：这里需要根据实际集群配置调整
	etcdConfig := &EtcdBackupConfig{
		Endpoint: "https://127.0.0.1:2379", // 默认etcd端点
		CAFile:   "/etc/kubernetes/pki/etcd/ca.crt",
		CertFile: "/etc/kubernetes/pki/etcd/server.crt",
		KeyFile:  "/etc/kubernetes/pki/etcd/server.key",
		DataDir:  "/var/lib/etcd",
		Timeout:  30 * time.Second,
	}

	// 如果kubeconfig中有server地址，优先使用
	if clusterConfig.Server != "" {
		// 提取etcd端点（假设使用默认端口2379）
		// 实际中可能需要从集群配置中获取
		etcdConfig.Endpoint = clusterConfig.Server
		if !strings.Contains(etcdConfig.Endpoint, ":") {
			etcdConfig.Endpoint = etcdConfig.Endpoint + ":2379"
		}
	}

	// TODO: 实际实现中需要从集群中获取证书文件路径
	// 这里使用默认路径，实际中可能需要通过SSH或其他方式获取

	return etcdConfig, nil
}

// StopControlPlaneComponents 停止控制平面组件
func (s *EtcdBackupService) StopControlPlaneComponents(ctx context.Context) error {
	// 通过systemd停止控制平面组件
	components := []string{
		"kube-apiserver",
		"kube-controller-manager",
		"kube-scheduler",
	}

	for _, component := range components {
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "stop", component)
		if err := cmd.Run(); err != nil {
			// 记录错误但不返回，因为组件可能已经停止
			fmt.Printf("Warning: failed to stop %s: %v\n", component, err)
		}
	}

	// 等待组件完全停止
	time.Sleep(5 * time.Second)

	return nil
}

// StartControlPlaneComponents 启动控制平面组件
func (s *EtcdBackupService) StartControlPlaneComponents(ctx context.Context) error {
	// 按顺序启动控制平面组件
	components := []string{
		"etcd",
		"kube-apiserver",
		"kube-controller-manager",
		"kube-scheduler",
	}

	for _, component := range components {
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "start", component)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start %s: %w", component, err)
		}

		// 等待组件启动
		time.Sleep(2 * time.Second)
	}

	return nil
}

// CheckControlPlaneStatus 检查控制平面组件状态
func (s *EtcdBackupService) CheckControlPlaneStatus(ctx context.Context) error {
	components := []string{
		"etcd",
		"kube-apiserver",
		"kube-controller-manager",
		"kube-scheduler",
	}

	for _, component := range components {
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "is-active", component)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s is not running", component)
		}
	}

	return nil
}

// WaitForEtcdReady 等待etcd就绪
func (s *EtcdBackupService) WaitForEtcdReady(ctx context.Context, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for etcd to be ready")
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "etcdctl", "endpoint", "health")
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}
}
