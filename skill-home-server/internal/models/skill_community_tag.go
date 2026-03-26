package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SkillCommunityTag struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SkillID   uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_skill_user_tag" json:"skill_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_skill_user_tag" json:"user_id"`
	Tag       string    `gorm:"size:64;not null;index;uniqueIndex:idx_skill_user_tag" json:"tag"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SkillCommunityTag) TableName() string {
	return "skill_community_tags"
}

func (s *SkillCommunityTag) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type SkillCommunityTagSummary struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}
