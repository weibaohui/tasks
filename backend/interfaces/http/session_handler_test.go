/**
 * SessionHandler 单元测试
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

// mockSessionRepositoryForHandler - 用于测试的 Session 仓库模拟
type mockSessionRepositoryForHandler struct {
	sessions     map[domain.SessionID]*domain.Session
	sessionKeys  map[string]*domain.Session
	saveErr      error
	findByKeyErr error
	findByIDErr  error
	deleteErr    error
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

func (r *mockSessionRepositoryForHandler) FindByKey(ctx context.Context, key string) (*domain.Session, error) {
	if r.findByKeyErr != nil {
		return nil, r.findByKeyErr
	}
	return r.sessionKeys[key], nil
}

func (r *mockSessionRepositoryForHandler) Delete(ctx context.Context, id domain.SessionID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	session, ok := r.sessions[id]
	if ok {
		delete(r.sessions, id)
		delete(r.sessionKeys, session.SessionKey())
	}
	return nil
}

func (r *mockSessionRepositoryForHandler) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, session := range r.sessions {
		if session.UserCode() == userCode {
			result = append(result, session)
		}
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) FindByChannelCode(ctx context.Context, channelCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, session := range r.sessions {
		if session.ChannelCode() == channelCode {
			result = append(result, session)
		}
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) FindAll(ctx context.Context) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, session := range r.sessions {
		result = append(result, session)
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) Update(ctx context.Context, session *domain.Session) error {
	r.sessions[session.ID()] = session
	r.sessionKeys[session.SessionKey()] = session
	return nil
}

func (r *mockSessionRepositoryForHandler) FindBySessionKey(ctx context.Context, sessionKey string) (*domain.Session, error) {
	if r.findByKeyErr != nil {
		return nil, r.findByKeyErr
	}
	return r.sessionKeys[sessionKey], nil
}

func (r *mockSessionRepositoryForHandler) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, session := range r.sessions {
		if session.UserCode() == userCode {
			result = append(result, session)
		}
	}
	return result, nil
}

func (r *mockSessionRepositoryForHandler) DeleteBySessionKey(ctx context.Context, sessionKey string) error {
	session, ok := r.sessionKeys[sessionKey]
	if ok {
		delete(r.sessions, session.ID())
		delete(r.sessionKeys, sessionKey)
	}
	return nil
}

func (r *mockSessionRepositoryForHandler) DeleteByChannelCode(ctx context.Context, channelCode string) error {
	for key, session := range r.sessionKeys {
		if session.ChannelCode() == channelCode {
			delete(r.sessions, session.ID())
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

	c, w := setupGinContext("POST", "/sessions", []byte("invalid json"))
	handler.CreateSession(c)

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
	c, w := setupGinContext("POST", "/sessions", []byte(body))
	handler.CreateSession(c)

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

	c, w := setupGinContext("GET", "/sessions", nil)
	handler.ListSessions(c)

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

	c, w := setupGinContext("GET", "/sessions?user_code=user-001", nil)
	handler.ListSessions(c)

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

	c, w := setupGinContext("GET", "/session", nil)
	handler.GetSession(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_GetSession_NotFound(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	c, w := setupGinContext("GET", "/session?session_key=nonexistent", nil)
	handler.GetSession(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestSessionHandler_DeleteSession_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	c, w := setupGinContext("DELETE", "/session", nil)
	handler.DeleteSession(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_TouchSession_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	c, w := setupGinContext("PUT", "/sessions//touch", nil)
	handler.TouchSession(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_UpdateSessionMetadata_NoKey(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	c, w := setupGinContext("PUT", "/sessions//metadata", []byte("{}"))
	handler.UpdateSessionMetadata(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestSessionHandler_UpdateSessionMetadata_InvalidJSON(t *testing.T) {
	repo := newMockSessionRepositoryForHandler()
	svc := application.NewSessionApplicationService(repo, newMockSessionIDGeneratorForHandler("sess"))
	handler := NewSessionHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/sessions/key-001/metadata", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "sessionKey", Value: "key-001"}}

	handler.UpdateSessionMetadata(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
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
