package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type ConversationRecordHandler struct {
	recordService *application.ConversationRecordApplicationService
}

func NewConversationRecordHandler(recordService *application.ConversationRecordApplicationService) *ConversationRecordHandler {
	return &ConversationRecordHandler{recordService: recordService}
}

type CreateConversationRecordRequest struct {
	TraceID          string `json:"trace_id"`
	SpanID           string `json:"span_id"`
	ParentSpanID     string `json:"parent_span_id"`
	EventType        string `json:"event_type"`
	Timestamp        *int64 `json:"timestamp"`
	SessionKey       string `json:"session_key"`
	Role             string `json:"role"`
	Content          string `json:"content"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	ReasoningTokens  int    `json:"reasoning_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	UserCode         string `json:"user_code"`
	AgentCode        string `json:"agent_code"`
	ChannelCode      string `json:"channel_code"`
	ChannelType      string `json:"channel_type"`
}

func (h *ConversationRecordHandler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	var req CreateConversationRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	var timestamp *time.Time
	if req.Timestamp != nil && *req.Timestamp > 0 {
		t := time.UnixMilli(*req.Timestamp)
		timestamp = &t
	}
	record, err := h.recordService.CreateRecord(r.Context(), application.CreateConversationRecordCommand{
		TraceID:          req.TraceID,
		SpanID:           req.SpanID,
		ParentSpanID:     req.ParentSpanID,
		EventType:        req.EventType,
		Timestamp:        timestamp,
		SessionKey:       req.SessionKey,
		Role:             req.Role,
		Content:          req.Content,
		PromptTokens:     req.PromptTokens,
		CompletionTokens: req.CompletionTokens,
		TotalTokens:      req.TotalTokens,
		ReasoningTokens:  req.ReasoningTokens,
		CachedTokens:     req.CachedTokens,
		UserCode:         req.UserCode,
		AgentCode:        req.AgentCode,
		ChannelCode:      req.ChannelCode,
		ChannelType:      req.ChannelType,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(conversationRecordToMap(record))
}

func (h *ConversationRecordHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	records, err := h.recordService.ListRecords(r.Context(), application.ListConversationRecordsQuery{
		TraceID:     r.URL.Query().Get("trace_id"),
		SessionKey:  r.URL.Query().Get("session_key"),
		UserCode:    r.URL.Query().Get("user_code"),
		AgentCode:   r.URL.Query().Get("agent_code"),
		ChannelCode: r.URL.Query().Get("channel_code"),
		EventType:   r.URL.Query().Get("event_type"),
		Role:        r.URL.Query().Get("role"),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		resp = append(resp, conversationRecordToMap(record))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ConversationRecordHandler) GetRecord(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	record, err := h.recordService.GetRecord(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(conversationRecordToMap(record))
}

func conversationRecordToMap(record *domain.ConversationRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":                record.ID().String(),
		"trace_id":          record.TraceID(),
		"span_id":           record.SpanID(),
		"parent_span_id":    record.ParentSpanID(),
		"event_type":        record.EventType(),
		"timestamp":         record.Timestamp().UnixMilli(),
		"session_key":       record.SessionKey(),
		"role":              record.Role(),
		"content":           record.Content(),
		"prompt_tokens":     record.PromptTokens(),
		"completion_tokens": record.CompletionTokens(),
		"total_tokens":      record.TotalTokens(),
		"reasoning_tokens":  record.ReasoningTokens(),
		"cached_tokens":     record.CachedTokens(),
		"user_code":         record.UserCode(),
		"agent_code":        record.AgentCode(),
		"channel_code":      record.ChannelCode(),
		"channel_type":      record.ChannelType(),
		"created_at":        record.CreatedAt().UnixMilli(),
	}
}
