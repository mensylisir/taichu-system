package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MachineService 机器服务
type MachineService struct {
	machineRepo *repository.MachineRepository
}

// NewMachineService 创建机器服务
func NewMachineService(machineRepo *repository.MachineRepository) *MachineService {
	return &MachineService{
		machineRepo: machineRepo,
	}
}

// CreateMachine 创建机器
func (s *MachineService) CreateMachine(machine *model.Machine) error {
	err := s.machineRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		var existing model.Machine
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", machine.ID).First(&existing).Error; err == nil {
			return errors.New("machine with this ID already exists")
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		var existingIP model.Machine
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("ip_address = ?", machine.IPAddress).First(&existingIP).Error; err == nil {
			return errors.New("machine with this IP already exists")
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		return tx.Create(machine).Error
	})

	return err
}

// GetMachine 获取机器
func (s *MachineService) GetMachine(id uuid.UUID) (*model.Machine, error) {
	return s.machineRepo.GetByID(id)
}

// ListMachines 获取机器列表
func (s *MachineService) ListMachines(page, limit int, status, role string) ([]*model.Machine, int64, error) {
	return s.machineRepo.List(page, limit, status, role)
}

// UpdateMachine 更新机器
func (s *MachineService) UpdateMachine(machine *model.Machine) error {
	return s.machineRepo.Update(machine)
}

// UpdateMachineStatus 更新机器状态
func (s *MachineService) UpdateMachineStatus(id uuid.UUID, status string) error {
	return s.machineRepo.UpdateStatus(id, status)
}

// DeleteMachine 删除机器
func (s *MachineService) DeleteMachine(id uuid.UUID) error {
	return s.machineRepo.Delete(id)
}

// GetMachinesByIDs 根据ID列表获取机器
func (s *MachineService) GetMachinesByIDs(ids []uuid.UUID) ([]*model.Machine, error) {
	return s.machineRepo.GetByIDs(ids)
}

// GetAvailableMachines 获取可用机器
func (s *MachineService) GetAvailableMachines() ([]*model.Machine, error) {
	return s.machineRepo.GetAvailableMachines()
}

// GetMachinesByRole 根据角色获取机器
func (s *MachineService) GetMachinesByRole(role string) ([]*model.Machine, error) {
	return s.machineRepo.GetByRole(role)
}

// ValidateClusterNodes 验证集群节点配置
func (s *MachineService) ValidateClusterNodes(machineIDs []uuid.UUID) error {
	machines, err := s.machineRepo.GetByIDs(machineIDs)
	if err != nil {
		return err
	}

	if len(machines) == 0 {
		return errors.New("no machines found")
	}

	// 检查是否有master节点
	hasMaster := false
	for _, machine := range machines {
		if machine.Role == "master" {
			hasMaster = true
			break
		}
	}

	if !hasMaster {
		return errors.New("at least one master node is required")
	}

	// 检查机器状态
	for _, machine := range machines {
		if machine.Status != "available" {
			return fmt.Errorf("machine %s is not available (status: %s)", machine.Name, machine.Status)
		}
	}

	return nil
}
