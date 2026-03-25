package http

import (
	"encoding/json"
	"net/http"

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

type ProviderEmbeddingModelRequest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
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
	ProviderKey           *string                          `json:"provider_key"`
	ProviderName          *string                          `json:"provider_name"`
	APIKey                *string                          `json:"api_key"`
	APIBase               *string                          `json:"api_base"`
	ProviderType          *string                          `json:"provider_type"`
	ExtraHeaders          *map[string]string               `json:"extra_headers"`
	SupportedModels       *[]ProviderModelInfoRequest      `json:"supported_models"`
	DefaultModel          *string                          `json:"default_model"`
	IsDefault             *bool                            `json:"is_default"`
	Priority              *int                             `json:"priority"`
	AutoMerge             *bool                            `json:"auto_merge"`
	IsActive              *bool                            `json:"is_active"`
	EmbeddingModels       *[]ProviderEmbeddingModelRequest `json:"embedding_models"`
	DefaultEmbeddingModel *string                          `json:"default_embedding_model"`
}

type UpdateEmbeddingRequest struct {
	EmbeddingModels       []ProviderEmbeddingModelRequest `json:"embedding_models"`
	DefaultEmbeddingModel string                          `json:"default_embedding_model"`
}

func (h *LLMProviderHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	provider, err := h.providerService.Create(r.Context(), application.CreateProviderCommand{
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(providerToMap(provider))
}

func (h *LLMProviderHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	userCode := r.URL.Query().Get("user_code")
	if userCode == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "user_code is required"})
		return
	}

	providers, err := h.providerService.List(r.Context(), userCode)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	resp := make([]map[string]interface{}, 0, len(providers))
	for _, provider := range providers {
		resp = append(resp, providerToMap(provider))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *LLMProviderHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	provider, err := h.providerService.Get(r.Context(), domain.NewLLMProviderID(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(providerToMap(provider))
}

func (h *LLMProviderHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	var req UpdateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	provider, err := h.providerService.Update(r.Context(), application.UpdateProviderCommand{
		ID:                    domain.NewLLMProviderID(id),
		ProviderKey:           req.ProviderKey,
		ProviderName:          req.ProviderName,
		APIKey:                req.APIKey,
		APIBase:               req.APIBase,
		ProviderType:          req.ProviderType,
		ExtraHeaders:          req.ExtraHeaders,
		SupportedModels:       toDomainModelInfosPtr(req.SupportedModels),
		DefaultModel:          req.DefaultModel,
		IsDefault:             req.IsDefault,
		Priority:              req.Priority,
		AutoMerge:             req.AutoMerge,
		IsActive:              req.IsActive,
		EmbeddingModels:       toDomainEmbeddingModelsPtr(req.EmbeddingModels),
		DefaultEmbeddingModel: req.DefaultEmbeddingModel,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(providerToMap(provider))
}

func (h *LLMProviderHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.providerService.Delete(r.Context(), domain.NewLLMProviderID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (h *LLMProviderHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	result, err := h.providerService.TestConnection(r.Context(), domain.NewLLMProviderID(id))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (h *LLMProviderHandler) GetEmbeddingModels(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	provider, err := h.providerService.Get(r.Context(), domain.NewLLMProviderID(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"embedding_models":        provider.EmbeddingModels(),
		"default_embedding_model": provider.DefaultEmbeddingModel(),
		"has_embedding_models":    provider.HasEmbeddingModels(),
	})
}

func (h *LLMProviderHandler) UpdateEmbeddingModels(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	var req UpdateEmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	embeddingModels := toDomainEmbeddingModels(req.EmbeddingModels)
	defaultEmbeddingModel := req.DefaultEmbeddingModel
	provider, err := h.providerService.Update(r.Context(), application.UpdateProviderCommand{
		ID:                    domain.NewLLMProviderID(id),
		EmbeddingModels:       &embeddingModels,
		DefaultEmbeddingModel: &defaultEmbeddingModel,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(providerToMap(provider))
}

func providerToMap(provider *domain.LLMProvider) map[string]interface{} {
	return map[string]interface{}{
		"id":                      provider.ID().String(),
		"user_code":               provider.UserCode(),
		"provider_key":            provider.ProviderKey(),
		"provider_name":           provider.ProviderName(),
		"api_base":                provider.APIBase(),
		"provider_type":           provider.ProviderType(),
		"extra_headers":           provider.ExtraHeaders(),
		"supported_models":        provider.SupportedModels(),
		"default_model":           provider.DefaultModel(),
		"is_default":              provider.IsDefault(),
		"priority":                provider.Priority(),
		"auto_merge":              provider.AutoMerge(),
		"embedding_models":        provider.EmbeddingModels(),
		"default_embedding_model": provider.DefaultEmbeddingModel(),
		"is_active":               provider.IsActive(),
		"created_at":              provider.CreatedAt().UnixMilli(),
		"updated_at":              provider.UpdatedAt().UnixMilli(),
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

func toDomainEmbeddingModels(models []ProviderEmbeddingModelRequest) []domain.EmbeddingModelInfo {
	if len(models) == 0 {
		return nil
	}
	out := make([]domain.EmbeddingModelInfo, 0, len(models))
	for _, item := range models {
		out = append(out, domain.EmbeddingModelInfo{
			ID:         item.ID,
			Name:       item.Name,
			Dimensions: item.Dimensions,
		})
	}
	return out
}

func toDomainEmbeddingModelsPtr(models *[]ProviderEmbeddingModelRequest) *[]domain.EmbeddingModelInfo {
	if models == nil {
		return nil
	}
	out := toDomainEmbeddingModels(*models)
	return &out
}
