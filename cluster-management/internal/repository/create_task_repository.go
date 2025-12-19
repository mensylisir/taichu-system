package repository

import (
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

// CreateTaskRepository 创建任务数据访问层
type CreateTaskRepository struct {
	db *gorm.DB
}

// NewCreateTaskRepository 创建任务仓库
func NewCreateTaskRepository(db *gorm.DB) *CreateTaskRepository {
	return &CreateTaskRepository{db: db}
}

// Create 创建任务
func (r *CreateTaskRepository) Create(task *model.CreateTask) error {
	return r.db.Create(task).Error
}

// GetByID 根据ID获取任务
func (r *CreateTaskRepository) GetByID(id uuid.UUID) (*model.CreateTask, error) {
	var task model.CreateTask
	if err := r.db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// Update 更新任务
func (r *CreateTaskRepository) Update(task *model.CreateTask) error {
	return r.db.Save(task).Error
}

// UpdateStatus 更新任务状态
func (r *CreateTaskRepository) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Model(&model.CreateTask{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateProgress 更新任务进度
func (r *CreateTaskRepository) UpdateProgress(id uuid.UUID, progress int, currentStep string) error {
	return r.db.Model(&model.CreateTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"progress":      progress,
			"current_step":  currentStep,
		}).Error
}

// AppendLogs 追加日志
func (r *CreateTaskRepository) AppendLogs(id uuid.UUID, log string) error {
	return r.db.Model(&model.CreateTask{}).
		Where("id = ?", id).
		Update("logs", gorm.Expr("logs || ?", log)).Error
}

// List 获取任务列表
func (r *CreateTaskRepository) List(page, limit int) ([]*model.CreateTask, int64, error) {
	var tasks []*model.CreateTask
	var total int64

	query := r.db.Model(&model.CreateTask{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetByStatus 根据状态获取任务
func (r *CreateTaskRepository) GetByStatus(status string) ([]*model.CreateTask, error) {
	var tasks []*model.CreateTask
	if err := r.db.Where("status = ?", status).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetRunningTasks 获取运行中的任务
func (r *CreateTaskRepository) GetRunningTasks() ([]*model.CreateTask, error) {
	return r.GetByStatus("running")
}
