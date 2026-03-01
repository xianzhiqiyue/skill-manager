package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// Claims JWT 声明
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Auth 认证中间件
func Auth(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Missing authorization header"})
			c.Abort()
			return
		}

		// 支持 Bearer Token 或 API Key
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization format"})
			c.Abort()
			return
		}

		token := parts[1]

		// 尝试解析 JWT
		claims, err := parseJWT(token)
		if err == nil {
			// JWT 验证成功
			var user models.User
			if err := db.First(&user, "id = ?", claims.UserID).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "User not found"})
				c.Abort()
				return
			}
			c.Set("user", &user)
			c.Next()
			return
		}

		// 尝试验证 API Key
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

// parseJWT 解析 JWT Token
func parseJWT(tokenString string) (*Claims, error) {
	cfg := config.Get()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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
	// 查找 API Key
	var apiKey models.APIKey
	if err := db.Preload("User").First(&apiKey, "key_hash = ?", hashKey(key)).Error; err != nil {
		return nil, err
	}

	// 检查是否过期
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	// 更新最后使用时间
	now := time.Now()
	apiKey.LastUsedAt = &now
	db.Save(&apiKey)

	return &apiKey.User, nil
}

// hashKey 哈希 API Key
func hashKey(key string) string {
	// 实际应该使用 bcrypt 或 SHA-256
	hash, _ := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	return string(hash)
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
