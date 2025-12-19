package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

// CreateClusterService 集群创建服务
type CreateClusterService struct {
	taskRepo       *repository.CreateTaskRepository
	machineService *MachineService
	configGen      *ConfigGenerator
}

// NewCreateClusterService 创建集群创建服务
func NewCreateClusterService(
	taskRepo *repository.CreateTaskRepository,
	machineService *MachineService,
	configGen *ConfigGenerator,
) *CreateClusterService {
	return &CreateClusterService{
		taskRepo:       taskRepo,
		machineService: machineService,
		configGen:      configGen,
	}
}

// CreateCluster 创建集群
func (s *CreateClusterService) CreateCluster(req CreateClusterRequest) (*model.CreateTask, error) {
	// 验证机器配置
	if err := s.machineService.ValidateClusterNodes(req.MachineIDs); err != nil {
		return nil, err
	}

	// 获取机器信息
	machines, err := s.machineService.GetMachinesByIDs(req.MachineIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get machines: %w", err)
	}

	// 生成配置文件
	configYaml, err := s.configGen.GenerateConfig(req, machines)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	// 创建任务记录
	task := &model.CreateTask{
		ClusterName:       req.ClusterName,
		MachineIDs:        convertUUIDsToJSONMap(req.MachineIDs),
		ConfigYaml:        configYaml,
		Status:            "pending",
		Progress:          0,
		CurrentStep:       "Preparing configuration",
		ArtifactPath:      req.ArtifactPath,
		WithPackages:      req.WithPackages,
		AutoApprove:       req.AutoApprove,
		KubernetesVersion: req.Kubernetes.Version,
		NetworkPlugin:     req.Network.Plugin,
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 异步执行创建任务
	go s.executeCreateTask(task.ID)

	return task, nil
}

// GetTask 获取创建任务
func (s *CreateClusterService) GetTask(id uuid.UUID) (*model.CreateTask, error) {
	return s.taskRepo.GetByID(id)
}

// ListTasks 获取任务列表
func (s *CreateClusterService) ListTasks(page, limit int) ([]*model.CreateTask, int64, error) {
	return s.taskRepo.List(page, limit)
}

// executeCreateTask 执行创建任务（异步）
func (s *CreateClusterService) executeCreateTask(taskID uuid.UUID) {
	// 获取任务
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return
	}

	// 更新状态为运行中
	s.taskRepo.UpdateStatus(taskID, "running")
	s.taskRepo.UpdateProgress(taskID, 0, "Starting cluster creation")

	// 创建临时配置文件
	configFile, err := s.createTempConfigFile(task.ConfigYaml)
	if err != nil {
		s.taskRepo.UpdateStatus(taskID, "failed")
		s.taskRepo.UpdateProgress(taskID, 0, fmt.Sprintf("Failed to create config file: %v", err))
		return
	}
	defer os.Remove(configFile)

	// 准备 kk 命令
	cmdArgs := []string{"create", "cluster", "-f", configFile}
	if task.ArtifactPath != "" {
		cmdArgs = append(cmdArgs, "-a", task.ArtifactPath)
	}
	if task.WithPackages {
		cmdArgs = append(cmdArgs, "--with-packages")
	}
	if task.AutoApprove {
		cmdArgs = append(cmdArgs, "--yes")
	}

	// 启动命令
	cmd := exec.Command("kk", cmdArgs...)
	cmd.Dir = "." // 设置工作目录

	// 设置环境变量
	cmd.Env = append(os.Environ(),
		"KK_ZONE=cn", // 使用中国区域
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.taskRepo.UpdateStatus(taskID, "failed")
		s.taskRepo.UpdateProgress(taskID, 0, fmt.Sprintf("Failed to create stdout pipe: %v", err))
		return
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.taskRepo.UpdateStatus(taskID, "failed")
		s.taskRepo.UpdateProgress(taskID, 0, fmt.Sprintf("Failed to create stderr pipe: %v", err))
		return
	}
	defer stderr.Close()

	// 更新开始时间
	now := time.Now()
	task.StartedAt = &now
	s.taskRepo.Update(task)

	// 启动命令
	if err := cmd.Start(); err != nil {
		s.taskRepo.UpdateStatus(taskID, "failed")
		s.taskRepo.UpdateProgress(taskID, 0, fmt.Sprintf("Failed to start command: %v", err))
		return
	}

	// 实时读取日志
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理 stdout
	go s.readOutput(ctx, taskID, stdout, false)
	// 处理 stderr
	go s.readOutput(ctx, taskID, stderr, true)

	// 等待命令完成
	err = cmd.Wait()

	// 更新完成时间
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err != nil {
		s.taskRepo.UpdateStatus(taskID, "failed")
		s.taskRepo.UpdateProgress(taskID, task.Progress, fmt.Sprintf("Cluster creation failed: %v", err))
	} else {
		s.taskRepo.UpdateStatus(taskID, "success")
		s.taskRepo.UpdateProgress(taskID, 100, "Cluster creation completed successfully")
	}

	s.taskRepo.Update(task)
}

