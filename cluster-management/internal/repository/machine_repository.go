package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// MachineRepository 机器数据访问层
type MachineRepository struct {
	db *gorm.DB
}

// NewMachineRepository 创建机器仓库
func NewMachineRepository(db *gorm.DB) *MachineRepository {
	return &MachineRepository{db: db}
}

// Create 创建机器
func (r *MachineRepository) Create(machine *model.Machine) error {
	return r.db.Create(machine).Error
}

// GetByID 根据ID获取机器
func (r *MachineRepository) GetByID(id uuid.UUID) (*model.Machine, error) {
	var machine model.Machine
	if err := r.db.First(&machine, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &machine, nil
}

// GetByIDs 根据ID列表批量获取机器
func (r *MachineRepository) GetByIDs(ids []uuid.UUID) ([]*model.Machine, error) {
	var machines []*model.Machine
	if err := r.db.Where("id IN ?", ids).Find(&machines).Error; err != nil {
		return nil, err
	}
	return machines, nil
}

// List 获取机器列表
func (r *MachineRepository) List(page, limit int, status, role string) ([]*model.Machine, int64, error) {
	var machines []*model.Machine
	var total int64

	query := r.db.Model(&model.Machine{})

	// 状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 角色过滤
	if role != "" {
		query = query.Where("role = ?", role)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&machines).Error; err != nil {
		return nil, 0, err
	}

	return machines, total, nil
}

// Update 更新机器
func (r *MachineRepository) Update(machine *model.Machine) error {
	return r.db.Save(machine).Error
}

// UpdateStatus 更新机器状态
func (r *MachineRepository) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Model(&model.Machine{}).Where("id = ?", id).Update("status", status).Error
}

// Delete 删除机器
func (r *MachineRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Machine{}, "id = ?", id).Error
}

// GetAvailableMachines 获取可用的机器
func (r *MachineRepository) GetAvailableMachines() ([]*model.Machine, error) {
	var machines []*model.Machine
	if err := r.db.Where("status = ?", "available").Find(&machines).Error; err != nil {
		return nil, err
	}
	return machines, nil
}

// GetByRole 根据角色获取机器
func (r *MachineRepository) GetByRole(role string) ([]*model.Machine, error) {
	var machines []*model.Machine
	if err := r.db.Where("role = ?", role).Find(&machines).Error; err != nil {
		return nil, err
	}
	return machines, nil
}
