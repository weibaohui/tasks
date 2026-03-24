package channel

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Message 聊天消息
type Message struct {
	Role      string    `json:"role"`       // user, assistant, system
	Content   string    `json:"content"`    // 消息内容
	Timestamp time.Time `json:"timestamp"`  // 时间戳
	TraceID   string   `json:"trace_id"`   // 追踪ID
	SpanID    string   `json:"span_id"`    // 跨度ID
}

// Session 会话
type Session struct {
	key       string
	messages  []Message
	context   context.Context
	cancel    context.CancelFunc
	createdAt time.Time
	updatedAt time.Time
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewSession 创建新会话
func NewSession(key string, logger *zap.Logger) *Session {
	return &Session{
		key:       key,
		messages:  make([]Message, 0),
		createdAt: time.Now(),
		updatedAt: time.Now(),
		logger:    logger,
	}
}

// Key 返回会话键
func (s *Session) Key() string {
	return s.key
}

// Messages 返回消息列表
func (s *Session) Messages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Message(nil), s.messages...)
}

// AddMessage 添加消息
func (s *Session) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	s.messages = append(s.messages, msg)
	s.updatedAt = time.Now()
}

// SetContext 设置上下文
func (s *Session) SetContext(ctx context.Context, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.context = ctx
	s.cancel = cancel
}

// Context 获取上下文
func (s *Session) Context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context
}

// Cancel 取消会话
func (s *Session) Cancel() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// CreatedAt 返回创建时间
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt 返回更新时间
func (s *Session) UpdatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.updatedAt
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewSessionManager 创建会话管理器
func NewSessionManager(logger *zap.Logger) *SessionManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SessionManager{
		sessions: make(map[string]*Session),
		logger:    logger,
	}
}

// GetOrCreate 获取或创建会话
func (m *SessionManager) GetOrCreate(key string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[key]; exists {
		return session
	}

	session := NewSession(key, m.logger)
	m.sessions[key] = session
	m.logger.Info("创建新会话",
		zap.String("key", key),
		zap.Int("total", len(m.sessions)),
	)
	return session
}

// Get 获取会话
func (m *SessionManager) Get(key string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[key]
}

// Delete 删除会话
func (m *SessionManager) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session, exists := m.sessions[key]; exists {
		session.Cancel()
		delete(m.sessions, key)
		m.logger.Info("删除会话",
			zap.String("key", key),
			zap.Int("remaining", len(m.sessions)),
		)
	}
}

// List 返回所有会话键
func (m *SessionManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.sessions))
	for k := range m.sessions {
		keys = append(keys, k)
	}
	return keys
}

// Count 返回会话数量
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// CleanupInactive 清理不活跃的会话
func (m *SessionManager) CleanupInactive(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, session := range m.sessions {
		if now.Sub(session.updatedAt) > maxAge {
			session.Cancel()
			delete(m.sessions, key)
			removed++
		}
	}

	if removed > 0 {
		m.logger.Info("清理不活跃会话",
			zap.Int("removed", removed),
			zap.Int("remaining", len(m.sessions)),
		)
	}
	return removed
}