// readOutput 读取命令输出
func (s *CreateClusterService) readOutput(ctx context.Context, taskID uuid.UUID, reader io.ReadCloser, isError bool) {
	scanner := bufio.NewScanner(reader)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !scanner.Scan() {
				return
			}

			line := scanner.Text()
			timestamp := time.Now().Format("2006-01-02 15:04:05")

			// 格式化日志行
			logLine := fmt.Sprintf("[%s] %s\n", timestamp, line)
			if isError {
				logLine = fmt.Sprintf("[%s] ERROR: %s\n", timestamp, line)
			}

			// 追加日志
			s.taskRepo.AppendLogs(taskID, logLine)

			// 解析进度
			s.updateProgressFromLog(taskID, line)
		}
	}
}

// updateProgressFromLog 从日志中更新进度
func (s *CreateClusterService) updateProgressFromLog(taskID uuid.UUID, logLine string) {
	// 解析不同阶段的进度
	var progress int
	var currentStep string

	switch {
	case strings.Contains(logLine, "Checking an existing installation"):
		progress = 5
		currentStep = "Checking existing installation"
	case strings.Contains(logLine, "Preparing for installation"):
		progress = 10
		currentStep = "Preparing for installation"
	case strings.Contains(logLine, "Downloading"):
		progress = 20
		currentStep = "Downloading required packages"
	case strings.Contains(logLine, "downloading kubeadm"):
		progress = 30
		currentStep = "Downloading kubeadm binary"
	case strings.Contains(logLine, "pulling images"):
		progress = 40
		currentStep = "Pulling container images"
	case strings.Contains(logLine, "Installing"):
		progress = 60
		currentStep = "Installing Kubernetes components"
	case strings.Contains(logLine, "Configuring"):
		progress = 80
		currentStep = "Configuring cluster"
	case strings.Contains(logLine, "Installing kubesphere"):
		progress = 90
		currentStep = "Installing KubeSphere"
	case strings.Contains(logLine, "successfully installed") || strings.Contains(logLine, "KubeSphere is successfully installed"):
		progress = 100
		currentStep = "Installation completed"
	}

	if progress > 0 {
		s.taskRepo.UpdateProgress(taskID, progress, currentStep)
	}
}

// createTempConfigFile 创建临时配置文件
func (s *CreateClusterService) createTempConfigFile(configYaml string) (string, error) {
	tmpFile, err := os.CreateTemp("", "kubekey-config-*.yaml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(configYaml); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// convertUUIDsToJSONMap 转换UUID列表为JSON
func convertUUIDsToJSONMap(ids []uuid.UUID) map[string]interface{} {
	result := make(map[string]interface{}, len(ids))
	for i, id := range ids {
		result[fmt.Sprintf("%d", i)] = id.String()
	}
	return result
}
