package service

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type SecurityPolicyService struct {
	clusterManager      *ClusterManager
	encryptionSvc       *EncryptionService
	securityPolicyRepo  *repository.SecurityPolicyRepository
	clusterRepo         *repository.ClusterRepository
}

func NewSecurityPolicyService(
	clusterManager *ClusterManager,
	encryptionSvc *EncryptionService,
	securityPolicyRepo *repository.SecurityPolicyRepository,
	clusterRepo *repository.ClusterRepository,
) *SecurityPolicyService {
	return &SecurityPolicyService{
		clusterManager:      clusterManager,
		encryptionSvc:       encryptionSvc,
		securityPolicyRepo:  securityPolicyRepo,
		clusterRepo:         clusterRepo,
	}
}

func (s *SecurityPolicyService) GetSecurityPolicy(clusterID string) (*model.SecurityPolicy, error) {
	policy, err := s.securityPolicyRepo.GetByClusterID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get security policy: %w", err)
	}
	return policy, nil
}

func (s *SecurityPolicyService) SyncSecurityPolicyFromKubernetes(clusterID string) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	kubeconfig, err := s.encryptionSvc.Decrypt(
		cluster.KubeconfigEncrypted,
		cluster.KubeconfigNonce,
	)
	if err != nil {
		return fmt.Errorf("failed to decrypt kubeconfig: %w", err)
	}

	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	policy := &model.SecurityPolicy{
		ClusterID: cluster.ID,
	}

	// 检查Pod Security Policy
	// 注意：PodSecurityPolicy在Kubernetes 1.25+中已被废弃
	// 这里我们检查Pod Security Standard通过检查命名空间标签
	policy.PodSecurityStandard = s.getPodSecurityStandard(ctx, clientset)

	// 检查Network Policies
	netpols, err := clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err == nil {
		policy.NetworkPoliciesEnabled = len(netpols.Items) > 0
		policy.NetworkPoliciesCount = len(netpols.Items)
	}

	// 检查RBAC
	roles, err := clientset.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
	if err == nil {
		policy.RBACEnabled = len(roles.Items) > 0
		policy.RBACRolesCount = len(roles.Items)
	}

	// 检查审计日志
	// 这需要访问kube-apiserver配置，这里简化处理
	policy.AuditLoggingEnabled = false
	policy.AuditLoggingMode = "none"

	if err := s.securityPolicyRepo.Upsert(policy); err != nil {
		return fmt.Errorf("failed to upsert security policy: %w", err)
	}

	return nil
}

func (s *SecurityPolicyService) getPodSecurityStandard(ctx context.Context, clientset *kubernetes.Clientset) string {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "baseline"
	}

	// 检查是否有PSS标签
	for _, ns := range namespaces.Items {
		if pss, ok := ns.Labels["pod-security.kubernetes.io/enforce"]; ok {
			return pss
		}
	}

	return "baseline"
}
