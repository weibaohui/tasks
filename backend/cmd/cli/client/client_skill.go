package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

// ListSkills 获取技能列表
func (c *Client) ListSkills(ctx context.Context) ([]Skill, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/skills", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result struct {
		Items []Skill `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result.Items, nil
}

// GetSkill 获取技能详情
func (c *Client) GetSkill(ctx context.Context, name string) (*Skill, error) {
	path := "/skills/detail?name=" + name
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Skill
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}
