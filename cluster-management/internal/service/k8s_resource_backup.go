package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type ResourceBackupService struct {
	clientset *kubernetes.Clientset
}

func NewResourceBackupService(clientset *kubernetes.Clientset) *ResourceBackupService {
	return &ResourceBackupService{
		clientset: clientset,
	}
}

// BackupResources 备份Kubernetes资源
func (s *ResourceBackupService) BackupResources(ctx context.Context, backupPath string) error {
	// 创建资源目录
	resourcesPath := filepath.Join(backupPath, "resources")
	if err := os.MkdirAll(resourcesPath, 0755); err != nil {
		return fmt.Errorf("failed to create resources directory: %w", err)
	}

	fmt.Println("[RESOURCE-BACKUP] Starting resource backup via clientset")

	// 按依赖顺序备份资源
	resourceTypes := []string{
		"namespaces",
		"persistentvolumes",
		"persistentvolumeclaims",
		"configmaps",
		"secrets",
		"services",
		"endpoints",
		"pods",
		"deployments",
		"statefulsets",
		"daemonsets",
		"ingresses",
		"networkpolicies",
		"roles",
		"rolebindings",
		"clusterroles",
		"clusterrolebindings",
		"serviceaccounts",
		"poddisruptionbudgets",
		"podsecuritypolicies",
		"limitranges",
		"resourcequotas",
		"horizontalpodautoscalers",
		"verticalpodautoscalers",
		"certificatesigningrequests",
		"leases",
		"events",
	}

	for _, resourceType := range resourceTypes {
		if err := s.backupResourceTypeViaClientset(ctx, resourcesPath, resourceType); err != nil {
			fmt.Printf("[RESOURCE-BACKUP] Warning: failed to backup %s: %v\n", resourceType, err)
			continue
		}
	}

	fmt.Println("[RESOURCE-BACKUP] Resource backup completed successfully")
	return nil
}

// backupResourceTypeViaClientset 使用clientset备份特定类型的资源
func (s *ResourceBackupService) backupResourceTypeViaClientset(ctx context.Context, resourcesPath, resourceType string) error {
	resourceDir := filepath.Join(resourcesPath, resourceType)
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return fmt.Errorf("failed to create resource directory: %w", err)
	}

	fmt.Printf("[RESOURCE-BACKUP] Backing up resource type: %s\n", resourceType)

	// 根据资源类型选择合适的客户端方法
	switch resourceType {
	case "namespaces":
		return s.backupNamespaces(ctx, resourceDir)
	case "persistentvolumes":
		return s.backupPersistentVolumes(ctx, resourceDir)
	case "persistentvolumeclaims":
		return s.backupPersistentVolumeClaims(ctx, resourceDir)
	case "configmaps":
		return s.backupConfigMaps(ctx, resourceDir)
	case "secrets":
		return s.backupSecrets(ctx, resourceDir)
	case "services":
		return s.backupServices(ctx, resourceDir)
	case "pods":
		return s.backupPods(ctx, resourceDir)
	case "deployments":
		return s.backupDeployments(ctx, resourceDir)
	case "statefulsets":
		return s.backupStatefulSets(ctx, resourceDir)
	case "daemonsets":
		return s.backupDaemonSets(ctx, resourceDir)
	case "ingresses":
		return s.backupIngresses(ctx, resourceDir)
	default:
		fmt.Printf("[RESOURCE-BACKUP] Skipping unsupported resource type: %s\n", resourceType)
		return nil
	}
}

// 备份命名空间
func (s *ResourceBackupService) backupNamespaces(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("[RESOURCE-BACKUP] Found %d namespaces\n", len(namespaces.Items))

	for _, ns := range namespaces.Items {
		// 跳过系统命名空间
		if isSystemNamespace(ns.Name) {
			fmt.Printf("[RESOURCE-BACKUP] Skipping system namespace: %s\n", ns.Name)
			continue
		}

		filename := filepath.Join(resourceDir, fmt.Sprintf("namespace-%s.yaml", ns.Name))
		data, err := yaml.Marshal(ns)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}

		fmt.Printf("[RESOURCE-BACKUP] Backed up namespace: %s\n", ns.Name)
	}

	return nil
}

