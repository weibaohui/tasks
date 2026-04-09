/**
 * UserHandler 单元测试
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

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

// mockUserRepository - 用于测试的 User 仓库模拟
type mockUserRepository struct {
	users     map[domain.UserID]*domain.User
	usernames map[string]*domain.User
	userCodes map[domain.UserCode]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:     make(map[domain.UserID]*domain.User),
		usernames: make(map[string]*domain.User),
		userCodes: make(map[domain.UserCode]*domain.User),
	}
}

func (r *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	r.users[user.ID()] = user
	r.usernames[user.Username()] = user
	r.userCodes[user.UserCode()] = user
	return nil
}

func (r *mockUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return r.users[id], nil
}

func (r *mockUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.usernames[username], nil
}

func (r *mockUserRepository) FindByUserCode(ctx context.Context, userCode domain.UserCode) (*domain.User, error) {
	return r.userCodes[userCode], nil
}

func (r *mockUserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
	var result []*domain.User
	for _, user := range r.users {
		result = append(result, user)
	}
	return result, nil
}

func (r *mockUserRepository) Delete(ctx context.Context, id domain.UserID) error {
	delete(r.users, id)
	return nil
}

// mockUserIDGenerator - 用于测试的 ID 生成器模拟
type mockUserIDGenerator struct {
	prefix string
	count  int
}

func newMockUserIDGenerator(prefix string) *mockUserIDGenerator {
	return &mockUserIDGenerator{prefix: prefix}
}

func (g *mockUserIDGenerator) Generate() string {
	g.count++
	return g.prefix + "-id-" + strconv.Itoa(g.count)
}

// setupGinContext 创建用于测试的 Gin 上下文
func setupGinContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewBuffer(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	return c, w
}

func TestUserHandler_CreateUser_InvalidJSON(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))
	handler := NewUserHandler(svc)

	c, w := setupGinContext("POST", "/users", []byte("invalid json"))
	handler.CreateUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))
	handler := NewUserHandler(svc)

	body := `{
		"username": "testuser",
		"email": "test@example.com",
		"display_name": "Test User",
		"password": "password123"
	}`
	c, w := setupGinContext("POST", "/users", []byte(body))
	handler.CreateUser(c)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp["username"] != "testuser" {
		t.Errorf("期望 username 为 testuser, 实际为 %v", resp["username"])
	}
}

func TestUserHandler_ListUsers(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))

	// 先创建一个 user
	svc.CreateUser(context.Background(), application.CreateUserCommand{
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Password:    "password123",
	})

	handler := NewUserHandler(svc)

	c, w := setupGinContext("GET", "/users", nil)
	handler.ListUsers(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if len(resp) != 1 {
		t.Errorf("期望 1 个 user, 实际为 %d", len(resp))
	}
}

func TestUserHandler_GetUser_NoID(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))
	handler := NewUserHandler(svc)

	c, w := setupGinContext("GET", "/user", nil)
	handler.GetUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))
	handler := NewUserHandler(svc)

	c, w := setupGinContext("GET", "/user?id=nonexistent", nil)
	handler.GetUser(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestUserHandler_UpdateUser_NoID(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))
	handler := NewUserHandler(svc)

	c, w := setupGinContext("PUT", "/user", []byte("{}"))
	handler.UpdateUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserHandler_DeleteUser_NoID(t *testing.T) {
	repo := newMockUserRepository()
	svc := application.NewUserApplicationService(repo, newMockUserIDGenerator("u"))
	handler := NewUserHandler(svc)

	c, w := setupGinContext("DELETE", "/user", nil)
	handler.DeleteUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserToMap(t *testing.T) {
	user, _ := domain.NewUser(
		domain.NewUserID("u-001"),
		domain.NewUserCode("user-001"),
		"testuser",
		"test@example.com",
		"Test User",
		"hash",
	)

	result := userToMap(user)

	if result["id"] != "u-001" {
		t.Errorf("期望 id 为 u-001, 实际为 %v", result["id"])
	}

	if result["username"] != "testuser" {
		t.Errorf("期望 username 为 testuser, 实际为 %v", result["username"])
	}

	if result["email"] != "test@example.com" {
		t.Errorf("期望 email 为 test@example.com, 实际为 %v", result["email"])
	}

	if result["display_name"] != "Test User" {
		t.Errorf("期望 display_name 为 Test User, 实际为 %v", result["display_name"])
	}
}
