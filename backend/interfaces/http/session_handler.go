package http

import (
	"encoding/json"
	"net/http"
	"strings"

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

func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	session, err := h.sessionService.CreateSession(r.Context(), application.CreateSessionCommand{
		UserCode:    req.UserCode,
		ChannelCode: req.ChannelCode,
		AgentCode:   req.AgentCode,
		SessionKey:  req.SessionKey,
		ExternalID:  req.ExternalID,
		Metadata:    req.Metadata,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sessionToMap(session))
}

func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userCode := r.URL.Query().Get("user_code")
	channelCode := r.URL.Query().Get("channel_code")

	var (
		sessions []*domain.Session
		err      error
	)
	if userCode != "" {
		sessions, err = h.sessionService.ListUserSessions(r.Context(), userCode)
	} else if channelCode != "" {
		sessions, err = h.sessionService.ListChannelSessions(r.Context(), channelCode)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "user_code or channel_code is required"})
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(sessions))
	for _, session := range sessions {
		resp = append(resp, sessionToMap(session))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionKey := r.URL.Query().Get("session_key")
	if sessionKey == "" {
		sessionKey = extractSessionKey(r.URL.Path)
	}
	if sessionKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	session, err := h.sessionService.GetSessionByKey(r.Context(), sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(sessionToMap(session))
}

func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionKey := r.URL.Query().Get("session_key")
	if sessionKey == "" {
		sessionKey = extractSessionKey(r.URL.Path)
	}
	if sessionKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	if err := h.sessionService.DeleteSession(r.Context(), sessionKey); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (h *SessionHandler) TouchSession(w http.ResponseWriter, r *http.Request) {
	sessionKey := extractSessionKey(r.URL.Path)
	if sessionKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	if err := h.sessionService.TouchSession(r.Context(), sessionKey); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (h *SessionHandler) GetSessionMetadata(w http.ResponseWriter, r *http.Request) {
	sessionKey := extractSessionKey(r.URL.Path)
	if sessionKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	metadata, err := h.sessionService.GetSessionMetadata(r.Context(), sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(metadata)
}

func (h *SessionHandler) UpdateSessionMetadata(w http.ResponseWriter, r *http.Request) {
	sessionKey := extractSessionKey(r.URL.Path)
	if sessionKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "session_key is required"})
		return
	}
	metadata := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	if err := h.sessionService.UpdateSessionMetadata(r.Context(), application.UpdateSessionMetadataCommand{
		SessionKey: sessionKey,
		Metadata:   metadata,
	}); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
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

func extractSessionKey(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "sessions" && i+1 < len(parts) {
			key := parts[i+1]
			if key != "" {
				return key
			}
		}
	}
	return ""
}
