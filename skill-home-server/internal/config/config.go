package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Search    SearchConfig    `mapstructure:"search"`
	SoulStore SoulStoreConfig `mapstructure:"soulstore"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port     int    `mapstructure:"port"`
	Mode     string `mapstructure:"mode"`
	BasePath string `mapstructure:"base_path"`
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
	// PublicBaseURL must already point at the object root, such as
	// https://cdn.example.com/skill-home-assets.
	PublicBaseURL string `mapstructure:"public_base_url"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret           string `mapstructure:"jwt_secret"`
	TokenExpire         int    `mapstructure:"token_expire_hours"`
	APIKeyPrefix        string `mapstructure:"api_key_prefix"`
	BootstrapSuperAdmin string `mapstructure:"bootstrap_super_admin"`
}

// SearchConfig 搜索配置
type SearchConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Engine    string `mapstructure:"engine"`
	MeiliHost string `mapstructure:"meili_host"`
	MeiliKey  string `mapstructure:"meili_key"`
}

type SoulStoreConfig struct {
	BaseURL           string `mapstructure:"base_url"`
	SSOSecret         string `mapstructure:"sso_secret"`
	SSOExchangePath   string `mapstructure:"sso_exchange_path"`
	SSOTimeoutSeconds int    `mapstructure:"sso_timeout_seconds"`
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
	cfg.Server.BasePath = NormalizeBasePath(cfg.Server.BasePath)

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
	if v := os.Getenv("SKILL_HOME_SERVER_BASE_PATH"); v != "" {
		cfg.Server.BasePath = v
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
	if v := os.Getenv("SKILL_HOME_STORAGE_PUBLIC_BASE_URL"); v != "" {
		cfg.Storage.PublicBaseURL = v
	}
	applySoulstoreSkillOSSFallback()

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
	if v := os.Getenv("SKILL_HOME_AUTH_BOOTSTRAP_SUPER_ADMIN"); v != "" {
		cfg.Auth.BootstrapSuperAdmin = v
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

	// SoulStore SSO
	if v := os.Getenv("SKILL_HOME_SOULSTORE_BASE_URL"); v != "" {
		cfg.SoulStore.BaseURL = v
	}
	if v := os.Getenv("SKILL_HOME_SOULSTORE_SSO_SECRET"); v != "" {
		cfg.SoulStore.SSOSecret = v
	}
	if v := os.Getenv("SOULSTORE_SKILL_HOME_SSO_SECRET"); v != "" && cfg.SoulStore.SSOSecret == "" {
		cfg.SoulStore.SSOSecret = v
	}
	if v := os.Getenv("SKILL_HOME_SOULSTORE_SSO_EXCHANGE_PATH"); v != "" {
		cfg.SoulStore.SSOExchangePath = v
	}
	if v := os.Getenv("SKILL_HOME_SOULSTORE_SSO_TIMEOUT_SECONDS"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil {
			cfg.SoulStore.SSOTimeoutSeconds = seconds
		}
	}
}

func applySoulstoreSkillOSSFallback() {
	if !hasSoulstoreSkillOSSConfig() {
		return
	}

	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_TYPE"); !ok {
		cfg.Storage.Type = "s3"
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_ENDPOINT"); !ok {
		if endpoint := soulstoreSkillOSSEndpoint(); endpoint != "" {
			cfg.Storage.Endpoint = endpoint
		}
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_ACCESS_KEY"); !ok {
		if v := os.Getenv("SOULSTORE_SKILL_OSS_ACCESS_KEY_ID"); v != "" {
			cfg.Storage.AccessKey = v
		}
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_SECRET_KEY"); !ok {
		if v := os.Getenv("SOULSTORE_SKILL_OSS_ACCESS_KEY_SECRET"); v != "" {
			cfg.Storage.SecretKey = v
		}
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_BUCKET"); !ok {
		if v := os.Getenv("SOULSTORE_SKILL_OSS_BUCKET_RELEASE"); v != "" {
			cfg.Storage.Bucket = v
		}
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_REGION"); !ok {
		if v := os.Getenv("SOULSTORE_SKILL_OSS_REGION"); v != "" {
			cfg.Storage.Region = v
		}
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_USE_SSL"); !ok {
		cfg.Storage.UseSSL = soulstoreSkillOSSUseSSL()
	}
	if _, ok := os.LookupEnv("SKILL_HOME_STORAGE_PUBLIC_BASE_URL"); !ok {
		if publicBaseURL := soulstoreSkillOSSPublicBaseURL(); publicBaseURL != "" {
			cfg.Storage.PublicBaseURL = publicBaseURL
		}
	}
}

func hasSoulstoreSkillOSSConfig() bool {
	keys := []string{
		"SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT",
		"SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT",
		"SOULSTORE_SKILL_OSS_ACCESS_KEY_ID",
		"SOULSTORE_SKILL_OSS_ACCESS_KEY_SECRET",
		"SOULSTORE_SKILL_OSS_BUCKET_RELEASE",
		"SOULSTORE_SKILL_OSS_REGION",
	}

	for _, key := range keys {
		if os.Getenv(key) != "" {
			return true
		}
	}

	return false
}

func soulstoreSkillOSSEndpoint() string {
	useInternal := strings.EqualFold(os.Getenv("SOULSTORE_SKILL_OSS_USE_INTERNAL_ENDPOINT"), "true")
	if useInternal {
		if endpoint := sanitizeStorageEndpoint(os.Getenv("SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT")); endpoint != "" {
			return endpoint
		}
	}

	if endpoint := sanitizeStorageEndpoint(os.Getenv("SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT")); endpoint != "" {
		return endpoint
	}

	return sanitizeStorageEndpoint(os.Getenv("SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT"))
}

func soulstoreSkillOSSPublicBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT")); v != "" {
		return ensureURLScheme(v, "https")
	}

	return ""
}

