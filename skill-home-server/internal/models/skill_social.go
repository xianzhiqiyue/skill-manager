package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SkillLike 记录用户对 skill 的点赞关系。
type SkillLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SkillID   uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_skill_like_user" json:"skill_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_skill_like_user" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (SkillLike) TableName() string {
	return "skill_likes"
}

func (s *SkillLike) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// SkillInstallEvent 记录 CLI 或 Web 侧安装成功事件，用于安装量统计。
type SkillInstallEvent struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SkillID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"skill_id"`
	UserID        *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Version       string     `gorm:"size:20;index" json:"version,omitempty"`
	Target        string     `gorm:"size:32;index" json:"target,omitempty"`
	InstallMode   string     `gorm:"size:32" json:"install_mode,omitempty"`
	ClientVersion string     `gorm:"size:64" json:"client_version,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (SkillInstallEvent) TableName() string {
	return "skill_install_events"
}

func (s *SkillInstallEvent) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// SkillShareEvent 记录分享动作。当前不做计数排序，只保留后续运营分析能力。
type SkillShareEvent struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SkillID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"skill_id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Channel   string     `gorm:"size:32;index" json:"channel,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (SkillShareEvent) TableName() string {
	return "skill_share_events"
}

func (s *SkillShareEvent) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
