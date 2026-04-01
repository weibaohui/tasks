package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var requirementReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "分析PR并创建需求",
	Example: `  taskmanager requirement review <pr_url> <project_id>
  taskmanager requirement review <owner/repo> <pr_number> <project_id>
  taskmanager requirement review owner/repo 123 prj_xxx --title '修复登录bug'`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Print(`{"error":"missing required arguments"}`)
			return
		}

		title, _ := cmd.Flags().GetString("title")

		var prURL, owner, repo string
		var prNumber int

		// 解析参数
		if strings.HasPrefix(args[0], "http") {
			// 完整URL格式
			prURL = args[0]
			parts := strings.Split(strings.TrimSuffix(args[0], "/"), "/")
			// URL格式: https://github.com/owner/repo/pull/123
			if len(parts) < 5 {
				fmt.Print(`{"error":"invalid URL format"}`)
				return
			}
			owner = parts[len(parts)-4]
			repo = parts[len(parts)-3]
			// 提取 prNumber
			prNumStr := parts[len(parts)-1]
			if _, err := fmt.Sscanf(prNumStr, "%d", &prNumber); err != nil {
				fmt.Printf(`{"error":"invalid PR number: %s"}`, prNumStr)
				return
			}
		} else {
			// owner/repo 格式
			if len(args) < 3 {
				fmt.Print(`{"error":"owner/repo format requires at least 3 arguments"}`)
				return
			}
			ownerRepo := args[0]
			prNumberStr := args[1]

			parts := strings.Split(ownerRepo, "/")
			if len(parts) != 2 {
				fmt.Print(`{"error":"invalid owner/repo format"}`)
				return
			}
			owner = parts[0]
			repo = parts[1]

			if _, err := fmt.Sscanf(prNumberStr, "%d", &prNumber); err != nil {
				fmt.Printf(`{"error":"PR number must be numeric: %s"}`, prNumberStr)
				return
			}
			prURL = fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, prNumber)
		}

		projectID := args[len(args)-1]

		// 登录获取 token
		token, err := login(defaultAdminUsername, defaultAdminPassword)
		if err != nil {
			fmt.Printf(`{"error":"login failed: %v"}`, err)
			return
		}

		// 获取PR信息
		prInfo, err := fetchPRInfo(owner, repo, prNumber)
		if err != nil {
			fmt.Printf(`{"error":"fetch PR info failed: %v"}`, err)
			return
		}

		// 获取PR评论
		comments, err := fetchPRComments(owner, repo, prNumber)
		if err != nil {
			comments = []PRComment{}
		}

		// 生成需求内容
		generatedTitle := title
		if generatedTitle == "" {
			generatedTitle = fmt.Sprintf("PR #%d: %s", prNumber, prInfo.Title)
		}

		description := "## PR 信息\n\n"
		description += fmt.Sprintf("- PR: %s\n", prURL)
		description += fmt.Sprintf("- 标题: %s\n", prInfo.Title)
		description += fmt.Sprintf("- 作者: %s\n", prInfo.Author.Login)
		description += fmt.Sprintf("- 状态: %s\n", prInfo.State)
		description += fmt.Sprintf("- 创建时间: %s\n", prInfo.CreatedAt)
		description += "\n## PR 描述\n\n" + prInfo.Body + "\n\n"

		if len(comments) > 0 {
			description += "## PR 评论\n\n"
			for _, c := range comments {
				description += fmt.Sprintf("### %s (%s)\n%s\n\n", c.Author, c.CreatedAt, c.Body)
			}
		}

		// 调用创建需求 API
		reqBody := map[string]string{
			"project_id":          projectID,
			"title":               generatedTitle,
			"description":         description,
			"acceptance_criteria": "根据PR评论内容确定验收标准",
		}
		reqJSON, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements", bytes.NewBuffer(reqJSON))
		if err != nil {
			fmt.Printf(`{"error":"create request failed: %v"}`, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(`{"error":"create requirement request failed: %v"}`, err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			fmt.Printf(`{"error":"create requirement failed: %s","detail":%s}`, resp.Status, string(body))
			return
		}

		fmt.Print(string(body))
	},
}

// PRInfo PR信息
type PRInfo struct {
	Title     string
	Body      string
	Author    struct{ Login string }
	State     string
	CreatedAt string
}

// PRComment PR评论
type PRComment struct {
	Author    string
	Body      string
	CreatedAt string
}

// fetchPRInfo 获取PR信息
func fetchPRInfo(owner, repo string, prNumber int) (*PRInfo, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%s/%s#%d", owner, repo, prNumber),
		"--json", "title,body,author,state,createdAt", "-q", ".")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR info: %w", err)
	}

	var prInfo PRInfo
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}
	return &prInfo, nil
}

// fetchPRComments 获取PR评论
func fetchPRComments(owner, repo string, prNumber int) ([]PRComment, error) {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, prNumber),
		"--jq", ".[] | {author: .user.login, body: .body, createdAt: .created_at}", "-s")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR comments: %w", err)
	}

	var comments []PRComment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse PR comments: %w", err)
	}
	return comments, nil
}

func init() {
	requirementReviewCmd.Flags().StringP("title", "t", "", "需求标题 (可选)")
}