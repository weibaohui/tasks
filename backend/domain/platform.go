package domain

import (
	"context"
	"strings"
	"time"
)

// PlatformType 平台类型
type PlatformType string

const (
	PlatformTypeGitHub  PlatformType = "github"
	PlatformTypeAtomGit PlatformType = "atom_git"
)

func (t PlatformType) IsValid() bool {
	switch t {
	case PlatformTypeGitHub, PlatformTypeAtomGit:
		return true
	default:
		return false
	}
}

// DetectPlatformType 根据仓库 URL 自动检测平台类型
func DetectPlatformType(repo string) PlatformType {
	repo = strings.ToLower(repo)

	// AtomGit (gitcode.com) 优先检测
	if strings.Contains(repo, "gitcode.com") {
		return PlatformTypeAtomGit
	}

	// GitHub 检测
	if strings.Contains(repo, "github.com") {
		return PlatformTypeGitHub
	}

	// 默认返回 GitHub（向后兼容）
	return PlatformTypeGitHub
}

// PlatformProvider 平台操作接口
type PlatformProvider interface {
	// PlatformType 返回平台类型
	PlatformType() PlatformType

	// NormalizeRepo 将仓库URL或路径转换为标准格式 (owner/repo)
	NormalizeRepo(repo string) string

	// =========================================================================
	// Webhook 操作
	// =========================================================================

	// CreateWebhook 创建 webhook，返回 webhook ID
	CreateWebhook(ctx context.Context, repo, webhookURL string) (int64, error)

	// DeleteWebhook 删除 webhook
	DeleteWebhook(ctx context.Context, repo string, webhookID int64) error

	// UpdateWebhookURL 更新 webhook URL
	UpdateWebhookURL(ctx context.Context, repo string, webhookID int64, newURL string) error

	// FindExistingWebhook 查找已存在的 webhook，返回 webhook ID（未找到返回0）
	FindExistingWebhook(ctx context.Context, repo string) (int64, error)

	// GetWebhookURL 获取 webhook 当前的配置 URL
	GetWebhookURL(ctx context.Context, repo string, webhookID int64) (string, error)

	// =========================================================================
	// PR 操作
	// =========================================================================

	// ListOpenPRs 列出所有 open 的 PR
	ListOpenPRs(ctx context.Context, repo string) ([]PRInfo, error)

	// GetPR 获取 PR 详情
	GetPR(ctx context.Context, repo string, prNumber int) (*PRDetail, error)

	// CreatePR 创建 PR
	CreatePR(ctx context.Context, repo string, opts CreatePROptions) error

	// CommentPR 在 PR 下评论
	CommentPR(ctx context.Context, repo string, prNumber int, body string) error

	// MergePR 合并 PR
	MergePR(ctx context.Context, repo string, prNumber int) error

	// GetPRChecks 获取 PR 的 CI 检查状态
	GetPRChecks(ctx context.Context, repo string, prNumber int) ([]CheckInfo, error)

	// GetPRDiff 获取 PR 的 diff
	GetPRDiff(ctx context.Context, repo string, prNumber int) (string, error)

	// =========================================================================
	// Issue 操作
	// =========================================================================

	// ListOpenIssues 列出所有 open 的 Issue
	ListOpenIssues(ctx context.Context, repo string) ([]IssueInfo, error)

	// GetIssue 获取 Issue 详情
	GetIssue(ctx context.Context, repo string, issueNumber int) (*IssueDetail, error)

	// CommentIssue 在 Issue 下评论
	CommentIssue(ctx context.Context, repo string, issueNumber int, body string) error

	// ListIssueComments 列出 Issue 的所有评论
	ListIssueComments(ctx context.Context, repo string, issueNumber int) ([]CommentInfo, error)

	// =========================================================================
	// Workflow 操作
	// =========================================================================

	// ListWorkflowRuns 列出工作流运行
	ListWorkflowRuns(ctx context.Context, repo string, limit int) ([]WorkflowRunInfo, error)

	// GetWorkflowRun 获取工作流运行详情
	GetWorkflowRun(ctx context.Context, repo string, runID int64) (*WorkflowRunDetail, error)

	// GetWorkflowRunLogs 获取工作流失败的日志
	GetWorkflowRunLogs(ctx context.Context, repo string, runID int64) (string, error)
}

// =========================================================================
// 数据结构
// =========================================================================

// PRInfo PR 概要信息
type PRInfo struct {
	Number  int
	Title   string
	Author  string
	Body    string
	URL     string
	State   string
	HeadRef string
	BaseRef string
}

// PRDetail PR 详细信息
type PRDetail struct {
	PRInfo
	Mergeable   bool
	Comments    []CommentInfo
	Additions   int
	Deletions   int
	ChangedFiles int
}

// CreatePROptions 创建 PR 的选项
type CreatePROptions struct {
	Title       string
	Body        string
	Head        string
	Base        string
	IssueNumber int // 关联的 Issue（会在 body 中添加 Closes #issueNumber）
}

// IssueInfo Issue 概要信息
type IssueInfo struct {
	Number  int
	Title   string
	Body    string
	Author  string
	URL     string
	State   string
	Labels  []string
}

// IssueDetail Issue 详细信息
type IssueDetail struct {
	IssueInfo
	Comments []CommentInfo
}

// CommentInfo 评论信息
type CommentInfo struct {
	ID        string
	Author    string
	Body      string
	CreatedAt time.Time
}

// CheckInfo CI 检查信息
type CheckInfo struct {
	ID        string
	Name      string
	Status    string
	State     string
	Conclusion string
	URL       string
}

// WorkflowRunInfo 工作流运行概要
type WorkflowRunInfo struct {
	ID        int64
	Name      string
	Status    string
	Conclusion string
	HeadBranch string
	HeadSHA   string
	CreatedAt time.Time
}

// WorkflowRunDetail 工作流运行详细信息
type WorkflowRunDetail struct {
	WorkflowRunInfo
	Jobs []WorkflowJobInfo
}

// WorkflowJobInfo 工作流任务信息
type WorkflowJobInfo struct {
	ID        int64
	Name      string
	Status    string
	Conclusion string
	StartedAt time.Time
	CompletedAt *time.Time
}
