package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type BackupHandler struct {
	backupService  *service.BackupService
	restoreService *service.RestoreService
	auditService   *service.AuditService
}

type CreateBackupRequest struct {
	BackupName    string `json:"backup_name" binding:"required,min=1,max=100"`
	BackupType    string `json:"backup_type" binding:"required,oneof=etcd resource"`
	RetentionDays int    `json:"retention_days" binding:"omitempty,min=1,max=3650"`
}

type BackupSummary struct {
	ID                uuid.UUID `json:"id"`
	BackupName        string    `json:"backup_name"`
	BackupType        string    `json:"backup_type"`
	Status            string    `json:"status"`
	StorageLocation   string    `json:"storage_location"`
	StorageSizeBytes  int64     `json:"storage_size_bytes"`
	SnapshotTimestamp string    `json:"snapshot_timestamp"`
	CreatedAt         string    `json:"created_at"`
	CompletedAt       string    `json:"completed_at"`
}

type BackupListResponse struct {
	Backups []BackupSummary `json:"backups"`
	Total   int64           `json:"total"`
}

type BackupDetailResponse struct {
	ID                uuid.UUID `json:"id"`
	BackupName        string    `json:"backup_name"`
	BackupType        string    `json:"backup_type"`
	Status            string    `json:"status"`
	StorageLocation   string    `json:"storage_location"`
	StorageSizeBytes  int64     `json:"storage_size_bytes"`
	SnapshotTimestamp string    `json:"snapshot_timestamp"`
	RetentionDays     int       `json:"retention_days"`
	CreatedAt         string    `json:"created_at"`
	CompletedAt       string    `json:"completed_at"`
	ErrorMessage      string    `json:"error_message,omitempty"`
}

type RestoreBackupRequest struct {
	RestoreName string `json:"restore_name" binding:"required,min=1,max=100"`
}

type RestoreProgressResponse struct {
	RestoreID     string  `json:"restore_id"`
	Status        string  `json:"status"`
	Progress      float64 `json:"progress"`
	CurrentStep   string  `json:"current_step"`
	StartTime     string  `json:"start_time"`
	EstimatedTime int     `json:"estimated_time,omitempty"`
}

type CreateBackupScheduleRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=100"`
	CronExpr      string `json:"cron_expr" binding:"required,cron"`
	BackupType    string `json:"backup_type" binding:"required,oneof=etcd resource"`
	RetentionDays int    `json:"retention_days" binding:"omitempty,min=1,max=3650"`
	Enabled       bool   `json:"enabled"`
	CreatedBy     string `json:"created_by" binding:"required"`

	// etcd配置
	EtcdEndpoints string `json:"etcd_endpoints"`
	EtcdCaCert    string `json:"etcd_ca_cert"`
	EtcdCert      string `json:"etcd_cert"`
	EtcdKey       string `json:"etcd_key"`
	EtcdDataDir   string `json:"etcd_data_dir"`
	EtcdctlPath   string `json:"etcdctl_path"`
	SshUsername   string `json:"ssh_username"`
	SshPassword   string `json:"ssh_password"`

	// 部署方式配置
	EtcdDeploymentType string `json:"etcd_deployment_type"`
	K8sDeploymentType  string `json:"k8s_deployment_type"`
}

type UpdateBackupScheduleRequest struct {
	CronExpr      string `json:"cron_expr" binding:"omitempty,cron"`
	BackupType    string `json:"backup_type" binding:"omitempty,oneof=etcd resource"`
	RetentionDays int    `json:"retention_days" binding:"omitempty,min=1,max=3650"`
	Enabled       bool   `json:"enabled"`

	// etcd配置
	EtcdEndpoints string `json:"etcd_endpoints"`
	EtcdCaCert    string `json:"etcd_ca_cert"`
	EtcdCert      string `json:"etcd_cert"`
	EtcdKey       string `json:"etcd_key"`
	EtcdDataDir   string `json:"etcd_data_dir"`
	EtcdctlPath   string `json:"etcdctl_path"`
	SshUsername   string `json:"ssh_username"`
	SshPassword   string `json:"ssh_password"`

	// 部署方式配置
	EtcdDeploymentType string `json:"etcd_deployment_type"`
	K8sDeploymentType  string `json:"k8s_deployment_type"`
}

// CreateEtcdBackupRequest etcd备份请求
type CreateEtcdBackupRequest struct {
	BackupName    string `json:"backup_name" binding:"required,min=1,max=100"`
	BackupType    string `json:"backup_type" binding:"required,oneof=etcd"`
	RetentionDays int    `json:"retention_days" binding:"omitempty,min=1,max=3650"`
}

