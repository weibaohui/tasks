package atomgit

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain"
)

// Provider AtomGit 平台 Provider 实现
// 注意：AtomGit API 可能与 GitHub 有所不同，需要根据实际情况调整实现
type Provider struct {
	// TODO: 添加必要的配置，如 API endpoint、认证信息等
}

// NewProvider 创建 AtomGit Provider
func NewProvider() *Provider {
	return &Provider{}
}

// PlatformType 返回平台类型
func (p *Provider) PlatformType() domain.PlatformType {
	return domain.PlatformTypeAtomGit
}

// NormalizeRepo 将仓库URL或路径转换为标准格式 (owner/repo)
// AtomGit 仓库格式可能是：
//   - git@gitcode.com:owner/repo
//   - https://gitcode.com/owner/repo
func (p *Provider) NormalizeRepo(repo string) string {
	if strings.HasPrefix(repo, "git@gitcode.com:") {
		repo = strings.TrimPrefix(repo, "git@gitcode.com:")
	}
	if strings.HasPrefix(repo, "https://gitcode.com/") {
		repo = strings.TrimPrefix(repo, "https://gitcode.com/")
	}
	return strings.TrimSuffix(repo, ".git")
}

// CreateWebhook 创建 AtomGit webhook，返回 webhook ID
func (p *Provider) CreateWebhook(ctx context.Context, repo, webhookURL string) (int64, error) {
	// TODO: 实现 AtomGit webhook 创建
	// 需要确认 AtomGit 的 API 端点和请求格式
	return 0, fmt.Errorf("not implemented: AtomGit CreateWebhook")
}

// DeleteWebhook 删除 AtomGit webhook
func (p *Provider) DeleteWebhook(ctx context.Context, repo string, webhookID int64) error {
	// TODO: 实现 AtomGit webhook 删除
	return fmt.Errorf("not implemented: AtomGit DeleteWebhook")
}

// UpdateWebhookURL 更新 webhook 的 URL
func (p *Provider) UpdateWebhookURL(ctx context.Context, repo string, webhookID int64, newURL string) error {
	// TODO: 实现 AtomGit webhook URL 更新
	return fmt.Errorf("not implemented: AtomGit UpdateWebhookURL")
}

// FindExistingWebhook 查找是否已存在 webhook，返回 webhook ID（未找到返回0）
func (p *Provider) FindExistingWebhook(ctx context.Context, repo string) (int64, error) {
	// TODO: 实现 AtomGit webhook 查询
	return 0, fmt.Errorf("not implemented: AtomGit FindExistingWebhook")
}

// GetWebhookURL 获取 webhook 的当前配置 URL
func (p *Provider) GetWebhookURL(ctx context.Context, repo string, webhookID int64) (string, error) {
	// TODO: 实现 AtomGit webhook URL 获取
	return "", fmt.Errorf("not implemented: AtomGit GetWebhookURL")
}

// ListOpenPRs 列出所有 open 的 PR
func (p *Provider) ListOpenPRs(ctx context.Context, repo string) ([]domain.PRInfo, error) {
	// TODO: 实现 AtomGit PR 列表
	// AtomGit 可能使用 glab 或其他 CLI 工具
	return nil, fmt.Errorf("not implemented: AtomGit ListOpenPRs")
}

// GetPR 获取 PR 详情
func (p *Provider) GetPR(ctx context.Context, repo string, prNumber int) (*domain.PRDetail, error) {
	// TODO: 实现 AtomGit PR 获取
	return nil, fmt.Errorf("not implemented: AtomGit GetPR")
}

// CreatePR 创建 PR
func (p *Provider) CreatePR(ctx context.Context, repo string, opts domain.CreatePROptions) error {
	// TODO: 实现 AtomGit PR 创建
	return fmt.Errorf("not implemented: AtomGit CreatePR")
}

// CommentPR 在 PR 下评论
func (p *Provider) CommentPR(ctx context.Context, repo string, prNumber int, body string) error {
	// TODO: 实现 AtomGit PR 评论
	return fmt.Errorf("not implemented: AtomGit CommentPR")
}

// MergePR 合并 PR
func (p *Provider) MergePR(ctx context.Context, repo string, prNumber int) error {
	// TODO: 实现 AtomGit PR 合并
	return fmt.Errorf("not implemented: AtomGit MergePR")
}

// GetPRChecks 获取 PR 的 CI 检查状态
func (p *Provider) GetPRChecks(ctx context.Context, repo string, prNumber int) ([]domain.CheckInfo, error) {
	// TODO: 实现 AtomGit PR 检查状态
	return nil, fmt.Errorf("not implemented: AtomGit GetPRChecks")
}

// GetPRDiff 获取 PR 的 diff
func (p *Provider) GetPRDiff(ctx context.Context, repo string, prNumber int) (string, error) {
	// TODO: 实现 AtomGit PR diff
	return "", fmt.Errorf("not implemented: AtomGit GetPRDiff")
}

// ListOpenIssues 列出所有 open 的 Issue
func (p *Provider) ListOpenIssues(ctx context.Context, repo string) ([]domain.IssueInfo, error) {
	// TODO: 实现 AtomGit Issue 列表
	return nil, fmt.Errorf("not implemented: AtomGit ListOpenIssues")
}

// GetIssue 获取 Issue 详情
func (p *Provider) GetIssue(ctx context.Context, repo string, issueNumber int) (*domain.IssueDetail, error) {
	// TODO: 实现 AtomGit Issue 获取
	return nil, fmt.Errorf("not implemented: AtomGit GetIssue")
}

// CommentIssue 在 Issue 下评论
func (p *Provider) CommentIssue(ctx context.Context, repo string, issueNumber int, body string) error {
	// TODO: 实现 AtomGit Issue 评论
	return fmt.Errorf("not implemented: AtomGit CommentIssue")
}

// ListIssueComments 列出 Issue 的所有评论
func (p *Provider) ListIssueComments(ctx context.Context, repo string, issueNumber int) ([]domain.CommentInfo, error) {
	// TODO: 实现 AtomGit Issue 评论列表
	return nil, fmt.Errorf("not implemented: AtomGit ListIssueComments")
}

// ListWorkflowRuns 列出工作流运行
func (p *Provider) ListWorkflowRuns(ctx context.Context, repo string, limit int) ([]domain.WorkflowRunInfo, error) {
	// TODO: 实现 AtomGit 工作流运行列表
	return nil, fmt.Errorf("not implemented: AtomGit ListWorkflowRuns")
}

// GetWorkflowRun 获取工作流运行详情
func (p *Provider) GetWorkflowRun(ctx context.Context, repo string, runID int64) (*domain.WorkflowRunDetail, error) {
	// TODO: 实现 AtomGit 工作流运行详情
	return nil, fmt.Errorf("not implemented: AtomGit GetWorkflowRun")
}

// GetWorkflowRunLogs 获取工作流失败的日志
func (p *Provider) GetWorkflowRunLogs(ctx context.Context, repo string, runID int64) (string, error) {
	// TODO: 实现 AtomGit 工作流日志获取
	return "", fmt.Errorf("not implemented: AtomGit GetWorkflowRunLogs")
}
