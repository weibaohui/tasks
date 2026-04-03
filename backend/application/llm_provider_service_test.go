package application

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// mockTestConnectionRunner 用于测试的 fake runner
type mockTestConnectionRunner struct {
	returnErr   error
	called      bool
	lastConfig  *llm.Config
}

func (m *mockTestConnectionRunner) RunTest(ctx context.Context, config *llm.Config) error {
	m.called = true
	m.lastConfig = config
	return m.returnErr
}

// mockLLMProviderRepoForService 模拟 LLMProviderRepository
type mockLLMProviderRepoForService struct {
	providers           map[string]*domain.LLMProvider
	errSave             error
	errFindByID         error
	errFindByUserCode   error
	errDelete           error
	errClearDefault     error
	clearDefaultCalled  bool
	clearedExcludeID    *domain.LLMProviderID
}

func newMockLLMProviderRepoForService() *mockLLMProviderRepoForService {
	return &mockLLMProviderRepoForService{
		providers: make(map[string]*domain.LLMProvider),
	}
}

func (m *mockLLMProviderRepoForService) Save(ctx context.Context, provider *domain.LLMProvider) error {
	if m.errSave != nil {
		return m.errSave
	}
	m.providers[provider.ID().String()] = provider
	return nil
}

func (m *mockLLMProviderRepoForService) FindByID(ctx context.Context, id domain.LLMProviderID) (*domain.LLMProvider, error) {
	if m.errFindByID != nil {
		return nil, m.errFindByID
	}
	provider, ok := m.providers[id.String()]
	if !ok {
		return nil, nil
	}
	return provider, nil
}

func (m *mockLLMProviderRepoForService) FindByUserCode(ctx context.Context, userCode string) ([]*domain.LLMProvider, error) {
	if m.errFindByUserCode != nil {
		return nil, m.errFindByUserCode
	}
	var result []*domain.LLMProvider
	for _, provider := range m.providers {
		if provider.UserCode() == userCode {
			result = append(result, provider)
		}
	}
	return result, nil
}

func (m *mockLLMProviderRepoForService) FindByProviderKey(ctx context.Context, providerKey string) (*domain.LLMProvider, error) {
	for _, provider := range m.providers {
		if provider.ProviderKey() == providerKey {
			return provider, nil
		}
	}
	return nil, nil
}

func (m *mockLLMProviderRepoForService) FindDefaultActive(ctx context.Context, userCode string) (*domain.LLMProvider, error) {
	for _, provider := range m.providers {
		if provider.UserCode() == userCode && provider.IsDefault() && provider.IsActive() {
			return provider, nil
		}
	}
	return nil, nil
}

func (m *mockLLMProviderRepoForService) ClearDefaultByUserCode(ctx context.Context, userCode string, excludeID *domain.LLMProviderID) error {
	m.clearDefaultCalled = true
	m.clearedExcludeID = excludeID
	if m.errClearDefault != nil {
		return m.errClearDefault
	}
	for _, provider := range m.providers {
		if provider.UserCode() == userCode && provider.IsDefault() {
			if excludeID == nil || provider.ID().String() != excludeID.String() {
				provider.SetDefault(false)
			}
		}
	}
	return nil
}

func (m *mockLLMProviderRepoForService) Delete(ctx context.Context, id domain.LLMProviderID) error {
	if m.errDelete != nil {
		return m.errDelete
	}
	delete(m.providers, id.String())
	return nil
}

// mockLLMProviderIDGen 模拟 ID 生成器
type mockLLMProviderIDGen struct {
	count int
}

func (m *mockLLMProviderIDGen) Generate() string {
	m.count++
	return "provider-id-" + strconv.Itoa(m.count)
}

func setupTestLLMProviderSvc() (*LLMProviderApplicationService, *mockLLMProviderRepoForService, *mockLLMProviderIDGen) {
	return setupTestLLMProviderSvcWithRunner(nil)
}

func setupTestLLMProviderSvcWithRunner(runner TestConnectionRunner) (*LLMProviderApplicationService, *mockLLMProviderRepoForService, *mockLLMProviderIDGen) {
	repo := newMockLLMProviderRepoForService()
	idGen := &mockLLMProviderIDGen{}
	var svc *LLMProviderApplicationService
	if runner != nil {
		svc = NewLLMProviderApplicationServiceWithRunner(repo, idGen, runner)
	} else {
		svc = NewLLMProviderApplicationService(repo, idGen)
	}
	return svc, repo, idGen
}

