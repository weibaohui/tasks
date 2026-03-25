package application

import (
	"context"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	m.users[user.ID().String()] = user
	return nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	user, ok := m.users[id.String()]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username() == username {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) FindByUserCode(ctx context.Context, userCode domain.UserCode) (*domain.User, error) {
	for _, user := range m.users {
		if user.UserCode().String() == userCode.String() {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
	var result []*domain.User
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id domain.UserID) error {
	delete(m.users, id.String())
	return nil
}

type mockUserIDGenerator struct {
	count int
}

func (m *mockUserIDGenerator) Generate() string {
	m.count++
	return "user-id-" + string(rune('0'+m.count))
}

func setupTestUserService() *UserApplicationService {
	repo := newMockUserRepository()
	idGen := &mockUserIDGenerator{}
	return NewUserApplicationService(repo, idGen)
}

func TestCreateUser(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, CreateUserCommand{
		Username:    "testuser",
		Email:      "test@example.com",
		DisplayName: "Test User",
		Password:    "password123",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if user.Username() != "testuser" {
		t.Errorf("期望 username 为 'testuser', 实际为 '%s'", user.Username())
	}

	if user.Email() != "test@example.com" {
		t.Errorf("期望 email 为 'test@example.com', 实际为 '%s'", user.Email())
	}

	if user.DisplayName() != "Test User" {
		t.Errorf("期望 display_name 为 'Test User', 实际为 '%s'", user.DisplayName())
	}

	if !user.IsActive() {
		t.Error("新创建的用户应该是激活状态")
	}

	if user.UserCode().String() == "" {
		t.Error("user_code 不应该为空")
	}
}

func TestCreateUser_WithPasswordHash(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	// 使用已哈希的密码
	passwordHash := "sha256$" + "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8" // password
	user, err := svc.CreateUser(ctx, CreateUserCommand{
		Username:     "hashuser",
		Email:       "hash@example.com",
		DisplayName: "Hash User",
		PasswordHash: passwordHash,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if user.Username() != "hashuser" {
		t.Errorf("期望 username 为 'hashuser', 实际为 '%s'", user.Username())
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, CreateUserCommand{
		Username: "duplicate",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("第一次创建不应该出错: %v", err)
	}

	_, err = svc.CreateUser(ctx, CreateUserCommand{
		Username: "duplicate",
		Password: "password456",
	})
	if err != ErrUsernameDuplicated {
		t.Errorf("期望 ErrUsernameDuplicated, 实际为 %v", err)
	}
}

func TestGetUser(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username: "getuser",
		Password: "password123",
	})

	user, err := svc.GetUser(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if user.Username() != "getuser" {
		t.Errorf("期望 username 为 'getuser', 实际为 '%s'", user.Username())
	}
}

func TestGetUser_NotFound(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	_, err := svc.GetUser(ctx, domain.NewUserID("non-existent"))
	if err != ErrUserNotFound {
		t.Errorf("期望 ErrUserNotFound, 实际为 %v", err)
	}
}

func TestListUsers(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	svc.CreateUser(ctx, CreateUserCommand{Username: "user1", Password: "pass"})
	svc.CreateUser(ctx, CreateUserCommand{Username: "user2", Password: "pass"})

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(users) != 2 {
		t.Errorf("期望 2 个用户, 实际为 %d", len(users))
	}
}

func TestUpdateUser(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username:    "updateuser",
		Email:       "old@example.com",
		DisplayName: "Old Name",
		Password:    "password123",
	})

	isActive := false
	updated, err := svc.UpdateUser(ctx, UpdateUserCommand{
		ID:          created.ID(),
		Email:       "new@example.com",
		DisplayName: "New Name",
		IsActive:    &isActive,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Email() != "new@example.com" {
		t.Errorf("期望 email 为 'new@example.com', 实际为 '%s'", updated.Email())
	}

	if updated.DisplayName() != "New Name" {
		t.Errorf("期望 display_name 为 'New Name', 实际为 '%s'", updated.DisplayName())
	}

	if updated.IsActive() {
		t.Error("用户应该是非激活状态")
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	_, err := svc.UpdateUser(ctx, UpdateUserCommand{
		ID:       domain.NewUserID("non-existent"),
		Email:    "new@example.com",
	})
	if err != ErrUserNotFound {
		t.Errorf("期望 ErrUserNotFound, 实际为 %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username: "deleteuser",
		Password: "password123",
	})

	err := svc.DeleteUser(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	_, err = svc.GetUser(ctx, created.ID())
	if err != ErrUserNotFound {
		t.Errorf("期望 ErrUserNotFound, 实际为 %v", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	err := svc.DeleteUser(ctx, domain.NewUserID("non-existent"))
	if err != ErrUserNotFound {
		t.Errorf("期望 ErrUserNotFound, 实际为 %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	svc.CreateUser(ctx, CreateUserCommand{
		Username: "authuser",
		Password: "correctpassword",
	})

	user, err := svc.Authenticate(ctx, "authuser", "correctpassword")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if user.Username() != "authuser" {
		t.Errorf("期望 username 为 'authuser', 实际为 '%s'", user.Username())
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	svc.CreateUser(ctx, CreateUserCommand{
		Username: "authuser",
		Password: "correctpassword",
	})

	_, err := svc.Authenticate(ctx, "authuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("期望 ErrInvalidCredentials, 实际为 %v", err)
	}
}

func TestAuthenticate_UserNotFound(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	_, err := svc.Authenticate(ctx, "nonexistent", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("期望 ErrInvalidCredentials, 实际为 %v", err)
	}
}

func TestAuthenticate_InactiveUser(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	user, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username: "inactiveuser",
		Password: "password123",
	})

	// 停用用户
	isActive := false
	svc.UpdateUser(ctx, UpdateUserCommand{
		ID:       user.ID(),
		IsActive: &isActive,
	})

	_, err := svc.Authenticate(ctx, "inactiveuser", "password123")
	if err != ErrUserInactive {
		t.Errorf("期望 ErrUserInactive, 实际为 %v", err)
	}
}

func TestBuildStoredPasswordValue(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		passwordHash string
		expectHasPrefix bool
	}{
		{"plain password", "mypassword", "", true},
		{"sha256 prefix", "", "sha256$abc123", true},
		{"sha256: prefix", "", "sha256:abc123", true},
		{"hex 64 chars", "", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.BuildStoredPasswordValue(tt.password, tt.passwordHash)
			if tt.expectHasPrefix && result == "" {
				t.Error("期望非空结果")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	// Create a user to test VerifyPassword method
	svc := setupTestUserService()
	ctx := context.Background()

	user, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username: "verifytest",
		Password: "testpassword",
	})

	// Test correct password
	if !user.VerifyPassword("testpassword") {
		t.Error("正确的密码应该验证通过")
	}

	// Test wrong password
	if user.VerifyPassword("wrongpassword") {
		t.Error("错误的密码应该验证失败")
	}
}

func TestHashPassword(t *testing.T) {
	// Test through user creation
	svc := setupTestUserService()
	ctx := context.Background()

	user, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username: "hashtest",
		Password: "testpassword",
	})

	// The stored hash should have sha256$ prefix
	hash := user.PasswordHash()
	if len(hash) < 7 || hash[:7] != "sha256$" {
		t.Errorf("密码哈希应该有 sha256$ 前缀, 实际为: %s", hash)
	}
}

func TestUserTimestamps(t *testing.T) {
	svc := setupTestUserService()
	ctx := context.Background()

	before := time.Now()
	user, _ := svc.CreateUser(ctx, CreateUserCommand{
		Username: "timeuser",
		Password: "password123",
	})
	after := time.Now()

	if user.CreatedAt().Before(before) || user.CreatedAt().After(after) {
		t.Error("created_at 时间戳不在合理范围内")
	}
}