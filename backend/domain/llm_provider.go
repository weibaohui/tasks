package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrLLMProviderIDRequired  = errors.New("llm provider id is required")
	ErrProviderKeyRequired    = errors.New("provider key is required")
	ErrProviderUserCodeNeeded = errors.New("user code is required")
)

type LLMProviderID struct {
	value string
}

func NewLLMProviderID(value string) LLMProviderID {
	return LLMProviderID{value: value}
}

func (id LLMProviderID) String() string {
	return id.value
}

type ModelInfo struct {
	ID        string
	Name      string
	MaxTokens int
}

type LLMProvider struct {
	id              LLMProviderID
	userCode        string
	providerKey     string
	providerName    string
	apiKey          string
	apiBase         string
	providerType    string // API 格式：openai, anthropic
	extraHeaders    map[string]string
	supportedModels []ModelInfo
	defaultModel    string
	isDefault       bool
	priority        int
	autoMerge       bool
	isActive        bool
	createdAt       time.Time
	updatedAt       time.Time
}

func NewLLMProvider(
	id LLMProviderID,
	userCode string,
	providerKey string,
	providerName string,
	apiKey string,
	apiBase string,
) (*LLMProvider, error) {
	if id.String() == "" {
		return nil, ErrLLMProviderIDRequired
	}
	if strings.TrimSpace(userCode) == "" {
		return nil, ErrProviderUserCodeNeeded
	}
	if strings.TrimSpace(providerKey) == "" {
		return nil, ErrProviderKeyRequired
	}

	now := time.Now()
	return &LLMProvider{
		id:           id,
		userCode:     userCode,
		providerKey:  providerKey,
		providerName: providerName,
		apiKey:       apiKey,
		apiBase:      apiBase,
		extraHeaders: map[string]string{},
		autoMerge:    true,
		isActive:     true,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func (p *LLMProvider) ID() LLMProviderID               { return p.id }
func (p *LLMProvider) UserCode() string                { return p.userCode }
func (p *LLMProvider) ProviderKey() string             { return p.providerKey }
func (p *LLMProvider) ProviderName() string            { return p.providerName }
func (p *LLMProvider) APIKey() string                  { return p.apiKey }
func (p *LLMProvider) APIBase() string                 { return p.apiBase }
func (p *LLMProvider) ProviderType() string            { return p.providerType }
func (p *LLMProvider) ExtraHeaders() map[string]string { return cloneHeaders(p.extraHeaders) }
func (p *LLMProvider) SupportedModels() []ModelInfo    { return cloneModels(p.supportedModels) }
func (p *LLMProvider) DefaultModel() string            { return p.defaultModel }
func (p *LLMProvider) IsDefault() bool                 { return p.isDefault }
func (p *LLMProvider) Priority() int                   { return p.priority }
func (p *LLMProvider) AutoMerge() bool                 { return p.autoMerge }
func (p *LLMProvider) IsActive() bool                  { return p.isActive }
func (p *LLMProvider) CreatedAt() time.Time            { return p.createdAt }
func (p *LLMProvider) UpdatedAt() time.Time            { return p.updatedAt }

func (p *LLMProvider) UpdateProfile(providerKey, providerName, apiKey, apiBase string) error {
	if strings.TrimSpace(providerKey) == "" {
		return ErrProviderKeyRequired
	}
	p.providerKey = providerKey
	p.providerName = providerName
	p.apiKey = apiKey
	p.apiBase = apiBase
	p.updatedAt = time.Now()
	return nil
}

func (p *LLMProvider) SetExtraHeaders(headers map[string]string) {
	p.extraHeaders = cloneHeaders(headers)
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetSupportedModels(models []ModelInfo) {
	p.supportedModels = cloneModels(models)
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetDefaultModel(model string) {
	p.defaultModel = model
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetDefault(isDefault bool) {
	p.isDefault = isDefault
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetPriority(priority int) {
	p.priority = priority
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetAutoMerge(autoMerge bool) {
	p.autoMerge = autoMerge
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetActive(isActive bool) {
	p.isActive = isActive
	p.updatedAt = time.Now()
}

func (p *LLMProvider) SetProviderType(providerType string) {
	p.providerType = providerType
	p.updatedAt = time.Now()
}

type LLMProviderSnapshot struct {
	ID              LLMProviderID
	UserCode        string
	ProviderKey     string
	ProviderName    string
	APIKey          string
	APIBase         string
	ProviderType    string
	ExtraHeaders    map[string]string
	SupportedModels []ModelInfo
	DefaultModel    string
	IsDefault       bool
	Priority        int
	AutoMerge       bool
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (p *LLMProvider) ToSnapshot() LLMProviderSnapshot {
	return LLMProviderSnapshot{
		ID:              p.id,
		UserCode:        p.userCode,
		ProviderKey:     p.providerKey,
		ProviderName:    p.providerName,
		APIKey:          p.apiKey,
		APIBase:         p.apiBase,
		ProviderType:    p.providerType,
		ExtraHeaders:    cloneHeaders(p.extraHeaders),
		SupportedModels: cloneModels(p.supportedModels),
		DefaultModel:    p.defaultModel,
		IsDefault:       p.isDefault,
		Priority:        p.priority,
		AutoMerge:       p.autoMerge,
		IsActive:        p.isActive,
		CreatedAt:       p.createdAt,
		UpdatedAt:       p.updatedAt,
	}
}

func (p *LLMProvider) FromSnapshot(snap LLMProviderSnapshot) {
	p.id = snap.ID
	p.userCode = snap.UserCode
	p.providerKey = snap.ProviderKey
	p.providerName = snap.ProviderName
	p.apiKey = snap.APIKey
	p.apiBase = snap.APIBase
	p.providerType = snap.ProviderType
	p.extraHeaders = cloneHeaders(snap.ExtraHeaders)
	p.supportedModels = cloneModels(snap.SupportedModels)
	p.defaultModel = snap.DefaultModel
	p.isDefault = snap.IsDefault
	p.priority = snap.Priority
	p.autoMerge = snap.AutoMerge
	p.isActive = snap.IsActive
	p.createdAt = snap.CreatedAt
	p.updatedAt = snap.UpdatedAt
}

func cloneHeaders(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneModels(in []ModelInfo) []ModelInfo {
	if len(in) == 0 {
		return nil
	}
	out := make([]ModelInfo, len(in))
	copy(out, in)
	return out
}

// LLMProviderConfig LLM Provider 配置，用于传递给基础设施层创建实际的 Provider
type LLMProviderConfig struct {
	providerKey  string // Provider 标识：openai, kimi, deepseek 等
	ProviderType string // API 格式：openai (OpenAI 兼容格式), anthropic (Claude 原生格式)
	Model        string
	APIKey       string
	BaseURL      string
	Temperature  float64
	MaxTokens    int
}

// ProviderType 常量
const (
	ProviderTypeOpenAI    = "openai"    // OpenAI 兼容格式
	ProviderTypeAnthropic = "anthropic" // Anthropic/Claude 原生格式
)

// NewLLMProviderConfig 创建 LLM Provider 配置
func NewLLMProviderConfig(providerKey, model, apiKey, baseURL string) *LLMProviderConfig {
	return &LLMProviderConfig{
		providerKey:  providerKey,
		ProviderType: ProviderTypeOpenAI, // 默认为 OpenAI 兼容格式
		Model:        model,
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Temperature:  0.7,
		MaxTokens:    4096,
	}
}

// SetProviderType 设置 API 类型
func (c *LLMProviderConfig) SetProviderType(providerType string) *LLMProviderConfig {
	c.ProviderType = providerType
	return c
}

func (c *LLMProviderConfig) ProviderKey() string     { return c.providerKey }
func (c *LLMProviderConfig) APIFormat() string       { return c.ProviderType }
func (c *LLMProviderConfig) ModelName() string       { return c.Model }
func (c *LLMProviderConfig) GetAPIKey() string       { return c.APIKey }
func (c *LLMProviderConfig) GetBaseURL() string      { return c.BaseURL }
func (c *LLMProviderConfig) GetTemperature() float64 { return c.Temperature }
func (c *LLMProviderConfig) GetMaxTokens() int       { return c.MaxTokens }