func TestLLMProviderService_Create(t *testing.T) {
	svc, repo, idGen := setupTestLLMProviderSvc()
	ctx := context.Background()

	autoMerge := true
	provider, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:        "user-001",
		ProviderKey:     "openai",
		ProviderName:    "OpenAI",
		APIKey:          "sk-test-key",
		APIBase:         "https://api.openai.com",
		ProviderType:    "openai",
		ExtraHeaders:    map[string]string{"X-Custom": "value"},
		SupportedModels: []domain.ModelInfo{{ID: "gpt-4", Name: "GPT-4", MaxTokens: 8192}},
		DefaultModel:    "gpt-4",
		IsDefault:       false,
		Priority:        10,
		AutoMerge:       &autoMerge,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证 ID 生成
	expectedID := "provider-id-1"
	if provider.ID().String() != expectedID {
		t.Errorf("期望 id 为 '%s', 实际为 '%s'", expectedID, provider.ID().String())
	}

	// 验证基本字段
	if provider.UserCode() != "user-001" {
		t.Errorf("期望 userCode 为 'user-001', 实际为 '%s'", provider.UserCode())
	}
	if provider.ProviderKey() != "openai" {
		t.Errorf("期望 providerKey 为 'openai', 实际为 '%s'", provider.ProviderKey())
	}
	if provider.ProviderName() != "OpenAI" {
		t.Errorf("期望 providerName 为 'OpenAI', 实际为 '%s'", provider.ProviderName())
	}
	if provider.APIKey() != "sk-test-key" {
		t.Errorf("期望 apiKey 为 'sk-test-key', 实际为 '%s'", provider.APIKey())
	}
	if provider.APIBase() != "https://api.openai.com" {
		t.Errorf("期望 apiBase 为 'https://api.openai.com', 实际为 '%s'", provider.APIBase())
	}
	if provider.ProviderType() != "openai" {
		t.Errorf("期望 providerType 为 'openai', 实际为 '%s'", provider.ProviderType())
	}
	if provider.DefaultModel() != "gpt-4" {
		t.Errorf("期望 defaultModel 为 'gpt-4', 实际为 '%s'", provider.DefaultModel())
	}
	if provider.Priority() != 10 {
		t.Errorf("期望 priority 为 10, 实际为 %d", provider.Priority())
	}
	if !provider.AutoMerge() {
		t.Error("期望 autoMerge 为 true, 实际为 false")
	}
	if provider.IsDefault() {
		t.Error("期望 isDefault 为 false, 实际为 true")
	}

	// 验证保存到仓库
	if len(repo.providers) != 1 {
		t.Errorf("期望 repo 中有 1 个 provider, 实际为 %d", len(repo.providers))
	}

	// 验证 ID 生成器调用次数
	if idGen.count != 1 {
		t.Errorf("期望 idGen count 为 1, 实际为 %d", idGen.count)
	}
}

func TestLLMProviderService_Create_WithDefault(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 先创建一个默认 provider
	existingProvider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("existing-id"),
		"user-001",
		"kimi",
		"Kimi",
		"sk-kimi",
		"https://api.kimi.com",
	)
	existingProvider.SetDefault(true)
	repo.providers["existing-id"] = existingProvider

	// 创建一个新的默认 provider
	autoMerge := false
	provider, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "openai",
		ProviderName: "OpenAI",
		APIKey:       "sk-openai",
		APIBase:      "https://api.openai.com",
		IsDefault:    true,
		AutoMerge:    &autoMerge,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证 ClearDefaultByUserCode 被调用
	if !repo.clearDefaultCalled {
		t.Error("期望 ClearDefaultByUserCode 被调用")
	}

	// 验证新 provider 是默认的
	if !provider.IsDefault() {
		t.Error("期望新创建的 provider 是默认的")
	}
}

