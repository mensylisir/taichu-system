package service

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
)

type ImportService struct {
	importRepo     *repository.ImportRecordRepository
	clusterRepo    *repository.ClusterRepository
	encryptionSvc  *EncryptionService
	clusterManager *ClusterManager
	clusterService *ClusterService
	// 新增：三级分类模型相关
	tenantRepo         *repository.TenantRepository
	environmentRepo    *repository.EnvironmentRepository
	applicationRepo    *repository.ApplicationRepository
	quotaRepo          *repository.QuotaRepository
	resourceClassifier *ResourceClassifier
}

type ImportRecordWithDetails struct {
	*model.ImportRecord
}

func NewImportService(
	importRepo *repository.ImportRecordRepository,
	clusterRepo *repository.ClusterRepository,
	encryptionSvc *EncryptionService,
	clusterManager *ClusterManager,
	clusterService *ClusterService,
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
) *ImportService {
	return &ImportService{
		importRepo:      importRepo,
		clusterRepo:     clusterRepo,
		encryptionSvc:   encryptionSvc,
		clusterManager:  clusterManager,
		clusterService:  clusterService,
		tenantRepo:      tenantRepo,
		environmentRepo: environmentRepo,
		applicationRepo: applicationRepo,
		quotaRepo:       quotaRepo,
	}
}

