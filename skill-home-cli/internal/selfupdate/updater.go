package selfupdate

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skill-home/cli/pkg/archive"
)

const (
	DefaultRepo         = "xianzhiqiyue/skill-manager"
	defaultAPIBaseURL   = "https://api.github.com"
	defaultDownloadBase = "https://github.com"
	defaultHostedBase   = "http://47.122.112.210:8080/releases"
	envReleaseRepo      = "SKILL_HOME_RELEASE_REPO"
	envHostedBaseURL    = "SKILL_HOME_RELEASES_BASE_URL"
	supportedBinaryName = "skill-home"
)

type Updater struct {
	Repo            string
	CurrentVersion  string
	ExecutablePath  string
	GOOS            string
	GOARCH          string
	APIBaseURL      string
	DownloadBaseURL string
	HostedReleasesBaseURL string
	Client          *http.Client
}

func (u Updater) Update(targetVersion string) (string, error) {
	u.applyDefaults()

	if u.GOOS == "windows" {
		return "", errors.New("Windows 当前暂不支持 self-update，请重新运行安装脚本")
	}

	version, err := u.resolveTargetVersion(targetVersion)
	if err != nil {
		return "", err
	}

	platform, archiveExt, binaryName, err := resolveAsset(u.GOOS, u.GOARCH)
	if err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "skill-home-self-update-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	assetName := fmt.Sprintf("%s-%s.%s", supportedBinaryName, platform, archiveExt)
	archivePath := filepath.Join(tmpDir, assetName)
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	extractDir := filepath.Join(tmpDir, "extract")

	if err := u.downloadReleaseAsset(version, assetName, archivePath, checksumsPath); err != nil {
		return "", err
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %w", err)
	}
	if err := archive.ExtractAuto(archivePath, extractDir); err != nil {
		return "", fmt.Errorf("解压发布包失败: %w", err)
	}

	extractedBinary := filepath.Join(extractDir, binaryName)
	if _, err := os.Stat(extractedBinary); err != nil {
		return "", fmt.Errorf("发布包中缺少可执行文件 %s", binaryName)
	}

	if err := installUpdatedBinary(extractedBinary, u.ExecutablePath); err != nil {
		return "", err
	}

	return version, nil
}

func (u *Updater) applyDefaults() {
	if strings.TrimSpace(u.Repo) == "" {
		u.Repo = ResolveRepo()
	}
	if strings.TrimSpace(u.ExecutablePath) == "" {
		if path, err := os.Executable(); err == nil {
			u.ExecutablePath = path
		}
	}
	if strings.TrimSpace(u.GOOS) == "" {
		u.GOOS = runtime.GOOS
	}
	if strings.TrimSpace(u.GOARCH) == "" {
		u.GOARCH = runtime.GOARCH
	}
	if strings.TrimSpace(u.APIBaseURL) == "" {
		u.APIBaseURL = defaultAPIBaseURL
	}
	if strings.TrimSpace(u.DownloadBaseURL) == "" {
		u.DownloadBaseURL = defaultDownloadBase
	}
	if strings.TrimSpace(u.HostedReleasesBaseURL) == "" {
		u.HostedReleasesBaseURL = ResolveHostedReleasesBaseURL()
	}
	if u.Client == nil {
		u.Client = http.DefaultClient
	}
}

func ResolveRepo() string {
	if repo := strings.TrimSpace(os.Getenv(envReleaseRepo)); repo != "" {
		return repo
	}
	return DefaultRepo
}

func ResolveHostedReleasesBaseURL() string {
	if baseURL := strings.TrimSpace(os.Getenv(envHostedBaseURL)); baseURL != "" {
		return strings.TrimRight(baseURL, "/")
	}
	return defaultHostedBase
}

func (u Updater) resolveTargetVersion(targetVersion string) (string, error) {
	if normalized := normalizeVersion(targetVersion); normalized != "" {
		return normalized, nil
	}

	if version, err := u.resolveHostedLatestVersion(); err == nil {
		return version, nil
	}

	return u.resolveGitHubLatestVersion()
}