func soulstoreSkillOSSUseSSL() bool {
	rawEndpoint := os.Getenv("SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT")
	if !strings.EqualFold(os.Getenv("SOULSTORE_SKILL_OSS_USE_INTERNAL_ENDPOINT"), "true") {
		rawEndpoint = os.Getenv("SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT")
	}

	rawEndpoint = strings.TrimSpace(rawEndpoint)
	switch {
	case strings.HasPrefix(strings.ToLower(rawEndpoint), "http://"):
		return false
	case strings.HasPrefix(strings.ToLower(rawEndpoint), "https://"):
		return true
	default:
		return true
	}
}

func sanitizeStorageEndpoint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if !strings.Contains(trimmed, "://") {
		return strings.TrimSuffix(trimmed, "/")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://"), "/")
	}

	return strings.TrimSuffix(parsed.Host, "/")
}

func ensureURLScheme(raw string, defaultScheme string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		return strings.TrimRight(trimmed, "/")
	}

	return strings.TrimRight(defaultScheme+"://"+trimmed, "/")
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "development")
	viper.SetDefault("server.base_path", "")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.bucket", "skill-home")
	viper.SetDefault("auth.token_expire_hours", 24)
	viper.SetDefault("search.enabled", true)
	viper.SetDefault("soulstore.base_url", "https://soulstore.ciqtek.com")
	viper.SetDefault("soulstore.sso_exchange_path", "/api/v1/skill-home/sso/exchange")
	viper.SetDefault("soulstore.sso_timeout_seconds", 10)
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
	cfg.SoulStore.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.SoulStore.BaseURL), "/")
	if strings.TrimSpace(cfg.SoulStore.SSOExchangePath) == "" {
		cfg.SoulStore.SSOExchangePath = "/api/v1/skill-home/sso/exchange"
	}
	if cfg.SoulStore.SSOTimeoutSeconds <= 0 {
		cfg.SoulStore.SSOTimeoutSeconds = 10
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

func NormalizeBasePath(basePath string) string {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" || trimmed == "/" {
		return ""
	}

	return "/" + strings.Trim(trimmed, "/")
}

func JoinBasePath(basePath string, path string) string {
	normalizedBasePath := NormalizeBasePath(basePath)
	if path == "" || path == "/" {
		if normalizedBasePath == "" {
			return "/"
		}
		return normalizedBasePath + "/"
	}

	normalizedPath := "/" + strings.TrimPrefix(path, "/")
	if normalizedBasePath == "" {
		return normalizedPath
	}

	return normalizedBasePath + normalizedPath
}
