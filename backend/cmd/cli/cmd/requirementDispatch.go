package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

		ctx := context.Background()

		// 查找需求
		req, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(requirementID))
		if err != nil {
			fmt.Printf("查找需求失败: %v\n", err)
			return
		}
		if req == nil {
			fmt.Printf("需求不存在: %s\n", requirementID)
			return
		}

		// 查找项目获取派发配置
		project, err := projectRepo.FindByID(ctx, req.ProjectID())
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}
		if project == nil {
			fmt.Printf("项目不存在: %s\n", req.ProjectID().String())
			return
		}

		agentCode := project.AgentCode()
		channelCode := project.DispatchChannelCode()
		sessionKey := project.DispatchSessionKey()

		if agentCode == "" || sessionKey == "" {
			fmt.Printf("项目未配置派发信息\n")
			fmt.Printf("  项目ID: %s\n", project.ID().String())
			fmt.Printf("  项目名称: %s\n", project.Name())
			fmt.Printf("  AgentCode: %s\n", agentCode)
			fmt.Printf("  SessionKey: %s\n", sessionKey)
			return
		}

		if channelCode == "" {
			channelCode = "feishu"
		}

		// 登录获取 token
		token, err := login()
		if err != nil {
			fmt.Printf("登录失败: %v\n", err)
			return
		}

		// 调用派发 API
		reqBody := map[string]string{
			"requirement_id": requirementID,
			"agent_code":    agentCode,
			"channel_code":  channelCode,
			"session_key":   sessionKey,
		}
		reqJSON, _ := json.Marshal(reqBody)

		httpReq, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements/dispatch", bytes.NewBuffer(reqJSON))
		if err != nil {
			fmt.Printf("创建请求失败: %v\n", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			fmt.Printf("派发请求失败: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("派发需求失败: %s\n%s\n", resp.Status, string(body))
			return
		}

		var result struct {
			RequirementID   string `json:"requirement_id"`
			TaskID         string `json:"task_id"`
			WorkspacePath  string `json:"workspace_path"`
			ReplicaAgentCode string `json:"replica_agent_code"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("解析响应失败: %v\n", err)
			return
		}

		fmt.Printf("需求派发成功！\n需求ID: %s\n任务ID: %s\n工作空间: %s\n分身AgentCode: %s\n",
			result.RequirementID, result.TaskID, result.WorkspacePath, result.ReplicaAgentCode)
	},
}

func login() (string, error) {
	reqBody := map[string]string{
		"username": defaultAdminUsername,
		"password": defaultAdminPassword,
	}
	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/auth/login", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
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
