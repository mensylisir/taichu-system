package repository

import (

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type NodeRepository struct {
	db *gorm.DB
}

func NewNodeRepository(db *gorm.DB) *NodeRepository {
	return &NodeRepository{db: db}
}

func (r *NodeRepository) Create(node *model.Node) error {
	return r.db.Create(node).Error
}

func (r *NodeRepository) GetByID(id uuid.UUID) (*model.Node, error) {
	var node model.Node
	if err := r.db.First(&node, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *NodeRepository) GetByName(clusterID, name string) (*model.Node, error) {
	var node model.Node
	if err := r.db.Where("cluster_id = ? AND name = ?", clusterID, name).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *NodeRepository) List(clusterID, nodeType, status string, page, limit int) ([]*model.Node, int64, error) {
	var nodes []*model.Node
	var total int64

	query := r.db.Model(&model.Node{})

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if nodeType != "" {
		query = query.Where("type = ?", nodeType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("name").Find(&nodes).Error; err != nil {
		return nil, 0, err
	}

	return nodes, total, nil
}

func (r *NodeRepository) Update(node *model.Node) error {
	return r.db.Save(node).Error
}

func (r *NodeRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Node{}, "id = ?", id).Error
}

func (r *NodeRepository) Upsert(nodes []model.Node) error {
	for i := range nodes {
		node := &nodes[i]
		if err := r.db.Where("cluster_id = ? AND name = ?", node.ClusterID, node.Name).
			Assign(node).
			FirstOrCreate(node).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *NodeRepository) UpsertSingle(node *model.Node) error {
	// 检查节点是否已存在
	var existingNode model.Node
	err := r.db.Where("cluster_id = ? AND name = ?", node.ClusterID, node.Name).First(&existingNode).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 如果节点存在，更新它
	if err == nil {
		// 检查现有节点的ID是否为空（全零UUID）
		if existingNode.ID == uuid.Nil {
			// 如果现有节点的ID为空，生成一个新的UUID
			existingNode.ID = uuid.New()
			// 更新数据库中的ID
			if err := r.db.Model(&existingNode).Update("id", existingNode.ID).Error; err != nil {
				return err
			}
		}
		// 保留原有的ID
		node.ID = existingNode.ID
		return r.db.Model(&existingNode).Updates(node).Error
	}

	// 如果节点不存在，创建它
	return r.db.Create(node).Error
}

func (r *NodeRepository) GetByClusterID(clusterID string) ([]model.Node, error) {
	var nodes []model.Node
	err := r.db.Where("cluster_id = ?", clusterID).Find(&nodes).Error
	return nodes, err
}

func (r *NodeRepository) DeleteByClusterID(clusterID string) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.Node{}).Error
}
