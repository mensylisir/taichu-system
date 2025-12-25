package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/middleware"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type AuthHandler struct {
	authService  *service.AuthService
	auditService *service.AuditService
}

func NewAuthHandler(authService *service.AuthService, auditService *service.AuditService) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		auditService: auditService,
	}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	authResp, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		if h.auditService != nil {
			h.auditService.CreateAuditEvent(
				uuid.Nil,
				"auth",
				"login",
				"user",
				req.Username,
				req.Username,
				c.ClientIP(),
				c.GetHeader("User-Agent"),
				nil,
				nil,
				map[string]interface{}{
					"username": req.Username,
				},
				constants.StatusFailed,
			)
		}
		utils.Error(c, utils.ErrCodeUnauthorized, "Login failed: %v", err)
		return
	}

	if h.auditService != nil {
		h.auditService.CreateAuditEvent(
			uuid.Nil,
			"auth",
			"login",
			"user",
			req.Username,
			req.Username,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			nil,
			nil,
			map[string]interface{}{
				"username": req.Username,
			},
			constants.StatusSuccess,
		)
	}

	utils.Success(c, 200, authResp)
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	user, err := h.authService.Register(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		utils.Error(c, utils.ErrCodeAlreadyExists, "Registration failed: %v", err)
		return
	}

	utils.Success(c, 200, user)
}

// RefreshToken 刷新令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, utils.ErrCodeValidationFailed, "Invalid request body: %v", err)
		return
	}

	user, err := h.authService.ValidateToken(req.Token)
	if err != nil {
		utils.Error(c, utils.ErrCodeUnauthorized, "Invalid token: %v", err)
		return
	}

	authResp, err := h.authService.RefreshToken(user.ID.String())
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Token refresh failed: %v", err)
		return
	}

	utils.Success(c, 200, authResp)
}

// Profile 获取当前用户信息
func (h *AuthHandler) Profile(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		utils.Error(c, utils.ErrCodeBadRequest, "Invalid authorization header")
		return
	}

	user, err := h.authService.ValidateToken(authHeader[7:])
	if err != nil {
		utils.Error(c, utils.ErrCodeUnauthorized, "Invalid token: %v", err)
		return
	}

	utils.Success(c, 200, user)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	var username string
	if len(authHeader) >= 7 && authHeader[:7] == "Bearer " {
		if user, err := h.authService.ValidateToken(authHeader[7:]); err == nil {
			username = user.Username
		}
	}

	if h.auditService != nil {
		h.auditService.CreateAuditEvent(
			uuid.Nil,
			"auth",
			"logout",
			"user",
			username,
			username,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			nil,
			nil,
			map[string]interface{}{
				"username": username,
			},
			constants.StatusSuccess,
		)
	}

	utils.Success(c, 200, gin.H{
		"message": "Successfully logged out",
	})
}

// GenerateToken 生成令牌（用于测试）
func (h *AuthHandler) GenerateToken(c *gin.Context) {
	token, err := middleware.GenerateToken("test-id", "test-user", "admin", "your-secret-key")
	if err != nil {
		utils.Error(c, utils.ErrCodeInternalError, "Failed to generate token: %v", err)
		return
	}

	utils.Success(c, 200, gin.H{
		"token": token,
	})
}
