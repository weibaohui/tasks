package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type User struct {
	ID         string `json:"id"`
	UserCode   string `json:"user_code"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	IsAdmin    bool   `json:"is_admin"`
	IsActive   bool   `json:"is_active"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// ListUsers 获取用户列表
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/users", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateUserAPIRequest 创建用户请求
type CreateUserAPIRequest struct {
	UserCode   string `json:"user_code"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	IsAdmin    bool   `json:"is_admin"`
	IsActive   bool   `json:"is_active"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty"`
}

// CreateUser 创建用户
func (c *Client) CreateUser(ctx context.Context, req CreateUserAPIRequest) (*User, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/users", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateUserAPIRequest 更新用户请求
type UpdateUserAPIRequest struct {
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
	IsAdmin    *bool   `json:"is_admin,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	Email      *string `json:"email,omitempty"`
	Department *string `json:"department,omitempty"`
}

// UpdateUser 更新用户
func (c *Client) UpdateUser(ctx context.Context, id string, req UpdateUserAPIRequest) (*User, error) {
	path := fmt.Sprintf("/users?id=%s", id)
	resp, err := c.doRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteUser 删除用户
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	path := "/users?id=" + id
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

// ==================== Skill APIs ====================
