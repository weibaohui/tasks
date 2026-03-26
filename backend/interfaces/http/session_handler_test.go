/**
 * SessionHandler 单元测试
 */
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

// mockSessionRepositoryForHandler - 用于测试的 Session 仓库模拟
type mockSessionRepositoryForHandler struct {
	sessions      map[domain.SessionID]*domain.Session
	sessionKeys   map[string]*domain.Session
	saveErr       error
	findByKeyErr  error
	findByIDErr   error
	deleteErr     error
}

func newMockSessionRepositoryForHandler() *mockSessionRepositoryForHandler {
	return &mockSessionRepositoryForHandler{
		sessions:    make(map[domain.SessionID]*domain.Session),
		sessionKeys: make(map[string]*domain.Session),
	}
}

func (r *mockSessionRepositoryForHandler) Save(ctx context.Context, session *domain.Session) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.sessions[session.ID()] = session
	r.sessionKeys[session.SessionKey()] = session
	return nil
}

func (r *mockSessionRepositoryForHandler) FindByID(ctx context.Context, id domain.SessionID) (*domain.Session, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	return r.sessions[id], nil
}

func (r *mockSessionRepositoryForHandler) FindBySessionKey(ctx context.Context, key string) (*domain.Session, error) {
	if r.findByKeyErr != nil {
		return nil, r.findByKeyErr
	}
	return r.sessionKeys[key], nil
}

func (r *mockSessionRepositoryForHandler) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range r.sessions {
		if s.UserCode() == userCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range r.sessions {
		if s.UserCode() == userCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) FindByChannelCode(ctx context.Context, channelCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range r.sessions {
		if s.ChannelCode() == channelCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) DeleteBySessionKey(ctx context.Context, sessionKey string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	// Find session by key and delete from both maps
	if session, exists := r.sessionKeys[sessionKey]; exists {
		delete(r.sessions, session.ID())
	}
	delete(r.sessionKeys, sessionKey)
	return nil
}

func (r *mockSessionRepositoryForHandler) DeleteByChannelCode(ctx context.Context, channelCode string) error {
	for key, s := range r.sessionKeys {
		if s.ChannelCode() == channelCode {
			delete(r.sessionKeys, key)
		}
	}
	return nil
}

// mockSessionIDGeneratorForHandler - 用于测试的 ID 生成器模拟
type mockSessionIDGeneratorForHandler struct {
	prefix string
	count  int
}

func newMockSessionIDGeneratorForHandler(prefix string) *mockSessionIDGeneratorForHandler {
	return &mockSessionIDGeneratorForHandler{prefix: prefix}
}

func (g *mockSessionIDGeneratorForHandler) Generate() string {
	g.count++
	return g.prefix + "-" + strconv.Itoa(g.count)
}

func TestSessionHandler_CreateSession_InvalidJSON(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("POST", "/sessions", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_CreateSession_Success(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	body := `{
		"user_code": "user-001",
		"channel_code": "channel-001",
		"agent_code": "agent-001",
		"session_key": "key-001"
	}`
	req := httptest.NewRequest("POST", "/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateSession(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp["session_key"] != "key-001" {
		t.Errorf("期望 session_key 为 key-001, 实际为 %v", resp["session_key"])
	}
}

func TestSessionHandler_ListSessions_NoParams(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("GET", "/sessions", nil)
	w := httptest.NewRecorder()

	handler.ListSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_ListSessions_ByUserCode(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))

	// 先创建一个 session
	_, err := svc.CreateSession(context.Background(), application.CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})
	if err != nil {
		t.Fatalf("CreateSession 失败: %v", err)
	}

	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("GET", "/sessions?user_code=user-001", nil)
	w := httptest.NewRecorder()

	handler.ListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if len(resp) != 1 {
		t.Errorf("期望 1 个 session, 实际为 %d", len(resp))
	}
}

func TestSessionHandler_GetSession_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("GET", "/session", nil)
	w := httptest.NewRecorder()

	handler.GetSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_GetSession_NotFound(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("GET", "/session?session_key=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.GetSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestSessionHandler_DeleteSession_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("DELETE", "/session", nil)
	w := httptest.NewRecorder()

	handler.DeleteSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_TouchSession_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("PUT", "/sessions//touch", nil)
	w := httptest.NewRecorder()

	handler.TouchSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_UpdateSessionMetadata_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("PUT", "/sessions//metadata", strings.NewReader("{}"))
	w := httptest.NewRecorder()

	handler.UpdateSessionMetadata(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_UpdateSessionMetadata_InvalidJSON(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	req := httptest.NewRequest("PUT", "/sessions/key-001/metadata", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handler.UpdateSessionMetadata(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestExtractSessionKey(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sessions/key-001", "key-001"},
		{"/sessions/my-key/metadata", "my-key"},
		{"/sessions/", ""},
		{"/other/key", ""},
		{"/sessions", ""},
	}

	for _, tt := range tests {
		result := extractSessionKey(tt.path)
		if result != tt.expected {
			t.Errorf("extractSessionKey(%q) = %q, 期望 %q", tt.path, result, tt.expected)
		}
	}
}

func TestSessionToMap(t *testing.T) {
	session, err := domain.NewSession(
		domain.NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)
	if err != nil {
		t.Fatalf("创建 Session 失败: %v", err)
	}

	result := sessionToMap(session)

	if result["id"] != "sess-001" {
		t.Errorf("期望 id 为 sess-001, 实际为 %v", result["id"])
	}

	if result["user_code"] != "user-001" {
		t.Errorf("期望 user_code 为 user-001, 实际为 %v", result["user_code"])
	}

	if result["channel_code"] != "channel-001" {
		t.Errorf("期望 channel_code 为 channel-001, 实际为 %v", result["channel_code"])
	}

	if result["session_key"] != "key-001" {
		t.Errorf("期望 session_key 为 key-001, 实际为 %v", result["session_key"])
	}

	if result["external_id"] != "ext-001" {
		t.Errorf("期望 external_id 为 ext-001, 实际为 %v", result["external_id"])
	}
}