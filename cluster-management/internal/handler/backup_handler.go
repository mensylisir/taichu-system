package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type BackupHandler struct {
	backupService *service.BackupService
}

type CreateBackupRequest struct {
	BackupName    string `json:"backup_name" binding:"required"`
	BackupType    string `json:"backup_type" binding:"required"`
	RetentionDays int    `json:"retention_days"`
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
	RestoreName string `json:"restore_name"`
}

type RestoreProgressResponse struct {
	RestoreID     string  `json:"restore_id"`
	Status        string  `json:"status"`
	Progress      float64 `json:"progress"`
	CurrentStep   string  `json:"current_step"`
	StartTime     string  `json:"start_time"`
	EstimatedTime int     `json:"estimated_time,omitempty"`
}

func NewBackupHandler(backupService *service.BackupService) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
	}
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	backup, err := h.backupService.CreateBackup(id.String(), req.BackupName, req.BackupType, req.RetentionDays)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to create backup: %v", err)
		return
	}

	go h.backupService.ExecuteBackup(backup.ID.String())

	utils.Success(c, http.StatusCreated, backup)
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
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

	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	backupUUID, err := uuid.Parse(backupID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid backup ID")
		return
	}

	backup, err := h.backupService.GetBackup(clusterUUID.String(), backupUUID.String())
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Backup not found")
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

	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	backupUUID, err := uuid.Parse(backupID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid backup ID")
		return
	}

	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	restoreID, err := h.backupService.RestoreBackup(clusterUUID.String(), backupUUID.String(), req.RestoreName)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to restore backup: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message":    "Backup restoration started",
		"restore_id": restoreID,
	})
}

func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	clusterID := c.Param("id")
	backupID := c.Param("backupId")

	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	backupUUID, err := uuid.Parse(backupID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid backup ID")
		return
	}

	err = h.backupService.DeleteBackup(clusterUUID.String(), backupUUID.String())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to delete backup: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"message": "Backup deleted successfully",
	})
}

func (h *BackupHandler) ListBackupSchedules(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	schedules, err := h.backupService.ListBackupSchedules(id.String())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list backup schedules: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, schedules)
}

func (h *BackupHandler) GetRestoreProgress(c *gin.Context) {
	restoreID := c.Param("restoreId")

	restoreUUID, err := uuid.Parse(restoreID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid restore ID")
		return
	}

	progress, err := h.backupService.GetRestoreProgress(restoreUUID.String())
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
