package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrUserTokenIDRequired    = errors.New("user token id is required")
	ErrUserTokenNameRequired  = errors.New("user token name is required")
	ErrUserTokenValueEmpty   = errors.New("user token value is empty")
	ErrUserTokenNotFound     = errors.New("user token not found")
)

// UserTokenID 用户Token ID
type UserTokenID struct {
	value string
}

func NewUserTokenID(value string) UserTokenID {
	return UserTokenID{value: value}
}

func (id UserTokenID) String() string {
	return id.value
}

// UserToken 长期API Token
type UserToken struct {
	id          UserTokenID
	userID      UserID
	name        string
	description string
	tokenHash   string // 存储哈希值，不存储原始token
	expiresAt   *time.Time // nil表示永久不过期
	createdAt   time.Time
	lastUsedAt  *time.Time
	isActive    bool
}

func NewUserToken(
	id UserTokenID,
	userID UserID,
	name string,
	description string,
	tokenHash string,
	expiresAt *time.Time,
) (*UserToken, error) {
	if id.String() == "" {
		return nil, ErrUserTokenIDRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrUserTokenNameRequired
	}
	if strings.TrimSpace(tokenHash) == "" {
		return nil, ErrUserTokenValueEmpty
	}

	now := time.Now()
	return &UserToken{
		id:          id,
		userID:      userID,
		name:        name,
		description: description,
		tokenHash:   tokenHash,
		expiresAt:   expiresAt,
		createdAt:   now,
		lastUsedAt:  nil,
		isActive:    true,
	}, nil
}

func (t *UserToken) ID() UserTokenID {
	return t.id
}

func (t *UserToken) UserID() UserID {
	return t.userID
}

func (t *UserToken) Name() string {
	return t.name
}

func (t *UserToken) Description() string {
	return t.description
}

func (t *UserToken) TokenHash() string {
	return t.tokenHash
}

func (t *UserToken) ExpiresAt() *time.Time {
	return t.expiresAt
}

func (t *UserToken) IsExpired() bool {
	if t.expiresAt == nil {
		return false // 永久token不会过期
	}
	return time.Now().After(*t.expiresAt)
}

func (t *UserToken) IsActive() bool {
	return t.isActive
}

func (t *UserToken) CreatedAt() time.Time {
	return t.createdAt
}

func (t *UserToken) LastUsedAt() *time.Time {
	return t.lastUsedAt
}

func (t *UserToken) Deactivate() {
	t.isActive = false
}

func (t *UserToken) UpdateLastUsed() {
	now := time.Now()
	t.lastUsedAt = &now
}

// UserTokenSnapshot 用于序列化
type UserTokenSnapshot struct {
	ID          UserTokenID `json:"id"`
	UserID      UserID      `json:"user_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ExpiresAt   *int64     `json:"expires_at,omitempty"` // Unix毫秒，nil表示永久
	CreatedAt   int64      `json:"created_at"`
	LastUsedAt  *int64     `json:"last_used_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	IsExpired   bool       `json:"is_expired"`
}

func (t *UserToken) ToSnapshot() UserTokenSnapshot {
	snap := UserTokenSnapshot{
		ID:          t.id,
		UserID:      t.userID,
		Name:        t.name,
		Description: t.description,
		CreatedAt:   t.createdAt.UnixMilli(),
		IsActive:    t.isActive,
		IsExpired:   t.IsExpired(),
	}
	if t.expiresAt != nil {
		expiresAtMs := t.expiresAt.UnixMilli()
		snap.ExpiresAt = &expiresAtMs
	}
	if t.lastUsedAt != nil {
		lastUsedAtMs := t.lastUsedAt.UnixMilli()
		snap.LastUsedAt = &lastUsedAtMs
	}
	return snap
}
