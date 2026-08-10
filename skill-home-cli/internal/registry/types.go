package registry

import (
	"time"
)

// Skill 技能信息
type Skill struct {
	ID                 string                      `json:"id"`
	Namespace          string                      `json:"namespace"`
	Name               string                      `json:"name"`
	OwnerID            string                      `json:"owner_id,omitempty"`
	OwnerUsername      string                      `json:"owner_username,omitempty"`
	OwnerDisplayNameZh string                      `json:"owner_display_name_zh,omitempty"`
	Description        string                      `json:"description"`
	Category           string                      `json:"category,omitempty"`
	DescriptionZh      string                      `json:"description_zh,omitempty"`
	Author             string                      `json:"author"`
	Tags               []string                    `json:"tags,omitempty"`
	License            string                      `json:"license,omitempty"`
	Homepage           string                      `json:"homepage,omitempty"`
	DownloadCount      int64                       `json:"download_count"`
	LikeCount          int64                       `json:"like_count"`
	InstallCount       int64                       `json:"install_count"`
	Rating             float64                     `json:"rating"`
	RatingCount        int64                       `json:"rating_count"`
	IsPublic           bool                        `json:"is_public"`
	IsOwnerOnly        bool                        `json:"is_owner_only"`
	IsDeprecated       bool                        `json:"is_deprecated"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
	LatestVersion      string                      `json:"latest_version,omitempty"`
	DownloadURL        string                      `json:"download_url,omitempty"`
	Owner              *User                       `json:"owner,omitempty"`
	Versions           []SkillVersion              `json:"versions,omitempty"`
	UserRating         *SkillRating                `json:"user_rating,omitempty"`
	ViewerLiked        bool                        `json:"viewer_liked,omitempty"`
	Credentials        []SkillCredentialDescriptor `json:"credentials,omitempty"`
}

// SkillCredentialDescriptor 技能凭证描述
type SkillCredentialDescriptor struct {
	ID          string `json:"id"`
	Env         string `json:"env"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Input       string `json:"input,omitempty"`
	HelpURL     string `json:"help_url,omitempty"`
	Group       string `json:"group,omitempty"`
}

// SkillCollaborator skill 协作者权限。
type SkillCollaborator struct {
	ID            string    `json:"id"`
	SkillID       string    `json:"skill_id"`
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	DisplayNameZh string    `json:"display_name_zh,omitempty"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UpsertCollaboratorRequest 新增或更新协作者。
type UpsertCollaboratorRequest struct {
	Username string `json:"username"`
	Role     string `json:"role,omitempty"`
}

// SkillVersion 技能版本
type SkillVersion struct {
	ID           string      `json:"id"`
	SkillID      string      `json:"skill_id"`
	Version      string      `json:"version"`
	Manifest     *Manifest   `json:"manifest"`
	Dependencies []string    `json:"dependencies,omitempty"`
	SizeBytes    int64       `json:"size_bytes"`
	Checksum     string      `json:"checksum"`
	ScanStatus   string      `json:"scan_status"`
	ScanResult   *ScanResult `json:"scan_result,omitempty"`
	PublishedBy  string      `json:"published_by"`
	PublishedAt  time.Time   `json:"published_at"`
}

// SkillRating 技能评分
type SkillRating struct {
	ID        string    `json:"id"`
	SkillID   string    `json:"skill_id"`
	UserID    string    `json:"user_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Manifest 技能元数据
type Manifest struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category,omitempty"`
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
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ScanResult 安全扫描结果
type ScanResult struct {
	Status    string      `json:"status"`
	Summary   string      `json:"summary"`
	Issues    []ScanIssue `json:"issues"`
	ScannedAt time.Time   `json:"scanned_at"`
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

// AuthResponse 登录/注册响应
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// OAuthDeviceAuthorizationResponse starts the OAuth device authorization flow.
type OAuthDeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// OAuthDeviceTokenResponse is returned once the browser approves CLI access.
type OAuthDeviceTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	APIKeyName  string `json:"api_key_name"`
	User        User   `json:"user"`
}

// User 用户信息
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Namespace     string    `json:"namespace"`
	DisplayNameZh string    `json:"display_name_zh,omitempty"`
	Email         string    `json:"email"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
	Results []Skill `json:"results"`
}

// CatalogVersionResponse 目录版本响应
type CatalogVersionResponse struct {
	CatalogVersion int64     `json:"catalog_version"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ListSkillsOptions 技能列表查询选项
type ListSkillsOptions struct {
	Namespace string
	Query     string
	Tags      []string
	Page      int
	PerPage   int
}

// PublishRequest 发布请求
type PublishRequest struct {
	Namespace     string   `json:"namespace,omitempty"`
	Name          string   `json:"name,omitempty"`
	Version       string   `json:"version,omitempty"`
	Description   string   `json:"description,omitempty"`
	DescriptionZh string   `json:"description_zh,omitempty"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	License       string   `json:"license,omitempty"`
	IsPublic      *bool    `json:"is_public,omitempty"`
	IsOwnerOnly   *bool    `json:"is_owner_only,omitempty"`
	Force         bool     `json:"force,omitempty"`
}

type UpdateSkillRequest struct {
	Description   string   `json:"description"`
	DescriptionZh string   `json:"description_zh,omitempty"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	License       string   `json:"license,omitempty"`
	IsPublic      bool     `json:"is_public"`
	IsOwnerOnly   bool     `json:"is_owner_only"`
	IsDeprecated  bool     `json:"is_deprecated"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	PublishedAt string `json:"published_at"`
}

// CreateSkillFeedbackRequest 提交轻量 Skill 使用反馈。
type CreateSkillFeedbackRequest struct {
	FeedbackType string `json:"feedback_type"`
	Content      string `json:"content"`
}

// SkillFeedback Skill 使用反馈。
type SkillFeedback struct {
	ID                  string    `json:"id"`
	SkillID             string    `json:"skill_id"`
	UserID              string    `json:"user_id"`
	FeedbackType        string    `json:"feedback_type"`
	Content             string    `json:"content"`
	Status              string    `json:"status"`
	ResolutionNote      string    `json:"resolution_note,omitempty"`
	SkillNamespace      string    `json:"skill_namespace,omitempty"`
	SkillName           string    `json:"skill_name,omitempty"`
	AuthorUsername      string    `json:"author_username,omitempty"`
	AuthorDisplayNameZh string    `json:"author_display_name_zh,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type InstallEventRequest struct {
	Version       string `json:"version,omitempty"`
	Target        string `json:"target,omitempty"`
	InstallMode   string `json:"install_mode,omitempty"`
	ClientVersion string `json:"client_version,omitempty"`
}

type InstallEventResponse struct {
	InstallCount int64 `json:"install_count"`
	Skill        Skill `json:"skill"`
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

// AuditLog 审计日志
type AuditLog struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditLogList 审计日志列表
type AuditLogList struct {
	Total   int        `json:"total"`
	Page    int        `json:"page"`
	PerPage int        `json:"per_page"`
	Results []AuditLog `json:"results"`
}
