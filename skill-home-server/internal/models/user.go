package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:32;not null" json:"username"`
	Email     string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password  string         `gorm:"size:255" json:"-"` // 不序列化到 JSON
	AvatarURL string         `gorm:"size:500" json:"avatar_url,omitempty"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Skills  []Skill   `gorm:"foreignKey:OwnerID" json:"-"`
	APIKeys []APIKey  `gorm:"foreignKey:UserID" json:"-"`
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// APIKey API Key 模型
type APIKey struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	KeyHash    string         `gorm:"size:255;not null" json:"-"`
	Name       string         `gorm:"size:100" json:"name"`
	Prefix     string         `gorm:"size:16" json:"prefix"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// AuditLog 审计日志
type AuditLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Action       string    `gorm:"size:50;not null" json:"action"`
	ResourceType string    `gorm:"size:50;not null" json:"resource_type"`
	ResourceID   *uuid.UUID `gorm:"type:uuid" json:"resource_id,omitempty"`
	Metadata     JSON      `gorm:"type:jsonb" json:"metadata,omitempty"`
	IPAddress    string    `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent    string    `gorm:"size:500" json:"user_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
