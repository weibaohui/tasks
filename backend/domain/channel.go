package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrChannelIDRequired       = errors.New("channel id is required")
	ErrChannelCodeRequired     = errors.New("channel code is required")
	ErrChannelUserCodeRequired = errors.New("channel user code is required")
	ErrChannelNameRequired     = errors.New("channel name is required")
	ErrChannelTypeInvalid      = errors.New("invalid channel type")
)

type ChannelID struct {
	value string
}

func NewChannelID(value string) ChannelID {
	return ChannelID{value: value}
}

func (id ChannelID) String() string {
	return id.value
}

type ChannelCode struct {
	value string
}

func NewChannelCode(value string) ChannelCode {
	return ChannelCode{value: value}
}

func (c ChannelCode) String() string {
	return c.value
}

type ChannelType string

const (
	ChannelTypeFeishu    ChannelType = "feishu"
	ChannelTypeWebSocket ChannelType = "websocket"
)

func (t ChannelType) IsValid() bool {
	switch t {
	case ChannelTypeFeishu, ChannelTypeWebSocket:
		return true
	default:
		return false
	}
}

type Channel struct {
	id        ChannelID
	code      ChannelCode
	userCode  string
	agentCode string
	name      string
	typ       ChannelType
	isActive  bool
	allowFrom []string
	config    map[string]interface{}
	createdAt time.Time
	updatedAt time.Time
}

func NewChannel(
	id ChannelID,
	code ChannelCode,
	userCode string,
	name string,
	typ ChannelType,
) (*Channel, error) {
	if id.String() == "" {
		return nil, ErrChannelIDRequired
	}
	if code.String() == "" {
		return nil, ErrChannelCodeRequired
	}
	if strings.TrimSpace(userCode) == "" {
		return nil, ErrChannelUserCodeRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrChannelNameRequired
	}
	if !typ.IsValid() {
		return nil, ErrChannelTypeInvalid
	}

	now := time.Now()
	return &Channel{
		id:        id,
		code:      code,
		userCode:  userCode,
		name:      name,
		typ:       typ,
		isActive:  true,
		allowFrom: []string{},
		config:    map[string]interface{}{},
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (c *Channel) ID() ChannelID            { return c.id }
func (c *Channel) ChannelCode() ChannelCode { return c.code }
func (c *Channel) UserCode() string         { return c.userCode }
func (c *Channel) AgentCode() string        { return c.agentCode }
func (c *Channel) Name() string             { return c.name }
func (c *Channel) Type() ChannelType        { return c.typ }
func (c *Channel) IsActive() bool           { return c.isActive }
func (c *Channel) AllowFrom() []string      { return append([]string(nil), c.allowFrom...) }
func (c *Channel) Config() map[string]interface{} {
	m := cloneMap(c.config)
	if c.agentCode != "" {
		m["agent_code"] = c.agentCode
	}
	if c.code.String() != "" {
		m["channel_code"] = c.code.String()
	}
	return m
}
func (c *Channel) CreatedAt() time.Time { return c.createdAt }
func (c *Channel) UpdatedAt() time.Time { return c.updatedAt }

func (c *Channel) UpdateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrChannelNameRequired
	}
	c.name = name
	c.updatedAt = time.Now()
	return nil
}

func (c *Channel) UpdateConfig(config map[string]interface{}) {
	c.config = cloneMap(config)
	c.updatedAt = time.Now()
}

func (c *Channel) SetAllowFrom(allowFrom []string) {
	c.allowFrom = append([]string(nil), allowFrom...)
	c.updatedAt = time.Now()
}

func (c *Channel) BindAgent(agentCode string) {
	c.agentCode = agentCode
	c.updatedAt = time.Now()
}

func (c *Channel) SetActive(isActive bool) {
	c.isActive = isActive
	c.updatedAt = time.Now()
}

type ChannelSnapshot struct {
	ID        ChannelID
	Code      ChannelCode
	UserCode  string
	AgentCode string
	Name      string
	Type      ChannelType
	IsActive  bool
	AllowFrom []string
	Config    map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Channel) ToSnapshot() ChannelSnapshot {
	return ChannelSnapshot{
		ID:        c.id,
		Code:      c.code,
		UserCode:  c.userCode,
		AgentCode: c.agentCode,
		Name:      c.name,
		Type:      c.typ,
		IsActive:  c.isActive,
		AllowFrom: append([]string(nil), c.allowFrom...),
		Config:    cloneMap(c.config),
		CreatedAt: c.createdAt,
		UpdatedAt: c.updatedAt,
	}
}

func (c *Channel) FromSnapshot(snap ChannelSnapshot) {
	c.id = snap.ID
	c.code = snap.Code
	c.userCode = snap.UserCode
	c.agentCode = snap.AgentCode
	c.name = snap.Name
	c.typ = snap.Type
	c.isActive = snap.IsActive
	c.allowFrom = append([]string(nil), snap.AllowFrom...)
	c.config = cloneMap(snap.Config)
	c.createdAt = snap.CreatedAt
	c.updatedAt = snap.UpdatedAt
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
