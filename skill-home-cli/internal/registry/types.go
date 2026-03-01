package registry

import (
	"time"
)

// Skill 技能信息
type Skill struct {
	ID            string    `json:"id"`
	Namespace     string    `json:"namespace"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	DescriptionZh string    `json:"description_zh,omitempty"`
	Author        string    `json:"author"`
	Tags          []string  `json:"tags,omitempty"`
	License       string    `json:"license,omitempty"`
	Homepage      string    `json:"homepage,omitempty"`
	DownloadCount int64     `json:"download_count"`
	Rating        float64   `json:"rating"`
	RatingCount   int64     `json:"rating_count"`
	IsPublic      bool      `json:"is_public"`
	IsDeprecated  bool      `json:"is_deprecated"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LatestVersion string    `json:"latest_version,omitempty"`
}

// SkillVersion 技能版本
type SkillVersion struct {
	ID           string    `json:"id"`
	SkillID      string    `json:"skill_id"`
	Version      string    `json:"version"`
	Manifest     *Manifest `json:"manifest"`
	Dependencies []string  `json:"dependencies,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	Checksum     string    `json:"checksum"`
	ScanStatus   string    `json:"scan_status"`
	ScanResult   *ScanResult `json:"scan_result,omitempty"`
	PublishedBy  string    `json:"published_by"`
	PublishedAt  time.Time `json:"published_at"`
}

// Manifest 技能元数据
type Manifest struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Namespace     string                 `json:"namespace,omitempty"`
	DescriptionZh string                 `json:"description_zh,omitempty"`
	Author        string                 `json:"author,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	License       string                 `json:"license,omitempty"`
	Homepage      string                 `json:"homepage,omitempty"`
	Repository    string                 `json:"repository,omitempty"`
	Requires      []string               `json:"requires,omitempty"`
	IDEConfig     map[string]interface{} `json:"ide_config,omitempty"`
	Permissions   []string               `json:"permissions,omitempty"`
	Engines       map[string]string      `json:"engines,omitempty"`
}

// ScanResult 安全扫描结果
type ScanResult struct {
	Status   string      `json:"status"`
	Summary  string      `json:"summary"`
	Issues   []ScanIssue `json:"issues"`
	ScannedAt time.Time  `json:"scanned_at"`
}

// ScanIssue 扫描问题
type ScanIssue struct {
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Match      string `json:"match"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// User 用户信息
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
	Results []Skill `json:"results"`
}

// PublishRequest 发布请求
type PublishRequest struct {
	Namespace string `json:"namespace,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	PublishedAt string `json:"published_at"`
}

// APIError API 错误响应
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

// APIKey API Key 信息
type APIKey struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Prefix     string    `json:"prefix"` // 前 8 位，用于显示
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse 创建 API Key 响应
type CreateAPIKeyResponse struct {
	APIKey
	Key string `json:"key"` // 仅创建时返回完整 key
}
