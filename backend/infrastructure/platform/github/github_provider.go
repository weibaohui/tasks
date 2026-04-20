package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"taskmanager/backend/domain"
)

// ExecCommand 用于创建执行命令，测试时可替换
var ExecCommand = defaultExecCommand

func defaultExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// Provider GitHub 平台 Provider 实现
type Provider struct{}

// NewProvider 创建 GitHub Provider
func NewProvider() *Provider {
	return &Provider{}
}

// PlatformType 返回平台类型
func (p *Provider) PlatformType() domain.PlatformType {
	return domain.PlatformTypeGitHub
}

// NormalizeRepo 将仓库URL或路径转换为标准格式 (owner/repo)
func (p *Provider) NormalizeRepo(repo string) string {
	if strings.HasPrefix(repo, "git@github.com:") {
		repo = strings.TrimPrefix(repo, "git@github.com:")
	}
	if strings.HasPrefix(repo, "https://github.com/") {
		repo = strings.TrimPrefix(repo, "https://github.com/")
	}
	return strings.TrimSuffix(repo, ".git")
}

// CreateWebhook 创建 GitHub webhook，返回 webhook ID
func (p *Provider) CreateWebhook(ctx context.Context, repo, webhookURL string) (int64, error) {
	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"*"},
		"config": map[string]interface{}{
			"url":          webhookURL,
			"content_type": "json",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := ExecCommand("gh", "api", fmt.Sprintf("repos/%s/hooks", repo), "-X", "POST", "--input", "-")
	cmd.Stdin = bytes.NewReader(payloadBytes)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook: %w (stderr: %s)", err, stderr.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if id, ok := response["id"].(float64); ok {
		return int64(id), nil
	}

	return 0, fmt.Errorf("webhook id not found in response")
}

