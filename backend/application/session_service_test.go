/**
 * SessionApplicationService 单元测试
 */
package application

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// mockSessionRepository - 用于测试的 Session 仓库模拟
type mockSessionRepository struct {
	sessions     map[domain.SessionID]*domain.Session
	sessionKeys  map[string]*domain.Session
	saveErr      error
	findByKeyErr error
	findByIDErr  error
	deleteErr    error
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions:    make(map[domain.SessionID]*domain.Session),
		sessionKeys: make(map[string]*domain.Session),
	}
}

func (r *mockSessionRepository) Save(ctx context.Context, session *domain.Session) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.sessions[session.ID()] = session
	r.sessionKeys[session.SessionKey()] = session
	return nil
}

func (r *mockSessionRepository) FindByID(ctx context.Context, id domain.SessionID) (*domain.Session, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	return r.sessions[id], nil
}

func (r *mockSessionRepository) FindBySessionKey(ctx context.Context, key string) (*domain.Session, error) {
	if r.findByKeyErr != nil {
		return nil, r.findByKeyErr
	}
	return r.sessionKeys[key], nil
}

func (r *mockSessionRepository) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range r.sessions {
		if s.UserCode() == userCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSessionRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range r.sessions {
		if s.UserCode() == userCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSessionRepository) FindByChannelCode(ctx context.Context, channelCode string) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range r.sessions {
		if s.ChannelCode() == channelCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *mockSessionRepository) DeleteBySessionKey(ctx context.Context, sessionKey string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.sessionKeys, sessionKey)
	return nil
}

func (r *mockSessionRepository) DeleteByChannelCode(ctx context.Context, channelCode string) error {
	for key, s := range r.sessionKeys {
		if s.ChannelCode() == channelCode {
			delete(r.sessionKeys, key)
		}
	}
	return nil
}

// mockSessionIDGenerator - 用于测试的 ID 生成器模拟
type mockSessionIDGenerator struct {
	prefix string
	count  int
}

func newMockSessionIDGenerator(prefix string) *mockSessionIDGenerator {
	return &mockSessionIDGenerator{prefix: prefix}
}

func (g *mockSessionIDGenerator) Generate() string {
	g.count++
	return g.prefix + "-" + strconv.Itoa(g.count)
}

func TestSessionApplicationService_CreateSession(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	session, err := svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		AgentCode:   "agent-001",
		SessionKey:  "key-001",
		ExternalID:  "ext-001",
		Metadata:    map[string]interface{}{"k": "v"},
	})

	if err != nil {
		t.Fatalf("CreateSession 失败: %v", err)
	}

	if session.UserCode() != "user-001" {
		t.Errorf("期望 UserCode 为 user-001, 实际为 %s", session.UserCode())
	}

	if session.SessionKey() != "key-001" {
		t.Errorf("期望 SessionKey 为 key-001, 实际为 %s", session.SessionKey())
	}

	metadata := session.Metadata()
	if metadata["k"] != "v" {
		t.Errorf("期望 metadata k 为 v, 实际为 %v", metadata["k"])
	}
}

func TestSessionApplicationService_GetSessionByKey(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	// 先创建
	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})

	// 再获取
	session, err := svc.GetSessionByKey(context.Background(), "key-001")

	if err != nil {
		t.Fatalf("GetSessionByKey 失败: %v", err)
	}

	if session.SessionKey() != "key-001" {
		t.Errorf("期望 SessionKey 为 key-001, 实际为 %s", session.SessionKey())
	}
}

func TestSessionApplicationService_GetSessionByKey_NotFound(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	_, err := svc.GetSessionByKey(context.Background(), "nonexistent")

	if err != ErrSessionNotFound {
		t.Errorf("期望返回 ErrSessionNotFound, 实际返回 %v", err)
	}
}

func TestSessionApplicationService_GetSessionByID(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	// 先创建
	session, _ := svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})

	// 再获取
	found, err := svc.GetSessionByID(context.Background(), session.ID())

	if err != nil {
		t.Fatalf("GetSessionByID 失败: %v", err)
	}

	if found.ID() != session.ID() {
		t.Errorf("期望 ID 匹配")
	}
}

