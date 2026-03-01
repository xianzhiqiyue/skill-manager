package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

// Register 用户注册
func Register(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		// 检查邮箱是否已存在
		var existingUser models.User
		if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"code": "EMAIL_EXISTS", "message": "Email already registered"})
			return
		}

		// 检查用户名是否已存在
		if err := db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"code": "USERNAME_EXISTS", "message": "Username already taken"})
			return
		}

		// 密码哈希
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process password"})
			return
		}

		// 创建用户
		user := models.User{
			Username: req.Username,
			Email:    req.Email,
			Password: string(passwordHash),
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		// 生成 JWT
		token, err := generateToken(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate token"})
			return
		}

		// 返回响应
		resp := AuthResponse{
			Token: token,
		}
		resp.User.ID = user.ID.String()
		resp.User.Username = user.Username
		resp.User.Email = user.Email

		c.JSON(http.StatusCreated, resp)
	}
}

// Login 用户登录
func Login(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"})
			return
		}

		// 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"})
			return
		}

		// 生成 JWT
		token, err := generateToken(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate token"})
			return
		}

		// 返回响应
		resp := AuthResponse{
			Token: token,
		}
		resp.User.ID = user.ID.String()
		resp.User.Username = user.Username
		resp.User.Email = user.Email

		c.JSON(http.StatusOK, resp)
	}
}

// generateToken 生成 JWT Token
func generateToken(user *models.User) (string, error) {
	cfg := config.Get()
	expireHours := cfg.Auth.TokenExpire
	if expireHours == 0 {
		expireHours = 24
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(time.Hour * time.Duration(expireHours)).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Auth.JWTSecret))
}
