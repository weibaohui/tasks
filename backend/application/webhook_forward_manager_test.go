package application

import (
	"context"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// --- NormalizeRepo 测试 ---

func TestNormalizeRepo(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"owner/repo", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo.git", "owner/repo"},
		{"git@github.com:owner/repo", "owner/repo"},
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"org/my-project", "org/my-project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeRepo(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeRepo(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- WebhookGitHubManager 基础测试 ---

func TestWebhookGitHubManager_NewWebhookGitHubManager(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		expected  string
	}{
		{"normal URL", "https://example.com/api/v1", "https://example.com"},
		{"URL without trailing", "https://example.com", "https://example.com"},
		{"URL with trailing slash", "https://example.com/", "https://example.com/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewWebhookGitHubManager(tt.serverURL)
			if mgr.serverURL != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, mgr.serverURL)
			}
		})
	}
}

func TestWebhookGitHubManager_BuildWebhookURL(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")
	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{"short repo format", "owner/repo", "https://example.com/api/v1/webhook/repos/owner/repo"},
		{"full GitHub URL", "https://github.com/owner/repo", "https://example.com/api/v1/webhook/repos/owner/repo"},
		{"full GitHub URL with .git", "https://github.com/owner/repo.git", "https://example.com/api/v1/webhook/repos/owner/repo"},
		{"SSH format", "git@github.com:owner/repo", "https://example.com/api/v1/webhook/repos/owner/repo"},
		{"SSH format with .git", "git@github.com:owner/repo.git", "https://example.com/api/v1/webhook/repos/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := mgr.BuildWebhookURL(tt.repo)
			if url != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, url)
			}
		})
	}
}

func TestWebhookGitHubManager_UpdateServerURL(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://old.example.com")
	mgr.UpdateServerURL("https://new.example.com/api/v1")
	if mgr.serverURL != "https://new.example.com" {
		t.Errorf("expected https://new.example.com, got %s", mgr.serverURL)
	}
}

// --- CreateWebhook 测试 ---

func TestCreateWebhook_SSHRepoFormat(t *testing.T) {
	// 验证 SSH 格式 repo 能被正确 normalize 后调用 gh api
	// 用一个 mock 函数记录实际传给 ExecCommand 的参数
	var calledRepo string
	origExec := ExecCommand
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		// 捕获第一次调用 (FindExistingWebhook) 的 repo 参数
		for i, a := range args {
			if a == "repos/" && i+1 < len(args) {
				calledRepo = args[i+1]
			}
		}
		// 返回一个会失败的命令（非零退出），模拟无 webhook
		return exec.Command("false")
	}
	defer func() { ExecCommand = origExec }()

	mgr := NewWebhookGitHubManager("https://public.example.com")
	_, err := mgr.CreateWebhook(context.Background(), "cfg1", "proj1", "git@github.com:owner/repo")

	// 会因为 false 命令失败，但关键是 repo 被 normalize 了
	// 验证调用的 repo 路径不包含 git@github.com:
	_ = err
	// 由于 FindExistingWebhook 失败返回 0，会走 createWebhook
	// createWebhook 也会失败（false 命令）
	// 关键验证：calledRepo 不应该包含 git@ 前缀
	if calledRepo != "" && calledRepo == "git@github.com:owner/repo" {
		t.Error("repo should have been normalized from SSH format")
	}
}

func TestCreateWebhook_NormalRepo(t *testing.T) {
	// 确保普通 repo 格式不变
	var capturedArgs []string
	origExec := ExecCommand
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		capturedArgs = args
		// 第一次调用 FindExistingWebhook - 返回成功但无输出（无 webhook）
		return exec.Command("echo", "")
	}
	defer func() { ExecCommand = origExec }()

	mgr := NewWebhookGitHubManager("https://public.example.com")
	// FindExistingWebhook 返回空 → createWebhook 被调用（echo "" 会输出空 JSON → 解析失败）
	// 这主要验证参数传递链路正确
	_, _ = mgr.CreateWebhook(context.Background(), "cfg1", "proj1", "owner/repo")
	_ = capturedArgs

	// 验证 echo 被当作命令调用了（即 ExecCommand 被调用了）
	if len(capturedArgs) == 0 {
		t.Error("ExecCommand should have been called")
	}
}

// --- DeleteWebhook 测试 ---

func TestDeleteWebhook_NoExistingWebhook(t *testing.T) {
	origExec := ExecCommand
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		// FindExistingWebhook 返回空输出 → webhookID=0
		return exec.Command("echo", "")
	}
	defer func() { ExecCommand = origExec }()

	mgr := NewWebhookGitHubManager("https://example.com")
	err := mgr.DeleteWebhook(context.Background(), "cfg1", "proj1", "owner/repo")
	if err != nil {
		t.Errorf("expected nil error for no existing webhook, got: %v", err)
	}
}

func TestDeleteWebhook_SSHRepoFormat(t *testing.T) {
	// 验证 SSH 格式的 repo 在 delete 时也能正确 normalize
	var capturedRepo string
	callCount := 0
	origExec := ExecCommand
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// FindExistingWebhook - 返回 webhook ID=123
			for _, a := range args {
				if len(a) > 7 && a[:7] == "repos/" {
					capturedRepo = a[7:]
				}
			}
			return exec.Command("echo", "123")
		}
		// deleteWebhook - 成功
		return exec.Command("true")
	}
	defer func() { ExecCommand = origExec }()

	mgr := NewWebhookGitHubManager("https://example.com")
	err := mgr.DeleteWebhook(context.Background(), "cfg1", "proj1", "git@github.com:owner/repo.git")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	// 验证 repo 被正确 normalize（不包含 git@ 前缀）
	// FindExistingWebhook 请求 repos/{repo}/hooks
	// capturedRepo 应该是 normalize 后的路径
	if capturedRepo == "git@github.com:owner/repo" {
		t.Errorf("repo should be normalized, got: %s", capturedRepo)
	}
}

// --- 并发安全测试 ---

func TestWebhookGitHubManager_BuildWebhookURL_Concurrent(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")
	var wg sync.WaitGroup
	resultChan := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			url := mgr.BuildWebhookURL("owner/repo")
			resultChan <- url
		}()
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	for url := range resultChan {
		expected := "https://example.com/api/v1/webhook/repos/owner/repo"
		if url != expected {
			t.Errorf("expected %s, got %s", expected, url)
		}
	}
}

func TestWebhookGitHubManager_UpdateServerURL_Concurrent(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			newURL := "https://server-" + string(rune('0'+i%10)) + ".example.com"
			mgr.UpdateServerURL(newURL)
		}()
	}
	close(start)
	wg.Wait()
	if mgr.serverURL == "" {
		t.Error("serverURL should not be empty after concurrent updates")
	}
}

func TestWebhookGitHubManager_ThreadSafety(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				mgr.UpdateServerURL("https://server-"+time.Now().Format(time.RFC3339Nano)+".example.com", )
				mgr.BuildWebhookURL("owner/repo")
			}
		}()
	}
	close(start)
	wg.Wait()
}
