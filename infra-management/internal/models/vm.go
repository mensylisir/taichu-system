package models

import "time"

type VM struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	OS        string    `json:"os"`
	Status    string    `json:"status"`
	CPU       string    `json:"cpu"`
	Memory    string    `json:"memory"`
	Storage   string    `json:"storage"`
	IP        string    `json:"ip"`
	Cluster   string    `json:"cluster"`
	CreatedAt time.Time `json:"createdAt"`
}

type VMRequest struct {
	Name    string `json:"name" binding:"required"`
	OS      string `json:"os" binding:"required"`
	Status  string `json:"status" binding:"required"`
	CPU     string `json:"cpu" binding:"required"`
	Memory  string `json:"memory" binding:"required"`
	Storage string `json:"storage" binding:"required"`
	IP      string `json:"ip" binding:"required"`
	Cluster string `json:"cluster" binding:"required"`
}

type VMResponse struct {
	Name         string `json:"name"`
	OS           string `json:"os"`
	Status       string `json:"status"`
	StatusClass  string `json:"statusClass"`
	CPU          string `json:"cpu"`
	Memory       string `json:"memory"`
	Storage      string `json:"storage"`
	IP           string `json:"ip"`
	Cluster      string `json:"cluster"`
	ClusterClass string `json:"clusterClass"`
	CreatedAt    string `json:"createdAt"`
}

func (v *VM) ToResponse() *VMResponse {
	statusClass := getStatusClass(v.Status)
	clusterClass := getClusterClass(v.Cluster)

	return &VMResponse{
		Name:         v.Name,
		OS:           v.OS,
		Status:       v.Status,
		StatusClass:  statusClass,
		CPU:          v.CPU,
		Memory:       v.Memory,
		Storage:      v.Storage,
		IP:           v.IP,
		Cluster:      v.Cluster,
		ClusterClass: clusterClass,
		CreatedAt:    v.CreatedAt.Format("2006-01-02"),
	}
}

func getStatusClass(status string) string {
	switch status {
	case "运行中":
		return "success"
	case "已停止":
		return "warning"
	case "故障":
		return "destructive"
	default:
		return "secondary"
	}
}

func getClusterClass(cluster string) string {
	switch cluster {
	case "生产集群":
		return "info"
	case "测试集群":
		return "secondary"
	default:
		return "secondary"
	}
}