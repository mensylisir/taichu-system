package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JSONMap map[string]interface{}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

type Cluster struct {
	ID                  uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name                string    `json:"name" gorm:"uniqueIndex;not null;size:255"`
	Description         string    `json:"description" gorm:"type:text"`
	KubeconfigEncrypted string    `json:"-" gorm:"column:kubeconfig_encrypted;not null"`
	KubeconfigNonce     string    `json:"-" gorm:"column:kubeconfig_nonce;not null;size:32"`
	Version             string    `json:"version" gorm:"size:50;default:'1.0.0'"`
	Labels              JSONMap   `json:"labels" gorm:"type:jsonb;default:'{}'"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt           *time.Time `json:"deleted_at" gorm:"index"`
	CreatedBy           string    `json:"created_by" gorm:"size:100;default:'system'"`
	UpdatedBy           string    `json:"updated_by" gorm:"size:100;default:'system'"`

	State *ClusterState `json:"state,omitempty" gorm:"foreignKey:ClusterID"`
}

func (Cluster) TableName() string {
	return "clusters"
}

type ClusterState struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClusterID        uuid.UUID `json:"cluster_id" gorm:"uniqueIndex;not null;index"`
	Status           string    `json:"status" gorm:"size:20;default:'unknown'"`
	NodeCount        int       `json:"node_count" gorm:"default:0"`
	TotalCPUCores    int       `json:"total_cpu_cores" gorm:"default:0"`
	TotalMemoryBytes int64     `json:"total_memory_bytes" gorm:"default:0"`
	KubernetesVersion string   `json:"kubernetes_version" gorm:"size:50"`
	APIServerURL     string    `json:"api_server_url" gorm:"size:255"`
	LastHeartbeatAt  *time.Time `json:"last_heartbeat_at"`
	LastSyncAt       time.Time `json:"last_sync_at" gorm:"autoUpdateTime"`
	SyncError        string    `json:"sync_error" gorm:"type:text"`
	SyncSuccess      bool      `json:"sync_success" gorm:"default:false"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ClusterState) TableName() string {
	return "cluster_states"
}

type ClusterWithState struct {
	*Cluster
	State *ClusterState `json:"state"`
}

func (c *Cluster) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (c *Cluster) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Cluster) IsDeleted() bool {
	return c.DeletedAt != nil
}

func (c *Cluster) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	c.DeletedAt = &now
	return tx.Save(c).Error
}

func (c *Cluster) Restore(tx *gorm.DB) error {
	c.DeletedAt = nil
	return tx.Save(c).Error
}

func (j *JSONMap) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

func (j JSONMap) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}

func (j JSONMap) Get(key string) (interface{}, bool) {
	val, ok := j[key]
	return val, ok
}

func (j JSONMap) GetString(key string) (string, bool) {
	if val, ok := j[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

func (j JSONMap) Set(key string, value interface{}) {
	if j == nil {
		*j = make(JSONMap)
	}
	(*j)[key] = value
}

func (j JSONMap) Delete(key string) {
	if j == nil {
		return
	}
	delete(*j, key)
}
