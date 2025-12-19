package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/middleware"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/pkg/utils"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	authResp, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "Login failed: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, authResp)
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	// 设置默认角色
	if req.Role == "" {
		req.Role = "user"
	}

	user, err := h.authService.Register(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Registration failed: %v", err)
		return
	}

	utils.Success(c, http.StatusCreated, user)
}

// RefreshToken 刷新令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	// 验证令牌并获取用户信息
	user, err := h.authService.ValidateToken(req.Token)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "Invalid token: %v", err)
		return
	}

	// 生成新令牌
	authResp, err := h.authService.RefreshToken(user.ID.String())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Token refresh failed: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, authResp)
}

// Profile 获取当前用户信息
func (h *AuthHandler) Profile(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		utils.Error(c, http.StatusBadRequest, "Invalid authorization header")
		return
	}

	user, err := h.authService.ValidateToken(authHeader[7:]) // 去掉"Bearer "前缀
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "Invalid token: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, user)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 在实际应用中，可以将令牌加入黑名单或使用Redis存储已登出的令牌
	// 这里简化处理，只返回成功响应
	utils.Success(c, http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}

// GenerateToken 生成令牌（用于测试）
func (h *AuthHandler) GenerateToken(c *gin.Context) {
	// Generate a test token for development
	token, err := middleware.GenerateToken("test-id", "test-user", "admin", "your-secret-key")
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to generate token: %v", err)
		return
	}

	utils.Success(c, http.StatusOK, gin.H{
		"token": token,
	})
}
