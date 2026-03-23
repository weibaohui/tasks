package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrConversationRecordIDRequired        = errors.New("conversation record id is required")
	ErrConversationRecordTraceIDRequired   = errors.New("conversation record trace_id is required")
	ErrConversationRecordEventTypeRequired = errors.New("conversation record event_type is required")
)

type ConversationRecordID struct {
	value string
}

func NewConversationRecordID(value string) ConversationRecordID {
	return ConversationRecordID{value: value}
}

func (id ConversationRecordID) String() string {
	return id.value
}

type ConversationRecord struct {
	id               ConversationRecordID
	traceID          string
	spanID           string
	parentSpanID     string
	eventType        string
	timestamp        time.Time
	sessionKey       string
	role             string
	content          string
	promptTokens     int
	completionTokens int
	totalTokens      int
	reasoningTokens  int
	cachedTokens     int
	userCode         string
	agentCode        string
	channelCode      string
	channelType      string
	createdAt        time.Time
}

func NewConversationRecord(
	id ConversationRecordID,
	traceID string,
	eventType string,
) (*ConversationRecord, error) {
	if id.String() == "" {
		return nil, ErrConversationRecordIDRequired
	}
	if strings.TrimSpace(traceID) == "" {
		return nil, ErrConversationRecordTraceIDRequired
	}
	if strings.TrimSpace(eventType) == "" {
		return nil, ErrConversationRecordEventTypeRequired
	}

	now := time.Now()
	return &ConversationRecord{
		id:        id,
		traceID:   traceID,
		eventType: eventType,
		timestamp: now,
		createdAt: now,
	}, nil
}

func (r *ConversationRecord) ID() ConversationRecordID { return r.id }
func (r *ConversationRecord) TraceID() string          { return r.traceID }
func (r *ConversationRecord) SpanID() string           { return r.spanID }
func (r *ConversationRecord) ParentSpanID() string     { return r.parentSpanID }
func (r *ConversationRecord) EventType() string        { return r.eventType }
func (r *ConversationRecord) Timestamp() time.Time     { return r.timestamp }
func (r *ConversationRecord) SessionKey() string       { return r.sessionKey }
func (r *ConversationRecord) Role() string             { return r.role }
func (r *ConversationRecord) Content() string          { return r.content }
func (r *ConversationRecord) PromptTokens() int        { return r.promptTokens }
func (r *ConversationRecord) CompletionTokens() int    { return r.completionTokens }
func (r *ConversationRecord) TotalTokens() int         { return r.totalTokens }
func (r *ConversationRecord) ReasoningTokens() int     { return r.reasoningTokens }
func (r *ConversationRecord) CachedTokens() int        { return r.cachedTokens }
func (r *ConversationRecord) UserCode() string         { return r.userCode }
func (r *ConversationRecord) AgentCode() string        { return r.agentCode }
func (r *ConversationRecord) ChannelCode() string      { return r.channelCode }
func (r *ConversationRecord) ChannelType() string      { return r.channelType }
func (r *ConversationRecord) CreatedAt() time.Time     { return r.createdAt }

func (r *ConversationRecord) SetSpan(spanID, parentSpanID string) {
	r.spanID = spanID
	r.parentSpanID = parentSpanID
}

func (r *ConversationRecord) SetMessage(role, content string) {
	r.role = role
	r.content = content
}

func (r *ConversationRecord) SetScope(sessionKey, userCode, agentCode, channelCode, channelType string) {
	r.sessionKey = sessionKey
	r.userCode = userCode
	r.agentCode = agentCode
	r.channelCode = channelCode
	r.channelType = channelType
}

func (r *ConversationRecord) SetTokenUsage(prompt, completion, total, reasoning, cached int) {
	r.promptTokens = prompt
	r.completionTokens = completion
	r.totalTokens = total
	r.reasoningTokens = reasoning
	r.cachedTokens = cached
}

func (r *ConversationRecord) SetTimestamp(timestamp time.Time) {
	if timestamp.IsZero() {
		return
	}
	r.timestamp = timestamp
}

type ConversationRecordSnapshot struct {
	ID               ConversationRecordID
	TraceID          string
	SpanID           string
	ParentSpanID     string
	EventType        string
	Timestamp        time.Time
	SessionKey       string
	Role             string
	Content          string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	ReasoningTokens  int
	CachedTokens     int
	UserCode         string
	AgentCode        string
	ChannelCode      string
	ChannelType      string
	CreatedAt        time.Time
}

func (r *ConversationRecord) ToSnapshot() ConversationRecordSnapshot {
	return ConversationRecordSnapshot{
		ID:               r.id,
		TraceID:          r.traceID,
		SpanID:           r.spanID,
		ParentSpanID:     r.parentSpanID,
		EventType:        r.eventType,
		Timestamp:        r.timestamp,
		SessionKey:       r.sessionKey,
		Role:             r.role,
		Content:          r.content,
		PromptTokens:     r.promptTokens,
		CompletionTokens: r.completionTokens,
		TotalTokens:      r.totalTokens,
		ReasoningTokens:  r.reasoningTokens,
		CachedTokens:     r.cachedTokens,
		UserCode:         r.userCode,
		AgentCode:        r.agentCode,
		ChannelCode:      r.channelCode,
		ChannelType:      r.channelType,
		CreatedAt:        r.createdAt,
	}
}

func (r *ConversationRecord) FromSnapshot(snap ConversationRecordSnapshot) {
	r.id = snap.ID
	r.traceID = snap.TraceID
	r.spanID = snap.SpanID
	r.parentSpanID = snap.ParentSpanID
	r.eventType = snap.EventType
	r.timestamp = snap.Timestamp
	r.sessionKey = snap.SessionKey
	r.role = snap.Role
	r.content = snap.Content
	r.promptTokens = snap.PromptTokens
	r.completionTokens = snap.CompletionTokens
	r.totalTokens = snap.TotalTokens
	r.reasoningTokens = snap.ReasoningTokens
	r.cachedTokens = snap.CachedTokens
	r.userCode = snap.UserCode
	r.agentCode = snap.AgentCode
	r.channelCode = snap.ChannelCode
	r.channelType = snap.ChannelType
	r.createdAt = snap.CreatedAt
}
