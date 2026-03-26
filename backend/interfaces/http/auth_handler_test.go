/**
 * Auth Handler 测试
 */
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type mockAuthUserRepository struct {
	users map[string]*domain.User
}

func newMockAuthUserRepository() *mockAuthUserRepository {
	return &mockAuthUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockAuthUserRepository) Save(ctx context.Context, user *domain.User) error {
	m.users[user.ID().String()] = user
	return nil
}

func (m *mockAuthUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	user, ok := m.users[id.String()]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *mockAuthUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username() == username {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockAuthUserRepository) FindByUserCode(ctx context.Context, userCode domain.UserCode) (*domain.User, error) {
	for _, user := range m.users {
		if user.UserCode().String() == userCode.String() {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockAuthUserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
	var result []*domain.User
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, nil
}

func (m *mockAuthUserRepository) Delete(ctx context.Context, id domain.UserID) error {
	delete(m.users, id.String())
	return nil
}

type mockAuthIDGenerator struct {
	count int
}

func (m *mockAuthIDGenerator) Generate() string {
	m.count++
	return "user-id-" + strconv.Itoa(m.count)
}

func setupTestAuthHandler() (*AuthHandler, *mockAuthUserRepository) {
	repo := newMockAuthUserRepository()
	idGen := &mockAuthIDGenerator{}
	userService := application.NewUserApplicationService(repo, idGen)
	handler := NewAuthHandler(userService, "test-secret-key", 24*time.Hour)
	return handler, repo
}

func setupAuthMux(handler *AuthHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", handler.Login)
	mux.HandleFunc("/api/v1/auth/me", handler.Me)
	return mux
}

func TestLogin_Success(t *testing.T) {
	handler, repo := setupTestAuthHandler()
	mux := setupAuthMux(handler)
	ctx := context.Background()

	// 创建一个用户（密码是 "password123"）
	user, _ := domain.NewUser(
		domain.NewUserID("user-1"),
		domain.NewUserCode("usr_001"),
		"testuser",
		"test@example.com",
		"Test User",
		"sha256$5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8", // password
	)
	repo.Save(ctx, user)

	body := `{"username": "testuser", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["token"] == nil || resp["token"] == "" {
		t.Error("响应中应该包含 token")
	}

	if resp["expires_at"] == nil {
		t.Error("响应中应该包含 expires_at")
	}

	userMap, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("响应中应该包含 user")
	}

	if userMap["username"] != "testuser" {
		t.Errorf("期望 username 为 'testuser', 实际为 '%v'", userMap["username"])
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	handler, _ := setupTestAuthHandler()
	mux := setupAuthMux(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	handler, _ := setupTestAuthHandler()
	mux := setupAuthMux(handler)

	body := `{"username": "nonexistent", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	handler, repo := setupTestAuthHandler()
	mux := setupAuthMux(handler)
	ctx := context.Background()

	// 创建一个用户
	user, _ := domain.NewUser(
		domain.NewUserID("user-1"),
		domain.NewUserCode("usr_001"),
		"testuser",
		"test@example.com",
		"Test User",
		"sha256$5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8", // password
	)
	repo.Save(ctx, user)

	body := `{"username": "testuser", "password": "wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	handler, repo := setupTestAuthHandler()
	mux := setupAuthMux(handler)
	ctx := context.Background()

	// 创建一个用户并停用它
	user, _ := domain.NewUser(
		domain.NewUserID("user-1"),
		domain.NewUserCode("usr_001"),
		"inactiveuser",
		"test@example.com",
		"Test User",
		"sha256$5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8", // password
	)
	user.Deactivate()
	repo.Save(ctx, user)

	body := `{"username": "inactiveuser", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMe_WithValidToken(t *testing.T) {
	handler, repo := setupTestAuthHandler()
	mux := setupAuthMux(handler)
	ctx := context.Background()

	// 创建一个用户
	user, _ := domain.NewUser(
		domain.NewUserID("user-1"),
		domain.NewUserCode("usr_001"),
		"testuser",
		"test@example.com",
		"Test User",
		"sha256$5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
	)
	repo.Save(ctx, user)

	// 先登录获取 token
	loginBody := `{"username": "testuser", "password": "password"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	mux.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// 使用 token 访问 /me
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meW := httptest.NewRecorder()

	mux.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, meW.Code)
	}

	var meResp map[string]interface{}
	json.Unmarshal(meW.Body.Bytes(), &meResp)

	if meResp["username"] != "testuser" {
		t.Errorf("期望 username 为 'testuser', 实际为 '%v'", meResp["username"])
	}
}

func TestMe_WithoutToken(t *testing.T) {
	handler, _ := setupTestAuthHandler()
	mux := setupAuthMux(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMe_InvalidToken(t *testing.T) {
	handler, _ := setupTestAuthHandler()
	mux := setupAuthMux(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMe_ExpiredToken(t *testing.T) {
	handler, repo := setupTestAuthHandler()
	mux := setupAuthMux(handler)
	ctx := context.Background()

	// 创建一个用户
	user, _ := domain.NewUser(
		domain.NewUserID("user-1"),
		domain.NewUserCode("usr_001"),
		"testuser",
		"test@example.com",
		"Test User",
		"sha256$5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
	)
	repo.Save(ctx, user)

	// 手动创建一个过期的 token
	expiredHandler := NewAuthHandler(
		application.NewUserApplicationService(repo, &mockAuthIDGenerator{}),
		"test-secret-key",
		-1*time.Hour, // 负数的 TTL 导致 token 立即过期
	)

	body := `{"username": "testuser", "password": "password"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	expiredHandler.Login(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// 使用过期的 token 访问 /me
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meW := httptest.NewRecorder()

	mux.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusUnauthorized, meW.Code)
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid bearer", "Bearer abc123", "abc123"},
		{"lowercase bearer", "bearer xyz789", "xyz789"},
		{"empty header", "", ""},
		{"no bearer prefix", "abc123", ""},
		{"only bearer", "Bearer", ""},
		{"bearer with multiple spaces", "Bearer   token123", "token123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBearerToken(tt.header)
			if result != tt.expected {
				t.Errorf("期望 '%s', 实际为 '%s'", tt.expected, result)
			}
		})
	}
}

func TestSignPayload(t *testing.T) {
	sig1 := signPayload([]byte("secret"), "payload")
	sig2 := signPayload([]byte("secret"), "payload")
	sig3 := signPayload([]byte("secret"), "different")

	if sig1 != sig2 {
		t.Error("相同输入应该产生相同签名")
	}

	if sig1 == sig3 {
		t.Error("不同输入应该产生不同签名")
	}
}