// 备份ConfigMap
func (s *ResourceBackupService) backupConfigMaps(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		configMaps, err := s.clientset.CoreV1().ConfigMaps(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, cm := range configMaps.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("configmap-%s-%s.yaml", ns.Name, cm.Name))
			data, err := yaml.Marshal(cm)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up configmap: %s/%s\n", ns.Name, cm.Name)
		}
	}

	return nil
}

// 备份Secret
func (s *ResourceBackupService) backupSecrets(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		secrets, err := s.clientset.CoreV1().Secrets(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, secret := range secrets.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("secret-%s-%s.yaml", ns.Name, secret.Name))
			data, err := yaml.Marshal(secret)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up secret: %s/%s\n", ns.Name, secret.Name)
		}
	}

	return nil
}

// 备份Service
func (s *ResourceBackupService) backupServices(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		services, err := s.clientset.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, svc := range services.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("service-%s-%s.yaml", ns.Name, svc.Name))
			data, err := yaml.Marshal(svc)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up service: %s/%s\n", ns.Name, svc.Name)
		}
	}

	return nil
}

// 备份Pod
func (s *ResourceBackupService) backupPods(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		pods, err := s.clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, pod := range pods.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("pod-%s-%s.yaml", ns.Name, pod.Name))
			data, err := yaml.Marshal(pod)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up pod: %s/%s\n", ns.Name, pod.Name)
		}
	}

	return nil
}

// 备份Deployment
func (s *ResourceBackupService) backupDeployments(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		deployments, err := s.clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, deploy := range deployments.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("deployment-%s-%s.yaml", ns.Name, deploy.Name))
			data, err := yaml.Marshal(deploy)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up deployment: %s/%s\n", ns.Name, deploy.Name)
		}
	}

	return nil
}

// 备份StatefulSet
func (s *ResourceBackupService) backupStatefulSets(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		statefulSets, err := s.clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, ss := range statefulSets.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("statefulset-%s-%s.yaml", ns.Name, ss.Name))
			data, err := yaml.Marshal(ss)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up statefulset: %s/%s\n", ns.Name, ss.Name)
		}
	}

	return nil
}

// 备份DaemonSet
func (s *ResourceBackupService) backupDaemonSets(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		daemonSets, err := s.clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, ds := range daemonSets.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("daemonset-%s-%s.yaml", ns.Name, ds.Name))
			data, err := yaml.Marshal(ds)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up daemonset: %s/%s\n", ns.Name, ds.Name)
		}
	}

	return nil
}

// 备份Ingress
func (s *ResourceBackupService) backupIngresses(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		ingresses, err := s.clientset.NetworkingV1().Ingresses(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, ing := range ingresses.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("ingress-%s-%s.yaml", ns.Name, ing.Name))
			data, err := yaml.Marshal(ing)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up ingress: %s/%s\n", ns.Name, ing.Name)
		}
	}

	return nil
}

// 备份PersistentVolume
func (s *ResourceBackupService) backupPersistentVolumes(ctx context.Context, resourceDir string) error {
	pvs, err := s.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pv := range pvs.Items {
		filename := filepath.Join(resourceDir, fmt.Sprintf("persistentvolume-%s.yaml", pv.Name))
		data, err := yaml.Marshal(pv)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}

		fmt.Printf("[RESOURCE-BACKUP] Backed up persistentvolume: %s\n", pv.Name)
	}

	return nil
}

// 备份PersistentVolumeClaim
func (s *ResourceBackupService) backupPersistentVolumeClaims(ctx context.Context, resourceDir string) error {
	namespaces, err := s.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			continue
		}

		pvcs, err := s.clientset.CoreV1().PersistentVolumeClaims(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, pvc := range pvcs.Items {
			filename := filepath.Join(resourceDir, fmt.Sprintf("persistentvolumeclaim-%s-%s.yaml", ns.Name, pvc.Name))
			data, err := yaml.Marshal(pvc)
			if err != nil {
				return err
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			fmt.Printf("[RESOURCE-BACKUP] Backed up persistentvolumeclaim: %s/%s\n", ns.Name, pvc.Name)
		}
	}

	return nil
}