func (u Updater) resolveHostedLatestVersion() (string, error) {
	if strings.TrimSpace(u.HostedReleasesBaseURL) == "" {
		return "", errors.New("hosted releases base URL is empty")
	}

	requestURL := fmt.Sprintf("%s/latest.json", strings.TrimRight(u.HostedReleasesBaseURL, "/"))
	resp, err := u.Client.Get(requestURL)
	if err != nil {
		return "", fmt.Errorf("获取站内最新版本失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取站内最新版本失败: HTTP %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("解析站内最新版本响应失败: %w", err)
	}

	version := normalizeVersion(payload.TagName)
	if version == "" {
		return "", errors.New("站内最新版本响应缺少 tag_name")
	}
	return version, nil
}

func (u Updater) resolveGitHubLatestVersion() (string, error) {

	requestURL := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(u.APIBaseURL, "/"), u.Repo)
	resp, err := u.Client.Get(requestURL)
	if err != nil {
		return "", fmt.Errorf("获取最新版本失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取最新版本失败: HTTP %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("解析最新版本响应失败: %w", err)
	}

	version := normalizeVersion(payload.TagName)
	if version == "" {
		return "", errors.New("最新版本响应缺少 tag_name")
	}
	return version, nil
}

type releaseDownloadSource struct {
	name        string
	releaseURL  string
	checksumURL string
}

func (u Updater) releaseDownloadSources(version string, assetName string) []releaseDownloadSource {
	sources := make([]releaseDownloadSource, 0, 2)

	if hostedBase := strings.TrimRight(u.HostedReleasesBaseURL, "/"); hostedBase != "" {
		sources = append(sources, releaseDownloadSource{
			name:        "hosted",
			releaseURL:  fmt.Sprintf("%s/%s/%s", hostedBase, version, assetName),
			checksumURL: fmt.Sprintf("%s/%s/checksums.txt", hostedBase, version),
		})
	}

	githubBase := strings.TrimRight(u.DownloadBaseURL, "/")
	sources = append(sources, releaseDownloadSource{
		name:        "github",
		releaseURL:  fmt.Sprintf("%s/%s/releases/download/%s/%s", githubBase, u.Repo, version, assetName),
		checksumURL: fmt.Sprintf("%s/%s/releases/download/%s/checksums.txt", githubBase, u.Repo, version),
	})

	return sources
}

func (u Updater) downloadReleaseAsset(version string, assetName string, archivePath string, checksumsPath string) error {
	var lastErr error

	for _, source := range u.releaseDownloadSources(version, assetName) {
		if err := u.downloadFile(source.releaseURL, archivePath); err != nil {
			lastErr = fmt.Errorf("%s 下载发布包失败: %w", source.name, err)
			continue
		}
		if err := u.downloadFile(source.checksumURL, checksumsPath); err != nil {
			lastErr = fmt.Errorf("%s 下载校验文件失败: %w", source.name, err)
			continue
		}
		if err := verifyChecksum(archivePath, checksumsPath, assetName); err != nil {
			lastErr = fmt.Errorf("%s checksum 校验失败: %w", source.name, err)
			continue
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("下载发布包失败: no download sources configured")
}

func normalizeVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "v") {
		return trimmed
	}
	return "v" + trimmed
}

func resolveAsset(goos, goarch string) (platform string, archiveExt string, binaryName string, err error) {
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", "", "", fmt.Errorf("不支持的架构: %s", goarch)
	}

	switch goos {
	case "linux", "darwin":
		return goos + "-" + goarch, string(archive.FormatTarGz), supportedBinaryName, nil
	case "windows":
		return "windows-" + goarch, string(archive.FormatZip), supportedBinaryName + ".exe", nil
	default:
		return "", "", "", fmt.Errorf("不支持的操作系统: %s", goos)
	}
}

func (u Updater) downloadFile(url string, outputPath string) error {
	resp, err := u.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer outputFile.Close()

	if _, err := io.Copy(outputFile, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

func verifyChecksum(archivePath string, checksumsPath string, assetName string) error {
	expected, err := expectedChecksum(checksumsPath, assetName)
	if err != nil {
		return err
	}

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开发布包失败: %w", err)
	}
	defer archiveFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, archiveFile); err != nil {
		return fmt.Errorf("计算发布包校验值失败: %w", err)
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum 校验失败: %s", assetName)
	}
	return nil
}

func expectedChecksum(checksumsPath string, assetName string) (string, error) {
	checksumFile, err := os.Open(checksumsPath)
	if err != nil {
		return "", fmt.Errorf("打开校验文件失败: %w", err)
	}
	defer checksumFile.Close()

	scanner := bufio.NewScanner(checksumFile)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if filepath.Base(strings.TrimPrefix(fields[len(fields)-1], "*")) == assetName {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("读取校验文件失败: %w", err)
	}
	return "", fmt.Errorf("校验文件中未找到 %s", assetName)
}

func installUpdatedBinary(sourcePath string, targetPath string) error {
	backupPath := targetPath + ".bak"

	if err := os.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("清理旧备份失败: %w", err)
	}
	if err := os.Rename(targetPath, backupPath); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}

	restore := func(cause error) error {
		_ = os.Remove(targetPath)
		if err := os.Rename(backupPath, targetPath); err != nil {
			return fmt.Errorf("更新失败且回滚失败: %v (rollback: %w)", cause, err)
		}
		return fmt.Errorf("安装更新失败: %w", cause)
	}

	if err := copyFile(sourcePath, targetPath); err != nil {
		return restore(err)
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return restore(fmt.Errorf("获取新二进制信息失败: %w", err))
	}
	if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
		return restore(fmt.Errorf("设置执行权限失败: %w", err))
	}

	return nil
}

func copyFile(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("打开新二进制失败: %w", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("写入目标文件失败: %w", err)
	}
	return nil
}
