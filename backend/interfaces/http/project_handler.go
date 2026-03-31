package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type ProjectHandler struct {
	projectService      *application.ProjectApplicationService
	heartbeatScheduler  *application.HeartbeatScheduler
}

func NewProjectHandler(projectService *application.ProjectApplicationService, heartbeatScheduler *application.HeartbeatScheduler) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, heartbeatScheduler: heartbeatScheduler}
}

type CreateProjectRequest struct {
	Name          string   `json:"name"`
	GitRepoURL    string   `json:"git_repo_url"`
	DefaultBranch string   `json:"default_branch"`
	InitSteps     []string `json:"init_steps"`
}

type UpdateProjectRequest struct {
	ID                        string   `json:"id"`
	Name                      string   `json:"name"`
	GitRepoURL                string   `json:"git_repo_url"`
	DefaultBranch             string   `json:"default_branch"`
	InitSteps                 []string `json:"init_steps"`
	HeartbeatEnabled          bool     `json:"heartbeat_enabled"`
	HeartbeatIntervalMinutes  int      `json:"heartbeat_interval_minutes"`
	HeartbeatMDContent        string   `json:"heartbeat_md_content"`
	HeartbeatAgentID          string   `json:"heartbeat_agent_id"`
	DispatchChannelCode       string   `json:"dispatch_channel_code"`
	DispatchSessionKey        string   `json:"dispatch_session_key"`
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	project, err := h.projectService.CreateProject(r.Context(), application.CreateProjectCommand{
		Name:          req.Name,
		GitRepoURL:    req.GitRepoURL,
		DefaultBranch: req.DefaultBranch,
		InitSteps:     req.InitSteps,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(projectToMap(project))
}

func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	project, err := h.projectService.GetProject(r.Context(), domain.NewProjectID(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(projectToMap(project))
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projectService.ListProjects(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(projects))
	for _, project := range projects {
		resp = append(resp, projectToMap(project))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	project, err := h.projectService.UpdateProject(r.Context(), application.UpdateProjectCommand{
		ID:                        domain.NewProjectID(req.ID),
		Name:                      req.Name,
		GitRepoURL:                req.GitRepoURL,
		DefaultBranch:             req.DefaultBranch,
		InitSteps:                 req.InitSteps,
		HeartbeatEnabled:          req.HeartbeatEnabled,
		HeartbeatIntervalMinutes:  req.HeartbeatIntervalMinutes,
		HeartbeatMDContent:        req.HeartbeatMDContent,
		HeartbeatAgentID:          req.HeartbeatAgentID,
		DispatchChannelCode:       req.DispatchChannelCode,
		DispatchSessionKey:        req.DispatchSessionKey,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	// 刷新心跳调度
	if h.heartbeatScheduler != nil {
		if err := h.heartbeatScheduler.RefreshSchedule(context.Background()); err != nil {
			log.Printf("failed to refresh heartbeat schedule: %v", err)
		}
	}
	_ = json.NewEncoder(w).Encode(projectToMap(project))
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.projectService.DeleteProject(r.Context(), domain.NewProjectID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func projectToMap(project *domain.Project) map[string]interface{} {
	return map[string]interface{}{
		"id":                         project.ID().String(),
		"name":                       project.Name(),
		"git_repo_url":               project.GitRepoURL(),
		"default_branch":             project.DefaultBranch(),
		"init_steps":                 project.InitSteps(),
		"heartbeat_enabled":          project.HeartbeatEnabled(),
		"heartbeat_interval_minutes":  project.HeartbeatIntervalMinutes(),
		"heartbeat_md_content":       project.HeartbeatMDContent(),
		"heartbeat_agent_id":         project.HeartbeatAgentID(),
		"dispatch_channel_code":      project.DispatchChannelCode(),
		"dispatch_session_key":       project.DispatchSessionKey(),
		"created_at":                 project.CreatedAt().UnixMilli(),
		"updated_at":                 project.UpdatedAt().UnixMilli(),
	}
}