// DeleteWebhook 删除 GitHub webhook
func (p *Provider) DeleteWebhook(ctx context.Context, repo string, webhookID int64) error {
	cmd := ExecCommand("gh", "api", fmt.Sprintf("repos/%s/hooks/%d", repo, webhookID), "-X", "DELETE")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// UpdateWebhookURL 更新 webhook 的 URL
func (p *Provider) UpdateWebhookURL(ctx context.Context, repo string, webhookID int64, newURL string) error {
	payload := map[string]interface{}{
		"config": map[string]interface{}{
			"url": newURL,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := ExecCommand("gh", "api", fmt.Sprintf("repos/%s/hooks/%d", repo, webhookID), "-X", "PATCH", "--input", "-")
	cmd.Stdin = bytes.NewReader(payloadBytes)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// FindExistingWebhook 查找是否已存在 webhook，返回 webhook ID（未找到返回0）
func (p *Provider) FindExistingWebhook(ctx context.Context, repo string) (int64, error) {
	cmd := ExecCommand("gh", "api", fmt.Sprintf("repos/%s/hooks", repo), "--jq", "[.[] | select(.name == \"web\")] | .[0].id")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return 0, nil
	}

	idStr := strings.TrimSpace(out.String())
	if idStr == "" || idStr == "null" {
		return 0, nil
	}

	var id int64
	fmt.Sscanf(idStr, "%d", &id)
	return id, nil
}

// GetWebhookURL 获取 webhook 的当前配置 URL
func (p *Provider) GetWebhookURL(ctx context.Context, repo string, webhookID int64) (string, error) {
	cmd := ExecCommand("gh", "api", fmt.Sprintf("repos/%s/hooks/%d", repo, webhookID), "--jq", ".config.url")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get webhook URL: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

// ListOpenPRs 列出所有 open 的 PR
func (p *Provider) ListOpenPRs(ctx context.Context, repo string) ([]domain.PRInfo, error) {
	cmd := ExecCommand("gh", "pr", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,title,author,body,url,headRefName,baseRefName",
		"-q", ".")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w (stderr: %s)", err, stderr.String())
	}

	var prs []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		Author      struct {
			Login string `json:"login"`
		} `json:"author"`
		Body       string `json:"body"`
		URL        string `json:"url"`
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
	}

	if err := json.Unmarshal(out.Bytes(), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PRs: %w", err)
	}

	result := make([]domain.PRInfo, len(prs))
	for i, pr := range prs {
		result[i] = domain.PRInfo{
			Number:  pr.Number,
			Title:   pr.Title,
			Author:  pr.Author.Login,
			Body:    pr.Body,
			URL:     pr.URL,
			HeadRef: pr.HeadRefName,
			BaseRef: pr.BaseRefName,
			State:   "open",
		}
	}

	return result, nil
}

// GetPR 获取 PR 详情
func (p *Provider) GetPR(ctx context.Context, repo string, prNumber int) (*domain.PRDetail, error) {
	cmd := ExecCommand("gh", "pr", "view",
		fmt.Sprintf("%s#%d", repo, prNumber),
		"--json", "title,body,author,state,createdAt,headRefName,baseRefName,additions,deletions,changedFiles,comments",
		"-q", ".")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w (stderr: %s)", err, stderr.String())
	}

	var pr struct {
		Number        int    `json:"number"`
		Title         string `json:"title"`
		Author        struct {
			Login string `json:"login"`
		} `json:"author"`
		Body          string `json:"body"`
		State         string `json:"state"`
		CreatedAt     string `json:"createdAt"`
		HeadRefName   string `json:"headRefName"`
		BaseRefName   string `json:"baseRefName"`
		Additions     int    `json:"additions"`
		Deletions     int    `json:"deletions"`
		ChangedFiles  int    `json:"changedFiles"`
		Comments      []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"comments"`
	}

	if err := json.Unmarshal(out.Bytes(), &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	comments := make([]domain.CommentInfo, len(pr.Comments))
	for i, c := range pr.Comments {
		comments[i] = domain.CommentInfo{
			Author: c.Author.Login,
			Body:   c.Body,
		}
	}

	return &domain.PRDetail{
		PRInfo: domain.PRInfo{
			Number:  pr.Number,
			Title:   pr.Title,
			Author:  pr.Author.Login,
			Body:    pr.Body,
			State:   pr.State,
			HeadRef: pr.HeadRefName,
			BaseRef: pr.BaseRefName,
		},
		Mergeable:    true, // gh pr view 不直接返回 mergeable，需要单独查询
		Comments:    comments,
		Additions:   pr.Additions,
		Deletions:   pr.Deletions,
		ChangedFiles: pr.ChangedFiles,
	}, nil
}

// CreatePR 创建 PR
func (p *Provider) CreatePR(ctx context.Context, repo string, opts domain.CreatePROptions) error {
	body := opts.Body
	if opts.IssueNumber > 0 {
		if body != "" {
			body += "\n\n"
		}
		body += fmt.Sprintf("Closes #%d", opts.IssueNumber)
	}

	args := []string{"pr", "create", "--repo", repo, "--title", opts.Title, "--body", body, "--head", opts.Head, "--base", opts.Base}
	cmd := ExecCommand("gh", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create PR: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// CommentPR 在 PR 下评论
func (p *Provider) CommentPR(ctx context.Context, repo string, prNumber int, body string) error {
	cmd := ExecCommand("gh", "pr", "comment", fmt.Sprintf("%s#%d", repo, prNumber), "--body", body)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to comment PR: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// MergePR 合并 PR
func (p *Provider) MergePR(ctx context.Context, repo string, prNumber int) error {
	cmd := ExecCommand("gh", "pr", "merge", fmt.Sprintf("%s#%d", repo, prNumber), "--squash", "--delete-branch")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to merge PR: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// GetPRChecks 获取 PR 的 CI 检查状态
func (p *Provider) GetPRChecks(ctx context.Context, repo string, prNumber int) ([]domain.CheckInfo, error) {
	cmd := ExecCommand("gh", "pr", "checks", fmt.Sprintf("%s#%d", repo, prNumber), "--json", "id,name,status,state,conclusion,url")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR checks: %w (stderr: %s)", err, stderr.String())
	}

	var checks []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		State      string `json:"state"`
		Conclusion string `json:"conclusion"`
		URL        string `json:"url"`
	}

	if err := json.Unmarshal(out.Bytes(), &checks); err != nil {
		return nil, fmt.Errorf("failed to parse checks: %w", err)
	}

	result := make([]domain.CheckInfo, len(checks))
	for i, c := range checks {
		result[i] = domain.CheckInfo{
			ID:         c.ID,
			Name:       c.Name,
			Status:     c.Status,
			State:      c.State,
			Conclusion: c.Conclusion,
			URL:        c.URL,
		}
	}

	return result, nil
}

// GetPRDiff 获取 PR 的 diff
func (p *Provider) GetPRDiff(ctx context.Context, repo string, prNumber int) (string, error) {
	cmd := ExecCommand("gh", "pr", "diff", fmt.Sprintf("%s#%d", repo, prNumber))

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w (stderr: %s)", err, stderr.String())
	}

	return out.String(), nil
}

// ListOpenIssues 列出所有 open 的 Issue
func (p *Provider) ListOpenIssues(ctx context.Context, repo string) ([]domain.IssueInfo, error) {
	cmd := ExecCommand("gh", "issue", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,title,body,author,url,labels",
		"-q", ".")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w (stderr: %s)", err, stderr.String())
	}

	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		URL    string `json:"url"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	if err := json.Unmarshal(out.Bytes(), &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	result := make([]domain.IssueInfo, len(issues))
	for i, issue := range issues {
		labels := make([]string, len(issue.Labels))
		for j, l := range issue.Labels {
			labels[j] = l.Name
		}
		result[i] = domain.IssueInfo{
			Number: issue.Number,
			Title:  issue.Title,
			Body:   issue.Body,
			Author: issue.Author.Login,
			URL:    issue.URL,
			State:  "open",
			Labels: labels,
		}
	}

	return result, nil
}

// GetIssue 获取 Issue 详情
func (p *Provider) GetIssue(ctx context.Context, repo string, issueNumber int) (*domain.IssueDetail, error) {
	cmd := ExecCommand("gh", "issue", "view",
		fmt.Sprintf("%s#%d", repo, issueNumber),
		"--json", "title,body,author,state,createdAt,comments",
		"-q", ".")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w (stderr: %s)", err, stderr.String())
	}

	var issue struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Author    struct {
			Login string `json:"login"`
		} `json:"author"`
		State     string `json:"state"`
		CreatedAt string `json:"createdAt"`
		Comments  []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"comments"`
	}

	if err := json.Unmarshal(out.Bytes(), &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	comments := make([]domain.CommentInfo, len(issue.Comments))
	for i, c := range issue.Comments {
		comments[i] = domain.CommentInfo{
			Author: c.Author.Login,
			Body:   c.Body,
		}
	}

	return &domain.IssueDetail{
		IssueInfo: domain.IssueInfo{
			Number: issue.Number,
			Title:  issue.Title,
			Body:   issue.Body,
			Author: issue.Author.Login,
			State:  issue.State,
		},
		Comments: comments,
	}, nil
}

// CommentIssue 在 Issue 下评论
func (p *Provider) CommentIssue(ctx context.Context, repo string, issueNumber int, body string) error {
	cmd := ExecCommand("gh", "issue", "comment", fmt.Sprintf("%s#%d", repo, issueNumber), "--body", body)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to comment issue: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// ListIssueComments 列出 Issue 的所有评论
func (p *Provider) ListIssueComments(ctx context.Context, repo string, issueNumber int) ([]domain.CommentInfo, error) {
	cmd := ExecCommand("gh", "api",
		fmt.Sprintf("repos/%s/issues/%d/comments", repo, issueNumber),
		"--jq", ".[] | {author: .user.login, body: .body, createdAt: .created_at}",
		"-q", ".")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list issue comments: %w (stderr: %s)", err, stderr.String())
	}

	var rawComments []struct {
		Author    string `json:"author"`
		Body      string `json:"body"`
		CreatedAt string `json:"createdAt"`
	}

	if err := json.Unmarshal(out.Bytes(), &rawComments); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	comments := make([]domain.CommentInfo, len(rawComments))
	for i, c := range rawComments {
		comments[i] = domain.CommentInfo{
			Author: c.Author,
			Body:   c.Body,
		}
	}

	return comments, nil
}

// ListWorkflowRuns 列出工作流运行
func (p *Provider) ListWorkflowRuns(ctx context.Context, repo string, limit int) ([]domain.WorkflowRunInfo, error) {
	cmd := ExecCommand("gh", "run", "list",
		"--repo", repo,
		"--limit", fmt.Sprintf("%d", limit),
		"--json", "id,name,status,conclusion,headBranch,headSHA,createdAt",
		"-q", ".")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w (stderr: %s)", err, stderr.String())
	}

	var runs []struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HeadBranch string `json:"headBranch"`
		HeadSHA    string `json:"headSHA"`
		CreatedAt  string `json:"createdAt"`
	}

	if err := json.Unmarshal(out.Bytes(), &runs); err != nil {
		return nil, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	result := make([]domain.WorkflowRunInfo, len(runs))
	for i, r := range runs {
		createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
		result[i] = domain.WorkflowRunInfo{
			ID:         r.ID,
			Name:       r.Name,
			Status:     r.Status,
			Conclusion: r.Conclusion,
			HeadBranch: r.HeadBranch,
			HeadSHA:    r.HeadSHA,
			CreatedAt:  createdAt,
		}
	}

	return result, nil
}

// GetWorkflowRun 获取工作流运行详情
func (p *Provider) GetWorkflowRun(ctx context.Context, repo string, runID int64) (*domain.WorkflowRunDetail, error) {
	cmd := ExecCommand("gh", "run", "view",
		fmt.Sprintf("%d", runID),
		"--repo", repo,
		"--json", "id,name,status,conclusion,headBranch,headSHA,createdAt,jobs",
		"-q", ".")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w (stderr: %s)", err, stderr.String())
	}

	var run struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HeadBranch string `json:"headBranch"`
		HeadSHA    string `json:"headSHA"`
		CreatedAt  string `json:"createdAt"`
		Jobs       []struct {
			ID          int64   `json:"id"`
			Name        string  `json:"name"`
			Status      string  `json:"status"`
			Conclusion  string  `json:"conclusion"`
			StartedAt   string  `json:"startedAt"`
			CompletedAt *string `json:"completedAt"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(out.Bytes(), &run); err != nil {
		return nil, fmt.Errorf("failed to parse workflow run: %w", err)
	}

	jobs := make([]domain.WorkflowJobInfo, len(run.Jobs))
	for i, j := range run.Jobs {
		startedAt, _ := time.Parse(time.RFC3339, j.StartedAt)
		var completedAt *time.Time
		if j.CompletedAt != nil {
			t, _ := time.Parse(time.RFC3339, *j.CompletedAt)
			completedAt = &t
		}
		jobs[i] = domain.WorkflowJobInfo{
			ID:          j.ID,
			Name:        j.Name,
			Status:      j.Status,
			Conclusion:  j.Conclusion,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
		}
	}

	createdAt, _ := time.Parse(time.RFC3339, run.CreatedAt)

	return &domain.WorkflowRunDetail{
		WorkflowRunInfo: domain.WorkflowRunInfo{
			ID:         run.ID,
			Name:       run.Name,
			Status:     run.Status,
			Conclusion: run.Conclusion,
			HeadBranch: run.HeadBranch,
			HeadSHA:    run.HeadSHA,
			CreatedAt:  createdAt,
		},
		Jobs: jobs,
	}, nil
}

// GetWorkflowRunLogs 获取工作流失败的日志
func (p *Provider) GetWorkflowRunLogs(ctx context.Context, repo string, runID int64) (string, error) {
	cmd := ExecCommand("gh", "run", "view", fmt.Sprintf("%d", runID), "--repo", repo, "--log-failed")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get workflow run logs: %w (stderr: %s)", err, stderr.String())
	}

	return out.String(), nil
}
