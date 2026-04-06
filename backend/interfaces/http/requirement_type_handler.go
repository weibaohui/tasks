package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type RequirementTypeHandler struct {
	requirementTypeRepo domain.RequirementTypeEntityRepository
}

func NewRequirementTypeHandler(requirementTypeRepo domain.RequirementTypeEntityRepository) *RequirementTypeHandler {
	return &RequirementTypeHandler{
		requirementTypeRepo: requirementTypeRepo,
	}
}

type CreateRequirementTypeRequest struct {
	ProjectID   string `json:"project_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
}

func (h *RequirementTypeHandler) CreateRequirementType(w http.ResponseWriter, r *http.Request) {
	var req CreateRequirementTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	if req.ProjectID == "" || req.Code == "" || req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "project_id, code and name are required"})
		return
	}

	rt, err := domain.NewRequirementTypeEntity(
		domain.NewRequirementTypeEntityID(generateID()),
		domain.NewProjectID(req.ProjectID),
		req.Code,
		req.Name,
		req.Description,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	if req.Icon != "" {
		rt.SetIcon(req.Icon)
	}
	if req.Color != "" {
		rt.SetColor(req.Color)
	}

	if err := h.requirementTypeRepo.Save(context.Background(), rt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(h.requirementTypeToMap(rt))
}

func (h *RequirementTypeHandler) ListRequirementTypes(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}

	types, err := h.requirementTypeRepo.FindByProjectID(context.Background(), domain.NewProjectID(projectIDStr))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	resp := make([]map[string]interface{}, 0, len(types))
	for _, rt := range types {
		resp = append(resp, h.requirementTypeToMap(rt))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *RequirementTypeHandler) requirementTypeToMap(rt *domain.RequirementTypeEntity) map[string]interface{} {
	return map[string]interface{}{
		"id":              rt.ID().String(),
		"project_id":      rt.ProjectID().String(),
		"code":            rt.Code(),
		"name":            rt.Name(),
		"description":     rt.Description(),
		"icon":            rt.Icon(),
		"color":           rt.Color(),
		"sort_order":      rt.SortOrder(),
		"state_machine_id": rt.StateMachineID(),
		"created_at":      rt.CreatedAt().UnixMilli(),
		"updated_at":      rt.UpdatedAt().UnixMilli(),
	}
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}