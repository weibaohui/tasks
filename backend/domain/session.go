package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrSessionIDRequired      = errors.New("session id is required")
	ErrSessionUserCodeMissing = errors.New("session user code is required")
	ErrSessionChannelMissing  = errors.New("session channel code is required")
	ErrSessionKeyRequired     = errors.New("session key is required")
)

type SessionID struct {
	value string
}

func NewSessionID(value string) SessionID {
	return SessionID{value: value}
}

func (id SessionID) String() string {
	return id.value
}

type Session struct {
	id          SessionID
	userCode    string
	agentCode   string
	channelCode string
	sessionKey  string
	externalID  string
	lastActive  *time.Time
	metadata    map[string]interface{}
	createdAt   time.Time
	updatedAt   time.Time
}

func NewSession(
	id SessionID,
	userCode string,
	channelCode string,
	sessionKey string,
	externalID string,
	agentCode string,
) (*Session, error) {
	if id.String() == "" {
		return nil, ErrSessionIDRequired
	}
	if strings.TrimSpace(userCode) == "" {
		return nil, ErrSessionUserCodeMissing
	}
	if strings.TrimSpace(channelCode) == "" {
		return nil, ErrSessionChannelMissing
	}
	if strings.TrimSpace(sessionKey) == "" {
		return nil, ErrSessionKeyRequired
	}

	now := time.Now()
	lastActive := now
	return &Session{
		id:          id,
		userCode:    userCode,
		agentCode:   agentCode,
		channelCode: channelCode,
		sessionKey:  sessionKey,
		externalID:  externalID,
		lastActive:  &lastActive,
		metadata:    map[string]interface{}{},
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func (s *Session) ID() SessionID                    { return s.id }
func (s *Session) UserCode() string                 { return s.userCode }
func (s *Session) AgentCode() string                { return s.agentCode }
func (s *Session) ChannelCode() string              { return s.channelCode }
func (s *Session) SessionKey() string               { return s.sessionKey }
func (s *Session) ExternalID() string               { return s.externalID }
func (s *Session) LastActiveAt() *time.Time         { return cloneTimePtr(s.lastActive) }
func (s *Session) Metadata() map[string]interface{} { return cloneMap(s.metadata) }
func (s *Session) CreatedAt() time.Time             { return s.createdAt }
func (s *Session) UpdatedAt() time.Time             { return s.updatedAt }

func (s *Session) Touch() {
	now := time.Now()
	s.lastActive = &now
	s.updatedAt = now
}

func (s *Session) SetMetadata(metadata map[string]interface{}) {
	s.metadata = cloneMap(metadata)
	s.updatedAt = time.Now()
}

func (s *Session) SetAgentCode(agentCode string) {
	s.agentCode = agentCode
	s.updatedAt = time.Now()
}

type SessionSnapshot struct {
	ID          SessionID
	UserCode    string
	AgentCode   string
	ChannelCode string
	SessionKey  string
	ExternalID  string
	LastActive  *time.Time
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Session) ToSnapshot() SessionSnapshot {
	return SessionSnapshot{
		ID:          s.id,
		UserCode:    s.userCode,
		AgentCode:   s.agentCode,
		ChannelCode: s.channelCode,
		SessionKey:  s.sessionKey,
		ExternalID:  s.externalID,
		LastActive:  cloneTimePtr(s.lastActive),
		Metadata:    cloneMap(s.metadata),
		CreatedAt:   s.createdAt,
		UpdatedAt:   s.updatedAt,
	}
}

func (s *Session) FromSnapshot(snap SessionSnapshot) {
	s.id = snap.ID
	s.userCode = snap.UserCode
	s.agentCode = snap.AgentCode
	s.channelCode = snap.ChannelCode
	s.sessionKey = snap.SessionKey
	s.externalID = snap.ExternalID
	s.lastActive = cloneTimePtr(snap.LastActive)
	s.metadata = cloneMap(snap.Metadata)
	s.createdAt = snap.CreatedAt
	s.updatedAt = snap.UpdatedAt
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	c := *t
	return &c
}
