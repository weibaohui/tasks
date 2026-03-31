package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var requirementCompleteCmd = &cobra.Command{
	Use:   "complete",
	Short: "完成需求（创建 PR 后调用）",
	Example: `  taskmanager requirement complete --id <id> --pr-url <url>
  taskmanager requirement complete -i <id> -u <pr_url> -b <branch>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		prURL, _ := cmd.Flags().GetString("pr-url")
		branch, _ := cmd.Flags().GetString("branch")

		if id == "" || prURL == "" {
			fmt.Println("错误: --id 和 --pr-url 是必填参数")
			cmd.Usage()
			return
		}

		token, err := login()
		if err != nil {
			fmt.Printf("登录失败: %v\n", err)
			return
		}

		reqBody := map[string]string{
			"requirement_id": id,
			"pr_url":        prURL,
			"branch_name":   branch,
		}
		reqJSON, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements/pr", bytes.NewBuffer(reqJSON))
		if err != nil {
			fmt.Printf("创建请求失败: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("完成需求请求失败: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("完成需求失败: %s\n%s\n", resp.Status, string(body))
			return
		}

		var result struct {
			RequirementID string `json:"requirement_id"`
			PRURL        string `json:"pr_url"`
			BranchName   string `json:"branch_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("解析响应失败: %v\n", err)
			return
		}

		fmt.Printf("需求 %s 已标记为完成，PR: %s\n", result.RequirementID, result.PRURL)
	},
}

func init() {
	requirementCompleteCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementCompleteCmd.Flags().StringP("pr-url", "u", "", "PR URL (必填)")
	requirementCompleteCmd.Flags().StringP("branch", "b", "", "分支名 (可选)")
}
