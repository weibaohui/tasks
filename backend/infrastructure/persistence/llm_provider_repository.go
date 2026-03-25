package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteLLMProviderRepository struct {
	db *sql.DB
}

func NewSQLiteLLMProviderRepository(db *sql.DB) *SQLiteLLMProviderRepository {
	return &SQLiteLLMProviderRepository{db: db}
}

func (r *SQLiteLLMProviderRepository) Save(ctx context.Context, provider *domain.LLMProvider) error {
	snap := provider.ToSnapshot()
	extraHeaders, _ := json.Marshal(snap.ExtraHeaders)
	supportedModels, _ := json.Marshal(snap.SupportedModels)
	embeddingModels, _ := json.Marshal(snap.EmbeddingModels)

	query := `
		INSERT INTO llm_providers (
			id, user_code, provider_key, provider_name, api_key, api_base, provider_type, extra_headers,
			supported_models, default_model, is_default, priority, auto_merge,
			embedding_models, default_embedding_model, is_active, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider_key=excluded.provider_key,
			provider_name=excluded.provider_name,
			api_key=excluded.api_key,
			api_base=excluded.api_base,
			provider_type=excluded.provider_type,
			extra_headers=excluded.extra_headers,
			supported_models=excluded.supported_models,
			default_model=excluded.default_model,
			is_default=excluded.is_default,
			priority=excluded.priority,
			auto_merge=excluded.auto_merge,
			embedding_models=excluded.embedding_models,
			default_embedding_model=excluded.default_embedding_model,
			is_active=excluded.is_active,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.UserCode,
		snap.ProviderKey,
		snap.ProviderName,
		snap.APIKey,
		snap.APIBase,
		snap.ProviderType,
		extraHeaders,
		supportedModels,
		snap.DefaultModel,
		boolToInt(snap.IsDefault),
		snap.Priority,
		boolToInt(snap.AutoMerge),
		embeddingModels,
		snap.DefaultEmbeddingModel,
		boolToInt(snap.IsActive),
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteLLMProviderRepository) FindByID(ctx context.Context, id domain.LLMProviderID) (*domain.LLMProvider, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_code, provider_key,
		COALESCE(provider_name, '') as provider_name,
		COALESCE(api_key, '') as api_key,
		COALESCE(api_base, '') as api_base,
		COALESCE(provider_type, 'openai') as provider_type,
		COALESCE(extra_headers, '{}') as extra_headers,
		COALESCE(supported_models, '[]') as supported_models,
		COALESCE(default_model, '') as default_model,
		is_default, priority, auto_merge,
		COALESCE(embedding_models, '[]') as embedding_models,
		COALESCE(default_embedding_model, '') as default_embedding_model,
		is_active, created_at, updated_at
		FROM llm_providers WHERE id = ?`, id.String())
	return scanProvider(row)
}

func (r *SQLiteLLMProviderRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.LLMProvider, error) {
	query := `SELECT id, user_code, provider_key,
		COALESCE(provider_name, '') as provider_name,
		COALESCE(api_key, '') as api_key,
		COALESCE(api_base, '') as api_base,
		COALESCE(provider_type, 'openai') as provider_type,
		COALESCE(extra_headers, '{}') as extra_headers,
		COALESCE(supported_models, '[]') as supported_models,
		COALESCE(default_model, '') as default_model,
		is_default, priority, auto_merge,
		COALESCE(embedding_models, '[]') as embedding_models,
		COALESCE(default_embedding_model, '') as default_embedding_model,
		is_active, created_at, updated_at
		FROM llm_providers WHERE user_code = ? ORDER BY priority DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *SQLiteLLMProviderRepository) FindDefaultActive(ctx context.Context, userCode string) (*domain.LLMProvider, error) {
	query := `SELECT id, user_code, provider_key,
		COALESCE(provider_name, '') as provider_name,
		COALESCE(api_key, '') as api_key,
		COALESCE(api_base, '') as api_base,
		COALESCE(provider_type, 'openai') as provider_type,
		COALESCE(extra_headers, '{}') as extra_headers,
		COALESCE(supported_models, '[]') as supported_models,
		COALESCE(default_model, '') as default_model,
		is_default, priority, auto_merge,
		COALESCE(embedding_models, '[]') as embedding_models,
		COALESCE(default_embedding_model, '') as default_embedding_model,
		is_active, created_at, updated_at
		FROM llm_providers WHERE user_code = ? AND is_default = 1 AND is_active = 1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, userCode)
	return scanProvider(row)
}

func (r *SQLiteLLMProviderRepository) ClearDefaultByUserCode(ctx context.Context, userCode string, excludeID *domain.LLMProviderID) error {
	query := `UPDATE llm_providers SET is_default = 0 WHERE user_code = ?`
	args := []interface{}{userCode}
	if excludeID != nil {
		query += ` AND id != ?`
		args = append(args, excludeID.String())
	}
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *SQLiteLLMProviderRepository) Delete(ctx context.Context, id domain.LLMProviderID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM llm_providers WHERE id = ?`, id.String())
	return err
}

func scanProviders(rows *sql.Rows) ([]*domain.LLMProvider, error) {
	providers := make([]*domain.LLMProvider, 0)
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		if provider != nil {
			providers = append(providers, provider)
		}
	}
	return providers, rows.Err()
}

func scanProvider(scanner rowScanner) (*domain.LLMProvider, error) {
	var (
		idStr                 string
		userCode              string
		providerKey           string
		providerName          string
		apiKey                string
		apiBase               string
		providerType          string
		extraHeadersJSON      []byte
		supportedModelsJSON   []byte
		defaultModel          string
		isDefaultInt          int
		priority              int
		autoMergeInt          int
		embeddingModelsJSON   []byte
		defaultEmbeddingModel string
		isActiveInt           int
		createdAtUnix         int64
		updatedAtUnix         int64
	)

	err := scanner.Scan(
		&idStr,
		&userCode,
		&providerKey,
		&providerName,
		&apiKey,
		&apiBase,
		&providerType,
		&extraHeadersJSON,
		&supportedModelsJSON,
		&defaultModel,
		&isDefaultInt,
		&priority,
		&autoMergeInt,
		&embeddingModelsJSON,
		&defaultEmbeddingModel,
		&isActiveInt,
		&createdAtUnix,
		&updatedAtUnix,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	extraHeaders := map[string]string{}
	_ = json.Unmarshal(extraHeadersJSON, &extraHeaders)
	var supportedModels []domain.ModelInfo
	_ = json.Unmarshal(supportedModelsJSON, &supportedModels)
	var embeddingModels []domain.EmbeddingModelInfo
	_ = json.Unmarshal(embeddingModelsJSON, &embeddingModels)

	provider := &domain.LLMProvider{}
	provider.FromSnapshot(domain.LLMProviderSnapshot{
		ID:                    domain.NewLLMProviderID(idStr),
		UserCode:              userCode,
		ProviderKey:           providerKey,
		ProviderName:          providerName,
		APIKey:                apiKey,
		APIBase:               apiBase,
		ProviderType:          providerType,
		ExtraHeaders:          extraHeaders,
		SupportedModels:       supportedModels,
		DefaultModel:          defaultModel,
		IsDefault:             isDefaultInt == 1,
		Priority:              priority,
		AutoMerge:             autoMergeInt == 1,
		EmbeddingModels:       embeddingModels,
		DefaultEmbeddingModel: defaultEmbeddingModel,
		IsActive:              isActiveInt == 1,
		CreatedAt:             time.Unix(createdAtUnix, 0),
		UpdatedAt:             time.Unix(updatedAtUnix, 0),
	})
	return provider, nil
}
