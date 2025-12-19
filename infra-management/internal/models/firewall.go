package models

import "time"

type FirewallRule struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	Port      string    `json:"port"`
	SourceIP  string    `json:"sourceIp"`
	TargetIP  string    `json:"targetIp"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type FirewallRequest struct {
	Name      string `json:"name" binding:"required"`
	Protocol  string `json:"protocol" binding:"required"`
	Port      string `json:"port" binding:"required"`
	SourceIP  string `json:"sourceIp" binding:"required"`
	TargetIP  string `json:"targetIp" binding:"required"`
	Action    string `json:"action" binding:"required"`
	Status    string `json:"status" binding:"required"`
}

type FirewallResponse struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Port       string `json:"port"`
	SourceIP   string `json:"sourceIp"`
	TargetIP   string `json:"targetIp"`
	Action     string `json:"action"`
	ActionClass string `json:"actionClass"`
	Status     string `json:"status"`
	StatusClass string `json:"statusClass"`
	CreatedAt  string `json:"createdAt"`
}

func (f *FirewallRule) ToResponse() *FirewallResponse {
	actionClass := getActionClass(f.Action)
	statusClass := getFirewallStatusClass(f.Status)

	return &FirewallResponse{
		Name:       f.Name,
		Protocol:   f.Protocol,
		Port:       f.Port,
		SourceIP:   f.SourceIP,
		TargetIP:   f.TargetIP,
		Action:     f.Action,
		ActionClass: actionClass,
		Status:     f.Status,
		StatusClass: statusClass,
		CreatedAt:  f.CreatedAt.Format("2006-01-02"),
	}
}

func getActionClass(action string) string {
	switch action {
	case "允许":
		return "success"
	case "拒绝":
		return "destructive"
	default:
		return "secondary"
	}
}

func getFirewallStatusClass(status string) string {
	switch status {
	case "启用":
		return "success"
	case "禁用":
		return "warning"
	default:
		return "secondary"
	}
}