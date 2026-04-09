package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

func (h *ConversationRecordHandler) CreateRecord(c *gin.Context) {
	var req CreateConversationRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	var timestamp *time.Time
	if req.Timestamp != nil && *req.Timestamp > 0 {
		t := time.UnixMilli(*req.Timestamp)
		timestamp = &t
	}
	record, err := h.recordService.CreateRecord(c.Request.Context(), application.CreateConversationRecordCommand{
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
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, conversationRecordToMap(record))
}

func (h *ConversationRecordHandler) ListRecords(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	query := application.ListConversationRecordsQuery{
		TraceID:     c.Query("trace_id"),
		SessionKey:  c.Query("session_key"),
		UserCode:    c.Query("user_code"),
		AgentCode:   c.Query("agent_code"),
		ChannelCode: c.Query("channel_code"),
		EventType:   c.Query("event_type"),
		Role:        c.Query("role"),
		Limit:       limit,
		Offset:      offset,
	}
	records, err := h.recordService.ListRecords(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	total, err := h.recordService.CountRecords(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		resp = append(resp, conversationRecordToMap(record))
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"items": resp,
		"total": total,
	})
}

func (h *ConversationRecordHandler) GetRecord(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	record, err := h.recordService.GetRecord(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, conversationRecordToMap(record))
}

func (h *ConversationRecordHandler) GetRecordsBySession(c *gin.Context) {
	sessionKey := c.Param("sessionKey")
	records, err := h.recordService.ListRecords(c.Request.Context(), application.ListConversationRecordsQuery{
		SessionKey: sessionKey,
		Limit:      500,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		resp = append(resp, conversationRecordToMap(record))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ConversationRecordHandler) GetRecordsByTrace(c *gin.Context) {
	traceId := c.Param("traceId")
	records, err := h.recordService.ListRecords(c.Request.Context(), application.ListConversationRecordsQuery{
		TraceID: traceId,
		Limit:   500,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		resp = append(resp, conversationRecordToMap(record))
	}
	c.JSON(http.StatusOK, resp)
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

func (h *ConversationRecordHandler) GetStats(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	agentCodes := c.QueryArray("agent_codes")
	channelCodes := c.QueryArray("channel_codes")
	roles := c.QueryArray("roles")

	var startTime, endTime *time.Time
	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err == nil {
			startTime = &t
		}
	}
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err == nil {
			endTime = &t
		}
	}

	stats, err := h.recordService.GetStats(c.Request.Context(), application.GetConversationStatsQuery{
		StartTime:    startTime,
		EndTime:      endTime,
		AgentCodes:   agentCodes,
		ChannelCodes: channelCodes,
		Roles:        roles,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 转换 daily_trends 字段名：completion_tokens -> complete_tokens
	dailyTrends := make([]map[string]interface{}, 0, len(stats.DailyTrends))
	for _, dt := range stats.DailyTrends {
		dailyTrends = append(dailyTrends, map[string]interface{}{
			"date":            dt.Date,
			"prompt_tokens":   dt.PromptTokens,
			"complete_tokens": dt.CompletionTokens,
			"total_tokens":    dt.TotalTokens,
		})
	}

	tokenStats := map[string]interface{}{
		"total_prompt_tokens":     stats.TotalPromptTokens,
		"total_completion_tokens": stats.TotalCompletionTokens,
		"total_tokens":            stats.TotalTokens,
		"daily_trends":            dailyTrends,
	}

	agentDist := make([]map[string]interface{}, 0, len(stats.AgentDistribution))
	for _, a := range stats.AgentDistribution {
		agentDist = append(agentDist, map[string]interface{}{
			"code":   a.Code,
			"name":   a.Name,
			"count":  a.Count,
			"tokens": a.Tokens,
		})
	}

	channelDist := make([]map[string]interface{}, 0, len(stats.ChannelDistribution))
	for _, c2 := range stats.ChannelDistribution {
		channelDist = append(channelDist, map[string]interface{}{
			"type":  c2.Type,
			"count": c2.Count,
		})
	}

	roleDist := make([]map[string]interface{}, 0, len(stats.RoleDistribution))
	for _, r := range stats.RoleDistribution {
		roleDist = append(roleDist, map[string]interface{}{
			"role":  r.Role,
			"count": r.Count,
		})
	}

	sessionStats := map[string]interface{}{
		"total_sessions": stats.TotalSessions,
	}

	projectDist := make([]map[string]interface{}, 0, len(stats.ProjectDistribution))
	for _, p := range stats.ProjectDistribution {
		projectDist = append(projectDist, map[string]interface{}{
			"project_id": p.ProjectID,
			"name":       p.Name,
			"tokens":     p.Tokens,
		})
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"token_stats":          tokenStats,
		"agent_distribution":   agentDist,
		"channel_distribution": channelDist,
		"role_distribution":    roleDist,
		"project_distribution": projectDist,
		"session_stats":        sessionStats,
	})
}

// handleGetRecords 根据 query 参数分发到 GetRecord 或 ListRecords
func (h *ConversationRecordHandler) handleGetRecords(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetRecord(c)
		return
	}
	h.ListRecords(c)
}
