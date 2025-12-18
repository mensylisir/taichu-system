package model

import (
	"time"

	"github.com/google/uuid"
)

// CreateTask 集群创建任务模型
// 用于跟踪通过 kk 创建集群的异步任务
type CreateTask struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterName     string    `json:"cluster_name" gorm:"size:255;not null;index"`
	ClusterID       *uuid.UUID `json:"cluster_id" gorm:"index"` // 成功后关联的集群ID
	MachineIDs      JSONMap   `json:"machine_ids" gorm:"type:jsonb;default:'[]'"` // 使用的机器ID列表
	ConfigYaml      string    `json:"config_yaml" gorm:"type:text"` // 生成的配置文件内容
	Status          string    `json:"status" gorm:"size:50;default:'pending'"` // pending/running/success/failed
	Progress        int       `json:"progress" gorm:"default:0"` // 进度百分比 0-100
	CurrentStep     string    `json:"current_step" gorm:"size:255"` // 当前执行步骤
	Logs            string    `json:"logs" gorm:"type:text"` // 安装日志
	ErrorMsg        string    `json:"error_msg" gorm:"type:text"`
	ArtifactPath    string    `json:"artifact_path" gorm:"type:text"`
	WithPackages    bool      `json:"with_packages" gorm:"default:false"`
	AutoApprove     bool      `json:"auto_approve" gorm:"default:false"`
	KubernetesVersion string  `json:"kubernetes_version" gorm:"size:50"`
	NetworkPlugin   string    `json:"network_plugin" gorm:"size:50"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 返回表名
func (CreateTask) TableName() string {
	return "create_tasks"
}
