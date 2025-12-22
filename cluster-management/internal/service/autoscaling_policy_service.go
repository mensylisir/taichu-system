package service

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type AutoscalingPolicyService struct {
	clusterManager         *ClusterManager
	encryptionSvc          *EncryptionService
	autoscalingPolicyRepo  *repository.AutoscalingPolicyRepository
	clusterRepo            *repository.ClusterRepository
}

type HPAPolicy struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	MinReplicas     int    `json:"min_replicas"`
	MaxReplicas     int    `json:"max_replicas"`
	CurrentReplicas int    `json:"current_replicas"`
}

type AutoscalingPolicyWithDetails struct {
	*model.AutoscalingPolicy
	HPAPolicies []HPAPolicy `json:"hpa_policies"`
}

func NewAutoscalingPolicyService(
	clusterManager *ClusterManager,
	encryptionSvc *EncryptionService,
	autoscalingPolicyRepo *repository.AutoscalingPolicyRepository,
	clusterRepo *repository.ClusterRepository,
) *AutoscalingPolicyService {
	return &AutoscalingPolicyService{
		clusterManager:         clusterManager,
		encryptionSvc:          encryptionSvc,
		autoscalingPolicyRepo:  autoscalingPolicyRepo,
		clusterRepo:            clusterRepo,
	}
}

func (s *AutoscalingPolicyService) GetAutoscalingPolicy(clusterID string) (*AutoscalingPolicyWithDetails, error) {
	policy, err := s.autoscalingPolicyRepo.GetByClusterID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get autoscaling policy: %w", err)
	}

	hpaPolicies := s.getHPAPolicies(clusterID)

	return &AutoscalingPolicyWithDetails{
		AutoscalingPolicy: policy,
		HPAPolicies:       hpaPolicies,
	}, nil
}

func (s *AutoscalingPolicyService) SyncAutoscalingPolicyFromKubernetes(clusterID string) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt kubeconfig: %w", err)
	}

	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	policy := &model.AutoscalingPolicy{
		ClusterID: cluster.ID,
	}

	// 检查HPA
	hpas, err := clientset.AutoscalingV1().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	if err == nil {
		policy.HPACount = len(hpas.Items)
	}

	// 检查Cluster Autoscaler
	// 这通常通过检查Deployment来确认
	policy.ClusterAutoscalerEnabled = s.checkClusterAutoscaler(ctx, clientset)

	// 注意：VPA (Vertical Pod Autoscaler) 需要通过 autoscaling.k8s.io/v1 API 访问
	// 这里暂时注释掉，因为不同Kubernetes版本的API可能不同
	// vpaList, err := clientset.AutoscalingV2().VerticalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	// if err == nil {
	// 	policy.VPACount = len(vpaList.Items)
	// }
	policy.VPACount = 0

	if err := s.autoscalingPolicyRepo.Upsert(policy); err != nil {
		return fmt.Errorf("failed to upsert autoscaling policy: %w", err)
	}

	return nil
}

func (s *AutoscalingPolicyService) getHPAPolicies(clusterID string) []HPAPolicy {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil
	}

	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		return nil
	}

	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return nil
	}

	hpas, err := clientset.AutoscalingV1().HorizontalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	policies := make([]HPAPolicy, 0, len(hpas.Items))
	for _, hpa := range hpas.Items {
		policies = append(policies, HPAPolicy{
			Name:            hpa.Name,
			Namespace:       hpa.Namespace,
			MinReplicas:     int(*hpa.Spec.MinReplicas),
			MaxReplicas:     int(hpa.Spec.MaxReplicas),
			CurrentReplicas: int(hpa.Status.CurrentReplicas),
		})
	}

	return policies
}

func (s *AutoscalingPolicyService) checkClusterAutoscaler(ctx context.Context, clientset *kubernetes.Clientset) bool {
	// 检查kube-system命名空间中的cluster-autoscaler deployment
	_, err := clientset.AppsV1().Deployments("kube-system").Get(ctx, "cluster-autoscaler", metav1.GetOptions{})
	return err == nil
}
