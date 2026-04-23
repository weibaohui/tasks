package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

// --- 测试基础设施 ---
// setupGinContext 在 user_handler_test.go 中定义，此包内共享使用

// mockWebhookConfigRepo 内存 mock
type mockWebhookConfigRepo struct {
	configs map[string]*domain.GitHubWebhookConfig
	saveErr error
}

func newMockWebhookConfigRepo() *mockWebhookConfigRepo {
	return &mockWebhookConfigRepo{configs: make(map[string]*domain.GitHubWebhookConfig)}
}

func (r *mockWebhookConfigRepo) Save(_ context.Context, config *domain.GitHubWebhookConfig) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.configs[config.ID().String()] = config
	return nil
}
func (r *mockWebhookConfigRepo) FindByID(_ context.Context, id domain.GitHubWebhookConfigID) (*domain.GitHubWebhookConfig, error) {
	return r.configs[id.String()], nil
}
func (r *mockWebhookConfigRepo) FindByProjectID(_ context.Context, projectID domain.ProjectID) (*domain.GitHubWebhookConfig, error) {
	for _, c := range r.configs {
		if c.ProjectID().String() == projectID.String() {
			return c, nil
		}
	}
	return nil, nil
}
func (r *mockWebhookConfigRepo) FindAll(_ context.Context) ([]*domain.GitHubWebhookConfig, error) {
	var result []*domain.GitHubWebhookConfig
	for _, c := range r.configs {
		result = append(result, c)
	}
	return result, nil
}
func (r *mockWebhookConfigRepo) FindAllEnabled(_ context.Context) ([]*domain.GitHubWebhookConfig, error) {
	var result []*domain.GitHubWebhookConfig
	for _, c := range r.configs {
		if c.Enabled() {
			result = append(result, c)
		}
	}
	return result, nil
}
func (r *mockWebhookConfigRepo) Delete(_ context.Context, id domain.GitHubWebhookConfigID) error {
	delete(r.configs, id.String())
	return nil
}

// mockEventLogRepo 空实现
type mockEventLogRepo struct{}

func (r *mockEventLogRepo) Save(_ context.Context, _ *domain.WebhookEventLog) error { return nil }
func (r *mockEventLogRepo) FindByID(_ context.Context, _ domain.WebhookEventLogID) (*domain.WebhookEventLog, error) {
	return nil, nil
}
func (r *mockEventLogRepo) FindByProjectID(_ context.Context, _ domain.ProjectID, _, _ int) ([]*domain.WebhookEventLog, error) {
	return nil, nil
}
func (r *mockEventLogRepo) CountByProjectID(_ context.Context, _ domain.ProjectID) (int, error) {
	return 0, nil
}
func (r *mockEventLogRepo) DeleteByProjectID(_ context.Context, _ domain.ProjectID) error { return nil }
func (r *mockEventLogRepo) Delete(_ context.Context, _ domain.WebhookEventLogID) error     { return nil }

// mockBindingRepo 空实现
type mockBindingRepo struct{}

func (r *mockBindingRepo) Save(_ context.Context, _ *domain.WebhookHeartbeatBinding) error { return nil }
func (r *mockBindingRepo) FindByID(_ context.Context, _ domain.WebhookHeartbeatBindingID) (*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}
func (r *mockBindingRepo) FindByConfigID(_ context.Context, _ domain.GitHubWebhookConfigID) ([]*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}
func (r *mockBindingRepo) FindByConfigIDAndEventType(_ context.Context, _ domain.GitHubWebhookConfigID, _ string) ([]*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}
func (r *mockBindingRepo) Delete(_ context.Context, _ domain.WebhookHeartbeatBindingID) error          { return nil }
func (r *mockBindingRepo) DeleteByHeartbeatID(_ context.Context, _ domain.HeartbeatID) error           { return nil }
func (r *mockBindingRepo) FindByHeartbeatID(_ context.Context, _ domain.HeartbeatID) ([]*domain.WebhookHeartbeatBinding, error) {
	return nil, nil
}

// mockTriggeredHeartbeatRepo 空实现
type mockTriggeredHeartbeatRepo struct{}

func (r *mockTriggeredHeartbeatRepo) Save(_ context.Context, _ *domain.WebhookEventTriggeredHeartbeat) error { return nil }
func (r *mockTriggeredHeartbeatRepo) FindByEventLogID(_ context.Context, _ domain.WebhookEventLogID) ([]*domain.WebhookEventTriggeredHeartbeat, error) {
	return nil, nil
}
func (r *mockTriggeredHeartbeatRepo) DeleteByEventLogID(_ context.Context, _ domain.WebhookEventLogID) error { return nil }

// mockHeartbeatRepo 空实现
type mockHeartbeatRepo struct{}

func (r *mockHeartbeatRepo) Save(_ context.Context, _ *domain.Heartbeat) error              { return nil }
func (r *mockHeartbeatRepo) FindByID(_ context.Context, _ domain.HeartbeatID) (*domain.Heartbeat, error) {
	return nil, nil
}
func (r *mockHeartbeatRepo) FindByProjectID(_ context.Context, _ domain.ProjectID) ([]*domain.Heartbeat, error) {
	return nil, nil
}
func (r *mockHeartbeatRepo) FindAllEnabled(_ context.Context) ([]*domain.Heartbeat, error) { return nil, nil }
func (r *mockHeartbeatRepo) Delete(_ context.Context, _ domain.HeartbeatID) error          { return nil }

// mockIDGen 固定 ID 生成器
type mockIDGen struct{}

func (g *mockIDGen) Generate() string { return "mock-id-123" }

