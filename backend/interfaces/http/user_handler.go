package http

import (
	"encoding/json"
	"net/http"

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
}

type UpdateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	IsActive    *bool  `json:"is_active"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	user, err := h.userService.CreateUser(r.Context(), application.CreateUserCommand{
		Username:     req.Username,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: req.PasswordHash,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(userToMap(user))
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.ListUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToMap(user))
	}
	_ = json.NewEncoder(w).Encode(items)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	user, err := h.userService.GetUser(r.Context(), domain.NewUserID(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(userToMap(user))
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	user, err := h.userService.UpdateUser(r.Context(), application.UpdateUserCommand{
		ID:          domain.NewUserID(id),
		Email:       req.Email,
		DisplayName: req.DisplayName,
		IsActive:    req.IsActive,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(userToMap(user))
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.userService.DeleteUser(r.Context(), domain.NewUserID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
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
