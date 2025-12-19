package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type AuditHandler struct {
	auditService *service.AuditService
}

type AuditEventResponse struct {
	ID         uuid.UUID `json:"id"`
	EventType  string    `json:"event_type"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	User       string    `json:"user"`
	IPAddress  string    `json:"ip_address"`
	OldValue   map[string]interface{} `json:"old_value,omitempty"`
	NewValue   map[string]interface{} `json:"new_value,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Result     string    `json:"result"`
	Timestamp  string    `json:"timestamp"`
}

type AuditListResponse struct {
	Events []*AuditEventResponse `json:"events"`
	Total  int64                 `json:"total"`
	Page   int                   `json:"page"`
	Limit  int                   `json:"limit"`
}

func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

func (h *AuditHandler) ListAuditEvents(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	eventType := c.Query("event_type")
	action := c.Query("action")
	user := c.Query("user")
	result := c.Query("result")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	params := repository.AuditListParams{
		EventType: eventType,
		Action:    action,
		User:      user,
		Result:    result,
		Limit:     limit,
		Offset:    offset,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			params.StartTime = startTime
		}
	}

	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			params.EndTime = endTime
		}
	}

	events, total, err := h.auditService.GetAuditLogs(id, params)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to get audit events: %v", err)
		return
	}

	responses := make([]*AuditEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, &AuditEventResponse{
			ID:         event.ID,
			EventType:  event.EventType,
			Action:     event.Action,
			Resource:   event.Resource,
			ResourceID: event.ResourceID,
			User:       event.Username,
			IPAddress:  event.IPAddress,
			OldValue:   convertJSONMap(event.OldValue),
			NewValue:   convertJSONMap(event.NewValue),
			Details:    convertJSONMap(event.Details),
			Result:     event.Result,
			Timestamp:  event.Timestamp.Format(time.RFC3339),
		})
	}

	response := AuditListResponse{
		Events: responses,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}

	utils.Success(c, http.StatusOK, response)
}

// ListAllAuditEvents 获取所有审计日志（全局）
func (h *AuditHandler) ListAllAuditEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	eventType := c.Query("event_type")
	action := c.Query("action")
	user := c.Query("user")
	result := c.Query("result")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	params := repository.AuditListParams{
		EventType: eventType,
		Action:    action,
		User:      user,
		Result:    result,
		Limit:     limit,
		Offset:    offset,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			params.StartTime = startTime
		}
	}

	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			params.EndTime = endTime
		}
	}

	// 使用 uuid.Nil 表示查询所有集群的审计日志
	events, total, err := h.auditService.GetAuditLogs(uuid.Nil, params)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to get audit events: %v", err)
		return
	}

	responses := make([]*AuditEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, &AuditEventResponse{
			ID:         event.ID,
			EventType:  event.EventType,
			Action:     event.Action,
			Resource:   event.Resource,
			ResourceID: event.ResourceID,
			User:       event.Username,
			IPAddress:  event.IPAddress,
			OldValue:   convertJSONMap(event.OldValue),
			NewValue:   convertJSONMap(event.NewValue),
			Details:    convertJSONMap(event.Details),
			Result:     event.Result,
			Timestamp:  event.Timestamp.Format(time.RFC3339),
		})
	}

	response := AuditListResponse{
		Events: responses,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}

	utils.Success(c, http.StatusOK, response)
}

// CreateAuditEventRequest 创建审计事件请求
type CreateAuditEventRequest struct {
	ClusterID  string                 `json:"cluster_id"`
	EventType  string                 `json:"event_type" binding:"required"`
	Action     string                 `json:"action" binding:"required"`
	Resource   string                 `json:"resource" binding:"required"`
	ResourceID string                 `json:"resource_id"`
	Username   string                 `json:"username"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	OldValue   map[string]interface{} `json:"old_value"`
	NewValue   map[string]interface{} `json:"new_value"`
	Details    map[string]interface{} `json:"details"`
	Result     string                 `json:"result"`
	ErrorMsg   string                 `json:"error_msg"`
	Timestamp  string                 `json:"timestamp"`
}

// CreateAuditEvent 创建审计事件
func (h *AuditHandler) CreateAuditEvent(c *gin.Context) {
	var req CreateAuditEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	var clusterID uuid.UUID
	if req.ClusterID != "" {
		var err error
		clusterID, err = uuid.Parse(req.ClusterID)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
			return
		}
	}

	// 解析时间戳
	var timestamp time.Time
	if req.Timestamp != "" {
		var err error
		timestamp, err = time.Parse("2006-01-02 15:04:05.000 -07:00", req.Timestamp)
		if err != nil {
			timestamp, err = time.Parse(time.RFC3339, req.Timestamp)
			if err != nil {
				timestamp = time.Now()
			}
		}
	} else {
		timestamp = time.Now()
	}

	result := req.Result
	if result == "" {
		result = "success"
	}

	err := h.auditService.CreateAuditEventWithTimestamp(
		clusterID,
		req.EventType,
		req.Action,
		req.Resource,
		req.ResourceID,
		req.Username,
		req.IPAddress,
		req.UserAgent,
		req.OldValue,
		req.NewValue,
		req.Details,
		result,
		timestamp,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to create audit event: %v", err)
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{"message": "Audit event created successfully"})
}

func convertJSONMap(j map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range j {
		result[k] = v
	}
	return result
}
