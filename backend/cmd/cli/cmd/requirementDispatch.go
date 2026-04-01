package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var requirementDispatchCmd = &cobra.Command{
	Use:   "dispatch <requirement_id>",
	Short: "派发需求",
	Args:  cobra.ExactArgs(1),
	Example: `  taskmanager requirement dispatch <requirement_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		requirementID := args[0]

		requirementRepo, projectRepo, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 查找需求
		req, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(requirementID))
		if err != nil {
			fmt.Printf(`{"error":"find requirement failed: %v"}`, err)
			return
		}
		if req == nil {
			fmt.Printf(`{"error":"requirement not found: %s"}`, requirementID)
			return
		}

		// 查找项目获取派发配置
		project, err := projectRepo.FindByID(ctx, req.ProjectID())
		if err != nil {
			fmt.Printf(`{"error":"find project failed: %v"}`, err)
			return
		}
		if project == nil {
			fmt.Printf(`{"error":"project not found: %s"}`, req.ProjectID().String())
			return
		}

		agentCode := project.AgentCode()
		channelCode := project.DispatchChannelCode()
		sessionKey := project.DispatchSessionKey()

		if agentCode == "" || sessionKey == "" {
			result, _ := json.Marshal(map[string]string{
				"error":       "project dispatch not configured",
				"project_id":  project.ID().String(),
				"project":     project.Name(),
				"agent_code":  agentCode,
				"session_key": sessionKey,
			})
			fmt.Print(string(result))
			return
		}

		if channelCode == "" {
			channelCode = "feishu"
		}

		// 登录获取 token
		token, err := login(defaultAdminUsername, defaultAdminPassword)
		if err != nil {
			fmt.Printf(`{"error":"login failed: %v"}`, err)
			return
		}

		// 调用派发 API
		reqBody := map[string]string{
			"requirement_id": requirementID,
			"agent_code":     agentCode,
			"channel_code":   channelCode,
			"session_key":    sessionKey,
		}
		reqJSON, _ := json.Marshal(reqBody)

		httpReq, err := http.NewRequestWithContext(ctx, "POST", config.GetAPIBaseURL()+"/requirements/dispatch", bytes.NewBuffer(reqJSON))
		if err != nil {
			fmt.Printf(`{"error":"create request failed: %v"}`, err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			fmt.Printf(`{"error":"dispatch request failed: %v"}`, err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			fmt.Printf(`{"error":"dispatch failed: %s","detail":%s}`, resp.Status, string(body))
			return
		}

		fmt.Print(string(body))
	},
}

func login(username, password string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqBody := map[string]string{
		"username": username,
		"password": password,
	}
	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", config.GetAPIBaseURL()+"/auth/login", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed: %s", string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func init() {
	requirementDispatchCmd.Flags().StringP("username", "u", defaultAdminUsername, "用户名")
	requirementDispatchCmd.Flags().StringP("password", "p", defaultAdminPassword, "密码")
}
