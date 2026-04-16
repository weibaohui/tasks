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
	"github.com/weibh/taskmanager/cmd/cli/client"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var requirementDispatchCmd = &cobra.Command{
	Use:   "dispatch <requirement_id>",
	Short: "派发需求",
	Args:  cobra.ExactArgs(1),
	Example: `  taskmanager requirement dispatch <requirement_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		requirementID := args[0]

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 获取需求详情
		c := client.New()
		req, err := c.GetRequirement(ctx, requirementID)
		if err != nil {
			fmt.Printf(`{"error":"find requirement failed: %v"}`, err)
			return
		}

		// 查找项目获取派发配置
		project, err := c.GetProject(ctx, req.ProjectID)
		if err != nil {
			fmt.Printf(`{"error":"find project failed: %v"}`, err)
			return
		}

		agentCode := req.AssigneeAgentCode
		if agentCode == "" {
			agentCode = req.ReplicaAgentCode
		}
		// 如果需求没有指定 agent，使用项目的默认 agent
		if agentCode == "" {
			agentCode = project.DefaultAgentCode
		}
		channelCode := project.DispatchChannelCode
		sessionKey := project.DispatchSessionKey

		if agentCode == "" || sessionKey == "" {
			result, _ := json.Marshal(map[string]string{
				"error":       "project dispatch not configured",
				"project_id":  project.ID,
				"project":     project.Name,
				"agent_code":  agentCode,
				"session_key": sessionKey,
			})
			fmt.Print(string(result))
			return
		}

		if channelCode == "" {
			channelCode = "feishu"
		}

		// 从配置文件获取 token
		token := config.GetAPIToken()
		if token == "" {
			fmt.Print(`{"error":"API token not configured, please set api.token in ~/.taskmanager/config.yaml"}`)
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

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(httpReq)
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

func registerRequirementDispatchCommands() {
	// 不再需要 username/password 参数，使用配置文件中的 token
}
