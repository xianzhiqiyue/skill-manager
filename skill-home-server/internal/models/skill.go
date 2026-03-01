package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Skill 技能模型
type Skill struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Namespace     string         `gorm:"size:64;not null;index:idx_namespace_name,unique" json:"namespace"`
	Name          string         `gorm:"size:64;not null;index:idx_namespace_name,unique" json:"name"`
	OwnerID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"owner_id"`
	Description   string         `gorm:"size:500" json:"description"`
	DescriptionZh string         `gorm:"size:500" json:"description_zh,omitempty"`
	Author        string         `gorm:"size:255" json:"author,omitempty"`
	Tags          StringArray    `gorm:"type:text[]" json:"tags,omitempty"`
	License       string         `gorm:"size:50" json:"license,omitempty"`
	Homepage      string         `gorm:"size:500" json:"homepage,omitempty"`
	Repository    string         `gorm:"size:500" json:"repository,omitempty"`
	DownloadCount int64          `gorm:"default:0" json:"download_count"`
	RatingSum     int64          `gorm:"default:0" json:"-"`
	RatingCount   int64          `gorm:"default:0" json:"rating_count"`
	IsPublic      bool           `gorm:"default:true" json:"is_public"`
	IsDeprecated   bool           `gorm:"default:false" json:"is_deprecated"`
	LatestVersion  string         `gorm:"size:20" json:"latest_version,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Owner    User           `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Versions []SkillVersion `gorm:"foreignKey:SkillID;order:published_at desc" json:"versions,omitempty"`
}

// TableName 指定表名
func (Skill) TableName() string {
	return "skills"
}

// GetFullName 获取完整名称
func (s *Skill) GetFullName() string {
	return s.Namespace + "/" + s.Name
}

// GetRating 获取评分
func (s *Skill) GetRating() float64 {
	if s.RatingCount == 0 {
		return 0
	}
	return float64(s.RatingSum) / float64(s.RatingCount)
}

// SkillVersion 技能版本模型
type SkillVersion struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SkillID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_skill_version,unique" json:"skill_id"`
	Version      string         `gorm:"size:20;not null;index:idx_skill_version,unique" json:"version"`
	Manifest     JSON           `gorm:"type:jsonb" json:"manifest"`
	Dependencies StringArray    `gorm:"type:text[]" json:"dependencies,omitempty"`
	StoragePath  string         `gorm:"size:500;not null" json:"-"`
	SizeBytes    int64          `json:"size_bytes"`
	Checksum     string         `gorm:"size:64" json:"checksum"`
	ScanStatus   string         `gorm:"size:20;default:'pending'" json:"scan_status"`
	ScanResult   JSON           `gorm:"type:jsonb" json:"scan_result,omitempty"`
	PublishedBy  uuid.UUID      `gorm:"type:uuid;not null" json:"published_by"`
	PublishedAt  time.Time      `json:"published_at"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Skill Skill `gorm:"foreignKey:SkillID" json:"-"`
}

// TableName 指定表名
func (SkillVersion) TableName() string {
	return "skill_versions"
}

// JSON 自定义 JSON 类型
type JSON map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return json.Unmarshal([]byte(fmt.Sprintf("%v", v)), j)
	}
}

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// StringArray 字符串数组类型
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		// PostgreSQL text[] 返回格式如: {item1,item2}
		return parseArray(string(v), a)
	case string:
		return parseArray(v, a)
	default:
		return parseArray(fmt.Sprintf("%v", v), a)
	}
}

// Value 实现 driver.Valuer 接口
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

// parseArray 解析 PostgreSQL 数组格式
func parseArray(s string, a *StringArray) error {
	s = strings.Trim(s, "{}")
	if s == "" {
		*a = StringArray{}
		return nil
	}
	*a = strings.Split(s, ",")
	return nil
}

// SkillRating 技能评分
type SkillRating struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SkillID   uuid.UUID `gorm:"type:uuid;not null;index" json:"skill_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Rating    int       `gorm:"not null" json:"rating"`
	Comment   string    `gorm:"size:1000" json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
