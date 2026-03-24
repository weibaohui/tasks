/**
 * User 聚合根单元测试
 */
package domain

import (
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	user, err := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"zhangsan@example.com",
		"张三",
		"hashed_password_123",
	)

	if err != nil {
		t.Fatalf("创建User失败: %v", err)
	}

	if user.ID() != NewUserID("user-001") {
		t.Errorf("期望UserID为 user-001, 实际为 %s", user.ID())
	}

	if user.UserCode() != NewUserCode("user-zhangsan") {
		t.Errorf("期望UserCode为 user-zhangsan, 实际为 %s", user.UserCode())
	}

	if user.Username() != "zhangsan" {
		t.Errorf("期望Username为 zhangsan, 实际为 %s", user.Username())
	}

	if user.Email() != "zhangsan@example.com" {
		t.Errorf("期望Email为 zhangsan@example.com, 实际为 %s", user.Email())
	}

	if user.DisplayName() != "张三" {
		t.Errorf("期望DisplayName为 张三, 实际为 %s", user.DisplayName())
	}

	if user.PasswordHash() != "hashed_password_123" {
		t.Errorf("期望PasswordHash为 hashed_password_123, 实际为 %s", user.PasswordHash())
	}

	if !user.IsActive() {
		t.Error("期望IsActive为true")
	}

	if user.CreatedAt().IsZero() {
		t.Error("期望CreatedAt不为零值")
	}

	if user.UpdatedAt().IsZero() {
		t.Error("期望UpdatedAt不为零值")
	}
}

func TestNewUser_EmptyID(t *testing.T) {
	_, err := NewUser(
		NewUserID(""),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"",
		"",
		"hash",
	)

	if err != ErrUserIDRequired {
		t.Errorf("期望返回 ErrUserIDRequired, 实际返回 %v", err)
	}
}

func TestNewUser_EmptyUserCode(t *testing.T) {
	_, err := NewUser(
		NewUserID("user-001"),
		NewUserCode(""),
		"zhangsan",
		"",
		"",
		"hash",
	)

	if err != ErrUserCodeRequired {
		t.Errorf("期望返回 ErrUserCodeRequired, 实际返回 %v", err)
	}
}

func TestNewUser_EmptyUsername(t *testing.T) {
	_, err := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"   ", // 空白
		"",
		"",
		"hash",
	)

	if err != ErrUsernameRequired {
		t.Errorf("期望返回 ErrUsernameRequired, 实际返回 %v", err)
	}
}

func TestNewUser_EmptyPasswordHash(t *testing.T) {
	_, err := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"",
		"",
		"   ", // 空白
	)

	if err != ErrPasswordHashRequired {
		t.Errorf("期望返回 ErrPasswordHashRequired, 实际返回 %v", err)
	}
}

func TestUser_UpdateProfile(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"old@example.com",
		"旧名称",
		"hash",
	)

	originalUpdatedAt := user.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	user.UpdateProfile("new@example.com", "新名称")

	if user.Email() != "new@example.com" {
		t.Errorf("期望Email为 new@example.com, 实际为 %s", user.Email())
	}

	if user.DisplayName() != "新名称" {
		t.Errorf("期望DisplayName为 新名称, 实际为 %s", user.DisplayName())
	}

	if !user.UpdatedAt().After(originalUpdatedAt) {
		t.Error("期望UpdatedAt已更新")
	}
}

func TestUser_UpdateProfile_EmptyValues(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"old@example.com",
		"旧名称",
		"hash",
	)

	// 空字符串应该被接受
	user.UpdateProfile("", "")

	if user.Email() != "" {
		t.Errorf("期望Email为空, 实际为 %s", user.Email())
	}

	if user.DisplayName() != "" {
		t.Errorf("期望DisplayName为空, 实际为 %s", user.DisplayName())
	}
}

func TestUser_ChangePasswordHash(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"",
		"",
		"old_hash",
	)

	originalUpdatedAt := user.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	err := user.ChangePasswordHash("new_hash_456")
	if err != nil {
		t.Fatalf("ChangePasswordHash失败: %v", err)
	}

	if user.PasswordHash() != "new_hash_456" {
		t.Errorf("期望PasswordHash为 new_hash_456, 实际为 %s", user.PasswordHash())
	}

	if !user.UpdatedAt().After(originalUpdatedAt) {
		t.Error("期望UpdatedAt已更新")
	}
}

