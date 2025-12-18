package models

import "time"

type Storage struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Capacity  string    `json:"capacity"`
	IOPS      string    `json:"iops"`
	Status    string    `json:"status"`
	MountedTo *string   `json:"mountedTo"`
	CreatedAt time.Time `json:"createdAt"`
}

type StorageRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Capacity string `json:"capacity" binding:"required"`
	IOPS     string `json:"iops" binding:"required"`
	Status   string `json:"status" binding:"required"`
	MountedTo string `json:"mountedTo"`
}

type StorageResponse struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	TypeClass  string `json:"typeClass"`
	Capacity   string `json:"capacity"`
	IOPS       string `json:"iops"`
	Status     string `json:"status"`
	StatusClass string `json:"statusClass"`
	MountedTo  string `json:"mountedTo"`
	CreatedAt  string `json:"createdAt"`
}

func (s *Storage) ToResponse() *StorageResponse {
	typeClass := getStorageTypeClass(s.Type)
	statusClass := getStorageStatusClass(s.Status)

	mountedTo := ""
	if s.MountedTo != nil {
		mountedTo = *s.MountedTo
	}

	return &StorageResponse{
		Name:       s.Name,
		Type:       s.Type,
		TypeClass:  typeClass,
		Capacity:   s.Capacity,
		IOPS:       s.IOPS,
		Status:     s.Status,
		StatusClass: statusClass,
		MountedTo:  mountedTo,
		CreatedAt:  s.CreatedAt.Format("2006-01-02"),
	}
}

func getStorageTypeClass(storageType string) string {
	switch storageType {
	case "SSD":
		return "warning"
	case "HDD":
		return "secondary"
	default:
		return "secondary"
	}
}

func getStorageStatusClass(status string) string {
	switch status {
	case "已挂载":
		return "success"
	case "未挂载":
		return "warning"
	case "故障":
		return "destructive"
	default:
		return "secondary"
	}
}
