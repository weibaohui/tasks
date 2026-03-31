package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig  `yaml:"database"`
	API      APIConfig      `yaml:"api"`
	Logging  LoggingConfig   `yaml:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// APIConfig API 配置
type APIConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// Load 加载配置
// 优先级：环境变量 > 配置文件 > 默认值
func Load() (*Config, error) {
	cfg := defaultConfig()

	// 1. 尝试从配置文件加载
	configPath := getConfigPath()
	if configPath != "" {
		if err := loadFromFile(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
	}

	// 2. 环境变量覆盖
	applyEnvOverrides(cfg)

	return cfg, nil
}

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	home, _ := os.UserHomeDir()
	defaultDBPath := filepath.Join(home, ".taskmanager", "tasks.db")
	if _, err := os.Stat(defaultDBPath); os.IsNotExist(err) {
		// 如果默认路径不存在，使用项目内路径
		defaultDBPath = "./backend/tasks.db"
	}

	return &Config{
		Server: ServerConfig{
			Port: 8888,
		},
		Database: DatabaseConfig{
			Path: defaultDBPath,
		},
		API: APIConfig{
			BaseURL: "http://localhost:8888/api/v1",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	// 环境变量指定
	if path := os.Getenv("TASKMANAGER_CONFIG"); path != "" {
		return path
	}

	// 当前目录
	cwd, _ := os.Getwd()
	localPath := filepath.Join(cwd, "taskmanager.yaml")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// ~/.taskmanager/config.yaml
	home, _ := os.UserHomeDir()
	homePath := filepath.Join(home, ".taskmanager", "config.yaml")
	if _, err := os.Stat(homePath); err == nil {
		return homePath
	}

	return ""
}

// loadFromFile 从文件加载配置
func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

// applyEnvOverrides 应用环境变量覆盖
func applyEnvOverrides(cfg *Config) {
	// SERVER_PORT 环境变量
	if port := os.Getenv("TASKMANAGER_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	// DB_PATH 环境变量
	if dbPath := os.Getenv("TASKMANAGER_DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	// DB_PATH 简写形式
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" && cfg.Database.Path == "" {
		cfg.Database.Path = dbPath
	}

	// API_BASE_URL 环境变量
	if baseURL := os.Getenv("TASKMANAGER_API_BASE_URL"); baseURL != "" {
		cfg.API.BaseURL = baseURL
	}

	// API_BASE_URL 简写形式
	if baseURL := os.Getenv("API_BASE_URL"); baseURL != "" {
		cfg.API.BaseURL = baseURL
	}
}

// GetDatabasePath 获取数据库路径（兼容旧接口）
func GetDatabasePath() string {
	cfg, err := Load()
	if err != nil {
		// 回退到旧逻辑
		return getLegacyDBPath()
	}
	return cfg.Database.Path
}

// GetAPIBaseURL 获取 API Base URL
func GetAPIBaseURL() string {
	cfg, err := Load()
	if err != nil {
		return "http://localhost:8888/api/v1"
	}
	return cfg.API.BaseURL
}

// GetAPIToken 获取 API Token
func GetAPIToken() string {
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.API.Token
}

// getLegacyDBPath 回退到旧的路径逻辑
func getLegacyDBPath() string {
	if p := os.Getenv("TASKMANAGER_DB_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	cwd, _ := os.Getwd()
	if st, err := os.Stat(filepath.Join(cwd, "backend")); err == nil && st.IsDir() {
		return filepath.Join(cwd, "backend", "tasks.db")
	}
	if _, err := os.Stat("./tasks.db"); err == nil {
		return "./tasks.db"
	}
	return "./backend/tasks.db"
}

// EnsureConfigDir 确保配置目录存在
func EnsureConfigDir() error {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".taskmanager")
	return os.MkdirAll(configDir, 0755)
}

// WriteDefaultConfig 写入默认配置文件
func WriteDefaultConfig(path string) error {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8888,
		},
		Database: DatabaseConfig{
			Path: getLegacyDBPath(),
		},
		API: APIConfig{
			BaseURL: "http://localhost:8888/api/v1",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// InitConfig 初始化配置（创建默认配置如果不存在）
func InitConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".taskmanager", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		// 配置文件已存在
		return nil
	}

	return WriteDefaultConfig(configPath)
}

// FormatConfig 格式化配置为 YAML 字符串
func FormatConfig(cfg *Config) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExpandPath 展开路径中的 ~ 和环境变量
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}