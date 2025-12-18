package repository

import (
	"fmt"

	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type ClusterRepository struct {
	db *gorm.DB
}

type ListClustersParams struct {
	Page          int
	Limit         int
	Offset        int
	Status        string
	LabelSelector string
	Search        string
}

func NewClusterRepository(db *gorm.DB) *ClusterRepository {
	return &ClusterRepository{db: db}
}

func (r *ClusterRepository) Create(cluster *model.Cluster) error {
	return r.db.Create(cluster).Error
}

func (r *ClusterRepository) GetByID(id string) (*model.Cluster, error) {
	var cluster model.Cluster
	err := r.db.Where("id = ?", id).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *ClusterRepository) GetByName(name string) (*model.Cluster, error) {
	var cluster model.Cluster
	err := r.db.Where("name = ?", name).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *ClusterRepository) Update(cluster *model.Cluster) error {
	return r.db.Save(cluster).Error
}

func (r *ClusterRepository) Delete(id string) error {
	cluster, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Unscoped().Delete(cluster).Error
}

func (r *ClusterRepository) List(params ListClustersParams) ([]*model.Cluster, int64, error) {
	var clusters []*model.Cluster
	var total int64

	query := r.db.Model(&model.Cluster{})

	if params.Search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?",
			fmt.Sprintf("%%%s%%", params.Search),
			fmt.Sprintf("%%%s%%", params.Search))
	}

	if params.LabelSelector != "" {
		query = query.Where("labels @> ?", params.LabelSelector)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(params.Offset).Limit(params.Limit).
		Order("created_at DESC").
		Find(&clusters).Error

	return clusters, total, err
}

func (r *ClusterRepository) FindActiveClusters() ([]*model.Cluster, error) {
	var clusters []*model.Cluster
	err := r.db.Find(&clusters).Error
	return clusters, err
}

func (r *ClusterRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Cluster{}).Count(&count).Error
	return count, err
}

func (r *ClusterRepository) ExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Cluster{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ClusterRepository) GetWithState(id string) (*model.ClusterWithState, error) {
	var cluster model.Cluster
	err := r.db.Where("id = ?", id).First(&cluster).Error
	if err != nil {
		return nil, err
	}

	// 单独获取 ClusterState
	var state model.ClusterState
	err = r.db.Where("cluster_id = ?", cluster.ID).First(&state).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	return &model.ClusterWithState{
		Cluster: &cluster,
		State:   &state,
	}, nil
}

func (r *ClusterRepository) FindAllWithState() ([]*model.ClusterWithState, error) {
	var clusters []*model.Cluster
	err := r.db.Find(&clusters).Error
	if err != nil {
		return nil, err
	}

	result := make([]*model.ClusterWithState, 0, len(clusters))
	for _, cluster := range clusters {
		result = append(result, &model.ClusterWithState{
			Cluster: cluster,
			State:   cluster.State,
		})
	}

	return result, nil
}
