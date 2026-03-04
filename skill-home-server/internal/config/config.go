package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Search   SearchConfig   `mapstructure:"search"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// StorageConfig 对象存储配置
type StorageConfig struct {
	Type      string `mapstructure:"type"`
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	LocalPath string `mapstructure:"local_path"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret    string `mapstructure:"jwt_secret"`
	TokenExpire  int    `mapstructure:"token_expire_hours"`
	APIKeyPrefix string `mapstructure:"api_key_prefix"`
}

// SearchConfig 搜索配置
type SearchConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Engine    string `mapstructure:"engine"`
	MeiliHost string `mapstructure:"meili_host"`
	MeiliKey  string `mapstructure:"meili_key"`
}

var cfg *Config

// Load 加载配置
func Load() error {
	setDefaults()

	viper.SetEnvPrefix("SKILL_HOME")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/skill-home/")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 从环境变量读取（覆盖 viper 配置）
	loadFromEnv()

	return validate(cfg)
}

// loadFromEnv 从环境变量直接读取配置
func loadFromEnv() {
	// Server
	if v := os.Getenv("SKILL_HOME_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("SKILL_HOME_SERVER_MODE"); v != "" {
		cfg.Server.Mode = v
	}

	// Database
	if v := os.Getenv("SKILL_HOME_DATABASE_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("SKILL_HOME_DATABASE_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("SKILL_HOME_DATABASE_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("SKILL_HOME_DATABASE_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("SKILL_HOME_DATABASE_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("SKILL_HOME_DATABASE_SSL_MODE"); v != "" {
		cfg.Database.SSLMode = v
	}

	// Storage
	if v := os.Getenv("SKILL_HOME_STORAGE_TYPE"); v != "" {
		cfg.Storage.Type = v
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_ENDPOINT"); v != "" {
		cfg.Storage.Endpoint = v
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_ACCESS_KEY"); v != "" {
		cfg.Storage.AccessKey = v
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_SECRET_KEY"); v != "" {
		cfg.Storage.SecretKey = v
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_BUCKET"); v != "" {
		cfg.Storage.Bucket = v
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_REGION"); v != "" {
		cfg.Storage.Region = v
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_USE_SSL"); v != "" {
		cfg.Storage.UseSSL = v == "true"
	}
	if v := os.Getenv("SKILL_HOME_STORAGE_LOCAL_PATH"); v != "" {
		cfg.Storage.LocalPath = v
	}

	// Auth
	if v := os.Getenv("SKILL_HOME_AUTH_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("SKILL_HOME_AUTH_TOKEN_EXPIRE_HOURS"); v != "" {
		if hours, err := strconv.Atoi(v); err == nil {
			cfg.Auth.TokenExpire = hours
		}
	}
	if v := os.Getenv("SKILL_HOME_AUTH_API_KEY_PREFIX"); v != "" {
		cfg.Auth.APIKeyPrefix = v
	}

	// Search
	if v := os.Getenv("SKILL_HOME_SEARCH_ENABLED"); v != "" {
		cfg.Search.Enabled = v == "true"
	}
	if v := os.Getenv("SKILL_HOME_SEARCH_ENGINE"); v != "" {
		cfg.Search.Engine = v
	}
	if v := os.Getenv("SKILL_HOME_SEARCH_MEILI_HOST"); v != "" {
		cfg.Search.MeiliHost = v
	}
	if v := os.Getenv("SKILL_HOME_SEARCH_MEILI_KEY"); v != "" {
		cfg.Search.MeiliKey = v
	}
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "development")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.bucket", "skill-home")
	viper.SetDefault("auth.token_expire_hours", 24)
	viper.SetDefault("search.enabled", true)
}

func validate(cfg *Config) error {
	if cfg.Auth.JWTSecret == "" {
		if strings.EqualFold(cfg.Server.Mode, "production") {
			return fmt.Errorf("auth.jwt_secret is required in production")
		}
		cfg.Auth.JWTSecret = "dev-secret"
	}
	if cfg.Auth.TokenExpire <= 0 {
		cfg.Auth.TokenExpire = 24
	}
	return nil
}

// Get 获取配置
func Get() *Config {
	return cfg
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}
