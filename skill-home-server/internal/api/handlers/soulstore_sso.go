package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var soulStoreSSOHTTPClient = http.DefaultClient

type SoulStoreSSORequest struct {
	Ticket string `json:"ticket" binding:"required"`
}

type soulStoreSSOExchangeResponse struct {
	User soulStoreSSOUser `json:"user"`
}

type soulStoreSSOUser struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	Avatar       string `json:"avatar"`
	Role         string `json:"role"`
	IsAdmin      bool   `json:"is_admin"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

func SoulStoreSSOLogin(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SoulStoreSSORequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		exchanged, err := exchangeSoulStoreSSOTicket(c.Request.Context(), req.Ticket)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "SSO_EXCHANGE_FAILED", "message": err.Error()})
			return
		}

		user, err := upsertSoulStoreUser(db, exchanged.User)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		token, err := generateToken(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate token"})
			return
		}

		writeAuditLog(db, c, &user.ID, "auth.soulstore_sso", resourceTypeUser, &user.ID, models.JSON{
			"soulstore_user_id": user.SoulStoreUserID,
			"username":          user.Username,
		})

		c.JSON(http.StatusOK, buildAuthResponse(token, user))
	}
}

func exchangeSoulStoreSSOTicket(ctx context.Context, ticket string) (*soulStoreSSOExchangeResponse, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("skill-home config is not loaded")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.SoulStore.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("soulstore base url is not configured")
	}
	secret := strings.TrimSpace(cfg.SoulStore.SSOSecret)
	if secret == "" {
		secret = cfg.Auth.JWTSecret
	}
	if secret == "" {
		return nil, fmt.Errorf("soulstore sso secret is not configured")
	}

	body, err := json.Marshal(map[string]string{"ticket": strings.TrimSpace(ticket)})
	if err != nil {
		return nil, err
	}

	exchangePath := strings.TrimSpace(cfg.SoulStore.SSOExchangePath)
	if exchangePath == "" {
		exchangePath = "/api/v1/skill-home/sso/exchange"
	}
	if !strings.HasPrefix(exchangePath, "/") {
		exchangePath = "/" + exchangePath
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+exchangePath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-SoulStore-SSO-Secret", secret)

	client := soulStoreSSOHTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	if client.Timeout == 0 {
		client = &http.Client{Timeout: time.Duration(cfg.SoulStore.SSOTimeoutSeconds) * time.Second}
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("soulstore exchange returned %d: %s", response.StatusCode, strings.TrimSpace(string(limited)))
	}

	var exchanged soulStoreSSOExchangeResponse
	if err := json.NewDecoder(response.Body).Decode(&exchanged); err != nil {
		return nil, err
	}
	if strings.TrimSpace(exchanged.User.ID) == "" || strings.TrimSpace(exchanged.User.Username) == "" {
		return nil, fmt.Errorf("soulstore exchange response missing user identity")
	}
	return &exchanged, nil
}

func upsertSoulStoreUser(db *storage.Database, remote soulStoreSSOUser) (*models.User, error) {
	soulStoreUserID := strings.TrimSpace(remote.ID)
	username := normalizeSoulStoreUsername(remote.Username, soulStoreUserID)
	displayName := trimToRuneLimit(strings.TrimSpace(remote.DisplayName), 64)
	if displayName == "" {
		displayName = username
	}
	email := strings.TrimSpace(remote.Email)
	if email == "" {
		email = fmt.Sprintf("%s@soulstore.local", username)
	}

	var user models.User
	err := db.Where("soul_store_user_id = ?", soulStoreUserID).First(&user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		err = db.Where("username = ?", username).First(&user).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	isNew := user.ID == uuid.Nil

	user.SoulStoreUserID = soulStoreUserID
	user.Username = username
	user.DisplayNameZh = displayName
	user.Email = email
	user.AvatarURL = strings.TrimSpace(remote.Avatar)
	if isNew {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("soulstore-sso-disabled-password"), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(passwordHash)
	}
	user.IsActive = true
	user.IsAdmin = remote.IsAdmin || remote.IsSuperAdmin || strings.EqualFold(remote.Role, "admin") || strings.EqualFold(remote.Role, "superadmin")
	user.IsSuperAdmin = remote.IsSuperAdmin || strings.EqualFold(remote.Role, "superadmin")

	if isNew {
		if err := createUserRecord(db.DB, &user); err != nil {
			return nil, err
		}
		return &user, nil
	}

	if err := db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func buildAuthResponse(token string, user *models.User) AuthResponse {
	resp := AuthResponse{Token: token}
	resp.User.ID = user.ID.String()
	resp.User.Username = user.Username
	resp.User.DisplayNameZh = user.DisplayNameZh
	resp.User.Email = user.Email
	resp.User.IsAdmin = user.IsAdmin
	resp.User.IsSuperAdmin = user.IsSuperAdmin
	return resp
}

func normalizeSoulStoreUsername(username string, userID string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(username) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_':
			builder.WriteRune(r)
		case unicode.IsSpace(r), r == '.':
			builder.WriteRune('-')
		}
	}
	normalized := strings.Trim(builder.String(), "-_")
	if normalized == "" {
		normalized = "soulstore-" + shortHash(userID)
	}
	if len([]rune(normalized)) <= 32 {
		return normalized
	}
	prefix := trimToRuneLimit(normalized, 23)
	return strings.Trim(prefix, "-_") + "-" + shortHash(normalized+userID)
}

func trimToRuneLimit(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}