func TestSessionApplicationService_GetSessionByID_NotFound(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	_, err := svc.GetSessionByID(context.Background(), domain.NewSessionID("nonexistent"))

	if err != ErrSessionNotFound {
		t.Errorf("期望返回 ErrSessionNotFound, 实际返回 %v", err)
	}
}

func TestSessionApplicationService_ListUserSessions(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	// 创建多个 session
	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})
	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-002",
		SessionKey:  "key-002",
	})
	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-002",
		ChannelCode: "channel-001",
		SessionKey:  "key-003",
	})

	sessions, err := svc.ListUserSessions(context.Background(), "user-001")

	if err != nil {
		t.Fatalf("ListUserSessions 失败: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("期望 2 个 sessions, 实际为 %d", len(sessions))
	}
}

func TestSessionApplicationService_ListUserSessions_EmptyUserCode(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := &mockIDGenerator{}

	svc := NewSessionApplicationService(repo, idGen)

	_, err := svc.ListUserSessions(context.Background(), "")

	if err == nil {
		t.Error("期望错误")
	}
}

func TestSessionApplicationService_ListChannelSessions(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})
	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-002",
		ChannelCode: "channel-001",
		SessionKey:  "key-002",
	})

	sessions, err := svc.ListChannelSessions(context.Background(), "channel-001")

	if err != nil {
		t.Fatalf("ListChannelSessions 失败: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("期望 2 个 sessions, 实际为 %d", len(sessions))
	}
}

func TestSessionApplicationService_ListChannelSessions_EmptyChannelCode(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := &mockIDGenerator{}

	svc := NewSessionApplicationService(repo, idGen)

	_, err := svc.ListChannelSessions(context.Background(), "")

	if err == nil {
		t.Error("期望错误")
	}
}

func TestSessionApplicationService_DeleteSession(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})

	err := svc.DeleteSession(context.Background(), "key-001")

	if err != nil {
		t.Fatalf("DeleteSession 失败: %v", err)
	}

	// 验证已删除
	_, err = svc.GetSessionByKey(context.Background(), "key-001")
	if err != ErrSessionNotFound {
		t.Error("期望 session 已删除")
	}
}

func TestSessionApplicationService_DeleteSession_NotFound(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := &mockIDGenerator{}

	svc := NewSessionApplicationService(repo, idGen)

	err := svc.DeleteSession(context.Background(), "nonexistent")

	if err != ErrSessionNotFound {
		t.Errorf("期望返回 ErrSessionNotFound, 实际返回 %v", err)
	}
}

func TestSessionApplicationService_TouchSession(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
	})

	original, _ := svc.GetSessionByKey(context.Background(), "key-001")
	originalLastActive := original.LastActiveAt()

	time.Sleep(10 * time.Millisecond)

	err := svc.TouchSession(context.Background(), "key-001")
	if err != nil {
		t.Fatalf("TouchSession 失败: %v", err)
	}

	updated, _ := svc.GetSessionByKey(context.Background(), "key-001")
	if !updated.LastActiveAt().After(*originalLastActive) {
		t.Error("LastActiveAt 应该更新")
	}
}

func TestSessionApplicationService_UpdateSessionMetadata(t *testing.T) {
	repo := newMockSessionRepository()
	idGen := newMockSessionIDGenerator("sess")

	svc := NewSessionApplicationService(repo, idGen)

	svc.CreateSession(context.Background(), CreateSessionCommand{
		UserCode:    "user-001",
		ChannelCode: "channel-001",
		SessionKey:  "key-001",
		Metadata:    map[string]interface{}{"k1": "v1"},
	})

	err := svc.UpdateSessionMetadata(context.Background(), UpdateSessionMetadataCommand{
		SessionKey: "key-001",
		Metadata:   map[string]interface{}{"k2": "v2"},
	})

	if err != nil {
		t.Fatalf("UpdateSessionMetadata 失败: %v", err)
	}

	session, _ := svc.GetSessionByKey(context.Background(), "key-001")
	metadata := session.Metadata()

	if metadata["k2"] != "v2" {
		t.Errorf("期望 metadata k2 为 v2, 实际为 %v", metadata["k2"])
	}
}
