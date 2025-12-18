package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

// MachineHandler 机器处理器
type MachineHandler struct {
	machineService *service.MachineService
}

// NewMachineHandler 创建机器处理器
func NewMachineHandler(machineService *service.MachineService) *MachineHandler {
	return &MachineHandler{
		machineService: machineService,
	}
}

// CreateMachineRequest 创建机器请求
type CreateMachineRequest struct {
	Name             string            `json:"name" binding:"required"`
	IPAddress        string            `json:"ip_address" binding:"required"`
	InternalAddress  string            `json:"internal_address"`
	User             string            `json:"user" binding:"required"`
	Password         string            `json:"password" binding:"required"`
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
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	// 验证必填字段
	if req.Name == "" {
		utils.Error(c, http.StatusBadRequest, "Machine name is required")
		return
	}
	if req.IPAddress == "" {
		utils.Error(c, http.StatusBadRequest, "IP address is required")
		return
	}
	if req.User == "" {
		utils.Error(c, http.StatusBadRequest, "User is required")
		return
	}
	if req.Password == "" {
		utils.Error(c, http.StatusBadRequest, "Password is required")
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
		utils.Error(c, http.StatusInternalServerError, "Failed to create machine: %v", err)
		return
	}

	utils.Success(c, http.StatusCreated, machine)
}

// ListMachines 获取机器列表
func (h *MachineHandler) ListMachines(c *gin.Context) {
	// 解析查询参数
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	status := c.Query("status")
	role := c.Query("role")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		utils.Error(c, http.StatusBadRequest, "Invalid page parameter")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		utils.Error(c, http.StatusBadRequest, "Invalid limit parameter (1-100)")
		return
	}

	machines, total, err := h.machineService.ListMachines(page, limit, status, role)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to list machines: %v", err)
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

	id, err := uuid.Parse(machineID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid machine ID")
		return
	}

	machine, err := h.machineService.GetMachine(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Machine not found")
		return
	}

	utils.Success(c, http.StatusOK, machine)
}

// UpdateMachine 更新机器
func (h *MachineHandler) UpdateMachine(c *gin.Context) {
	machineID := c.Param("id")

	id, err := uuid.Parse(machineID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid machine ID")
		return
	}

	var req CreateMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	machine, err := h.machineService.GetMachine(id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Machine not found")
		return
	}

	// 更新字段
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
		utils.Error(c, http.StatusInternalServerError, "Failed to update machine: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, machine)
}

// DeleteMachine 删除机器
func (h *MachineHandler) DeleteMachine(c *gin.Context) {
	machineID := c.Param("id")

	id, err := uuid.Parse(machineID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid machine ID")
		return
	}

	if err := h.machineService.DeleteMachine(id); err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to delete machine: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "Machine deleted successfully"})
}

// UpdateMachineStatus 更新机器状态
func (h *MachineHandler) UpdateMachineStatus(c *gin.Context) {
	machineID := c.Param("id")

	id, err := uuid.Parse(machineID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid machine ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=available in-use deploying maintenance offline"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	if err := h.machineService.UpdateMachineStatus(id, req.Status); err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to update machine status: %v", err)
		return
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