func TestLLMProviderService_Create_ValidationError(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 测试空的 providerKey
	_, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "",
		ProviderName: "Test",
		APIKey:       "sk-test",
	})
	if err == nil {
		t.Error("期望 providerKey 为空时返回错误, 实际为 nil")
	}

	// 测试空的 userCode
	_, err = svc.Create(ctx, CreateProviderCommand{
		UserCode:     "",
		ProviderKey:  "openai",
		ProviderName: "Test",
		APIKey:       "sk-test",
	})
	if err == nil {
		t.Error("期望 userCode 为空时返回错误, 实际为 nil")
	}
}

func TestLLMProviderService_Create_SaveError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	repo.errSave = errors.New("database error")

	_, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "openai",
		ProviderName: "OpenAI",
		APIKey:       "sk-test",
	})

	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}

	if !errors.Is(err, repo.errSave) {
		t.Errorf("期望错误包含 'database error', 实际为 %v", err)
	}
}

func TestLLMProviderService_Create_ClearDefaultError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	repo.errClearDefault = errors.New("clear default error")

	_, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "openai",
		ProviderName: "OpenAI",
		APIKey:       "sk-test",
		IsDefault:    true,
	})

	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}

	if !errors.Is(err, repo.errClearDefault) {
		t.Errorf("期望错误包含 'clear default error', 实际为 %v", err)
	}
}

func TestLLMProviderService_Get(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-get-001"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-test",
		"https://api.openai.com",
	)
	repo.providers["provider-get-001"] = provider

	// 获取 provider
	found, err := svc.Get(ctx, domain.NewLLMProviderID("provider-get-001"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if found == nil {
		t.Fatal("期望找到 provider, 实际为 nil")
	}

	if found.ID().String() != "provider-get-001" {
		t.Errorf("期望 id 为 'provider-get-001', 实际为 '%s'", found.ID().String())
	}

	if found.ProviderKey() != "openai" {
		t.Errorf("期望 providerKey 为 'openai', 实际为 '%s'", found.ProviderKey())
	}
}

func TestLLMProviderService_Get_NotFound(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	_, err := svc.Get(ctx, domain.NewLLMProviderID("non-existent-id"))
	if err != ErrProviderNotFound {
		t.Errorf("期望 ErrProviderNotFound, 实际为 %v", err)
	}
}

func TestLLMProviderService_Get_RepoError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	repo.errFindByID = errors.New("database error")

	_, err := svc.Get(ctx, domain.NewLLMProviderID("provider-id"))
	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}
}

func TestLLMProviderService_List(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建多个 providers
	for i := 1; i <= 3; i++ {
		provider, _ := domain.NewLLMProvider(
			domain.NewLLMProviderID("provider-list-"+strconv.Itoa(i)),
			"user-001",
			"openai-"+strconv.Itoa(i),
			"OpenAI "+strconv.Itoa(i),
			"sk-test-"+strconv.Itoa(i),
			"https://api.openai.com",
		)
		repo.providers[provider.ID().String()] = provider
	}

	// 为其他用户创建 provider
	otherProvider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-other"),
		"user-002",
		"kimi",
		"Kimi",
		"sk-kimi",
		"https://api.kimi.com",
	)
	repo.providers["provider-other"] = otherProvider

	// 列出 user-001 的 providers
	providers, err := svc.List(ctx, "user-001")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(providers) != 3 {
		t.Errorf("期望 3 个 providers, 实际为 %d", len(providers))
	}

	// 验证所有返回的 provider 都属于 user-001
	for _, p := range providers {
		if p.UserCode() != "user-001" {
			t.Errorf("期望 userCode 为 'user-001', 实际为 '%s'", p.UserCode())
		}
	}
}

func TestLLMProviderService_List_Empty(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 列出没有 providers 的用户
	providers, err := svc.List(ctx, "user-no-providers")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(providers) != 0 {
		t.Errorf("期望 0 个 providers, 实际为 %d", len(providers))
	}
}

func TestLLMProviderService_List_RepoError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	repo.errFindByUserCode = errors.New("database error")

	_, err := svc.List(ctx, "user-001")
	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}
}

