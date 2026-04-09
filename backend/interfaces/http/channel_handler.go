package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type ChannelHandler struct {
	channelService *application.ChannelApplicationService
}

func NewChannelHandler(channelService *application.ChannelApplicationService) *ChannelHandler {
	return &ChannelHandler{channelService: channelService}
}

type ChannelTypeOption struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (h *ChannelHandler) ListChannelTypes(c *gin.Context) {
	resp := []ChannelTypeOption{
		{Key: string(domain.ChannelTypeFeishu), Name: "飞书"},
		{Key: string(domain.ChannelTypeWebSocket), Name: "WebSocket"},
	}
	c.JSON(http.StatusOK, resp)
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

func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	channel, err := h.channelService.CreateChannel(c.Request.Context(), application.CreateChannelCommand{
		UserCode:  req.UserCode,
		Name:      req.Name,
		Type:      domain.ChannelType(req.Type),
		Config:    req.Config,
		AllowFrom: req.AllowFrom,
		AgentCode: req.AgentCode,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, channelToMap(channel))
}

func (h *ChannelHandler) ListChannels(c *gin.Context) {
	userCode := c.Query("user_code")
	channels, err := h.channelService.ListChannels(c.Request.Context(), userCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(channels))
	for _, channel := range channels {
		resp = append(resp, channelToMap(channel))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ChannelHandler) GetChannel(c *gin.Context) {
	id := c.Query("id")
	code := c.Query("code")
	if id == "" && code == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id or code is required"})
		return
	}

	var (
		channel *domain.Channel
		err     error
	)
	if id != "" {
		channel, err = h.channelService.GetChannel(c.Request.Context(), domain.NewChannelID(id))
	} else {
		channel, err = h.channelService.GetChannelByCode(c.Request.Context(), domain.NewChannelCode(code))
	}
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, channelToMap(channel))
}

func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	channel, err := h.channelService.UpdateChannel(c.Request.Context(), application.UpdateChannelCommand{
		ID:        domain.NewChannelID(id),
		Name:      req.Name,
		Config:    req.Config,
		AllowFrom: req.AllowFrom,
		IsActive:  req.IsActive,
		AgentCode: req.AgentCode,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, channelToMap(channel))
}

func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.channelService.DeleteChannel(c.Request.Context(), domain.NewChannelID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// handleGetChannels 根据 query 参数分发到 GetChannel 或 ListChannels
func (h *ChannelHandler) handleGetChannels(c *gin.Context) {
	if c.Query("id") != "" || c.Query("code") != "" {
		h.GetChannel(c)
		return
	}
	h.ListChannels(c)
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
