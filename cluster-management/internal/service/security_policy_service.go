package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

// SecurityPolicyService 安全策略服务
type SecurityPolicyService struct {
	securityPolicyRepo *repository.SecurityPolicyRepository
	clusterRepo        *repository.ClusterRepository
	clusterManager     *ClusterManager
	encryptionService  *EncryptionService
	kubernetesService  KubernetesService
}

// NewSecurityPolicyService 创建新的安全策略服务
func NewSecurityPolicyService(
	securityPolicyRepo *repository.SecurityPolicyRepository,
	clusterRepo *repository.ClusterRepository,
	clusterManager *ClusterManager,
	encryptionService *EncryptionService,
	kubernetesService KubernetesService,
) *SecurityPolicyService {
	return &SecurityPolicyService{
		securityPolicyRepo: securityPolicyRepo,
		clusterRepo:        clusterRepo,
		clusterManager:     clusterManager,
		encryptionService:  encryptionService,
		kubernetesService:  kubernetesService,
	}
}

// GetSecurityPolicy 获取集群的安全策略信息
func (s *SecurityPolicyService) GetSecurityPolicy(ctx context.Context, clusterID string) (*model.SecurityPolicy, error) {
	return s.securityPolicyRepo.GetByClusterID(ctx, clusterID)
}

// SyncSecurityPolicyFromKubernetes 从Kubernetes集群同步安全策略信息
func (s *SecurityPolicyService) SyncSecurityPolicyFromKubernetes(ctx context.Context, clusterID string) error {
	// 获取集群kubeconfig
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 解密kubeconfig
	kubeconfig, err := s.encryptionService.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt kubeconfig: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, err := s.kubernetesService.GetClientset(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 创建安全策略检测器
	detector := NewSecurityPolicyDetector(clientset)

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
		ClusterID:              uuid.MustParse(clusterID),
		PodSecurityStandard:    podSecurityStatus.Standard,
		NetworkPoliciesEnabled: networkPolicyStatus.Enabled,
		RBACEnabled:            rbacStatus.Enabled,
		AuditLoggingEnabled:    auditLoggingStatus.Enabled,
		// 添加新字段
		RBACDetails:              rbacStatus.Details,
		NetworkPolicyDetails:     networkPolicyStatus.Details,
		PodSecurityDetails:       podSecurityStatus.Details,
		AuditLoggingDetails:      auditLoggingStatus.Details,
		NetworkPoliciesCount:     networkPolicyStatus.PoliciesCount,
		RBACRolesCount:           rbacStatus.RolesCount,
		CNIPlugin:                networkPolicyStatus.CNIPlugin,
		PodSecurityAdmissionMode: podSecurityStatus.AdmissionMode,
	}

	// 保存到数据库
	if err := s.securityPolicyRepo.Upsert(ctx, securityPolicy); err != nil {
		return fmt.Errorf("failed to save security policy: %w", err)
	}

	return nil
}
