package http

import (
	"encoding/json"
	"net/http"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type ChannelHandler struct {
	channelService *application.ChannelApplicationService
}

func NewChannelHandler(channelService *application.ChannelApplicationService) *ChannelHandler {
	return &ChannelHandler{channelService: channelService}
}

type CreateChannelRequest struct {
	UserCode  string                 `json:"user_code"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	AllowFrom []string               `json:"allow_from"`
	AgentCode string                 `json:"agent_code"`
}

type UpdateChannelRequest struct {
	Name      *string                 `json:"name"`
	Config    *map[string]interface{} `json:"config"`
	AllowFrom *[]string               `json:"allow_from"`
	IsActive  *bool                   `json:"is_active"`
	AgentCode *string                 `json:"agent_code"`
}

func (h *ChannelHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	channel, err := h.channelService.CreateChannel(r.Context(), application.CreateChannelCommand{
		UserCode:  req.UserCode,
		Name:      req.Name,
		Type:      domain.ChannelType(req.Type),
		Config:    req.Config,
		AllowFrom: req.AllowFrom,
		AgentCode: req.AgentCode,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(channelToMap(channel))
}

func (h *ChannelHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	userCode := r.URL.Query().Get("user_code")
	channels, err := h.channelService.ListChannels(r.Context(), userCode)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(channels))
	for _, channel := range channels {
		resp = append(resp, channelToMap(channel))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ChannelHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	code := r.URL.Query().Get("code")
	if id == "" && code == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id or code is required"})
		return
	}

	var (
		channel *domain.Channel
		err     error
	)
	if id != "" {
		channel, err = h.channelService.GetChannel(r.Context(), domain.NewChannelID(id))
	} else {
		channel, err = h.channelService.GetChannelByCode(r.Context(), domain.NewChannelCode(code))
	}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(channelToMap(channel))
}

func (h *ChannelHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	channel, err := h.channelService.UpdateChannel(r.Context(), application.UpdateChannelCommand{
		ID:        domain.NewChannelID(id),
		Name:      req.Name,
		Config:    req.Config,
		AllowFrom: req.AllowFrom,
		IsActive:  req.IsActive,
		AgentCode: req.AgentCode,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(channelToMap(channel))
}

func (h *ChannelHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.channelService.DeleteChannel(r.Context(), domain.NewChannelID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func channelToMap(channel *domain.Channel) map[string]interface{} {
	return map[string]interface{}{
		"id":           channel.ID().String(),
		"channel_code": channel.ChannelCode().String(),
		"user_code":    channel.UserCode(),
		"agent_code":   channel.AgentCode(),
		"name":         channel.Name(),
		"type":         string(channel.Type()),
		"is_active":    channel.IsActive(),
		"allow_from":   channel.AllowFrom(),
		"config":       channel.Config(),
		"created_at":   channel.CreatedAt().UnixMilli(),
		"updated_at":   channel.UpdatedAt().UnixMilli(),
	}
}
