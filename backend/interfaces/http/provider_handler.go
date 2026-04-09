package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type LLMProviderHandler struct {
	providerService *application.LLMProviderApplicationService
}

func NewLLMProviderHandler(providerService *application.LLMProviderApplicationService) *LLMProviderHandler {
	return &LLMProviderHandler{providerService: providerService}
}

type ProviderModelInfoRequest struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MaxTokens int    `json:"max_tokens"`
}

type CreateProviderRequest struct {
	UserCode        string                     `json:"user_code"`
	ProviderKey     string                     `json:"provider_key"`
	ProviderName    string                     `json:"provider_name"`
	APIKey          string                     `json:"api_key"`
	APIBase         string                     `json:"api_base"`
	ProviderType    string                     `json:"provider_type"`
	ExtraHeaders    map[string]string          `json:"extra_headers"`
	SupportedModels []ProviderModelInfoRequest `json:"supported_models"`
	DefaultModel    string                     `json:"default_model"`
	IsDefault       bool                       `json:"is_default"`
	Priority        int                        `json:"priority"`
	AutoMerge       *bool                      `json:"auto_merge"`
}

type UpdateProviderRequest struct {
	ProviderKey     *string                     `json:"provider_key"`
	ProviderName    *string                     `json:"provider_name"`
	APIKey          *string                     `json:"api_key"`
	APIBase         *string                     `json:"api_base"`
	ProviderType    *string                     `json:"provider_type"`
	ExtraHeaders    *map[string]string          `json:"extra_headers"`
	SupportedModels *[]ProviderModelInfoRequest `json:"supported_models"`
	DefaultModel    *string                     `json:"default_model"`
	IsDefault       *bool                       `json:"is_default"`
	Priority        *int                        `json:"priority"`
	AutoMerge       *bool                       `json:"auto_merge"`
	IsActive        *bool                       `json:"is_active"`
}

func (h *LLMProviderHandler) CreateProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	provider, err := h.providerService.Create(c.Request.Context(), application.CreateProviderCommand{
		UserCode:        req.UserCode,
		ProviderKey:     req.ProviderKey,
		ProviderName:    req.ProviderName,
		APIKey:          req.APIKey,
		APIBase:         req.APIBase,
		ProviderType:    req.ProviderType,
		ExtraHeaders:    req.ExtraHeaders,
		SupportedModels: toDomainModelInfos(req.SupportedModels),
		DefaultModel:    req.DefaultModel,
		IsDefault:       req.IsDefault,
		Priority:        req.Priority,
		AutoMerge:       req.AutoMerge,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, providerToMap(provider))
}

func (h *LLMProviderHandler) ListProviders(c *gin.Context) {
	userCode := c.Query("user_code")
	if userCode == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "user_code is required"})
		return
	}

	providers, err := h.providerService.List(c.Request.Context(), userCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	resp := make([]map[string]interface{}, 0, len(providers))
	for _, provider := range providers {
		resp = append(resp, providerToMap(provider))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *LLMProviderHandler) GetProvider(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	provider, err := h.providerService.Get(c.Request.Context(), domain.NewLLMProviderID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, providerToMap(provider))
}

func (h *LLMProviderHandler) UpdateProvider(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	provider, err := h.providerService.Update(c.Request.Context(), application.UpdateProviderCommand{
		ID:              domain.NewLLMProviderID(id),
		ProviderKey:     req.ProviderKey,
		ProviderName:    req.ProviderName,
		APIKey:          req.APIKey,
		APIBase:         req.APIBase,
		ProviderType:    req.ProviderType,
		ExtraHeaders:    req.ExtraHeaders,
		SupportedModels: toDomainModelInfosPtr(req.SupportedModels),
		DefaultModel:    req.DefaultModel,
		IsDefault:       req.IsDefault,
		Priority:        req.Priority,
		AutoMerge:       req.AutoMerge,
		IsActive:        req.IsActive,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, providerToMap(provider))
}

func (h *LLMProviderHandler) DeleteProvider(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.providerService.Delete(c.Request.Context(), domain.NewLLMProviderID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func (h *LLMProviderHandler) TestConnection(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	result, err := h.providerService.TestConnection(c.Request.Context(), domain.NewLLMProviderID(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleGetProviders 根据 query 参数分发到 GetProvider 或 ListProviders
func (h *LLMProviderHandler) handleGetProviders(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetProvider(c)
		return
	}
	h.ListProviders(c)
}

func providerToMap(provider *domain.LLMProvider) map[string]interface{} {
	return map[string]interface{}{
		"id":               provider.ID().String(),
		"user_code":        provider.UserCode(),
		"provider_key":     provider.ProviderKey(),
		"provider_name":    provider.ProviderName(),
		"api_base":         provider.APIBase(),
		"provider_type":    provider.ProviderType(),
		"extra_headers":    provider.ExtraHeaders(),
		"supported_models": provider.SupportedModels(),
		"default_model":    provider.DefaultModel(),
		"is_default":       provider.IsDefault(),
		"priority":         provider.Priority(),
		"auto_merge":       provider.AutoMerge(),
		"is_active":        provider.IsActive(),
		"created_at":       provider.CreatedAt().UnixMilli(),
		"updated_at":       provider.UpdatedAt().UnixMilli(),
	}
}

func toDomainModelInfos(models []ProviderModelInfoRequest) []domain.ModelInfo {
	if len(models) == 0 {
		return nil
	}
	out := make([]domain.ModelInfo, 0, len(models))
	for _, item := range models {
		out = append(out, domain.ModelInfo{
			ID:        item.ID,
			Name:      item.Name,
			MaxTokens: item.MaxTokens,
		})
	}
	return out
}

func toDomainModelInfosPtr(models *[]ProviderModelInfoRequest) *[]domain.ModelInfo {
	if models == nil {
		return nil
	}
	out := toDomainModelInfos(*models)
	return &out
}
