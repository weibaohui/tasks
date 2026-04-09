package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type SessionHandler struct {
	sessionService *application.SessionApplicationService
}

func NewSessionHandler(sessionService *application.SessionApplicationService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

type CreateSessionRequest struct {
	UserCode    string                 `json:"user_code"`
	ChannelCode string                 `json:"channel_code"`
	AgentCode   string                 `json:"agent_code"`
	SessionKey  string                 `json:"session_key"`
	ExternalID  string                 `json:"external_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	session, err := h.sessionService.CreateSession(c.Request.Context(), application.CreateSessionCommand{
		UserCode:    req.UserCode,
		ChannelCode: req.ChannelCode,
		AgentCode:   req.AgentCode,
		SessionKey:  req.SessionKey,
		ExternalID:  req.ExternalID,
		Metadata:    req.Metadata,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sessionToMap(session))
}

func (h *SessionHandler) ListSessions(c *gin.Context) {
	userCode := c.Query("user_code")
	channelCode := c.Query("channel_code")

	var (
		sessions []*domain.Session
		err      error
	)
	if userCode != "" {
		sessions, err = h.sessionService.ListUserSessions(c.Request.Context(), userCode)
	} else if channelCode != "" {
		sessions, err = h.sessionService.ListChannelSessions(c.Request.Context(), channelCode)
	} else {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "user_code or channel_code is required"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(sessions))
	for _, session := range sessions {
		resp = append(resp, sessionToMap(session))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *SessionHandler) GetSession(c *gin.Context) {
	// 优先从 query 参数获取
	sessionKey := c.Query("session_key")
	// 其次从路径参数获取
	if sessionKey == "" {
		sessionKey = c.Param("sessionKey")
	}
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	session, err := h.sessionService.GetSessionByKey(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessionToMap(session))
}

func (h *SessionHandler) DeleteSession(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	if err := h.sessionService.DeleteSession(c.Request.Context(), sessionKey); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// DeleteSessionByPath 通过路径参数删除 session
func (h *SessionHandler) DeleteSessionByPath(c *gin.Context) {
	sessionKey := c.Param("sessionKey")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	if err := h.sessionService.DeleteSession(c.Request.Context(), sessionKey); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func (h *SessionHandler) TouchSession(c *gin.Context) {
	sessionKey := c.Param("sessionKey")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	if err := h.sessionService.TouchSession(c.Request.Context(), sessionKey); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func (h *SessionHandler) GetSessionMetadata(c *gin.Context) {
	sessionKey := c.Param("sessionKey")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	metadata, err := h.sessionService.GetSessionMetadata(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, metadata)
}

func (h *SessionHandler) UpdateSessionMetadata(c *gin.Context) {
	sessionKey := c.Param("sessionKey")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	metadata := map[string]interface{}{}
	if err := c.ShouldBindJSON(&metadata); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	if err := h.sessionService.UpdateSessionMetadata(c.Request.Context(), application.UpdateSessionMetadataCommand{
		SessionKey: sessionKey,
		Metadata:   metadata,
	}); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// handleGetSessions 根据 query 参数分发到 GetSession 或 ListSessions
func (h *SessionHandler) handleGetSessions(c *gin.Context) {
	if c.Query("session_key") != "" {
		h.GetSession(c)
		return
	}
	h.ListSessions(c)
}

func sessionToMap(session *domain.Session) map[string]interface{} {
	lastActive := interface{}(nil)
	if session.LastActiveAt() != nil {
		lastActive = session.LastActiveAt().UnixMilli()
	}
	return map[string]interface{}{
		"id":             session.ID().String(),
		"user_code":      session.UserCode(),
		"agent_code":     session.AgentCode(),
		"channel_code":   session.ChannelCode(),
		"session_key":    session.SessionKey(),
		"external_id":    session.ExternalID(),
		"last_active_at": lastActive,
		"metadata":       session.Metadata(),
		"created_at":     session.CreatedAt().UnixMilli(),
		"updated_at":     session.UpdatedAt().UnixMilli(),
	}
}