func TestUser_ChangePasswordHash_Empty(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"",
		"",
		"old_hash",
	)

	err := user.ChangePasswordHash("   ")
	if err != ErrPasswordHashRequired {
		t.Errorf("期望返回 ErrPasswordHashRequired, 实际返回 %v", err)
	}

	// 密码不应被修改
	if user.PasswordHash() != "old_hash" {
		t.Errorf("密码不应被修改，期望 old_hash, 实际为 %s", user.PasswordHash())
	}
}

func TestUser_Activate(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"",
		"",
		"hash",
	)

	user.Deactivate()
	if user.IsActive() {
		t.Error("Deactivate后IsActive应为false")
	}

	originalUpdatedAt := user.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	user.Activate()
	if !user.IsActive() {
		t.Error("Activate后IsActive应为true")
	}

	if !user.UpdatedAt().After(originalUpdatedAt) {
		t.Error("期望UpdatedAt已更新")
	}
}

func TestUser_Deactivate(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"",
		"",
		"hash",
	)

	if !user.IsActive() {
		t.Error("初始IsActive应为true")
	}

	originalUpdatedAt := user.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	user.Deactivate()
	if user.IsActive() {
		t.Error("Deactivate后IsActive应为false")
	}

	if !user.UpdatedAt().After(originalUpdatedAt) {
		t.Error("期望UpdatedAt已更新")
	}
}

func TestUser_ToSnapshot(t *testing.T) {
	user, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"zhangsan@example.com",
		"张三",
		"hash123",
	)

	user.Deactivate()

	snap := user.ToSnapshot()

	if snap.ID != user.ID() {
		t.Errorf("ID不匹配")
	}

	if snap.UserCode != user.UserCode() {
		t.Errorf("UserCode不匹配")
	}

	if snap.Username != user.Username() {
		t.Errorf("Username不匹配")
	}

	if snap.Email != user.Email() {
		t.Errorf("Email不匹配")
	}

	if snap.DisplayName != user.DisplayName() {
		t.Errorf("DisplayName不匹配")
	}

	if snap.PasswordHash != user.PasswordHash() {
		t.Errorf("PasswordHash不匹配")
	}

	if snap.IsActive != user.IsActive() {
		t.Errorf("IsActive不匹配")
	}
}

func TestUser_FromSnapshot(t *testing.T) {
	original, _ := NewUser(
		NewUserID("user-001"),
		NewUserCode("user-zhangsan"),
		"zhangsan",
		"zhangsan@example.com",
		"张三",
		"hash123",
	)

	original.Deactivate()

	snap := original.ToSnapshot()

	restored := &User{}
	restored.FromSnapshot(snap)

	if restored.ID() != original.ID() {
		t.Errorf("ID不匹配")
	}

	if restored.UserCode() != original.UserCode() {
		t.Errorf("UserCode不匹配")
	}

	if restored.Username() != original.Username() {
		t.Errorf("Username不匹配")
	}

	if restored.Email() != original.Email() {
		t.Errorf("Email不匹配")
	}

	if restored.DisplayName() != original.DisplayName() {
		t.Errorf("DisplayName不匹配")
	}

	if restored.PasswordHash() != original.PasswordHash() {
		t.Errorf("PasswordHash不匹配")
	}

	if restored.IsActive() != original.IsActive() {
		t.Errorf("IsActive不匹配")
	}

	if restored.CreatedAt() != original.CreatedAt() {
		t.Errorf("CreatedAt不匹配")
	}

	if restored.UpdatedAt() != original.UpdatedAt() {
		t.Errorf("UpdatedAt不匹配")
	}
}

func TestUserID_String(t *testing.T) {
	id := NewUserID("user-001")
	if id.String() != "user-001" {
		t.Errorf("期望 user-001, 实际 %s", id.String())
	}
}

func TestUserCode_String(t *testing.T) {
	code := NewUserCode("user-zhangsan")
	if code.String() != "user-zhangsan" {
		t.Errorf("期望 user-zhangsan, 实际 %s", code.String())
	}
}
