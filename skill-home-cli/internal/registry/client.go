package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skill-home/cli/internal/config"
)

// Client 注册中心客户端
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建注册中心客户端
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = config.DefaultRegistryEndpoint
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// 处理路径中可能包含的查询参数
	baseURL := c.baseURL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(path, "/") {
		baseURL += "/"
	}
	u := baseURL + path

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}

	// 添加认证头
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// 添加其他头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// handleError 处理错误响应
func (c *Client) handleError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return &apiErr
}

// HealthCheck 健康检查
func (c *Client) HealthCheck() error {
	resp, err := c.doRequest("GET", "/health", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry is not healthy: %d", resp.StatusCode)
	}

	return nil
}

// GetCatalogVersion 获取目录版本信息
func (c *Client) GetCatalogVersion() (*CatalogVersionResponse, error) {
	resp, err := c.doRequest("GET", "/api/v1/catalog/version", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result CatalogVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Login 使用邮箱和密码登录
func (c *Client) Login(email, password string) (*AuthResponse, error) {
	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/login", bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Search 搜索技能
func (c *Client) Search(query, namespace string, tags []string, page, perPage int) (*SearchResult, error) {
	opts := ListSkillsOptions{
		Namespace: namespace,
		Query:     query,
		Tags:      tags,
		Page:      page,
		PerPage:   perPage,
	}
	return c.listSkills("/api/v1/search", opts)
}

// ListSkills 列出技能
func (c *Client) ListSkills(opts ListSkillsOptions) (*SearchResult, error) {
	return c.listSkills("/api/v1/skills", opts)
}

func (c *Client) listSkills(path string, opts ListSkillsOptions) (*SearchResult, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("q", opts.Query)
	}
	if opts.Namespace != "" {
		params.Set("namespace", strings.TrimPrefix(opts.Namespace, "@"))
	}
	for _, tag := range opts.Tags {
		params.Add("tag", tag)
	}
	if opts.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.PerPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSkill 获取技能信息
func (c *Client) GetSkill(namespace, name string) (*Skill, error) {
	path := fmt.Sprintf("/api/v1/skills/%s/%s", namespace, name)

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var skill Skill
	if err := json.NewDecoder(resp.Body).Decode(&skill); err != nil {
		return nil, err
	}

	return &skill, nil
}

// ListVersions 列出技能版本
func (c *Client) ListVersions(namespace, name string) ([]SkillVersion, error) {
	path := fmt.Sprintf("/api/v1/skills/%s/%s/versions", namespace, name)

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var versions []SkillVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}

// Publish 发布技能
func (c *Client) Publish(skillPath string, req *PublishRequest) (*PublishResponse, error) {
	resp, err := c.publishArchive("/api/v1/skills", skillPath, req)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.Code == "ALREADY_EXISTS" && req.Namespace != "" && req.Name != "" {
				return c.publishVersion(skillPath, req)
			}
			if apiErr.Code == "VALIDATION_FAILED" {
				return nil, fmt.Errorf("安全扫描失败: %s", apiErr.Message)
			}
		}
		return nil, err
	}

	return resp, nil
}

func (c *Client) publishVersion(skillPath string, req *PublishRequest) (*PublishResponse, error) {
	path := fmt.Sprintf("/api/v1/skills/%s/%s/versions", req.Namespace, req.Name)
	return c.publishArchive(path, skillPath, &PublishRequest{
		Version: req.Version,
		Force:   req.Force,
	})
}

func (c *Client) publishArchive(path, skillPath string, req *PublishRequest) (*PublishResponse, error) {
	// 创建 multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	file, err := os.Open(skillPath)
	if err != nil {
		return nil, fmt.Errorf("打开技能包失败: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("skill", filepath.Base(skillPath))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	// 添加其他字段
	if req.Namespace != "" {
		if err := writer.WriteField("namespace", req.Namespace); err != nil {
			return nil, err
		}
	}
	if req.Name != "" {
		if err := writer.WriteField("name", req.Name); err != nil {
			return nil, err
		}
	}
	if req.Version != "" {
		if err := writer.WriteField("version", req.Version); err != nil {
			return nil, err
		}
	}
	if req.Description != "" {
		if err := writer.WriteField("description", req.Description); err != nil {
			return nil, err
		}
	}
	if req.DescriptionZh != "" {
		if err := writer.WriteField("description_zh", req.DescriptionZh); err != nil {
			return nil, err
		}
	}
	if req.Category != "" {
		if err := writer.WriteField("category", req.Category); err != nil {
			return nil, err
		}
	}
	if len(req.Tags) > 0 {
		if err := writer.WriteField("tags", strings.Join(req.Tags, ",")); err != nil {
			return nil, err
		}
	}
	if req.License != "" {
		if err := writer.WriteField("license", req.License); err != nil {
			return nil, err
		}
	}
	if req.IsPublic != nil {
		if *req.IsPublic {
			if err := writer.WriteField("is_public", "true"); err != nil {
				return nil, err
			}
		} else {
			if err := writer.WriteField("is_public", "false"); err != nil {
				return nil, err
			}
		}
	}
	if req.IsOwnerOnly != nil {
		if *req.IsOwnerOnly {
			if err := writer.WriteField("is_owner_only", "true"); err != nil {
				return nil, err
			}
		} else {
			if err := writer.WriteField("is_owner_only", "false"); err != nil {
				return nil, err
			}
		}
	}
	if req.Force {
		if err := writer.WriteField("force", "true"); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// 发送请求
	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	resp, err := c.doRequest("POST", path, &buf, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func isAbsoluteDownloadURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs()
}

func shouldUseDownloadURL(requestedVersion string, skill *Skill) bool {
	if skill == nil || skill.DownloadURL == "" || skill.LatestVersion == "" {
		return false
	}

	if !isAbsoluteDownloadURL(skill.DownloadURL) {
		return false
	}

	if requestedVersion == "" {
		return true
	}

	return requestedVersion == skill.LatestVersion
}

func (c *Client) saveDownloadResponse(resp *http.Response, outputPath string) error {
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	return nil
}

func (c *Client) downloadFromURL(downloadURL, outputPath string) error {
	if isAbsoluteDownloadURL(downloadURL) {
		req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}

		return c.saveDownloadResponse(resp, outputPath)
	}

	resp, err := c.doRequest(http.MethodGet, downloadURL, nil, nil)
	if err != nil {
		return err
	}

	return c.saveDownloadResponse(resp, outputPath)
}

// Download 下载技能
func (c *Client) Download(namespace, name, version, outputPath string) error {
	skill, err := c.GetSkill(namespace, name)
	if err != nil {
		return err
	}

	if shouldUseDownloadURL(version, skill) {
		if err := c.downloadFromURL(skill.DownloadURL, outputPath); err != nil {
			return err
		}

		return nil
	}

	path := fmt.Sprintf("/api/v1/download/%s/%s/%s?format=zip", namespace, name, version)

	resp, err := c.doRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}

	return c.saveDownloadResponse(resp, outputPath)
}

// DeleteSkill 删除技能
func (c *Client) DeleteSkill(namespace, name string) error {
	path := fmt.Sprintf("/api/v1/skills/%s/%s", namespace, name)

	resp, err := c.doRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleError(resp)
}

// DeleteVersion 删除技能版本
func (c *Client) DeleteVersion(namespace, name, version string) error {
	path := fmt.Sprintf("/api/v1/skills/%s/%s/versions/%s", namespace, name, version)

	resp, err := c.doRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleError(resp)
}

// ListCollaborators 列出 skill 协作者。
func (c *Client) ListCollaborators(namespace, name string) ([]SkillCollaborator, error) {
	path := fmt.Sprintf("/api/v1/skills/%s/%s/collaborators", namespace, name)

	resp, err := c.doRequest(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result []SkillCollaborator
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpsertCollaborator 新增或更新 skill 协作者。
func (c *Client) UpsertCollaborator(namespace, name string, req *UpsertCollaboratorRequest) (*SkillCollaborator, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	path := fmt.Sprintf("/api/v1/skills/%s/%s/collaborators", namespace, name)
	resp, err := c.doRequest(http.MethodPost, path, bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result SkillCollaborator
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteCollaborator 移除 skill 协作者。
func (c *Client) DeleteCollaborator(namespace, name, username string) error {
	path := fmt.Sprintf("/api/v1/skills/%s/%s/collaborators/%s", namespace, name, url.PathEscape(strings.TrimPrefix(username, "@")))

	resp, err := c.doRequest(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleError(resp)
}

// UpdateSkill 更新技能元数据
func (c *Client) UpdateSkill(namespace, name string, req *UpdateSkillRequest) (*Skill, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	path := fmt.Sprintf("/api/v1/skills/%s/%s", namespace, name)
	resp, err := c.doRequest(http.MethodPut, path, bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var skill Skill
	if err := json.NewDecoder(resp.Body).Decode(&skill); err != nil {
		return nil, err
	}

	return &skill, nil
}

// GetCurrentUser 获取当前用户信息
func (c *Client) GetCurrentUser() (*User, error) {
	resp, err := c.doRequest("GET", "/api/v1/user", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserSkills 获取用户的技能列表
func (c *Client) GetUserSkills() ([]Skill, error) {
	resp, err := c.doRequest("GET", "/api/v1/user/skills", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var skills []Skill
	if err := json.NewDecoder(resp.Body).Decode(&skills); err != nil {
		return nil, err
	}

	return skills, nil
}

// CreateAPIKey 创建 API Key
func (c *Client) CreateAPIKey(req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := c.doRequest("POST", "/api/v1/user/api-keys", bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result CreateAPIKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RevokeAPIKey 撤销 API Key
func (c *Client) RevokeAPIKey(keyID string) error {
	path := fmt.Sprintf("/api/v1/user/api-keys/%s", keyID)

	resp, err := c.doRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleError(resp)
}

// ListAuditLogs 获取审计日志
func (c *Client) ListAuditLogs(page, perPage int, action string) (*AuditLogList, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if perPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", perPage))
	}
	if action != "" {
		params.Set("action", action)
	}

	path := "/api/v1/user/audit-logs"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result AuditLogList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateSkillFeedback 提交结构化 Skill 使用反馈。
func (c *Client) CreateSkillFeedback(namespace, name string, req *CreateSkillFeedbackRequest) (*SkillFeedback, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/v1/skills/%s/%s/feedback", namespace, name)
	resp, err := c.doRequest("POST", path, bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result SkillFeedback
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RecordInstallEvent 上报安装成功事件。该接口用于注册中心统计真实安装量。
func (c *Client) RecordInstallEvent(namespace, name string, req *InstallEventRequest) (*InstallEventResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	path := fmt.Sprintf("/api/v1/skills/%s/%s/install-events", namespace, name)
	resp, err := c.doRequest(http.MethodPost, path, bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return nil, err
	}

	var result InstallEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
