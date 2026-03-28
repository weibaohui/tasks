/**
 * SQLite User Repository 集成测试
 */
package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
)

func setupUserTestDB(t *testing.T) (*sql.DB, func()) {
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

func createTestUser(id, username, email string) *domain.User {
	user, _ := domain.NewUser(
		domain.NewUserID(id),
		domain.NewUserCode("usr_"+id),
		username,
		email,
		"Test User",
		"sha256$testhash",
	)
	return user
}

func TestSQLiteUserRepository_SaveAndFindByID(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	user := createTestUser("user-1", "testuser1", "test1@example.com")
	err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("保存用户失败: %v", err)
	}

	found, err := repo.FindByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("查找用户失败: %v", err)
	}

	if found.Username() != "testuser1" {
		t.Errorf("期望 username 为 'testuser1', 实际为 '%s'", found.Username())
	}

	if found.Email() != "test1@example.com" {
		t.Errorf("期望 email 为 'test1@example.com', 实际为 '%s'", found.Email())
	}
}

func TestSQLiteUserRepository_FindByUsername(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	user := createTestUser("user-1", "findbyusername", "test@example.com")
	repo.Save(ctx, user)

	found, err := repo.FindByUsername(ctx, "findbyusername")
	if err != nil {
		t.Fatalf("查找用户失败: %v", err)
	}

	if found == nil {
		t.Fatal("期望找到用户, 实际为 nil")
	}

	if found.ID().String() != user.ID().String() {
		t.Errorf("期望 ID 为 %s, 实际为 %s", user.ID(), found.ID())
	}
}

func TestSQLiteUserRepository_FindByUserCode(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	user := createTestUser("user-1", "findbyusercode", "test@example.com")
	repo.Save(ctx, user)

	found, err := repo.FindByUserCode(ctx, user.UserCode())
	if err != nil {
		t.Fatalf("查找用户失败: %v", err)
	}

	if found == nil {
		t.Fatal("期望找到用户, 实际为 nil")
	}

	if found.Username() != "findbyusercode" {
		t.Errorf("期望 username 为 'findbyusercode', 实际为 '%s'", found.Username())
	}
}

func TestSQLiteUserRepository_FindAll(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	repo.Save(ctx, createTestUser("user-1", "user1", "user1@example.com"))
	repo.Save(ctx, createTestUser("user-2", "user2", "user2@example.com"))
	repo.Save(ctx, createTestUser("user-3", "user3", "user3@example.com"))

	users, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("查找所有用户失败: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("期望 3 个用户, 实际为 %d", len(users))
	}
}

func TestSQLiteUserRepository_Update(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	user := createTestUser("user-1", "updateuser", "old@example.com")
	repo.Save(ctx, user)

	// 更新用户
	user.UpdateProfile("new@example.com", "New Display Name")
	err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("更新用户失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, user.ID())
	if found.Email() != "new@example.com" {
		t.Errorf("期望 email 为 'new@example.com', 实际为 '%s'", found.Email())
	}
	if found.DisplayName() != "New Display Name" {
		t.Errorf("期望 display_name 为 'New Display Name', 实际为 '%s'", found.DisplayName())
	}
}

func TestSQLiteUserRepository_ActivateDeactivate(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	user := createTestUser("user-1", "activeuser", "test@example.com")
	repo.Save(ctx, user)

	if !user.IsActive() {
		t.Error("新用户应该是激活状态")
	}

	user.Deactivate()
	repo.Save(ctx, user)

	found, _ := repo.FindByID(ctx, user.ID())
	if found.IsActive() {
		t.Error("用户应该是非激活状态")
	}

	found.Activate()
	repo.Save(ctx, found)

	found2, _ := repo.FindByID(ctx, user.ID())
	if !found2.IsActive() {
		t.Error("用户应该是激活状态")
	}
}

func TestSQLiteUserRepository_Delete(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	user := createTestUser("user-1", "deleteuser", "test@example.com")
	repo.Save(ctx, user)

	err := repo.Delete(ctx, user.ID())
	if err != nil {
		t.Fatalf("删除用户失败: %v", err)
	}

	found, _ := repo.FindByID(ctx, user.ID())
	if found != nil {
		t.Error("期望用户已被删除")
	}
}

func TestSQLiteUserRepository_NotFound(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, domain.NewUserID("non-existent"))
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if found != nil {
		t.Error("期望返回 nil")
	}

	found, err = repo.FindByUsername(ctx, "non-existent")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if found != nil {
		t.Error("期望返回 nil")
	}
}

func TestSQLiteUserRepository_Timestamps(t *testing.T) {
	db, cleanup := setupUserTestDB(t)
	defer cleanup()

	repo := NewSQLiteUserRepository(db)
	ctx := context.Background()

	before := time.Now().Add(-1 * time.Second)
	user := createTestUser("user-1", "timeuser", "test@example.com")
	repo.Save(ctx, user)
	after := time.Now().Add(1 * time.Second)

	found, _ := repo.FindByID(ctx, user.ID())

	if found.CreatedAt().Before(before) || found.CreatedAt().After(after) {
		t.Error("created_at 时间戳不在合理范围内")
	}
}