func TestLLMProviderService_Update(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-update-001"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-old-key",
		"https://api.openai.com",
	)
	provider.SetDefaultModel("gpt-3.5-turbo")
	provider.SetPriority(5)
	repo.providers["provider-update-001"] = provider

	// 更新所有字段
	newProviderKey := "anthropic"
	newProviderName := "Claude"
	newAPIKey := "sk-new-key"
	newAPIBase := "https://api.anthropic.com"
	newDefaultModel := "claude-3-opus"
	newPriority := 10
	newAutoMerge := false
	newIsActive := false
	newProviderType := "anthropic"
	newIsDefault := true

	updated, err := svc.Update(ctx, UpdateProviderCommand{
		ID:              domain.NewLLMProviderID("provider-update-001"),
		ProviderKey:     &newProviderKey,
		ProviderName:    &newProviderName,
		APIKey:          &newAPIKey,
		APIBase:         &newAPIBase,
		DefaultModel:    &newDefaultModel,
		Priority:        &newPriority,
		AutoMerge:       &newAutoMerge,
		IsActive:        &newIsActive,
		ProviderType:    &newProviderType,
		IsDefault:       &newIsDefault,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证更新的字段
	if updated.ProviderKey() != "anthropic" {
		t.Errorf("期望 providerKey 为 'anthropic', 实际为 '%s'", updated.ProviderKey())
	}
	if updated.ProviderName() != "Claude" {
		t.Errorf("期望 providerName 为 'Claude', 实际为 '%s'", updated.ProviderName())
	}
	if updated.APIKey() != "sk-new-key" {
		t.Errorf("期望 apiKey 为 'sk-new-key', 实际为 '%s'", updated.APIKey())
	}
	if updated.APIBase() != "https://api.anthropic.com" {
		t.Errorf("期望 apiBase 为 'https://api.anthropic.com', 实际为 '%s'", updated.APIBase())
	}
	if updated.DefaultModel() != "claude-3-opus" {
		t.Errorf("期望 defaultModel 为 'claude-3-opus', 实际为 '%s'", updated.DefaultModel())
	}
	if updated.Priority() != 10 {
		t.Errorf("期望 priority 为 10, 实际为 %d", updated.Priority())
	}
	if updated.AutoMerge() {
		t.Error("期望 autoMerge 为 false, 实际为 true")
	}
	if updated.IsActive() {
		t.Error("期望 isActive 为 false, 实际为 true")
	}
	if updated.ProviderType() != "anthropic" {
		t.Errorf("期望 providerType 为 'anthropic', 实际为 '%s'", updated.ProviderType())
	}
	if !updated.IsDefault() {
		t.Error("期望 isDefault 为 true, 实际为 false")
	}
}

func TestLLMProviderService_Update_PartialFields(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-partial-001"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	provider.SetDefaultModel("gpt-3.5-turbo")
	provider.SetPriority(5)
	repo.providers["provider-partial-001"] = provider

	// 只更新 providerName 和 priority
	newProviderName := "OpenAI Pro"
	newPriority := 20

	updated, err := svc.Update(ctx, UpdateProviderCommand{
		ID:           domain.NewLLMProviderID("provider-partial-001"),
		ProviderName: &newProviderName,
		Priority:     &newPriority,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证更新的字段
	if updated.ProviderName() != "OpenAI Pro" {
		t.Errorf("期望 providerName 为 'OpenAI Pro', 实际为 '%s'", updated.ProviderName())
	}
	if updated.Priority() != 20 {
		t.Errorf("期望 priority 为 20, 实际为 %d", updated.Priority())
	}

	// 验证未更新的字段保持不变
	if updated.ProviderKey() != "openai" {
		t.Errorf("期望 providerKey 保持为 'openai', 实际为 '%s'", updated.ProviderKey())
	}
	if updated.DefaultModel() != "gpt-3.5-turbo" {
		t.Errorf("期望 defaultModel 保持为 'gpt-3.5-turbo', 实际为 '%s'", updated.DefaultModel())
	}
}

func TestLLMProviderService_Update_SetDefault(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个默认 provider
	existingProvider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-existing"),
		"user-001",
		"kimi",
		"Kimi",
		"sk-kimi",
		"https://api.kimi.com",
	)
	existingProvider.SetDefault(true)
	repo.providers["provider-existing"] = existingProvider

	// 创建另一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-set-default"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-openai",
		"https://api.openai.com",
	)
	repo.providers["provider-set-default"] = provider

	// 将新 provider 设为默认
	isDefault := true
	updated, err := svc.Update(ctx, UpdateProviderCommand{
		ID:        domain.NewLLMProviderID("provider-set-default"),
		IsDefault: &isDefault,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证 ClearDefaultByUserCode 被调用，并且传入了正确的 excludeID
	if !repo.clearDefaultCalled {
		t.Error("期望 ClearDefaultByUserCode 被调用")
	}
	if repo.clearedExcludeID == nil || repo.clearedExcludeID.String() != "provider-set-default" {
		t.Error("期望 excludeID 为 'provider-set-default'")
	}

	if !updated.IsDefault() {
		t.Error("期望 updated provider 是默认的")
	}
}

func TestLLMProviderService_Update_NotFound(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	newProviderName := "Test"
	_, err := svc.Update(ctx, UpdateProviderCommand{
		ID:           domain.NewLLMProviderID("non-existent-id"),
		ProviderName: &newProviderName,
	})

	if err != ErrProviderNotFound {
		t.Errorf("期望 ErrProviderNotFound, 实际为 %v", err)
	}
}

func TestLLMProviderService_Update_SaveError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-save-error"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-save-error"] = provider

	repo.errSave = errors.New("database error")

	newProviderName := "Test"
	_, err := svc.Update(ctx, UpdateProviderCommand{
		ID:           domain.NewLLMProviderID("provider-save-error"),
		ProviderName: &newProviderName,
	})

	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}

	if !errors.Is(err, repo.errSave) {
		t.Errorf("期望错误包含 'database error', 实际为 %v", err)
	}
}

