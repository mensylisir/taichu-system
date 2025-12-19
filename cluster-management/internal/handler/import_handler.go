package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/internal/service/worker"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type ImportHandler struct {
	importService     *service.ImportService
	healthWorker      *worker.HealthCheckWorker
	resourceSyncWorker *worker.ResourceSyncWorker
	auditService      *service.AuditService
}

type ImportClusterRequest struct {
	ImportSource     string            `json:"import_source" binding:"required"`
	Name             string            `json:"name" binding:"required"`
	Description      string            `json:"description"`
	EnvironmentType  string            `json:"environment_type"`
	Kubeconfig       string            `json:"kubeconfig" binding:"required"`
	Labels           map[string]string `json:"labels"`
}

type ImportRecordSummary struct {
	ID               uuid.UUID  `json:"id"`
	ClusterID        string     `json:"cluster_id,omitempty"`
	ImportSource     string     `json:"import_source"`
	ImportStatus     string     `json:"import_status"`
	ImportedResources string     `json:"imported_resources"`
	ImportedBy       string    `json:"imported_by"`
	ImportedAt       string    `json:"imported_at"`
	CompletedAt      string    `json:"completed_at"`
}

type ImportStatusResponse struct {
	ImportID         uuid.UUID `json:"import_id"`
	ImportStatus     string    `json:"import_status"`
	Progress         int       `json:"progress"`
	CurrentStep      string    `json:"current_step"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	ValidationResults map[string]interface{} `json:"validation_results,omitempty"`
}

type ImportListResponse struct {
	Imports []ImportRecordSummary `json:"imports"`
	Total   int64                 `json:"total"`
}

func NewImportHandler(importService *service.ImportService, healthWorker *worker.HealthCheckWorker, resourceSyncWorker *worker.ResourceSyncWorker, auditService *service.AuditService) *ImportHandler {
	return &ImportHandler{
		importService:     importService,
		healthWorker:      healthWorker,
		resourceSyncWorker: resourceSyncWorker,
		auditService:      auditService,
	}
}

func (h *ImportHandler) ImportCluster(c *gin.Context) {
	var req ImportClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	importRecord, err := h.importService.ImportCluster(req.ImportSource, req.Name, req.Description, req.EnvironmentType, req.Kubeconfig, req.Labels)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to import cluster: %s", err.Error())
		return
	}

	// 启动异步导入任务
	log.Printf("Starting async import task for import record ID: %s", importRecord.ID.String())
	go h.importService.ExecuteImport(importRecord.ID.String())
	log.Printf("Async import task started for import record ID: %s", importRecord.ID.String())

	// 如果集群已创建，触发同步
	if importRecord.ClusterID != nil {
		if h.healthWorker != nil {
			log.Printf("Triggering health check for cluster ID: %s", importRecord.ClusterID.String())
			go h.healthWorker.TriggerSync(*importRecord.ClusterID)
		}

		if h.resourceSyncWorker != nil {
			log.Printf("Triggering resource sync for cluster ID: %s", importRecord.ClusterID.String())
			go h.resourceSyncWorker.TriggerSync(*importRecord.ClusterID)
		}
	}

	// 记录审计日志
	if h.auditService != nil {
		user := "api-user" // TODO: 从上下文中获取实际用户
		clusterID := uuid.Nil
		if importRecord.ClusterID != nil {
			clusterID = *importRecord.ClusterID
		}
		h.auditService.LogClusterOperation(
			clusterID,
			"import",
			"cluster",
			user,
			nil,
			map[string]interface{}{
				"import_id":       importRecord.ID,
				"import_source":   importRecord.ImportSource,
				"name":            req.Name,
				"description":     req.Description,
				"import_status":   importRecord.ImportStatus,
			},
		)
	}

	response := ImportRecordSummary{
		ID:               importRecord.ID,
		ClusterID:        func() string {
			if importRecord.ClusterID != nil {
				return importRecord.ClusterID.String()
			}
			return ""
		}(),
		ImportSource:     importRecord.ImportSource,
		ImportStatus:     importRecord.ImportStatus,
		ImportedResources: "pending",
		ImportedBy:       importRecord.ImportedBy,
		ImportedAt:       importRecord.ImportedAt.Format("2006-01-02T15:04:05Z07:00"),
		CompletedAt: func() string {
			if importRecord.CompletedAt != nil {
				return importRecord.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			}
			return ""
		}(),
	}

	utils.Success(c, http.StatusCreated, response)
}

func (h *ImportHandler) GetImportStatus(c *gin.Context) {
	importID := c.Param("importId")

	id, err := uuid.Parse(importID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid import ID")
		return
	}

	status, err := h.importService.GetImportStatus(id.String())
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Import not found")
		return
	}

	response := ImportStatusResponse{
		ImportID:         status.ID,
		ImportStatus:     status.ImportStatus,
		Progress:         h.calculateProgress(status.ImportStatus),
		CurrentStep:      h.getCurrentStep(status.ImportStatus),
		ErrorMessage:     status.ErrorMessage,
		ValidationResults: status.ValidationResults,
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *ImportHandler) ListImports(c *gin.Context) {
	importSource := c.Query("import_source")
	status := c.Query("status")

	imports, total, err := h.importService.ListImports(importSource, status)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list imports: %v", err)
		return
	}

	importSummaries := make([]ImportRecordSummary, 0, len(imports))
	for _, importRecord := range imports {
		var clusterIDStr string
		if importRecord.ClusterID != nil {
			clusterIDStr = importRecord.ClusterID.String()
		}
		
		importSummaries = append(importSummaries, ImportRecordSummary{
			ID:               importRecord.ID,
			ClusterID:        clusterIDStr,
			ImportSource:     importRecord.ImportSource,
			ImportStatus:     importRecord.ImportStatus,
			ImportedResources: "pending",
			ImportedBy:       importRecord.ImportedBy,
			ImportedAt:       importRecord.ImportedAt.Format("2006-01-02T15:04:05Z07:00"),
			CompletedAt: func() string {
				if importRecord.CompletedAt != nil {
					return importRecord.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
				}
				return ""
			}(),
		})
	}

	response := ImportListResponse{
		Imports: importSummaries,
		Total:   total,
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *ImportHandler) calculateProgress(status string) int {
	switch status {
	case "pending":
		return 0
	case "validating":
		return 25
	case "importing":
		return 50
	case "completed":
		return 100
	case "failed":
		return 0
	default:
		return 0
	}
}

func (h *ImportHandler) getCurrentStep(status string) string {
	switch status {
	case "pending":
		return "Waiting to start"
	case "validating":
		return "Validating kubeconfig"
	case "importing":
		return "Importing cluster resources"
	case "completed":
		return "Import completed successfully"
	case "failed":
		return "Import failed"
	default:
		return "Unknown"
	}
}