func (s *ImportService) ImportCluster(importSource, name, description, environmentType, region, kubeconfig string, labels map[string]string) (*model.ImportRecord, error) {
	log.Printf("Starting ImportCluster with importSource=%s, name=%s", importSource, name)

	// 验证kubeconfig
	isValid, err := s.clusterManager.ValidateKubeconfig(kubeconfig)
	if err != nil || !isValid {
		log.Printf("Invalid kubeconfig: %v", err)
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}

	log.Printf("Kubeconfig validation passed")

	// 创建导入记录
	importRecord := &model.ImportRecord{
		ImportSource: importSource,
		ImportStatus: "pending",
		ImportedBy:   "api-user",
		ValidationResults: map[string]interface{}{
			"kubeconfig_valid": true,
		},
		ImportedResources: map[string]interface{}{
			"nodes":       "pending",
			"namespaces":  "pending",
			"deployments": "pending",
			"services":    "pending",
		},
	}

	if err := s.importRepo.Create(importRecord); err != nil {
		log.Printf("Failed to create import record: %v", err)
		return nil, fmt.Errorf("failed to create import record: %w", err)
	}

	log.Printf("Import record created with ID: %s", importRecord.ID.String())

	// 创建集群记录
	encryptedKubeconfig, err := s.encryptionSvc.Encrypt(kubeconfig)
	if err != nil {
		log.Printf("Failed to encrypt kubeconfig: %v", err)
		return nil, fmt.Errorf("failed to encrypt kubeconfig: %w", err)
	}

	log.Printf("Kubeconfig encrypted successfully")

	clusterLabels := make(map[string]interface{})
	for k, v := range labels {
		clusterLabels[k] = v
	}

	// 检查同名集群是否已存在
	exists, err := s.clusterRepo.ExistsByName(name)
	if err != nil {
		log.Printf("Failed to check cluster existence: %v", err)
		return nil, fmt.Errorf("failed to check cluster existence: %w", err)
	}
	if exists {
		log.Printf("Cluster with name '%s' already exists", name)
		return nil, fmt.Errorf("cluster with name '%s' already exists", name)
	}

	cluster := &model.Cluster{
		Name:                name,
		Description:         description,
		KubeconfigEncrypted: encryptedKubeconfig,
		Labels:              clusterLabels,
		EnvironmentType:     environmentType,
		Region:              region,
		Provider:            "太初",
		CreatedBy:           "api-user",
		ImportSource:        importSource,
	}

	if err := s.clusterRepo.Create(cluster); err != nil {
		log.Printf("Failed to create cluster: %v", err)
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	log.Printf("Cluster created with ID: %s", cluster.ID.String())

	// 更新导入记录
	importRecord.ClusterID = &cluster.ID
	if err := s.importRepo.Update(importRecord); err != nil {
		log.Printf("Failed to update import record with cluster ID: %v", err)
		return nil, fmt.Errorf("failed to update import record: %w", err)
	}
	log.Printf("Import record updated with cluster ID: %s", cluster.ID.String())
	log.Printf("ImportCluster completed successfully")
	return importRecord, nil
}

func (s *ImportService) ExecuteImport(importID string) error {
	log.Printf("Starting ExecuteImport for importID: %s", importID)

	importRecord, err := s.importRepo.GetByID(importID)
	if err != nil {
		log.Printf("Failed to get import record %s: %v", importID, err)
		return fmt.Errorf("failed to get import record: %w", err)
	}

	log.Printf("Got import record: ID=%s, Status=%s, ClusterID=%v",
		importRecord.ID, importRecord.ImportStatus, importRecord.ClusterID)

	// 更新状态为validating
	importRecord.ImportStatus = "validating"
	if err := s.importRepo.Update(importRecord); err != nil {
		log.Printf("Failed to update import status to validating: %v", err)
		return fmt.Errorf("failed to update import status: %w", err)
	}

	log.Printf("Updated import status to validating")

	if importRecord.ClusterID == nil {
		log.Printf("ClusterID is nil for import record %s", importID)
		return s.handleImportError(importRecord, fmt.Errorf("cluster ID is nil"))
	}

	cluster, err := s.clusterRepo.GetByID(importRecord.ClusterID.String())
	if err != nil {
		log.Printf("Failed to get cluster %s: %v", importRecord.ClusterID.String(), err)
		return s.handleImportError(importRecord, fmt.Errorf("failed to get cluster: %w", err))
	}

	log.Printf("Got cluster: ID=%s, Name=%s", cluster.ID, cluster.Name)

	// 验证集群连接
	kubeconfig, err := s.encryptionSvc.Decrypt(cluster.KubeconfigEncrypted)
	if err != nil {
		log.Printf("Failed to decrypt kubeconfig: %v", err)
		return s.handleImportError(importRecord, fmt.Errorf("failed to decrypt kubeconfig: %w", err))
	}

	log.Printf("Successfully decrypted kubeconfig")

	// 更新状态为importing
	importRecord.ImportStatus = "importing"
	if err := s.importRepo.Update(importRecord); err != nil {
		log.Printf("Failed to update import status to importing: %v", err)
		return fmt.Errorf("failed to update import status: %w", err)
	}

	log.Printf("Updated import status to importing")

	// 执行导入操作
	log.Printf("Starting performImport...")
	if err := s.performImport(importRecord, cluster, kubeconfig); err != nil {
		log.Printf("performImport failed: %v", err)
		return s.handleImportError(importRecord, err)
	}

	log.Printf("performImport completed successfully")

	// 导入完成
	importRecord.ImportStatus = constants.StatusCompleted
	importRecord.CompletedAt = func() *time.Time { now := time.Now(); return &now }()
	importRecord.ValidationResults["import_completed"] = true

	log.Printf("Updating import status to completed")
	if err := s.importRepo.Update(importRecord); err != nil {
		log.Printf("Failed to update import status to completed: %v", err)
		return fmt.Errorf("failed to update import status: %w", err)
	}

	log.Printf("ExecuteImport completed successfully for importID: %s", importID)
	return nil
}

func (s *ImportService) performImport(importRecord *model.ImportRecord, cluster *model.Cluster, kubeconfig string) error {
	ctx := context.Background()
	clientset, err := s.clusterManager.GetClient(ctx, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	// 统计资源数量
	importRecord.ImportedResources["nodes"] = s.countResources(clientset, "nodes")
	importRecord.ImportedResources["namespaces"] = s.countResources(clientset, "namespaces")
	importRecord.ImportedResources["deployments"] = s.countResources(clientset, "deployments")
	importRecord.ImportedResources["services"] = s.countResources(clientset, "services")

	// ===== 新增：三级分类模型导入 =====
	// 1. 初始化预定义租户（system和default）
	if err := s.initializePredefinedTenants(); err != nil {
		log.Printf("Warning: Failed to initialize predefined tenants: %v", err)
	}

	// 2. 分类所有命名空间并创建环境记录
	classificationResults, err := s.classifyAndImportResources(cluster, clientset)
	if err != nil {
		log.Printf("Warning: Failed to classify resources: %v", err)
	} else {
		// 更新导入资源统计
		importRecord.ImportedResources["tenants"] = classificationResults.TenantCount
		importRecord.ImportedResources["environments"] = classificationResults.EnvironmentCount
		importRecord.ImportedResources["applications"] = classificationResults.ApplicationCount
		importRecord.ImportedResources["resource_quotas"] = classificationResults.QuotaCount
	}

	// 同步集群状态（包括版本、节点数等）
	err = s.clusterService.TriggerSync(cluster.ID.String())
	if err != nil {
		return fmt.Errorf("failed to trigger sync: %w", err)
	}

	return nil
}

// ClassificationResult 分类结果
type ClassificationResult struct {
	TenantCount      int `json:"tenant_count"`
	EnvironmentCount int `json:"environment_count"`
	ApplicationCount int `json:"application_count"`
	QuotaCount       int `json:"quota_count"`
}

// initializePredefinedTenants 初始化预定义租户
func (s *ImportService) initializePredefinedTenants() error {
	return s.tenantRepo.EnsurePredefinedTenants()
}

// classifyAndImportResources 分类并导入资源
func (s *ImportService) classifyAndImportResources(cluster *model.Cluster, clientset *kubernetes.Clientset) (*ClassificationResult, error) {
	// 创建资源分类器
	if s.resourceClassifier == nil {
		s.resourceClassifier = NewResourceClassifier(
			clientset,
			s.tenantRepo,
			s.environmentRepo,
			s.applicationRepo,
			s.quotaRepo,
		)
	}

	// 分类所有命名空间
	results, err := s.resourceClassifier.ClassifyAllNamespaces(cluster.ID.String())
	if err != nil {
		return nil, err
	}

	// 统计结果
	classificationResult := &ClassificationResult{
		TenantCount:      0,
		EnvironmentCount: 0,
		ApplicationCount: 0,
		QuotaCount:       0,
	}

	// 统计租户数量（去重）
	tenantMap := make(map[string]bool)
	for _, result := range results {
		tenantMap[result.AssignedTenant.ID.String()] = true

		classificationResult.EnvironmentCount++
		classificationResult.ApplicationCount += len(result.Applications)

		if result.ResourceQuota != nil {
			classificationResult.QuotaCount++
		}
	}
	classificationResult.TenantCount = len(tenantMap)

	log.Printf("Classification completed: %d tenants, %d environments, %d applications, %d quotas",
		classificationResult.TenantCount,
		classificationResult.EnvironmentCount,
		classificationResult.ApplicationCount,
		classificationResult.QuotaCount,
	)

	return classificationResult, nil
}

func (s *ImportService) countResources(clientset *kubernetes.Clientset, resourceType string) int {
	ctx := context.Background()
	switch resourceType {
	case "nodes":
		nodes, _ := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		return len(nodes.Items)
	case "namespaces":
		namespaces, _ := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		return len(namespaces.Items)
	case "deployments":
		deployments, _ := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
		return len(deployments.Items)
	case "services":
		services, _ := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
		return len(services.Items)
	default:
		return 0
	}
}

func (s *ImportService) handleImportError(importRecord *model.ImportRecord, err error) error {
	importRecord.ImportStatus = "failed"
	importRecord.ErrorMessage = err.Error()
	return s.importRepo.Update(importRecord)
}

func (s *ImportService) GetImportStatus(importID string) (*model.ImportRecord, error) {
	return s.importRepo.GetByID(importID)
}

func (s *ImportService) ListImports(importSource, status string) ([]*model.ImportRecord, int64, error) {
	return s.importRepo.List(importSource, status)
}

func (s *ImportService) DeleteImport(importID string) error {
	return s.importRepo.Delete(importID)
}
