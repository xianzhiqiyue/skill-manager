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
		baseURL = "https://registry.skill-home.dev"
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

// Search 搜索技能
func (c *Client) Search(query string, tags []string, page, perPage int) (*SearchResult, error) {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	for _, tag := range tags {
		params.Add("tag", tag)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if perPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", perPage))
	}

	path := "/api/v1/search"
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
		writer.WriteField("namespace", req.Namespace)
	}
	if req.Force {
		writer.WriteField("force", "true")
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// 发送请求
	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	resp, err := c.doRequest("POST", "/api/v1/skills", &buf, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		// 处理验证错误
		if apiErr, ok := err.(*APIError); ok && apiErr.Code == "VALIDATION_FAILED" {
			return nil, fmt.Errorf("安全扫描失败: %s", apiErr.Message)
		}
		return nil, err
	}

	var result PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Download 下载技能
func (c *Client) Download(namespace, name, version, outputPath string) error {
	path := fmt.Sprintf("/api/v1/download/%s/%s/%s", namespace, name, version)

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := c.handleError(resp); err != nil {
		return err
	}

	// 创建输出文件
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer out.Close()

	// 复制内容
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	return nil
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
	path := fmt.Sprintf("/skills/%s/%s/versions/%s", namespace, name, version)

	resp, err := c.doRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleError(resp)
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
	path := fmt.Sprintf("/user/api-keys/%s", keyID)

	resp, err := c.doRequest("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleError(resp)
}
