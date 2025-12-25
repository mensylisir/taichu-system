package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// MachineHandler 机器处理器
type MachineHandler struct {
	machineService *service.MachineService
	auditService   *service.AuditService
}

// NewMachineHandler 创建机器处理器
func NewMachineHandler(machineService *service.MachineService, auditService *service.AuditService) *MachineHandler {
	return &MachineHandler{
		machineService: machineService,
		auditService:   auditService,
	}
}

// CreateMachineRequest 创建机器请求
type CreateMachineRequest struct {
	Name             string            `json:"name" binding:"required,min=1,max=63"`
	IPAddress        string            `json:"ip_address" binding:"required,ip"`
	InternalAddress  string            `json:"internal_address" binding:"omitempty,ip"`
	User             string            `json:"user" binding:"required,min=1,max=50"`
	Password         string            `json:"password" binding:"required,min=6"`
	Role             string            `json:"role" binding:"required,oneof=master worker etcd registry"`
	ArtifactPath     string            `json:"artifact_path"`
	ImageRepo        string            `json:"image_repo"`
	RegistryAddress  string            `json:"registry_address"`
	Labels           map[string]string `json:"labels"`
}

// CreateMachine 创建机器
func (h *MachineHandler) CreateMachine(c *gin.Context) {
	var req CreateMachineRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	if req.Name == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "Machine name is required")
		return
	}
	if req.IPAddress == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "IP address is required")
		return
	}
	if req.User == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "User is required")
		return
	}
	if req.Password == "" {
		utils.Error(c, utils.ErrCodeValidationFailed, "Password is required")
		return
	}

	machine := &model.Machine{
		Name:             req.Name,
		IPAddress:        req.IPAddress,
		InternalAddress:  req.InternalAddress,
		User:             req.User,
		Password:         req.Password,
		Role:             req.Role,
		ArtifactPath:     req.ArtifactPath,
		ImageRepo:        req.ImageRepo,
		RegistryAddress:  req.RegistryAddress,
		Labels:           convertMachineLabelsToJSONMap(req.Labels),
	}

	if err := h.machineService.CreateMachine(machine); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to create machine: %v", err)
		return
	}

	if h.auditService != nil {
		user := "api-user"
		h.auditService.CreateAuditEvent(
			uuid.Nil,
			"machine",
			constants.EventTypeCreate,
			"machine",
			machine.ID.String(),
			user,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			nil,
			map[string]interface{}{
				"name":             machine.Name,
				"ip_address":       machine.IPAddress,
				"internal_address": machine.InternalAddress,
				"user":             machine.User,
				"role":             machine.Role,
			},
			map[string]interface{}{
				"operation": "create_machine",
			},
			constants.StatusSuccess,
		)
	}

	utils.Success(c, http.StatusCreated, machine)
}

// ListMachines 获取机器列表
func (h *MachineHandler) ListMachines(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	status := c.Query("status")
	role := c.Query("role")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid page parameter")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid limit parameter (1-100)")
		return
	}

	machines, total, err := h.machineService.ListMachines(page, limit, status, role)
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to list machines: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"machines": machines,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetMachine 获取机器详情
func (h *MachineHandler) GetMachine(c *gin.Context) {
	machineID := c.Param("id")

	id, err := utils.ParseUUID(machineID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid machine ID")
		return
	}

	machine, err := h.machineService.GetMachine(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Machine not found")
		return
	}

	utils.Success(c, http.StatusOK, machine)
}

// UpdateMachine 更新机器
func (h *MachineHandler) UpdateMachine(c *gin.Context) {
	machineID := c.Param("id")

	id, err := utils.ParseUUID(machineID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid machine ID")
		return
	}

	var req CreateMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	machine, err := h.machineService.GetMachine(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Machine not found")
		return
	}

	oldMachine := *machine
	machine.Name = req.Name
	machine.IPAddress = req.IPAddress
	machine.InternalAddress = req.InternalAddress
	machine.User = req.User
	if req.Password != "" {
		machine.Password = req.Password
	}
	machine.Role = req.Role
	machine.ArtifactPath = req.ArtifactPath
	machine.ImageRepo = req.ImageRepo
	machine.RegistryAddress = req.RegistryAddress
	machine.Labels = convertMachineLabelsToJSONMap(req.Labels)

	if err := h.machineService.UpdateMachine(machine); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to update machine: %v", err)
		return
	}

	if h.auditService != nil {
		user := "api-user"
		h.auditService.CreateAuditEvent(
			uuid.Nil,
			"machine",
			constants.EventTypeUpdate,
			"machine",
			machine.ID.String(),
			user,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			map[string]interface{}{
				"name":             oldMachine.Name,
				"ip_address":       oldMachine.IPAddress,
				"internal_address": oldMachine.InternalAddress,
				"user":             oldMachine.User,
				"role":             oldMachine.Role,
			},
			map[string]interface{}{
				"name":             machine.Name,
				"ip_address":       machine.IPAddress,
				"internal_address": machine.InternalAddress,
				"user":             machine.User,
				"role":             machine.Role,
			},
			map[string]interface{}{
				"operation": "update_machine",
			},
			constants.StatusSuccess,
		)
	}

	utils.Success(c, http.StatusOK, machine)
}

// DeleteMachine 删除机器
func (h *MachineHandler) DeleteMachine(c *gin.Context) {
	machineID := c.Param("id")

	id, err := utils.ParseUUID(machineID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid machine ID")
		return
	}

	machine, err := h.machineService.GetMachine(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Machine not found")
		return
	}

	if err := h.machineService.DeleteMachine(id); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to delete machine: %v", err)
		return
	}

	if h.auditService != nil {
		user := "api-user"
		h.auditService.CreateAuditEvent(
			uuid.Nil,
			"machine",
			constants.EventTypeDelete,
			"machine",
			id.String(),
			user,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			map[string]interface{}{
				"name":             machine.Name,
				"ip_address":       machine.IPAddress,
				"internal_address": machine.InternalAddress,
				"user":             machine.User,
				"role":             machine.Role,
			},
			nil,
			map[string]interface{}{
				"operation": "delete_machine",
			},
			constants.StatusSuccess,
		)
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "Machine deleted successfully"})
}

// UpdateMachineStatus 更新机器状态
func (h *MachineHandler) UpdateMachineStatus(c *gin.Context) {
	machineID := c.Param("id")

	id, err := utils.ParseUUID(machineID)
	if err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid machine ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=available in-use deploying maintenance offline"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	machine, err := h.machineService.GetMachine(id)
	if err != nil {
		utils.Error(c, utils.ErrCodeNotFound, "Machine not found")
		return
	}

	oldStatus := machine.Status

	if err := h.machineService.UpdateMachineStatus(id, req.Status); err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to update machine status: %v", err)
		return
	}

	if h.auditService != nil {
		user := "api-user"
		h.auditService.CreateAuditEvent(
			uuid.Nil,
			"machine",
			"update_status",
			"machine",
			id.String(),
			user,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			map[string]interface{}{
				"status": oldStatus,
			},
			map[string]interface{}{
				"status": req.Status,
			},
			map[string]interface{}{
				"operation": "update_machine_status",
			},
			constants.StatusSuccess,
		)
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "Machine status updated successfully"})
}

// convertMachineLabelsToJSONMap 转换标签为JSONMap
func convertMachineLabelsToJSONMap(labels map[string]string) model.JSONMap {
	jsonMap := make(model.JSONMap)
	for k, v := range labels {
		jsonMap[k] = v
	}
	return jsonMap
}
