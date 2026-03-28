/**
 * SQLite LLM Provider Repository 集成测试
 */
package persistence

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
)

func setupProviderTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	err = InitSchema(db)
	if err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createTestProvider(id, userCode, providerKey string) *domain.LLMProvider {
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID(id),
		userCode,
		providerKey,
		"Test Provider",
		"sk-test-key",
		"https://api.test.com",
	)
	return provider
}

func TestSQLiteLLMProviderRepository_SaveAndFindByID(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider := createTestProvider("prov-1", "usr_001", "openai")
	err := repo.Save(ctx, provider)
	if err != nil {
		t.Fatalf("保存 provider 失败: %v", err)
	}

	found, err := repo.FindByID(ctx, provider.ID())
	if err != nil {
		t.Fatalf("查找 provider 失败: %v", err)
	}

	if found.ProviderKey() != "openai" {
		t.Errorf("期望 provider_key 为 'openai', 实际为 '%s'", found.ProviderKey())
	}

	if found.ProviderName() != "Test Provider" {
		t.Errorf("期望 provider_name 为 'Test Provider', 实际为 '%s'", found.ProviderName())
	}
}

func TestSQLiteLLMProviderRepository_FindByUserCode(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider1 := createTestProvider("prov-1", "usr_001", "openai")
	provider2 := createTestProvider("prov-2", "usr_001", "claude")
	provider3 := createTestProvider("prov-3", "usr_002", "ollama")

	repo.Save(ctx, provider1)
	repo.Save(ctx, provider2)
	repo.Save(ctx, provider3)

	providers, err := repo.FindByUserCode(ctx, "usr_001")
	if err != nil {
		t.Fatalf("查找 providers 失败: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("期望 2 个 providers, 实际为 %d", len(providers))
	}
}

func TestSQLiteLLMProviderRepository_FindDefaultActive(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider1 := createTestProvider("prov-1", "usr_001", "openai")
	provider1.SetDefault(true)

	provider2 := createTestProvider("prov-2", "usr_001", "claude")
	provider2.SetDefault(false)

	repo.Save(ctx, provider1)
	repo.Save(ctx, provider2)

	defaultProvider, err := repo.FindDefaultActive(ctx, "usr_001")
	if err != nil {
		t.Fatalf("查找默认 provider 失败: %v", err)
	}

	if defaultProvider == nil {
		t.Fatal("期望找到默认 provider, 实际为 nil")
	}

	if defaultProvider.ProviderKey() != "openai" {
		t.Errorf("期望默认 provider 为 'openai', 实际为 '%s'", defaultProvider.ProviderKey())
	}
}

func TestSQLiteLLMProviderRepository_FindDefaultActive_Inactive(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider := createTestProvider("prov-1", "usr_001", "openai")
	provider.SetDefault(true)
	provider.SetActive(false)

	repo.Save(ctx, provider)

	defaultProvider, err := repo.FindDefaultActive(ctx, "usr_001")
	if err != nil {
		t.Fatalf("查找默认 provider 失败: %v", err)
	}

	if defaultProvider != nil {
		t.Error("期望返回 nil, 因为 provider 是非激活状态")
	}
}

func TestSQLiteLLMProviderRepository_ClearDefaultByUserCode(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider1 := createTestProvider("prov-1", "usr_001", "openai")
	provider1.SetDefault(true)

	provider2 := createTestProvider("prov-2", "usr_001", "claude")
	provider2.SetDefault(true)

	repo.Save(ctx, provider1)
	repo.Save(ctx, provider2)

	// 清除 usr_001 的所有默认设置
	err := repo.ClearDefaultByUserCode(ctx, "usr_001", nil)
	if err != nil {
		t.Fatalf("清除默认设置失败: %v", err)
	}

	// prov-1 应该是非默认
	found1, _ := repo.FindByID(ctx, provider1.ID())
	if found1.IsDefault() {
		t.Error("prov-1 应该是非默认状态")
	}

	// prov-2 应该是非默认
	found2, _ := repo.FindByID(ctx, provider2.ID())
	if found2.IsDefault() {
		t.Error("prov-2 应该是非默认状态")
	}
}

func TestSQLiteLLMProviderRepository_Delete(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider := createTestProvider("prov-1", "usr_001", "openai")
	repo.Save(ctx, provider)

	err := repo.Delete(ctx, provider.ID())
	if err != nil {
		t.Fatalf("删除 provider 失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, provider.ID())
	if found != nil {
		t.Error("期望 provider 已被删除")
	}
}

func TestSQLiteLLMProviderRepository_NotFound(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, domain.NewLLMProviderID("non-existent"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if found != nil {
		t.Error("期望返回 nil")
	}
}

func TestSQLiteLLMProviderRepository_Update(t *testing.T) {
	db, cleanup := setupProviderTestDB(t)
	defer cleanup()

	repo := NewSQLiteLLMProviderRepository(db)
	ctx := context.Background()

	provider := createTestProvider("prov-1", "usr_001", "openai")
	provider.SetPriority(1)
	provider.SetSupportedModels([]domain.ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", MaxTokens: 8000},
	})
	repo.Save(ctx, provider)

	// 更新
	provider.SetPriority(10)
	provider.UpdateProfile("openai", "Updated Provider", "sk-new-key", "https://api.new.com")
	err := repo.Save(ctx, provider)
	if err != nil {
		t.Fatalf("更新 provider 失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, provider.ID())
	if found.Priority() != 10 {
		t.Errorf("期望 priority 为 10, 实际为 %d", found.Priority())
	}
	if found.ProviderName() != "Updated Provider" {
		t.Errorf("期望 provider_name 为 'Updated Provider', 实际为 '%s'", found.ProviderName())
	}
}
