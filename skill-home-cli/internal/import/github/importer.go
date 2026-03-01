package github

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/skill-home/cli/internal/import/types"
)

// GitHubImporter GitHub 技能导入器
type GitHubImporter struct {
	url        string
	owner      string
	repo       string
	path       string
	ref        string
	isRelease  bool
	httpClient *http.Client
}

// NewImporter 创建 GitHub 导入器
func NewImporter(sourceURL string) (*GitHubImporter, error) {
	importer := &GitHubImporter{
		url:        sourceURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// 解析 GitHub URL
	if err := importer.parseURL(sourceURL); err != nil {
		return nil, err
	}

	return importer, nil
}

// parseURL 解析 GitHub URL
func (g *GitHubImporter) parseURL(sourceURL string) error {
	// 移除协议前缀
	url := strings.TrimPrefix(sourceURL, "gh://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// 匹配 github.com/owner/repo/path patterns
	if !strings.HasPrefix(url, "github.com/") {
		return fmt.Errorf("无效的 GitHub URL: %s", sourceURL)
	}

	url = strings.TrimPrefix(url, "github.com/")
	parts := strings.Split(url, "/")

	if len(parts) < 2 {
		return fmt.Errorf("无效的 GitHub URL，需要 owner/repo 格式: %s", sourceURL)
	}

	g.owner = parts[0]
	g.repo = parts[1]

	// 检查是否是 release 下载
	if len(parts) >= 4 && parts[2] == "releases" {
		g.isRelease = true
		g.ref = parts[len(parts)-1]
		return nil
	}

	// 提取路径和 ref (分支/tag)
	if len(parts) > 2 {
		// 检查是否有 @ref 后缀
		if idx := strings.Index(parts[len(parts)-1], "@"); idx > 0 {
			g.ref = parts[len(parts)-1][idx+1:]
			parts[len(parts)-1] = parts[len(parts)-1][:idx]
		}

		g.path = strings.Join(parts[2:], "/")
	}

	// 默认使用 main 分支
	if g.ref == "" {
		g.ref = "main"
	}

	return nil
}

// GetSkillInfo 获取技能信息
func (g *GitHubImporter) GetSkillInfo() (*types.SkillInfo, error) {
	// 尝试获取仓库信息
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", g.owner, g.repo)

	resp, err := g.httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	info := &types.SkillInfo{
		Source: "github",
		URL:    g.url,
		Name:   g.repo,
		Notes:  []string{},
	}

	if resp.StatusCode == http.StatusOK {
		var repo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Owner       struct {
				Login string `json:"login"`
			} `json:"owner"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&repo); err == nil {
			info.Name = repo.Name
			info.Description = repo.Description
			info.Author = repo.Owner.Login
		}
	}

	// 尝试获取 SKILL.md 中的版本信息
	version, _ := g.getVersionFromSkillMD()
	if version != "" {
		info.Version = version
	}

	info.Notes = append(info.Notes, fmt.Sprintf("从 GitHub 仓库 %s/%s 导入", g.owner, g.repo))
	if g.path != "" {
		info.Notes = append(info.Notes, fmt.Sprintf("子目录: %s", g.path))
	}
	info.Notes = append(info.Notes, fmt.Sprintf("分支/标签: %s", g.ref))

	return info, nil
}

// getVersionFromSkillMD 尝试从 SKILL.md 获取版本
func (g *GitHubImporter) getVersionFromSkillMD() (string, error) {
	rawURL := g.buildRawURL("SKILL.md")

	resp, err := g.httpClient.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SKILL.md 不存在")
	}

	content, _ := io.ReadAll(resp.Body)
	contentStr := string(content)

	// 简单解析 frontmatter 中的 version
	re := regexp.MustCompile(`(?m)^version:\s*(.+)$`)
	matches := re.FindStringSubmatch(contentStr)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	return "", nil
}

// Download 下载技能
func (g *GitHubImporter) Download(destPath string) error {
	var downloadURL string

	if g.isRelease {
		// 下载 release 资源
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/skill.tar.gz",
			g.owner, g.repo, g.ref)
	} else {
		// 下载仓库归档
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			g.owner, g.repo, g.ref)
	}

	// 创建临时文件
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("github-skill-%d.tar.gz", time.Now().Unix()))
	defer os.Remove(tempFile)

	// 下载文件
	if err := g.downloadFile(downloadURL, tempFile); err != nil {
		// 尝试使用 archive/main 路径
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.tar.gz",
			g.owner, g.repo)
		if err := g.downloadFile(downloadURL, tempFile); err != nil {
			return fmt.Errorf("下载失败: %w", err)
		}
	}

	// 解压文件
	if err := g.extractTarGz(tempFile, destPath); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	return nil
}

// downloadFile 下载文件
func (g *GitHubImporter) downloadFile(url, dest string) error {
	resp, err := g.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractTarGz 解压 tar.gz 文件
func (g *GitHubImporter) extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// 计算 strip 组件数
	stripComponents := 1

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 跳过非文件条目
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// 处理 strip 组件
		parts := strings.Split(header.Name, "/")
		if len(parts) <= stripComponents {
			continue
		}

		relPath := strings.Join(parts[stripComponents:], "/")

		// 如果指定了子路径，只提取该路径下的文件
		if g.path != "" {
			if !strings.HasPrefix(relPath, g.path) {
				continue
			}
			relPath = strings.TrimPrefix(relPath, g.path)
			relPath = strings.TrimPrefix(relPath, "/")
		}

		if relPath == "" {
			continue
		}

		targetPath := filepath.Join(dest, relPath)

		// 创建目录
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		// 写入文件
		outFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	return nil
}

// ConvertToSkill 转换为通用技能格式
func (g *GitHubImporter) ConvertToSkill(sourcePath string) (*types.Skill, error) {
	skill := &types.Skill{
		Version:    "0.1.0",
		License:    "MIT",
		References: make(map[string]string),
		Scripts:    make(map[string]string),
	}

	// 读取 SKILL.md
	skillMDPath := filepath.Join(sourcePath, "SKILL.md")
	if _, err := os.Stat(skillMDPath); err == nil {
		content, err := os.ReadFile(skillMDPath)
		if err == nil {
			skill.Content = string(content)
			// 解析元数据
			g.parseSkillMD(skill, string(content))
		}
	} else {
		// 如果没有 SKILL.md，尝试从 README.md 创建
		readmePath := filepath.Join(sourcePath, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			content, err := os.ReadFile(readmePath)
			if err == nil {
				skill.Content = g.convertReadmeToSkill(string(content))
				skill.Name = g.repo
				skill.Description = "从 GitHub 导入的技能"
			}
		}
	}

	// 读取 references
	refsDir := filepath.Join(sourcePath, "references")
	if entries, err := os.ReadDir(refsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, err := os.ReadFile(filepath.Join(refsDir, entry.Name()))
				if err == nil {
					skill.References[entry.Name()] = string(content)
				}
			}
		}
	}

	// 读取 scripts
	scriptsDir := filepath.Join(sourcePath, "scripts")
	if entries, err := os.ReadDir(scriptsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, err := os.ReadFile(filepath.Join(scriptsDir, entry.Name()))
				if err == nil {
					skill.Scripts[entry.Name()] = string(content)
				}
			}
		}
	}

	// 如果名称未设置，使用仓库名
	if skill.Name == "" {
		skill.Name = g.repo
	}

	// 添加来源标记
	skill.Content = g.addSourceHeader(skill.Content)

	return skill, nil
}

// buildRawURL 构建 GitHub raw 文件 URL
func (g *GitHubImporter) buildRawURL(filePath string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		g.owner, g.repo, g.ref, filePath)
}

// parseSkillMD 解析 SKILL.md 元数据
func (g *GitHubImporter) parseSkillMD(skill *types.Skill, content string) {
	// 简单的正则解析
	patterns := map[string]*regexp.Regexp{
		"name":    regexp.MustCompile(`(?m)^name:\s*(.+)$`),
		"version": regexp.MustCompile(`(?m)^version:\s*(.+)$`),
		"desc":    regexp.MustCompile(`(?m)^description:\s*(.+)$`),
		"author":  regexp.MustCompile(`(?m)^author:\s*(.+)$`),
		"license": regexp.MustCompile(`(?m)^license:\s*(.+)$`),
	}

	for key, re := range patterns {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			switch key {
			case "name":
				skill.Name = value
			case "version":
				skill.Version = value
			case "desc":
				skill.Description = value
			case "author":
				skill.Author = value
			case "license":
				skill.License = value
			}
		}
	}
}

// convertReadmeToSkill 将 README.md 转换为 SKILL.md 格式
func (g *GitHubImporter) convertReadmeToSkill(readme string) string {
	return fmt.Sprintf(`---
name: %s
version: 0.1.0
description: Imported from GitHub
tags: [imported, github]
license: MIT
---

# %s

从 GitHub 仓库导入的技能。

%s
`, g.repo, g.repo, readme)
}

// addSourceHeader 添加来源标记
func (g *GitHubImporter) addSourceHeader(content string) string {
	header := fmt.Sprintf("<!--\n  Source: GitHub\n  Repository: %s/%s\n  Imported: %s\n-->\n\n",
		g.owner, g.repo, time.Now().Format("2006-01-02"))
	return header + content
}