// CreateResourceBackupRequest 资源备份请求
type CreateResourceBackupRequest struct {
	BackupName    string `json:"backup_name" binding:"required,min=1,max=100"`
	BackupType    string `json:"backup_type" binding:"required,oneof=resource"`
	RetentionDays int    `json:"retention_days" binding:"omitempty,min=1,max=3650"`
}

func NewBackupHandler(backupService *service.BackupService, restoreService *service.RestoreService, auditService *service.AuditService) *BackupHandler {
	return &BackupHandler{
		backupService:  backupService,
		restoreService: restoreService,
		auditService:   auditService,
	}
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	backup, err := h.backupService.CreateBackup(id.String(), req.BackupName, req.BackupType, req.RetentionDays)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to create backup: %v", err)
		return
	}

	go h.backupService.ExecuteBackup(backup.ID.String())

	user := getUsernameFromContext(c)
	if user == "" {
		user = "system"
	}

	if h.auditService != nil {
		h.auditService.LogBackupOperation(
			id,
			constants.EventTypeCreate,
			backup.ID.String(),
			user,
			map[string]interface{}{
				"backup_name":    req.BackupName,
				"backup_type":    req.BackupType,
				"retention_days": req.RetentionDays,
			},
		)
	}

	utils.Success(c, http.StatusCreated, backup)
}

func (h *BackupHandler) CreateEtcdBackup(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	var req CreateEtcdBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	backup, err := h.backupService.CreateEtcdBackup(id.String(), req.BackupName, req.BackupType, req.RetentionDays)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to create etcd backup: %v", err)
		return
	}

	go h.backupService.ExecuteEtcdBackup(backup.ID.String())

	user := getUsernameFromContext(c)
	if user == "" {
		user = "system"
	}

	if h.auditService != nil {
		h.auditService.LogBackupOperation(
			id,
			"create_etcd",
			backup.ID.String(),
			user,
			map[string]interface{}{
				"backup_name":    req.BackupName,
				"backup_type":    req.BackupType,
				"retention_days": req.RetentionDays,
			},
		)
	}

	utils.Success(c, http.StatusCreated, backup)
}

func (h *BackupHandler) CreateResourceBackup(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	var req CreateResourceBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	backup, err := h.backupService.CreateResourceBackup(id.String(), req.BackupName, req.BackupType, req.RetentionDays)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to create resource backup: %v", err)
		return
	}

	go h.backupService.ExecuteResourceBackup(backup.ID.String())

	user := getUsernameFromContext(c)
	if user == "" {
		user = "system"
	}

	if h.auditService != nil {
		h.auditService.LogBackupOperation(
			id,
			"create_resource",
			backup.ID.String(),
			user,
			map[string]interface{}{
				"backup_name":    req.BackupName,
				"backup_type":    req.BackupType,
				"retention_days": req.RetentionDays,
			},
		)
	}

	utils.Success(c, http.StatusCreated, backup)
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	backups, total, err := h.backupService.ListBackups(id.String(), status, page, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list backups: %v", err)
		return
	}

	backupSummaries := make([]BackupSummary, 0, len(backups))
	for _, backup := range backups {
		backupSummaries = append(backupSummaries, BackupSummary{
			ID:                backup.ID,
			BackupName:        backup.BackupName,
			BackupType:        backup.BackupType,
			Status:            backup.Status,
			StorageLocation:   backup.StorageLocation,
			StorageSizeBytes:  backup.StorageSizeBytes,
			SnapshotTimestamp: backup.SnapshotTimestamp.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt:         backup.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			CompletedAt: func() string {
				if backup.CompletedAt != nil {
					return backup.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
				}
				return ""
			}(),
		})
	}

	response := BackupListResponse{
		Backups: backupSummaries,
		Total:   total,
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *BackupHandler) GetBackup(c *gin.Context) {
	clusterID := c.Param("id")
	backupID := c.Param("backupId")

	clusterUUID, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	backupUUID, err := utils.ParseUUID(backupID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid backup ID")
		return
	}

	backup, err := h.backupService.GetBackup(clusterUUID.String(), backupUUID.String())
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Backup not found")
		return
	}

	response := BackupDetailResponse{
		ID:                backup.ID,
		BackupName:        backup.BackupName,
		BackupType:        backup.BackupType,
		Status:            backup.Status,
		StorageLocation:   backup.StorageLocation,
		StorageSizeBytes:  backup.StorageSizeBytes,
		SnapshotTimestamp: backup.SnapshotTimestamp.Format("2006-01-02T15:04:05Z07:00"),
		RetentionDays:     backup.RetentionDays,
		CreatedAt:         backup.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		CompletedAt: func() string {
			if backup.CompletedAt != nil {
				return backup.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			}
			return ""
		}(),
		ErrorMessage: backup.ErrorMsg,
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	clusterID := c.Param("id")
	backupID := c.Param("backupId")

	clusterUUID, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	backupUUID, err := utils.ParseUUID(backupID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid backup ID")
		return
	}

	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	restoreID, err := h.restoreService.RestoreBackup(clusterUUID.String(), backupUUID.String(), req.RestoreName)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to restore backup: %v", err)
		return
	}

	user := getUsernameFromContext(c)
	if user == "" {
		user = "system"
	}

	if h.auditService != nil {
		h.auditService.LogBackupOperation(
			clusterUUID,
			"restore",
			backupUUID.String(),
			user,
			map[string]interface{}{
				"restore_name": req.RestoreName,
				"restore_id":   restoreID,
			},
		)
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message":    "Backup restoration started",
		"restore_id": restoreID,
	})
}

