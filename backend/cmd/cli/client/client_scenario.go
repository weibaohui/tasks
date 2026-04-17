package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HeartbeatScenario struct {
	ID          string                    `json:"id"`
	Code        string                    `json:"code"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Items       []HeartbeatScenarioItem   `json:"items"`
	Enabled     bool                      `json:"enabled"`
	IsBuiltIn   bool                      `json:"is_built_in"`
	CreatedAt   int64                     `json:"created_at"`
	UpdatedAt   int64                     `json:"updated_at"`
}

type HeartbeatScenarioItem struct {
	Name            string `json:"name"`
	IntervalMinutes int    `json:"interval_minutes"`
	MDContent       string `json:"md_content"`
	AgentCode       string `json:"agent_code"`
	RequirementType string `json:"requirement_type"`
	SortOrder       int    `json:"sort_order"`
}

// ListHeartbeatScenarios 获取心跳场景列表
func (c *Client) ListHeartbeatScenarios(ctx context.Context) ([]HeartbeatScenario, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/heartbeat-scenarios", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []HeartbeatScenario
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	return result, nil
}

// ApplyHeartbeatScenario 为项目应用场景
func (c *Client) ApplyHeartbeatScenario(ctx context.Context, projectID, scenarioCode string) error {
	req := map[string]string{
		"scenario_code": scenarioCode,
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/projects/"+projectID+"/apply-scenario", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}
	return nil
}
