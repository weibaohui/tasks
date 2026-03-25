/**
 * LLM Provider 相关类型定义
 */

export interface ProviderModelInfo {
  id: string;
  name: string;
  max_tokens: number;
}

export interface ProviderEmbeddingModelInfo {
  id: string;
  name: string;
  dimensions: number;
}

export interface LLMProvider {
  id: string;
  user_code: string;
  provider_key: string;
  provider_name: string;
  api_base: string;
  api_type: string;
  extra_headers: Record<string, string>;
  supported_models: ProviderModelInfo[];
  default_model: string;
  is_default: boolean;
  priority: number;
  auto_merge: boolean;
  embedding_models: ProviderEmbeddingModelInfo[];
  default_embedding_model: string;
  is_active: boolean;
  created_at: number;
  updated_at: number;
}

export interface CreateProviderRequest {
  user_code: string;
  provider_key: string;
  provider_name: string;
  api_key: string;
  api_base: string;
  api_type: string;
  extra_headers: Record<string, string>;
  supported_models: ProviderModelInfo[];
  default_model: string;
  is_default: boolean;
  priority: number;
  auto_merge?: boolean;
}

export interface UpdateProviderRequest {
  provider_key?: string;
  provider_name?: string;
  api_key?: string;
  api_base?: string;
  api_type?: string;
  extra_headers?: Record<string, string>;
  supported_models?: ProviderModelInfo[];
  default_model?: string;
  is_default?: boolean;
  priority?: number;
  auto_merge?: boolean;
  is_active?: boolean;
  embedding_models?: ProviderEmbeddingModelInfo[];
  default_embedding_model?: string;
}

export interface TestProviderResult {
  success: boolean;
  message: string;
  details?: Record<string, unknown>;
}
