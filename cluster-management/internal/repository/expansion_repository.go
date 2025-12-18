package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type ExpansionRepository struct {
	db *gorm.DB
}

func NewExpansionRepository(db *gorm.DB) *ExpansionRepository {
	return &ExpansionRepository{db: db}
}

func (r *ExpansionRepository) Create(expansion *model.ClusterExpansion) error {
	return r.db.Create(expansion).Error
}

func (r *ExpansionRepository) GetByID(id string) (*model.ClusterExpansion, error) {
	var expansion model.ClusterExpansion
	if err := r.db.First(&expansion, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &expansion, nil
}

func (r *ExpansionRepository) GetByClusterID(clusterID uuid.UUID) ([]*model.ClusterExpansion, error) {
	var expansions []*model.ClusterExpansion
	if err := r.db.Where("cluster_id = ?", clusterID).Order("created_at DESC").Find(&expansions).Error; err != nil {
		return nil, err
	}
	return expansions, nil
}

func (r *ExpansionRepository) Update(expansion *model.ClusterExpansion) error {
	return r.db.Save(expansion).Error
}

func (r *ExpansionRepository) DeleteByClusterID(clusterID uuid.UUID) error {
	return r.db.Where("cluster_id = ?", clusterID).Delete(&model.ClusterExpansion{}).Error
}