func TestLLMProviderService_Update_RepoFindError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	repo.errFindByID = errors.New("database error")

	newProviderName := "Test"
	_, err := svc.Update(ctx, UpdateProviderCommand{
		ID:           domain.NewLLMProviderID("provider-id"),
		ProviderName: &newProviderName,
	})

	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}
}

func TestLLMProviderService_Update_ClearDefaultError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-clear-error"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-clear-error"] = provider

	repo.errClearDefault = errors.New("clear default error")

	isDefault := true
	_, err := svc.Update(ctx, UpdateProviderCommand{
		ID:        domain.NewLLMProviderID("provider-clear-error"),
		IsDefault: &isDefault,
	})

	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}

	if !errors.Is(err, repo.errClearDefault) {
		t.Errorf("期望错误包含 'clear default error', 实际为 %v", err)
	}
}

func TestLLMProviderService_Update_WithExtraHeaders(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-headers"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-headers"] = provider

	// 更新 ExtraHeaders
	newHeaders := map[string]string{
		"X-Auth":  "token",
		"X-Trace": "trace-id",
	}

	updated, err := svc.Update(ctx, UpdateProviderCommand{
		ID:           domain.NewLLMProviderID("provider-headers"),
		ExtraHeaders: &newHeaders,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	headers := updated.ExtraHeaders()
	if len(headers) != 2 {
		t.Errorf("期望有 2 个 headers, 实际为 %d", len(headers))
	}
	if headers["X-Auth"] != "token" {
		t.Errorf("期望 X-Auth 为 'token', 实际为 '%s'", headers["X-Auth"])
	}
}

func TestLLMProviderService_Update_WithSupportedModels(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-models"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-models"] = provider

	// 更新 SupportedModels
	newModels := []domain.ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", MaxTokens: 8192},
		{ID: "gpt-3.5-turbo", Name: "GPT-3.5", MaxTokens: 4096},
	}

	updated, err := svc.Update(ctx, UpdateProviderCommand{
		ID:              domain.NewLLMProviderID("provider-models"),
		SupportedModels: &newModels,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	models := updated.SupportedModels()
	if len(models) != 2 {
		t.Errorf("期望有 2 个 models, 实际为 %d", len(models))
	}
	if models[0].ID != "gpt-4" {
		t.Errorf("期望第一个 model ID 为 'gpt-4', 实际为 '%s'", models[0].ID)
	}
}

func TestLLMProviderService_Delete(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-delete-001"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-delete-001"] = provider

	// 删除 provider
	err := svc.Delete(ctx, domain.NewLLMProviderID("provider-delete-001"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证已删除
	if _, exists := repo.providers["provider-delete-001"]; exists {
		t.Error("期望 provider 已被删除")
	}
}

func TestLLMProviderService_Delete_NotFound(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	err := svc.Delete(ctx, domain.NewLLMProviderID("non-existent-id"))
	if err != ErrProviderNotFound {
		t.Errorf("期望 ErrProviderNotFound, 实际为 %v", err)
	}
}

func TestLLMProviderService_Delete_RepoFindError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	repo.errFindByID = errors.New("database error")

	err := svc.Delete(ctx, domain.NewLLMProviderID("provider-id"))
	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}
}

