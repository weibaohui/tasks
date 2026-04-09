package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type UserHandler struct {
	userService *application.UserApplicationService
}

func NewUserHandler(userService *application.UserApplicationService) *UserHandler {
	return &UserHandler{userService: userService}
}

type CreateUserRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	PasswordHash string `json:"password_hash"`
	Password     string `json:"password"`
}

type UpdateUserRequest struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	IsActive    *bool   `json:"is_active"`
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), application.CreateUserCommand{
		Username:     req.Username,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: req.PasswordHash,
		Password:     req.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, userToMap(user))
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToMap(user))
	}
	c.JSON(http.StatusOK, items)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), domain.NewUserID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, userToMap(user))
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), application.UpdateUserCommand{
		ID:          domain.NewUserID(id),
		Email:       req.Email,
		DisplayName: req.DisplayName,
		IsActive:    req.IsActive,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, userToMap(user))
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.userService.DeleteUser(c.Request.Context(), domain.NewUserID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// handleGetUsers 根据 query 参数分发到 GetUser 或 ListUsers
func (h *UserHandler) handleGetUsers(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetUser(c)
		return
	}
	h.ListUsers(c)
}

func userToMap(user *domain.User) map[string]interface{} {
	return map[string]interface{}{
		"id":           user.ID().String(),
		"user_code":    user.UserCode().String(),
		"username":     user.Username(),
		"email":        user.Email(),
		"display_name": user.DisplayName(),
		"is_active":    user.IsActive(),
		"created_at":   user.CreatedAt().UnixMilli(),
		"updated_at":   user.UpdatedAt().UnixMilli(),
	}
}
