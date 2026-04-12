package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) GetAgent(ctx context.Context, id string) (*Agent, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/agents?id="+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// CreateAgentAPIRequest 创建 Agent 请求
type CreateAgentAPIRequest struct {
	UserCode              string   `json:"user_code"`
	Name                  string   `json:"name"`
	AgentType             string   `json:"agent_type"`
	Description           string   `json:"description"`
	IdentityContent       string   `json:"identity_content"`
	SoulContent           string   `json:"soul_content"`
	AgentsContent         string   `json:"agents_content"`
	UserContent           string   `json:"user_content"`
	ToolsContent          string   `json:"tools_content"`
	Model                 string   `json:"model"`
	LLMProviderID         string   `json:"llm_provider_id"`
	MaxTokens             int      `json:"max_tokens"`
	Temperature           float64  `json:"temperature"`
	MaxIterations         int      `json:"max_iterations"`
	HistoryMessages       int      `json:"history_messages"`
	SkillsList            []string `json:"skills_list"`
	ToolsList             []string `json:"tools_list"`
	IsDefault             bool     `json:"is_default"`
	EnableThinkingProcess bool     `json:"enable_thinking_process"`
}

// CreateAgent 创建 Agent
func (c *Client) CreateAgent(ctx context.Context, req CreateAgentAPIRequest) (*Agent, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/agents", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateAgentAPIRequest 更新 Agent 请求
type UpdateAgentAPIRequest struct {
	Name                  *string  `json:"name,omitempty"`
	AgentType             *string  `json:"agent_type,omitempty"`
	Description           *string  `json:"description,omitempty"`
	IdentityContent       *string  `json:"identity_content,omitempty"`
	SoulContent           *string  `json:"soul_content,omitempty"`
	AgentsContent         *string  `json:"agents_content,omitempty"`
	UserContent           *string  `json:"user_content,omitempty"`
	ToolsContent          *string  `json:"tools_content,omitempty"`
	Model                 *string  `json:"model,omitempty"`
	LLMProviderID         *string  `json:"llm_provider_id,omitempty"`
	MaxTokens             *int     `json:"max_tokens,omitempty"`
	Temperature           *float64 `json:"temperature,omitempty"`
	MaxIterations         *int     `json:"max_iterations,omitempty"`
	HistoryMessages       *int     `json:"history_messages,omitempty"`
	SkillsList            []string `json:"skills_list,omitempty"`
	ToolsList             []string `json:"tools_list,omitempty"`
	IsActive              *bool    `json:"is_active,omitempty"`
	IsDefault             *bool    `json:"is_default,omitempty"`
	EnableThinkingProcess *bool    `json:"enable_thinking_process,omitempty"`
}

// UpdateAgent 更新 Agent
func (c *Client) UpdateAgent(ctx context.Context, id string, req UpdateAgentAPIRequest) (*Agent, error) {
	path := fmt.Sprintf("/agents?id=%s", id)
	resp, err := c.doRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteAgent 删除 Agent
func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	path := "/agents?id=" + id
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// ==================== Channel APIs ====================
