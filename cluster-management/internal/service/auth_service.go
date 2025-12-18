package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/taichu-system/cluster-management/internal/model"
	"github.com/taichu-system/cluster-management/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	secretKey   string
	tokenExpiry time.Duration
}

// JWTClaims JWT声明结构
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(userRepo *repository.UserRepository, secretKey string, tokenExpiry time.Duration) *AuthService {
	if tokenExpiry == 0 {
		tokenExpiry = 24 * time.Hour // 默认24小时
	}
	return &AuthService{
		userRepo:    userRepo,
		secretKey:   secretKey,
		tokenExpiry: tokenExpiry,
	}
}

// Login 用户登录
func (s *AuthService) Login(username, password string) (*model.AuthResponse, error) {
	// 根据用户名查找用户
	repoUser, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(repoUser.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 转换为model.User
	user := &model.User{
		ID:       repoUser.ID,
		Username: repoUser.Username,
		Email:    repoUser.Email,
		Role:     repoUser.Role,
	}

	// 生成JWT令牌
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 更新最后登录时间
	now := time.Now()
	repoUser.LastLoginAt = &now
	if err := s.userRepo.Update(repoUser); err != nil {
		// 记录错误但不中断登录流程
		fmt.Printf("Failed to update last login time: %v\n", err)
	}

	return &model.AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(s.tokenExpiry),
		User:      user,
	}, nil
}

// Register 用户注册
func (s *AuthService) Register(username, email, password, role string) (*model.User, error) {
	// 检查用户名是否已存在
	if _, err := s.userRepo.GetByUsername(username); err == nil {
		return nil, errors.New("username already exists")
	}

	// 检查邮箱是否已存在
	if _, err := s.userRepo.GetByEmail(email); err == nil {
		return nil, errors.New("email already exists")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 创建仓库用户
	repoUser := &repository.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
	}

	if err := s.userRepo.Create(repoUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 返回用户信息，不包含密码
	return &model.User{
		ID:       repoUser.ID,
		Username: repoUser.Username,
		Email:    repoUser.Email,
		Role:     repoUser.Role,
	}, nil
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(userID string) (*model.AuthResponse, error) {
	// 根据ID查找用户
	repoUser, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 转换为model.User
	user := &model.User{
		ID:       repoUser.ID,
		Username: repoUser.Username,
		Email:    repoUser.Email,
		Role:     repoUser.Role,
	}

	// 生成新令牌
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &model.AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(s.tokenExpiry),
		User:      user,
	}, nil
}

// ValidateToken 验证令牌
func (s *AuthService) ValidateToken(tokenString string) (*model.User, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// 验证令牌有效性
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// 根据用户ID查找用户
		repoUser, err := s.userRepo.GetByID(claims.UserID)
		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}

		// 转换为model.User
		return &model.User{
			ID:       repoUser.ID,
			Username: repoUser.Username,
			Email:    repoUser.Email,
			Role:     repoUser.Role,
		}, nil
	}

	return nil, errors.New("invalid token")
}

// generateToken 生成JWT令牌
func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "cluster-management",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}