func TestLLMProviderService_Delete_RepoDeleteError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-delete-error"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-delete-error"] = provider
	repo.errDelete = errors.New("delete error")

	err := svc.Delete(ctx, domain.NewLLMProviderID("provider-delete-error"))
	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}
}

func TestLLMProviderService_TestConnection_NoAPIKey(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个没有 API Key 的 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-no-key"),
		"user-001",
		"openai",
		"OpenAI",
		"", // 空 API Key
		"https://api.openai.com",
	)
	repo.providers["provider-no-key"] = provider

	result, err := svc.TestConnection(ctx, domain.NewLLMProviderID("provider-no-key"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if result["success"] != false {
		t.Errorf("期望 success 为 false, 实际为 %v", result["success"])
	}

	message, ok := result["message"].(string)
	if !ok || message != "API Key 未配置" {
		t.Errorf("期望 message 为 'API Key 未配置', 实际为 '%v'", result["message"])
	}
}

func TestLLMProviderService_TestConnection_ProviderNotFound(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	_, err := svc.TestConnection(ctx, domain.NewLLMProviderID("non-existent-id"))
	if err != ErrProviderNotFound {
		t.Errorf("期望 ErrProviderNotFound, 实际为 %v", err)
	}
}

func TestLLMProviderService_NewService(t *testing.T) {
	repo := newMockLLMProviderRepoForService()
	idGen := &mockLLMProviderIDGen{}

	svc := NewLLMProviderApplicationService(repo, idGen)

	if svc == nil {
		t.Fatal("期望 svc 不为 nil")
	}

	// 测试服务是否正确初始化
	ctx := context.Background()
	provider, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "openai",
		ProviderName: "OpenAI",
		APIKey:       "sk-test",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if provider.ID().String() != "provider-id-1" {
		t.Errorf("期望 id 为 'provider-id-1', 实际为 '%s'", provider.ID().String())
	}
}

func TestLLMProviderService_Create_WithoutOptionalFields(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 只提供必需字段
	provider, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "openai",
		ProviderName: "OpenAI",
		APIKey:       "sk-test",
		APIBase:      "https://api.openai.com",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证默认值
	if provider.ProviderType() != "" {
		t.Errorf("期望 providerType 为空, 实际为 '%s'", provider.ProviderType())
	}
	if provider.Priority() != 0 {
		t.Errorf("期望 priority 为 0, 实际为 %d", provider.Priority())
	}
	if !provider.AutoMerge() {
		t.Error("期望 autoMerge 默认为 true (领域模型默认值)")
	}

	// 验证保存到仓库
	if len(repo.providers) != 1 {
		t.Errorf("期望 repo 中有 1 个 provider, 实际为 %d", len(repo.providers))
	}
}

func TestLLMProviderService_Create_WithNilAutoMerge(t *testing.T) {
	svc, _, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 不提供 AutoMerge (nil)
	provider, err := svc.Create(ctx, CreateProviderCommand{
		UserCode:     "user-001",
		ProviderKey:  "openai",
		ProviderName: "OpenAI",
		APIKey:       "sk-test",
		APIBase:      "https://api.openai.com",
		AutoMerge:    nil,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证使用领域模型默认值 (true)
	if !provider.AutoMerge() {
		t.Error("期望 autoMerge 使用默认值 true")
	}
}

func TestLLMProviderService_Update_UpdateProfileError(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-update-error"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	repo.providers["provider-update-error"] = provider

	// 尝试更新 providerKey 为空（应该失败）
	emptyKey := ""
	_, err := svc.Update(ctx, UpdateProviderCommand{
		ID:          domain.NewLLMProviderID("provider-update-error"),
		ProviderKey: &emptyKey,
	})

	if err == nil {
		t.Fatal("期望有验证错误, 实际为 nil")
	}
}

func TestLLMProviderService_Update_SetDefaultFalse(t *testing.T) {
	svc, repo, _ := setupTestLLMProviderSvc()
	ctx := context.Background()

	// 创建一个默认 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-unset-default"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-key",
		"https://api.openai.com",
	)
	provider.SetDefault(true)
	repo.providers["provider-unset-default"] = provider

	// 取消默认设置
	isDefault := false
	updated, err := svc.Update(ctx, UpdateProviderCommand{
		ID:        domain.NewLLMProviderID("provider-unset-default"),
		IsDefault: &isDefault,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.IsDefault() {
		t.Error("期望 isDefault 为 false")
	}

	// 验证 ClearDefaultByUserCode 没有被调用（因为设置为 false）
	if repo.clearDefaultCalled {
		t.Error("设置 IsDefault 为 false 时不应调用 ClearDefaultByUserCode")
	}
}

// TestLLMProviderService_TestConnection_WithSupportedModels 测试使用 SupportedModels[0] 作为模型
func TestLLMProviderService_TestConnection_WithSupportedModels(t *testing.T) {
	mockRunner := &mockTestConnectionRunner{returnErr: errors.New("connection failed")}
	svc, repo, _ := setupTestLLMProviderSvcWithRunner(mockRunner)
	ctx := context.Background()

	// 创建一个带有 SupportedModels 但没有 DefaultModel 的 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-with-models"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-test-key",
		"https://api.openai.com",
	)
	provider.SetSupportedModels([]domain.ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", MaxTokens: 8192},
		{ID: "gpt-3.5-turbo", Name: "GPT-3.5", MaxTokens: 4096},
	})
	// 不设置 DefaultModel，让它使用 SupportedModels[0].ID
	repo.providers["provider-with-models"] = provider

	result, err := svc.TestConnection(ctx, domain.NewLLMProviderID("provider-with-models"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证使用了正确的模型
	if !mockRunner.called {
		t.Error("期望 testRunner.RunTest 被调用")
	}
	if mockRunner.lastConfig == nil {
		t.Fatal("期望 lastConfig 不为 nil")
	}
	if mockRunner.lastConfig.Model != "gpt-4" {
		t.Errorf("期望模型为 'gpt-4', 实际为 '%s'", mockRunner.lastConfig.Model)
	}

	// 验证返回结果
	if result["success"] != false {
		t.Errorf("期望 success 为 false, 实际为 %v", result["success"])
	}
}

// TestLLMProviderService_TestConnection_WithDefaultModel 测试使用 DefaultModel 作为模型
func TestLLMProviderService_TestConnection_WithDefaultModel(t *testing.T) {
	mockRunner := &mockTestConnectionRunner{returnErr: errors.New("connection failed")}
	svc, repo, _ := setupTestLLMProviderSvcWithRunner(mockRunner)
	ctx := context.Background()

	// 创建一个同时有 DefaultModel 和 SupportedModels 的 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-default-model"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-test-key",
		"https://api.openai.com",
	)
	provider.SetDefaultModel("gpt-4-turbo")
	provider.SetSupportedModels([]domain.ModelInfo{
		{ID: "gpt-3.5-turbo", Name: "GPT-3.5", MaxTokens: 4096},
	})
	repo.providers["provider-default-model"] = provider

	result, err := svc.TestConnection(ctx, domain.NewLLMProviderID("provider-default-model"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证使用了 DefaultModel 而不是 SupportedModels[0]
	if !mockRunner.called {
		t.Error("期望 testRunner.RunTest 被调用")
	}
	if mockRunner.lastConfig == nil {
		t.Fatal("期望 lastConfig 不为 nil")
	}
	if mockRunner.lastConfig.Model != "gpt-4-turbo" {
		t.Errorf("期望模型为 'gpt-4-turbo', 实际为 '%s'", mockRunner.lastConfig.Model)
	}

	if result["success"] != false {
		t.Errorf("期望 success 为 false, 实际为 %v", result["success"])
	}
}

// TestLLMProviderService_TestConnection_FallbackModel 测试使用默认回退模型
func TestLLMProviderService_TestConnection_FallbackModel(t *testing.T) {
	mockRunner := &mockTestConnectionRunner{returnErr: errors.New("connection failed")}
	svc, repo, _ := setupTestLLMProviderSvcWithRunner(mockRunner)
	ctx := context.Background()

	// 创建一个既没有 DefaultModel 也没有 SupportedModels 的 provider
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-fallback"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-test-key",
		"https://api.openai.com",
	)
	// 不设置任何模型，让它使用默认的 "gpt-3.5-turbo"
	repo.providers["provider-fallback"] = provider

	result, err := svc.TestConnection(ctx, domain.NewLLMProviderID("provider-fallback"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证使用了回退模型
	if !mockRunner.called {
		t.Error("期望 testRunner.RunTest 被调用")
	}
	if mockRunner.lastConfig == nil {
		t.Fatal("期望 lastConfig 不为 nil")
	}
	if mockRunner.lastConfig.Model != "gpt-3.5-turbo" {
		t.Errorf("期望模型为 'gpt-3.5-turbo', 实际为 '%s'", mockRunner.lastConfig.Model)
	}

	if result["success"] != false {
		t.Errorf("期望 success 为 false, 实际为 %v", result["success"])
	}
}

// TestLLMProviderService_TestConnection_Success 测试连接成功场景
func TestLLMProviderService_TestConnection_Success(t *testing.T) {
	mockRunner := &mockTestConnectionRunner{returnErr: nil} // 成功场景
	svc, repo, _ := setupTestLLMProviderSvcWithRunner(mockRunner)
	ctx := context.Background()

	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("provider-success"),
		"user-001",
		"openai",
		"OpenAI",
		"sk-test-key",
		"https://api.openai.com",
	)
	provider.SetDefaultModel("gpt-4")
	provider.SetSupportedModels([]domain.ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", MaxTokens: 8192},
	})
	repo.providers["provider-success"] = provider

	result, err := svc.TestConnection(ctx, domain.NewLLMProviderID("provider-success"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// 验证传入 runner 的模型正确
	if !mockRunner.called {
		t.Error("期望 testRunner.RunTest 被调用")
	}
	if mockRunner.lastConfig == nil {
		t.Fatal("期望 lastConfig 不为 nil")
	}
	if mockRunner.lastConfig.Model != "gpt-4" {
		t.Errorf("期望传入 runner 的模型为 'gpt-4', 实际为 '%s'", mockRunner.lastConfig.Model)
	}

	// 验证返回结果
	if result["success"] != true {
		t.Errorf("期望 success 为 true, 实际为 %v", result["success"])
	}
	if result["message"] != "连接测试成功" {
		t.Errorf("期望 message 为 '连接测试成功', 实际为 '%v'", result["message"])
	}
	if result["model"] != "gpt-4" {
		t.Errorf("期望 model 为 'gpt-4', 实际为 '%v'", result["model"])
	}
}

// TestChooseModelForProvider 测试模型选择纯函数
func TestChooseModelForProvider(t *testing.T) {
	tests := []struct {
		name            string
		defaultModel    string
		supportedModels []domain.ModelInfo
		expected        string
	}{
		{
			name:            "使用 DefaultModel 当存在时",
			defaultModel:    "gpt-4-turbo",
			supportedModels: []domain.ModelInfo{{ID: "gpt-3.5", Name: "GPT-3.5"}},
			expected:        "gpt-4-turbo",
		},
		{
			name:            "使用 SupportedModels[0] 当 DefaultModel 为空",
			defaultModel:    "",
			supportedModels: []domain.ModelInfo{{ID: "gpt-4", Name: "GPT-4"}, {ID: "gpt-3.5", Name: "GPT-3.5"}},
			expected:        "gpt-4",
		},
		{
			name:            "使用回退模型当两者都为空",
			defaultModel:    "",
			supportedModels: []domain.ModelInfo{},
			expected:        "gpt-3.5-turbo",
		},
		{
			name:            "使用回退模型当 SupportedModels 为 nil",
			defaultModel:    "",
			supportedModels: nil,
			expected:        "gpt-3.5-turbo",
		},
		{
			name:            "优先使用 DefaultModel 即使 SupportedModels 为空",
			defaultModel:    "custom-model",
			supportedModels: []domain.ModelInfo{},
			expected:        "custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChooseModelForProvider(tt.defaultModel, tt.supportedModels)
			if result != tt.expected {
				t.Errorf("期望模型为 '%s', 实际为 '%s'", tt.expected, result)
			}
		})
	}
}
