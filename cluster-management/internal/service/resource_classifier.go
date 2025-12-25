package service

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
)

// ResourceClassifier 资源分类器 - 自动识别与归类
type ResourceClassifier struct {
	kubernetesClient *kubernetes.Clientset
	tenantRepo       *repository.TenantRepository
	environmentRepo  *repository.EnvironmentRepository
	applicationRepo  *repository.ApplicationRepository
	quotaRepo        *repository.QuotaRepository
}

// NewResourceClassifier 创建资源分类器
func NewResourceClassifier(
	kubeClient *kubernetes.Clientset,
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
) *ResourceClassifier {
	return &ResourceClassifier{
		kubernetesClient: kubeClient,
		tenantRepo:       tenantRepo,
		environmentRepo:  environmentRepo,
		applicationRepo:  applicationRepo,
		quotaRepo:        quotaRepo,
	}
}

// NamespaceClassificationResult 命名空间分类结果
type NamespaceClassificationResult struct {
	Namespace      corev1.Namespace
	AssignedTenant *model.Tenant
	Environment    *model.Environment
	Applications   []*model.Application
	ResourceQuota  *model.ResourceQuota
}

// ClassifyAllNamespaces 分类所有命名空间
func (rc *ResourceClassifier) ClassifyAllNamespaces(clusterID string) ([]*NamespaceClassificationResult, error) {
	ctx := context.Background()

	// 获取所有命名空间
	namespaces, err := rc.kubernetesClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var results []*NamespaceClassificationResult

	for _, ns := range namespaces.Items {
		result, err := rc.ClassifyNamespace(clusterID, ns)
		if err != nil {
			// 记录错误但继续处理其他命名空间
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// ClassifyNamespace 分类单个命名空间
func (rc *ResourceClassifier) ClassifyNamespace(clusterID string, namespace corev1.Namespace) (*NamespaceClassificationResult, error) {
	ctx := context.Background()

	// 1. 识别租户
	tenant, err := rc.identifyTenant(namespace)
	if err != nil {
		return nil, err
	}

	// 2. 创建或获取环境
	env, err := rc.createOrGetEnvironment(clusterID, tenant, namespace)
	if err != nil {
		return nil, err
	}

	// 3. 识别应用
	apps, err := rc.discoverApplications(ctx, env, namespace)
	if err != nil {
		return nil, err
	}

	// 4. 获取资源配额
	quota, err := rc.getResourceQuota(ctx, namespace.Name)
	if err != nil {
		// 命名空间可能没有ResourceQuota，这是正常的
		quota = nil
	}

	return &NamespaceClassificationResult{
		Namespace:      namespace,
		AssignedTenant: tenant,
		Environment:    env,
		Applications:   apps,
		ResourceQuota:  quota,
	}, nil
}

// identifyTenant 识别租户
func (rc *ResourceClassifier) identifyTenant(namespace corev1.Namespace) (*model.Tenant, error) {
	// 检查是否为系统命名空间
	if rc.isSystemNamespace(namespace) {
		// 获取或创建 system 租户
		return rc.tenantRepo.GetByName(model.PredefinedTenants.System)
	}

	// 业务命名空间 -> default 租户
	return rc.tenantRepo.GetByName(model.PredefinedTenants.Default)
}

// isSystemNamespace 判断是否为系统命名空间
func (rc *ResourceClassifier) isSystemNamespace(namespace corev1.Namespace) bool {
	nsName := namespace.Name

	// 系统命名空间列表
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"istio-system",
		"ingress-nginx",
		"cert-manager",
		"monitoring",
		"gatekeeper-system",
	}

	for _, sysNS := range systemNamespaces {
		if nsName == sysNS {
			return true
		}
	}

	// 检查是否以 kube- 开头
	if len(nsName) > 5 && nsName[:5] == "kube-" {
		return true
	}

	return false
}

// createOrGetEnvironment 创建或获取环境
func (rc *ResourceClassifier) createOrGetEnvironment(clusterID string, tenant *model.Tenant, namespace corev1.Namespace) (*model.Environment, error) {
	// 尝试获取现有环境
	env, err := rc.environmentRepo.GetByNamespace(clusterID, namespace.Name)
	if err == nil {
		return env, nil
	}

	// 如果不是记录未找到错误，返回错误
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 创建新环境
	displayName := namespace.Name
	if label, ok := namespace.Labels["env"]; ok {
		displayName = label
	}

	labels := make(model.JSONMap)
	for k, v := range namespace.Labels {
		labels[k] = v
	}

	env = &model.Environment{
		TenantID:    tenant.ID,
		ClusterID:   model.ParseUUID(clusterID),
		Namespace:   namespace.Name,
		DisplayName: displayName,
		Description: "自动导入的Kubernetes命名空间",
		Labels:      labels,
		Status:      model.EnvironmentStatusActive,
	}

	if err := rc.environmentRepo.Create(env); err != nil {
		return nil, err
	}

	return env, nil
}

// discoverApplications 发现应用
func (rc *ResourceClassifier) discoverApplications(ctx context.Context, env *model.Environment, namespace corev1.Namespace) ([]*model.Application, error) {
	var apps []*model.Application

	// 获取Deployments
	deployments, err := rc.kubernetesClient.AppsV1().Deployments(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 按标签聚合Deployments
	appGroups := make(map[string][]appsv1.Deployment)

	for _, deployment := range deployments.Items {
		appName := rc.extractAppLabel(deployment.Labels)
		if appName == "" {
			// 没有标签，使用Deployment名称作为应用名
			appName = deployment.Name
		}
		appGroups[appName] = append(appGroups[appName], deployment)
	}

	// 创建应用记录
	for appName, deployments := range appGroups {
		app := &model.Application{
			TenantID:        env.TenantID,
			EnvironmentID:   env.ID,
			Name:            appName,
			DisplayName:     appName,
			Description:     "自动发现的应用",
			Labels:          rc.extractCommonLabels(deployments[0].Labels),
			WorkloadTypes:   []string{"Deployment"},
			DeploymentCount: len(deployments),
		}

		if err := rc.applicationRepo.Create(app); err != nil {
			// 记录错误但继续
			continue
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// extractAppLabel 提取应用标签
func (rc *ResourceClassifier) extractAppLabel(labels map[string]string) string {
	// 优先级顺序
	if name, ok := labels[model.AppLabelKubernetesName]; ok {
		return name
	}
	if app, ok := labels[model.AppLabelStandard]; ok {
		return app
	}
	if k8sApp, ok := labels[model.AppLabelK8sApp]; ok {
		return k8sApp
	}
	return ""
}

// extractCommonLabels 提取公共标签
func (rc *ResourceClassifier) extractCommonLabels(labels map[string]string) model.JSONMap {
	commonLabels := make(map[string]interface{})
	importantKeys := []string{
		model.AppLabelKubernetesName,
		model.AppLabelStandard,
		model.AppLabelK8sApp,
		"version",
		"component",
		"tier",
	}

	for _, key := range importantKeys {
		if val, ok := labels[key]; ok {
			commonLabels[key] = val
		}
	}

	return commonLabels
}

// getResourceQuota 获取资源配额
func (rc *ResourceClassifier) getResourceQuota(ctx context.Context, namespaceName string) (*model.ResourceQuota, error) {
	quotaList, err := rc.kubernetesClient.CoreV1().ResourceQuotas(namespaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(quotaList.Items) == 0 {
		return nil, nil
	}

	quota := quotaList.Items[0]

	hardLimits := make(map[string]interface{})
	for k, v := range quota.Status.Hard {
		hardLimits[string(k)] = v.String()
	}

	used := make(map[string]interface{})
	for k, v := range quota.Status.Used {
		used[string(k)] = v.String()
	}

	return &model.ResourceQuota{
		HardLimits: hardLimits,
		Used:       used,
		Status:     "Active",
	}, nil
}
