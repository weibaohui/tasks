package atomgit

import (
	"context"
	"errors"
	"strings"

	"github.com/weibh/taskmanager/domain"
)

// ErrPlatformNotSupported AtomGit 平台尚未实现，用于调用方识别并跳过
var ErrPlatformNotSupported = errors.New("atomgit platform is not yet implemented, please implement before use")

// Provider AtomGit 平台 Provider 实现（stub）
// 注意：AtomGit API 尚未实现，所有操作均返回 ErrPlatformNotSupported
type Provider struct{}

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
	return 0, ErrPlatformNotSupported
}

// DeleteWebhook 删除 AtomGit webhook
func (p *Provider) DeleteWebhook(ctx context.Context, repo string, webhookID int64) error {
	return ErrPlatformNotSupported
}

// UpdateWebhookURL 更新 webhook 的 URL
func (p *Provider) UpdateWebhookURL(ctx context.Context, repo string, webhookID int64, newURL string) error {
	return ErrPlatformNotSupported
}

// FindExistingWebhook 查找是否已存在 webhook，返回 webhook ID（未找到返回0）
func (p *Provider) FindExistingWebhook(ctx context.Context, repo string) (int64, error) {
	return 0, ErrPlatformNotSupported
}

// GetWebhookURL 获取 webhook 的当前配置 URL
func (p *Provider) GetWebhookURL(ctx context.Context, repo string, webhookID int64) (string, error) {
	return "", ErrPlatformNotSupported
}

// ListOpenPRs 列出所有 open 的 PR
func (p *Provider) ListOpenPRs(ctx context.Context, repo string) ([]domain.PRInfo, error) {
	return nil, ErrPlatformNotSupported
}

// GetPR 获取 PR 详情
func (p *Provider) GetPR(ctx context.Context, repo string, prNumber int) (*domain.PRDetail, error) {
	return nil, ErrPlatformNotSupported
}

// CreatePR 创建 PR
func (p *Provider) CreatePR(ctx context.Context, repo string, opts domain.CreatePROptions) error {
	return ErrPlatformNotSupported
}

// CommentPR 在 PR 下评论
func (p *Provider) CommentPR(ctx context.Context, repo string, prNumber int, body string) error {
	return ErrPlatformNotSupported
}

// MergePR 合并 PR
func (p *Provider) MergePR(ctx context.Context, repo string, prNumber int) error {
	return ErrPlatformNotSupported
}

// GetPRChecks 获取 PR 的 CI 检查状态
func (p *Provider) GetPRChecks(ctx context.Context, repo string, prNumber int) ([]domain.CheckInfo, error) {
	return nil, ErrPlatformNotSupported
}

// GetPRDiff 获取 PR 的 diff
func (p *Provider) GetPRDiff(ctx context.Context, repo string, prNumber int) (string, error) {
	return "", ErrPlatformNotSupported
}

// ListOpenIssues 列出所有 open 的 Issue
func (p *Provider) ListOpenIssues(ctx context.Context, repo string) ([]domain.IssueInfo, error) {
	return nil, ErrPlatformNotSupported
}

// GetIssue 获取 Issue 详情
func (p *Provider) GetIssue(ctx context.Context, repo string, issueNumber int) (*domain.IssueDetail, error) {
	return nil, ErrPlatformNotSupported
}

// CommentIssue 在 Issue 下评论
func (p *Provider) CommentIssue(ctx context.Context, repo string, issueNumber int, body string) error {
	return ErrPlatformNotSupported
}

// ListIssueComments 列出 Issue 的所有评论
func (p *Provider) ListIssueComments(ctx context.Context, repo string, issueNumber int) ([]domain.CommentInfo, error) {
	return nil, ErrPlatformNotSupported
}

// ListWorkflowRuns 列出工作流运行
func (p *Provider) ListWorkflowRuns(ctx context.Context, repo string, limit int) ([]domain.WorkflowRunInfo, error) {
	return nil, ErrPlatformNotSupported
}

// GetWorkflowRun 获取工作流运行详情
func (p *Provider) GetWorkflowRun(ctx context.Context, repo string, runID int64) (*domain.WorkflowRunDetail, error) {
	return nil, ErrPlatformNotSupported
}

// GetWorkflowRunLogs 获取工作流失败的日志
func (p *Provider) GetWorkflowRunLogs(ctx context.Context, repo string, runID int64) (string, error) {
	return "", ErrPlatformNotSupported
}
