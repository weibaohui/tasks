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
	Database DatabaseConfig `yaml:"database"`
	API      APIConfig      `yaml:"api"`
	Logging  LoggingConfig  `yaml:"logging"`
	Agent    AgentConfig    `yaml:"agent"`
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
	BaseURL   string `yaml:"base_url"`    // API base URL (内部使用)
	PublicURL string `yaml:"public_url"`  // Public URL (用于 GitHub webhook回调)
	Token     string `yaml:"token"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	AIWorkSpaceRoot string `yaml:"ai_workspace_root"` // AI 工作区根目录
}

// Load 加载配置
// 优先级：环境变量 > 配置文件 > 默认值
func Load() (*Config, error) {
	cfg := defaultConfig()

	// 1. 尝试从配置文件加载
	configPath := GetConfigPath()
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
	defaultDBPath := filepath.Join(home, ".taskmanager", "data.db")

	return &Config{
		Server: ServerConfig{
			Port: 13618, // 正式环境默认端口
		},
		Database: DatabaseConfig{
			Path: defaultDBPath,
		},
		API: APIConfig{
			BaseURL:   "http://localhost:13618/api/v1",
			PublicURL: "", // 需要通过 tunnel 或手动配置获取公网地址
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Agent: AgentConfig{
			AIWorkSpaceRoot: "/tmp/ai-devops",
		},
	}
}

// GetConfigPath 获取配置文件路径（导出版本，供其他包使用）
// 优先级：环境变量 > 当前目录 taskmanager.yaml > ~/.taskmanager/config.yaml
func GetConfigPath() string {
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

// LoadFromPath 从指定路径加载配置
func LoadFromPath(path string) (*Config, error) {
	cfg := defaultConfig()
	if err := loadFromFile(path, cfg); err != nil {
		return nil, err
	}
	applyEnvOverrides(cfg)
	return cfg, nil
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

	// API_BASE_URL 环境变量
	if baseURL := os.Getenv("TASKMANAGER_API_BASE_URL"); baseURL != "" {
		cfg.API.BaseURL = baseURL
	}

	// API_BASE_URL 简写形式
	if baseURL := os.Getenv("API_BASE_URL"); baseURL != "" {
		cfg.API.BaseURL = baseURL
	}

	// TASKMANAGER_DB_PATH 环境变量
	if dbPath := os.Getenv("TASKMANAGER_DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	// AI_DEVOPS_WORKSPACE_ROOT 环境变量
	if workspaceRoot := os.Getenv("AI_DEVOPS_WORKSPACE_ROOT"); workspaceRoot != "" {
		cfg.Agent.AIWorkSpaceRoot = workspaceRoot
	}
}

// GetDatabasePath 获取数据库路径
// 优先使用 TASKMANAGER_DB_PATH 环境变量，其次使用配置文件中的路径
func GetDatabasePath() string {
	cfg, err := Load()
	if err != nil {
		// 如果加载失败，使用默认值 ~/.taskmanager/data.db
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".taskmanager", "data.db")
	}
	return ExpandPath(cfg.Database.Path)
}

// GetAPIBaseURL 获取 API Base URL
func GetAPIBaseURL() string {
	cfg, err := Load()
	if err != nil {
		return "http://localhost:13618/api/v1"
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

// GetAgentAIWorkSpaceRoot 获取 AI 工作区根目录
func GetAgentAIWorkSpaceRoot() string {
	cfg, err := Load()
	if err != nil {
		return "/tmp/ai-devops"
	}
	return cfg.Agent.AIWorkSpaceRoot
}

// GetServerPort 获取服务器端口
func GetServerPort() int {
	cfg, err := Load()
	if err != nil {
		return 13618
	}
	return cfg.Server.Port
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
			Port: 13618,
		},
		Database: DatabaseConfig{
			Path: filepath.Join("~", ".taskmanager", "data.db"),
		},
		API: APIConfig{
			BaseURL: "http://localhost:13618/api/v1",
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
// SaveConfig 保存配置到文件
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// 使用 0600 权限保护 API Token
	return os.WriteFile(path, data, 0600)
}

// GetPublicURL 获取公共 URL（用于 webhook 回调）
func GetPublicURL() string {
	cfg, err := Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.API.PublicURL
}

// UpdatePublicURL 更新配置文件中的 public_url
// 统一的配置更新方法，确保读取和写入使用相同的路径
func UpdatePublicURL(url string) error {
	cfg, err := Load()
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	cfg.API.PublicURL = url

	configPath := GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("未找到配置文件")
	}

	return SaveConfig(configPath, cfg)
}
