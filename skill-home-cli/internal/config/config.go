package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Version  string    `yaml:"version" mapstructure:"version"`
	Registry Registry  `yaml:"registry" mapstructure:"registry"`
	Local    Local     `yaml:"local" mapstructure:"local"`
	IDE      IDEConfig `yaml:"ide" mapstructure:"ide"`
	Sync     Sync      `yaml:"sync" mapstructure:"sync"`
	Security Security  `yaml:"security" mapstructure:"security"`
}

// Registry 注册中心配置
type Registry struct {
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	APIKey   string `yaml:"api_key" mapstructure:"api_key"`
	Timeout  int    `yaml:"timeout" mapstructure:"timeout"`
}

// Local 本地配置
type Local struct {
	SkillsDir        string `yaml:"skills_dir" mapstructure:"skills_dir"`
	DefaultNamespace string `yaml:"default_namespace" mapstructure:"default_namespace"`
}

// IDEConfig IDE 配置
type IDEConfig struct {
	Claude  IDE `yaml:"claude" mapstructure:"claude"`
	Copilot IDE `yaml:"copilot" mapstructure:"copilot"`
	Cursor  IDE `yaml:"cursor" mapstructure:"cursor"`
	Codex   IDE `yaml:"codex" mapstructure:"codex"`
}

// IDE 单个 IDE 配置
type IDE struct {
	Enabled     bool   `yaml:"enabled" mapstructure:"enabled"`
	ProjectPath string `yaml:"project_path" mapstructure:"project_path"`
	GlobalPath  string `yaml:"global_path,omitempty" mapstructure:"global_path"`
}

// Sync 同步配置
type Sync struct {
	Mode             string `yaml:"mode" mapstructure:"mode"`
	ConflictStrategy string `yaml:"conflict_strategy" mapstructure:"conflict_strategy"`
	AutoSyncOnPush   bool   `yaml:"auto_sync_on_push" mapstructure:"auto_sync_on_push"`
}

// Security 安全配置
type Security struct {
	ScanOnInstall      bool `yaml:"scan_on_install" mapstructure:"scan_on_install"`
	AllowRemoteScripts bool `yaml:"allow_remote_scripts" mapstructure:"allow_remote_scripts"`
}

var (
	// C 全局配置实例
	C *Config
)

// Init 初始化配置
func Init(configFile string) error {
	// 设置默认值
	setDefaults()

	// 设置配置文件搜索路径
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		configDir, err := getConfigDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 读取环境变量
	viper.SetEnvPrefix("SKILL_HOME")
	viper.AutomaticEnv()

	// 尝试读取配置文件（如果不存在则忽略）
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	// 绑定环境变量
	viper.BindEnv("registry.api_key", "SKILL_HOME_API_KEY")

	// 解析到结构体
	C = &Config{}
	if err := viper.Unmarshal(C); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}
	applyViperValues(C)

	// 扩展路径中的 ~
	expandPaths()

	return nil
}

func applyViperValues(cfg *Config) {
	cfg.Version = viper.GetString("version")

	cfg.Registry.Endpoint = viper.GetString("registry.endpoint")
	cfg.Registry.APIKey = viper.GetString("registry.api_key")
	cfg.Registry.Timeout = viper.GetInt("registry.timeout")

	cfg.Local.SkillsDir = viper.GetString("local.skills_dir")
	cfg.Local.DefaultNamespace = viper.GetString("local.default_namespace")

	cfg.IDE.Claude.Enabled = viper.GetBool("ide.claude.enabled")
	cfg.IDE.Claude.ProjectPath = viper.GetString("ide.claude.project_path")
	cfg.IDE.Claude.GlobalPath = viper.GetString("ide.claude.global_path")

	cfg.IDE.Copilot.Enabled = viper.GetBool("ide.copilot.enabled")
	cfg.IDE.Copilot.ProjectPath = viper.GetString("ide.copilot.project_path")
	cfg.IDE.Copilot.GlobalPath = viper.GetString("ide.copilot.global_path")

	cfg.IDE.Cursor.Enabled = viper.GetBool("ide.cursor.enabled")
	cfg.IDE.Cursor.ProjectPath = viper.GetString("ide.cursor.project_path")
	cfg.IDE.Cursor.GlobalPath = viper.GetString("ide.cursor.global_path")

	cfg.IDE.Codex.Enabled = viper.GetBool("ide.codex.enabled")
	cfg.IDE.Codex.ProjectPath = viper.GetString("ide.codex.project_path")
	cfg.IDE.Codex.GlobalPath = viper.GetString("ide.codex.global_path")

	cfg.Sync.Mode = viper.GetString("sync.mode")
	cfg.Sync.ConflictStrategy = viper.GetString("sync.conflict_strategy")
	cfg.Sync.AutoSyncOnPush = viper.GetBool("sync.auto_sync_on_push")

	cfg.Security.ScanOnInstall = viper.GetBool("security.scan_on_install")
	cfg.Security.AllowRemoteScripts = viper.GetBool("security.allow_remote_scripts")
}

// setDefaults 设置默认值
func setDefaults() {
	viper.SetDefault("version", "1.0")
	viper.SetDefault("registry.endpoint", "https://registry.skill-home.dev")
	viper.SetDefault("registry.timeout", 30)
	viper.SetDefault("local.skills_dir", "~/.skill-home/skills")
	viper.SetDefault("local.default_namespace", "@user")
	viper.SetDefault("ide.claude.enabled", true)
	viper.SetDefault("ide.claude.project_path", ".claude/skills")
	viper.SetDefault("ide.claude.global_path", "~/.claude/skills")
	viper.SetDefault("ide.copilot.enabled", false)
	viper.SetDefault("ide.copilot.project_path", ".github/skills")
	viper.SetDefault("ide.copilot.global_path", "~/.copilot/skills")
	viper.SetDefault("ide.cursor.enabled", true)
	viper.SetDefault("ide.cursor.project_path", ".cursor/rules")
	viper.SetDefault("ide.codex.enabled", true)
	viper.SetDefault("ide.codex.project_path", ".agents/skills")
	viper.SetDefault("ide.codex.global_path", "~/.agents/skills")
	viper.SetDefault("sync.mode", "auto")
	viper.SetDefault("sync.conflict_strategy", "project_wins")
	viper.SetDefault("sync.auto_sync_on_push", false)
	viper.SetDefault("security.scan_on_install", true)
	viper.SetDefault("security.allow_remote_scripts", false)
}

// getConfigDir 获取配置目录
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, ".config", "skill-home"), nil
}

// expandPaths 扩展路径中的 ~
func expandPaths() {
	home, _ := os.UserHomeDir()

	C.Local.SkillsDir = expandPath(C.Local.SkillsDir, home)
	C.IDE.Claude.GlobalPath = expandPath(C.IDE.Claude.GlobalPath, home)
	C.IDE.Copilot.GlobalPath = expandPath(C.IDE.Copilot.GlobalPath, home)
	C.IDE.Codex.GlobalPath = expandPath(C.IDE.Codex.GlobalPath, home)
}

// expandPath 扩展单个路径
func expandPath(path, home string) string {
	if path == "" {
		return path
	}
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(home, path[1:])
	}
	return path
}

// Save 保存配置到文件
func Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	return viper.WriteConfigAs(configFile)
}

// InitDefaultConfig 创建默认配置文件
func InitDefaultConfig() error {
	setDefaults()
	return Save()
}