// --- 辅助函数 ---

func newTestConfig(repo string) *domain.GitHubWebhookConfig {
	c, _ := domain.NewGitHubWebhookConfig(
		domain.NewGitHubWebhookConfigID("test-config-1"),
		domain.NewProjectID("test-project-1"),
		repo,
	)
	return c
}

// --- EnableWebhook 测试 ---

func TestEnableWebhook_ConfigNotFound(t *testing.T) {
	repo := newMockWebhookConfigRepo()
	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/nonexistent/enable", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.EnableWebhook(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestEnableWebhook_CreateWebhookSuccess(t *testing.T) {
	config := newTestConfig("owner/repo")
	repo := newMockWebhookConfigRepo()
	repo.configs["test-config-1"] = config

	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://public.example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	// Mock execCommand 让 CreateWebhook 成功
	origExec := application.ExecCommand
	callCount := 0
	application.ExecCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// FindExistingWebhook: 返回空（无已有 webhook）
			return exec.Command("echo", "")
		}
		// createWebhook: 返回 JSON 包含 id
		return exec.Command("echo", `{"id": 12345}`)
	}
	defer func() { application.ExecCommand = origExec }()

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/test-config-1/enable", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-config-1"}}

	handler.EnableWebhook(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", resp["enabled"])
	}
	if resp["webhook_url"] == nil || resp["webhook_url"] == "" {
		t.Error("expected webhook_url to be set")
	}
}

func TestEnableWebhook_CreateWebhookFails(t *testing.T) {
	config := newTestConfig("owner/repo")
	repo := newMockWebhookConfigRepo()
	repo.configs["test-config-1"] = config

	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://public.example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	// Mock execCommand 让 CreateWebhook 失败
	origExec := application.ExecCommand
	application.ExecCommand = func(name string, args ...string) *exec.Cmd {
		// 所有调用都失败
		return exec.Command("false")
	}
	defer func() { application.ExecCommand = origExec }()

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/test-config-1/enable", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-config-1"}}

	handler.EnableWebhook(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != float64(500) {
		t.Errorf("expected code 500, got %v", resp["code"])
	}
}

func TestEnableWebhook_SSHRepoFormat(t *testing.T) {
	// 验证 SSH 格式 repo 能正确 enable
	config := newTestConfig("git@github.com:owner/repo.git")
	repo := newMockWebhookConfigRepo()
	repo.configs["test-config-1"] = config

	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://public.example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	origExec := application.ExecCommand
	callCount := 0
	application.ExecCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("echo", "")
		}
		return exec.Command("echo", `{"id": 99999}`)
	}
	defer func() { application.ExecCommand = origExec }()

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/test-config-1/enable", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-config-1"}}

	handler.EnableWebhook(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SSH repo format, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["enabled"] != true {
		t.Errorf("expected enabled=true for SSH repo, got %v", resp["enabled"])
	}
}

// --- DisableWebhook 测试 ---

func TestDisableWebhook_ConfigNotFound(t *testing.T) {
	repo := newMockWebhookConfigRepo()
	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/nonexistent/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.DisableWebhook(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDisableWebhook_Success(t *testing.T) {
	config := newTestConfig("owner/repo")
	config.SetEnabled(true)
	repo := newMockWebhookConfigRepo()
	repo.configs["test-config-1"] = config

	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://public.example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	origExec := application.ExecCommand
	callCount := 0
	application.ExecCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// FindExistingWebhook: 返回 webhook ID
			return exec.Command("echo", "12345")
		}
		// deleteWebhook: 成功
		return exec.Command("true")
	}
	defer func() { application.ExecCommand = origExec }()

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/test-config-1/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-config-1"}}

	handler.DisableWebhook(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["enabled"] == true {
		t.Error("expected enabled=false after disable")
	}
}

func TestDisableWebhook_DeleteFailsStillReturns200(t *testing.T) {
	// handler 不因 delete 失败而报错，仍然返回 200
	config := newTestConfig("owner/repo")
	config.SetEnabled(true)
	repo := newMockWebhookConfigRepo()
	repo.configs["test-config-1"] = config

	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://public.example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	origExec := application.ExecCommand
	application.ExecCommand = func(name string, args ...string) *exec.Cmd {
		// 所有调用都失败
		return exec.Command("false")
	}
	defer func() { application.ExecCommand = origExec }()

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/test-config-1/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-config-1"}}

	handler.DisableWebhook(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 even when delete fails, got %d", w.Code)
	}
}

func TestDisableWebhook_NoExistingWebhook(t *testing.T) {
	config := newTestConfig("owner/repo")
	config.SetEnabled(true)
	repo := newMockWebhookConfigRepo()
	repo.configs["test-config-1"] = config

	svc := application.NewGitHubWebhookService(repo, &mockEventLogRepo{}, &mockBindingRepo{}, &mockHeartbeatRepo{}, &mockTriggeredHeartbeatRepo{}, nil, &mockIDGen{})
	mgr := application.NewWebhookGitHubManager("https://public.example.com")
	handler := NewGitHubWebhookHandler(svc, mgr, nil, nil, nil)

	origExec := application.ExecCommand
	application.ExecCommand = func(name string, args ...string) *exec.Cmd {
		// FindExistingWebhook: 返回空（无 webhook）
		return exec.Command("echo", "")
	}
	defer func() { application.ExecCommand = origExec }()

	c, w := setupGinContext("POST", "/api/v1/github-webhooks/configs/test-config-1/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-config-1"}}

	handler.DisableWebhook(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no webhook exists, got %d", w.Code)
	}
}
