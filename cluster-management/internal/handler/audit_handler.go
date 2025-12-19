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
			User:       event.User,
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

func convertJSONMap(j map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range j {
		result[k] = v
	}
	return result
}