func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	clusterID := c.Param("id")
	backupID := c.Param("backupId")

	clusterUUID, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	backupUUID, err := utils.ParseUUID(backupID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid backup ID")
		return
	}

	err = h.backupService.DeleteBackup(clusterUUID.String(), backupUUID.String())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to delete backup: %v", err)
		return
	}

	user := getUsernameFromContext(c)
	if user == "" {
		user = "system"
	}

	if h.auditService != nil {
		h.auditService.LogBackupOperation(
			clusterUUID,
			constants.EventTypeDelete,
			backupUUID.String(),
			user,
			map[string]interface{}{
				"backup_id": backupUUID.String(),
			},
		)
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message": "Backup deleted successfully",
	})
}

func (h *BackupHandler) ListBackupSchedules(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid cluster ID")
		return
	}

	schedules, err := h.backupService.ListBackupSchedules(id.String())
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to list backup schedules: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, schedules)
}

func (h *BackupHandler) GetRestoreProgress(c *gin.Context) {
	restoreID := c.Param("restoreId")

	restoreUUID, err := utils.ParseUUID(restoreID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid restore ID")
		return
	}

	progress, err := h.restoreService.GetRestoreProgress(restoreUUID.String())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to get restore progress: %v", err)
		return
	}

	response := RestoreProgressResponse{
		RestoreID:     progress.RestoreID,
		Status:        progress.Status,
		Progress:      progress.Progress,
		CurrentStep:   progress.CurrentStep,
		StartTime:     progress.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		EstimatedTime: progress.EstimatedTime,
	}

	utils.Success(c, http.StatusOK, response)
}

func (h *BackupHandler) CreateBackupSchedule(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := utils.ParseUUID(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	var req CreateBackupScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	schedule, err := h.backupService.CreateBackupSchedule(
		id.String(),
		req.Name,
		req.CronExpr,
		req.BackupType,
		req.RetentionDays,
		req.Enabled,
		req.EtcdEndpoints,
		req.EtcdCaCert,
		req.EtcdCert,
		req.EtcdKey,
		req.EtcdDataDir,
		req.EtcdctlPath,
		req.SshUsername,
		req.SshPassword,
		req.EtcdDeploymentType,
		req.K8sDeploymentType,
		req.CreatedBy,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to create backup schedule: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, schedule)
}

func (h *BackupHandler) UpdateBackupSchedule(c *gin.Context) {
	scheduleID := c.Param("scheduleId")

	scheduleUUID, err := utils.ParseUUID(scheduleID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid schedule ID")
		return
	}

	var req UpdateBackupScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	err = h.backupService.UpdateBackupSchedule(
		scheduleUUID.String(),
		req.CronExpr,
		req.BackupType,
		req.RetentionDays,
		req.Enabled,
		req.EtcdEndpoints,
		req.EtcdCaCert,
		req.EtcdCert,
		req.EtcdKey,
		req.EtcdDataDir,
		req.EtcdctlPath,
		req.SshUsername,
		req.SshPassword,
		req.EtcdDeploymentType,
		req.K8sDeploymentType,
	)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to update backup schedule: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message": "Backup schedule updated successfully",
	})
}

func (h *BackupHandler) DeleteBackupSchedule(c *gin.Context) {
	scheduleID := c.Param("scheduleId")

	scheduleUUID, err := utils.ParseUUID(scheduleID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	err = h.backupService.DeleteBackupSchedule(scheduleUUID.String())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to delete backup schedule: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message": "Backup schedule deleted successfully",
	})
}

func getUsernameFromContext(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		if usernameStr, ok := username.(string); ok {
			return usernameStr
		}
	}
	return ""
}