// RestoreResources 恢复Kubernetes资源
func (s *ResourceBackupService) RestoreResources(ctx context.Context, backupPath string) error {
	resourcesPath := filepath.Join(backupPath, "resources")
	if _, err := os.Stat(resourcesPath); os.IsNotExist(err) {
		return fmt.Errorf("resources backup not found at %s", resourcesPath)
	}

	fmt.Println("[RESOURCE-RESTORE] Starting resource restore")

	// 按依赖顺序恢复资源
	resourceTypes := []string{
		"namespaces",
		"persistentvolumes",
		"persistentvolumeclaims",
		"configmaps",
		"secrets",
		"services",
		"pods",
		"deployments",
		"statefulsets",
		"daemonsets",
		"ingresses",
	}

	for _, resourceType := range resourceTypes {
		if err := s.restoreResourceType(ctx, resourcesPath, resourceType); err != nil {
			fmt.Printf("[RESOURCE-RESTORE] Warning: failed to restore %s: %v\n", resourceType, err)
			continue
		}
	}

	fmt.Println("[RESOURCE-RESTORE] Resource restore completed successfully")
	return nil
}

// restoreResourceType 恢复特定类型的资源
func (s *ResourceBackupService) restoreResourceType(ctx context.Context, resourcesPath, resourceType string) error {
	resourceDir := filepath.Join(resourcesPath, resourceType)
	files, err := os.ReadDir(resourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read resource directory: %w", err)
	}

	fmt.Printf("[RESOURCE-RESTORE] Restoring resource type: %s\n", resourceType)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(resourceDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// 使用clientset应用资源到集群
		if err := s.applyResourceViaClientset(ctx, content, resourceType); err != nil {
			fmt.Printf("[RESOURCE-RESTORE] Warning: failed to restore %s: %v\n", file.Name(), err)
			continue
		}
	}

	return nil
}

// applyResourceViaClientset 使用clientset应用资源到集群
func (s *ResourceBackupService) applyResourceViaClientset(ctx context.Context, resourceContent []byte, resourceType string) error {
	// 解析YAML内容
	var resourceObj interface{}
	if err := yaml.Unmarshal(resourceContent, &resourceObj); err != nil {
		return fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	// 根据资源类型应用资源
	switch resourceType {
	case "namespaces":
		var ns corev1.Namespace
		if err := yaml.Unmarshal(resourceContent, &ns); err != nil {
			return err
		}
		_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	case "configmaps":
		var cm corev1.ConfigMap
		if err := yaml.Unmarshal(resourceContent, &cm); err != nil {
			return err
		}
		_, err := s.clientset.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, &cm, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create configmap: %w", err)
		}
	case "secrets":
		var secret corev1.Secret
		if err := yaml.Unmarshal(resourceContent, &secret); err != nil {
			return err
		}
		_, err := s.clientset.CoreV1().Secrets(secret.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
	case "services":
		var svc corev1.Service
		if err := yaml.Unmarshal(resourceContent, &svc); err != nil {
			return err
		}
		_, err := s.clientset.CoreV1().Services(svc.Namespace).Create(ctx, &svc, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	case "deployments":
		var deploy appsv1.Deployment
		if err := yaml.Unmarshal(resourceContent, &deploy); err != nil {
			return err
		}
		_, err := s.clientset.AppsV1().Deployments(deploy.Namespace).Create(ctx, &deploy, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	case "statefulsets":
		var ss appsv1.StatefulSet
		if err := yaml.Unmarshal(resourceContent, &ss); err != nil {
			return err
		}
		_, err := s.clientset.AppsV1().StatefulSets(ss.Namespace).Create(ctx, &ss, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create statefulset: %w", err)
		}
	case "daemonsets":
		var ds appsv1.DaemonSet
		if err := yaml.Unmarshal(resourceContent, &ds); err != nil {
			return err
		}
		_, err := s.clientset.AppsV1().DaemonSets(ds.Namespace).Create(ctx, &ds, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create daemonset: %w", err)
		}
	case "ingresses":
		var ing networkingv1.Ingress
		if err := yaml.Unmarshal(resourceContent, &ing); err != nil {
			return err
		}
		_, err := s.clientset.NetworkingV1().Ingresses(ing.Namespace).Create(ctx, &ing, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create ingress: %w", err)
		}
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return nil
}

// isSystemNamespace 检查是否为系统命名空间
func isSystemNamespace(namespace string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
		"kube-admission",
		"gatekeeper-system",
		"istio-system",
		"knative-eventing",
		"knative-serving",
		"logging",
		"monitoring",
	}

	for _, ns := range systemNamespaces {
		if namespace == ns {
			return true
		}
	}

	return false
}
