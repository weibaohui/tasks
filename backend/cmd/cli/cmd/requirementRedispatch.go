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
	"github.com/weibh/taskmanager/infrastructure/config"
)

var requirementCopyDispatchCmd = &cobra.Command{
	Use:   "copy-dispatch <requirement_id>",
	Short: "复制需求并派发",
	Args:  cobra.ExactArgs(1),
	Example: `  taskmanager requirement copy-dispatch <requirement_id>`,
	Long: `复制一份新需求（新 ID），标题增加"[重新派发]"标记，
重置所有状态为新建状态，然后派发新的这份需求。`,
	Run: func(cmd *cobra.Command, args []string) {
		requirementID := args[0]

		// 从配置文件获取 token
		token := config.GetAPIToken()
		if token == "" {
			printJSONError("API token not configured, please set api.token in ~/.taskmanager/config.yaml")
			return
		}

		// 调用复制并派发 API
		reqBody := map[string]string{
			"requirement_id": requirementID,
		}
		reqJSON, err := json.Marshal(reqBody)
		if err != nil {
			printJSONError("marshal request failed: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, "POST", config.GetAPIBaseURL()+"/requirements/copy-and-dispatch", bytes.NewBuffer(reqJSON))
		if err != nil {
			printJSONError("create request failed: %v", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			printJSONError("copy-dispatch request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			printJSONError("read response body failed: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			errResult := map[string]interface{}{
				"error":  fmt.Sprintf("copy-dispatch failed: %s", resp.Status),
				"detail": string(body),
			}
			jsonBytes, err := json.Marshal(errResult)
			if err != nil {
				printJSONError("marshal error response failed: %v", err)
				return
			}
			fmt.Print(string(jsonBytes))
			return
		}

		fmt.Print(string(body))
	},
}

func registerRequirementRedispatchCommands() {
	requirementCmd.AddCommand(requirementCopyDispatchCmd)
}
