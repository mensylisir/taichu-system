package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type EventHandler struct {
	eventService *service.EventService
}

type EventSummary struct {
	ID          uuid.UUID `json:"id"`
	EventType   string    `json:"event_type"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	Component   string    `json:"component"`
	Count       int       `json:"count"`
	FirstSeen   string    `json:"first_seen"`
	LastSeen    string    `json:"last_seen"`
	CreatedAt   string    `json:"created_at"`
}

type EventListResponse struct {
	Events  []EventSummary               `json:"events"`
	Total   int64                        `json:"total"`
	Summary service.EventSummaryStats `json:"summary"`
}

func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
	}
}

func (h *EventHandler) ListEvents(c *gin.Context) {
	clusterID := c.Param("id")

	id, err := uuid.Parse(clusterID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid cluster ID")
		return
	}

	severity := c.Query("severity")
	since := c.Query("since")

	// 解析since时间
	var sinceTime *time.Time
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			sinceTime = &t
		}
	}

	events, total, summary, err := h.eventService.ListEvents(id.String(), severity, sinceTime)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list events: %v", err)
		return
	}

	eventSummaries := make([]EventSummary, 0, len(events))
	for _, event := range events {
		eventSummaries = append(eventSummaries, EventSummary{
			ID:        event.ID,
			EventType: event.EventType,
			Message:   event.Message,
			Severity:  event.Severity,
			Component: event.Component,
			Count:     event.Count,
			FirstSeen: event.FirstTimestamp.Format("2006-01-02T15:04:05Z07:00"),
			LastSeen:  event.LastTimestamp.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt: event.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response := EventListResponse{
		Events:  eventSummaries,
		Total:   total,
		Summary: *summary,
	}

	utils.Success(c, http.StatusOK, response)
}
