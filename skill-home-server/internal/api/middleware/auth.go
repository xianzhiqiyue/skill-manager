package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Claims JWT 声明
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

const apiKeyPrefixLen = 8

// Auth 认证中间件
func Auth(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token, err := parseAuthorizationHeader(authHeader)
		if err != nil {
			message := "Invalid authorization format"
			if authHeader == "" {
				message = "Missing authorization header"
			}
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": message})
			c.Abort()
			return
		}

		claims, err := parseJWT(token)
		if err == nil {
			jwtUser, err := findUserByID(db, claims.UserID)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "User not found"})
				c.Abort()
				return
			}
			c.Set("user", jwtUser)
			c.Next()
			return
		}

		user, err := validateAPIKey(db, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Invalid token or API key"})
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件（认证失败时按匿名处理）
func OptionalAuth(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := parseAuthorizationHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		if claims, err := parseJWT(token); err == nil {
			if user, err := findUserByID(db, claims.UserID); err == nil {
				c.Set("user", user)
				c.Next()
				return
			}
		}

		if user, err := validateAPIKey(db, token); err == nil {
			c.Set("user", user)
		}
		c.Next()
	}
}

// parseJWT 解析 JWT Token
func parseJWT(tokenString string) (*Claims, error) {
	cfg := config.Get()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.Auth.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// validateAPIKey 验证 API Key
func validateAPIKey(db *storage.Database, key string) (*models.User, error) {
	prefix, ok := getAPIKeyPrefix(key)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	// 通过 prefix 缩小候选范围，再用 bcrypt 常量时间比较完整密钥。
	var apiKeys []models.APIKey
	if err := db.Preload("User").Where("prefix = ?", prefix).Find(&apiKeys).Error; err != nil {
		return nil, err
	}

	if len(apiKeys) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	now := time.Now()
	for i := range apiKeys {
		apiKey := apiKeys[i]

		if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(now) {
			continue
		}

		if err := bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(key)); err != nil {
			continue
		}

		db.Model(&models.APIKey{}).Where("id = ?", apiKey.ID).Update("last_used_at", now)

		if !apiKey.User.IsActive {
			return nil, gorm.ErrRecordNotFound
		}
		return &apiKey.User, nil
	}

	return nil, gorm.ErrRecordNotFound
}

func parseAuthorizationHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("invalid authorization format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	return token, nil
}

func findUserByID(db *storage.Database, userID string) (*models.User, error) {
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

func getAPIKeyPrefix(key string) (string, bool) {
	key = strings.TrimSpace(key)
	if len(key) < apiKeyPrefixLen {
		return "", false
	}
	return key[:apiKeyPrefixLen], true
}

// GenerateJWT 生成 JWT Token
func GenerateJWT(user *models.User) (string, error) {
	cfg := config.Get()

	claims := Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.Auth.TokenExpire) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "skill-home",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Auth.JWTSecret))
}
